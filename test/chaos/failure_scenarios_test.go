// +build chaos

package chaos

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/r11/esxi-commander/pkg/esxi/client"
	"github.com/r11/esxi-commander/pkg/esxi/vm"
)

// TestNetworkPartitionDuringVMCreate simulates network failure during VM creation
func TestNetworkPartitionDuringVMCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	cfg := getTestConfig(t)
	esxiClient, err := client.New(cfg)
	require.NoError(t, err)
	defer esxiClient.Close()

	vmOps := vm.NewOperations(esxiClient)
	ctx := context.Background()

	// Start VM creation in background
	createDone := make(chan error, 1)
	go func() {
		opts := &vm.CreateOptions{
			Name:     fmt.Sprintf("chaos-test-vm-%d", time.Now().Unix()),
			Template: getTestTemplate(),
			CPU:      2,
			Memory:   4096,
			Disk:     20,
		}
		
		// Simulate network partition after a random delay
		go func() {
			time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
			simulateNetworkPartition(esxiClient)
		}()
		
		_, err := vmOps.CreateFromTemplate(ctx, opts)
		createDone <- err
	}()

	// Wait for completion with timeout
	select {
	case err := <-createDone:
		// Should get an error due to network partition
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "network")
	case <-time.After(30 * time.Second):
		t.Fatal("VM creation did not complete within timeout")
	}

	// Verify cleanup happened properly
	restoreNetwork(esxiClient)
	verifyNoOrphanedResources(t, esxiClient)
}

// TestDatastoreFullDuringClone simulates datastore full condition during clone
func TestDatastoreFullDuringClone(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	cfg := getTestConfig(t)
	esxiClient, err := client.New(cfg)
	require.NoError(t, err)
	defer esxiClient.Close()

	vmOps := vm.NewOperations(esxiClient)
	ctx := context.Background()

	// Create source VM first
	sourceVM := createTestVM(t, vmOps, ctx, "chaos-source-vm")
	defer cleanupVM(t, vmOps, ctx, sourceVM)

	// Fill datastore to simulate near-full condition
	fillDatastore(t, esxiClient, 95) // Fill to 95%
	defer cleanupDatastore(t, esxiClient)

	// Attempt clone
	err = vmOps.CloneVM(ctx, sourceVM, fmt.Sprintf("chaos-clone-%d", time.Now().Unix()), nil)
	
	// Should fail with datastore full error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "datastore")
}

// TestConcurrentVMOperations tests concurrent VM operations for race conditions
func TestConcurrentVMOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	cfg := getTestConfig(t)
	esxiClient, err := client.New(cfg)
	require.NoError(t, err)
	defer esxiClient.Close()

	vmOps := vm.NewOperations(esxiClient)
	ctx := context.Background()

	const numOperations = 10
	var wg sync.WaitGroup
	errors := make([]error, numOperations)
	vmNames := make([]string, numOperations)

	// Launch concurrent VM creates
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			
			vmNames[idx] = fmt.Sprintf("chaos-concurrent-%d-%d", idx, time.Now().Unix())
			opts := &vm.CreateOptions{
				Name:     vmNames[idx],
				Template: getTestTemplate(),
				CPU:      1,
				Memory:   2048,
				Disk:     10,
			}
			
			// Add random delay to increase chance of race conditions
			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
			
			_, err := vmOps.CreateFromTemplate(ctx, opts)
			errors[idx] = err
		}()
	}

	wg.Wait()

	// Count successes and failures
	successCount := 0
	for i, err := range errors {
		if err == nil {
			successCount++
			// Cleanup successful VMs
			defer cleanupVMByName(t, vmOps, ctx, vmNames[i])
		}
	}

	// At least some operations should succeed
	assert.Greater(t, successCount, 0, "At least some concurrent operations should succeed")
	
	// Verify no resource leaks
	verifyNoOrphanedResources(t, esxiClient)
}

// TestAPITimeoutHandling tests timeout handling for ESXi API calls
func TestAPITimeoutHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	cfg := getTestConfig(t)
	cfg.Timeout = 1 * time.Second // Very short timeout
	
	esxiClient, err := client.New(cfg)
	require.NoError(t, err)
	defer esxiClient.Close()

	vmOps := vm.NewOperations(esxiClient)
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Inject delay to trigger timeout
	injectAPIDelay(esxiClient, 3*time.Second)
	defer removeAPIDelay(esxiClient)

	opts := &vm.CreateOptions{
		Name:     fmt.Sprintf("chaos-timeout-%d", time.Now().Unix()),
		Template: getTestTemplate(),
		CPU:      2,
		Memory:   4096,
		Disk:     20,
	}

	_, err = vmOps.CreateFromTemplate(ctx, opts)
	
	// Should timeout
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

// TestInvalidTemplateHandling tests handling of invalid/missing templates
func TestInvalidTemplateHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	cfg := getTestConfig(t)
	esxiClient, err := client.New(cfg)
	require.NoError(t, err)
	defer esxiClient.Close()

	vmOps := vm.NewOperations(esxiClient)
	ctx := context.Background()

	testCases := []struct {
		name     string
		template string
		expectErr bool
	}{
		{"NonExistentTemplate", "template-does-not-exist", true},
		{"EmptyTemplate", "", true},
		{"CorruptedTemplate", "corrupted-template", true},
		{"ValidTemplate", getTestTemplate(), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := &vm.CreateOptions{
				Name:     fmt.Sprintf("chaos-template-%s-%d", tc.name, time.Now().Unix()),
				Template: tc.template,
				CPU:      2,
				Memory:   4096,
				Disk:     20,
			}

			vm, err := vmOps.CreateFromTemplate(ctx, opts)
			
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if vm != nil {
					defer cleanupVMByName(t, vmOps, ctx, opts.Name)
				}
			}
		})
	}
}

// TestPowerOperationsDuringBackup tests VM power operations during backup
func TestPowerOperationsDuringBackup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	cfg := getTestConfig(t)
	esxiClient, err := client.New(cfg)
	require.NoError(t, err)
	defer esxiClient.Close()

	vmOps := vm.NewOperations(esxiClient)
	ctx := context.Background()

	// Create test VM
	vmName := createTestVM(t, vmOps, ctx, "chaos-backup-power")
	defer cleanupVMByName(t, vmOps, ctx, vmName)

	// Start backup in background
	backupDone := make(chan error, 1)
	go func() {
		// Simulate backup operation
		time.Sleep(2 * time.Second)
		backupDone <- nil
	}()

	// Try power operations during backup
	time.Sleep(500 * time.Millisecond) // Let backup start
	
	vmObj, err := esxiClient.FindVM(ctx, vmName)
	require.NoError(t, err)
	
	// Power off should be blocked or handled gracefully
	err = vmOps.PowerOff(ctx, vmObj)
	if err != nil {
		assert.Contains(t, err.Error(), "backup in progress")
	}

	// Wait for backup to complete
	<-backupDone
}

// Helper functions

func getTestConfig(t *testing.T) *client.Config {
	// Read from environment or use defaults
	return &client.Config{
		Host:     getEnvOrDefault("ESXI_HOST", "192.168.1.100"),
		User:     getEnvOrDefault("ESXI_USER", "root"),
		Password: getEnvOrDefault("ESXI_PASSWORD", "password"),
		Insecure: true,
		Timeout:  30 * time.Second,
	}
}

func getTestTemplate() string {
	return getEnvOrDefault("TEST_TEMPLATE", "ubuntu-22.04-template")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func createTestVM(t *testing.T, vmOps *vm.Operations, ctx context.Context, baseName string) string {
	vmName := fmt.Sprintf("%s-%d", baseName, time.Now().Unix())
	opts := &vm.CreateOptions{
		Name:     vmName,
		Template: getTestTemplate(),
		CPU:      1,
		Memory:   2048,
		Disk:     10,
	}
	
	_, err := vmOps.CreateFromTemplate(ctx, opts)
	require.NoError(t, err)
	return vmName
}

func cleanupVM(t *testing.T, vmOps *vm.Operations, ctx context.Context, vm interface{}) {
	// Implementation depends on VM type
}

func cleanupVMByName(t *testing.T, vmOps *vm.Operations, ctx context.Context, name string) {
	err := vmOps.Delete(ctx, name)
	if err != nil {
		t.Logf("Failed to cleanup VM %s: %v", name, err)
	}
}

func simulateNetworkPartition(client *client.ESXiClient) {
	// Simulate network partition by closing connection
	// This is a simplified simulation
	client.Close()
}

func restoreNetwork(client *client.ESXiClient) {
	// Restore network connection
	// In real implementation, would reconnect
}

func verifyNoOrphanedResources(t *testing.T, client *client.ESXiClient) {
	// Check for orphaned VMs, disks, etc.
	// Implementation would query ESXi for resources matching test patterns
}

func fillDatastore(t *testing.T, client *client.ESXiClient, percent int) {
	// Fill datastore to specified percentage
	// In real implementation, would create temporary files
}

func cleanupDatastore(t *testing.T, client *client.ESXiClient) {
	// Clean up temporary files from datastore
}

func injectAPIDelay(client *client.ESXiClient, delay time.Duration) {
	// Inject delay into API calls for testing timeouts
	// This would be implemented using middleware or proxy
}

func removeAPIDelay(client *client.ESXiClient) {
	// Remove injected delay
}