//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type VM struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	IP        string    `json:"ip"`
	CPU       int       `json:"cpu"`
	Memory    int       `json:"memory"`
	Disk      int       `json:"disk"`
	Template  string    `json:"template"`
	Created   time.Time `json:"created"`
	UUID      string    `json:"uuid"`
	MACAddr   string    `json:"mac_address"`
}

var (
	testConfig struct {
		ESXiHost     string
		ESXiUser     string
		ESXiPassword string
		Template     string
		NetworkCIDR  string
		Gateway      string
		DNS          string
		SSHKeyPath   string
		TestIP1      string
		TestIP2      string
		TestIP3      string
	}
)

func init() {
	// Load test environment
	envFile := filepath.Join("..", ".env")
	if err := godotenv.Load(envFile); err != nil {
		// Try alternative path
		if err := godotenv.Load("test/.env"); err != nil {
			fmt.Printf("Warning: Could not load .env file: %v\n", err)
		}
	}

	testConfig.ESXiHost = os.Getenv("ESXI_HOST")
	testConfig.ESXiUser = os.Getenv("ESXI_USER")
	testConfig.ESXiPassword = os.Getenv("ESXI_PASSWORD")
	testConfig.Template = os.Getenv("ESXI_TEMPLATE")
	testConfig.NetworkCIDR = os.Getenv("TEST_NETWORK_CIDR")
	testConfig.Gateway = os.Getenv("TEST_GATEWAY")
	testConfig.DNS = os.Getenv("TEST_DNS")
	testConfig.SSHKeyPath = os.Getenv("TEST_SSH_KEY_PATH")
	testConfig.TestIP1 = os.Getenv("TEST_VM_IP1")
	testConfig.TestIP2 = os.Getenv("TEST_VM_IP2")
	testConfig.TestIP3 = os.Getenv("TEST_VM_IP3")
}

func skipIfNoESXi(t *testing.T) {
	if testConfig.ESXiHost == "" {
		t.Skip("Skipping test: ESXI_HOST not configured")
	}
}

func getTestSSHKey() string {
	if testConfig.SSHKeyPath != "" {
		data, err := ioutil.ReadFile(testConfig.SSHKeyPath)
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	return "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample test@integration"
}

func TestVMLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	skipIfNoESXi(t)

	vmName := fmt.Sprintf("test-vm-%d", time.Now().Unix())
	sshKey := getTestSSHKey()

	// Create VM with cloud-init
	cmd := exec.Command("./ceso", "vm", "create", vmName,
		"--template", testConfig.Template,
		"--ip", testConfig.TestIP1,
		"--gateway", testConfig.Gateway,
		"--dns", testConfig.DNS,
		"--ssh-key", sshKey,
		"--cpu", "2",
		"--memory", "4096",
		"--disk", "40")

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "VM creation failed: %s", output)
	t.Logf("VM created: %s", vmName)

	// Ensure cleanup
	defer func() {
		cmd = exec.Command("./ceso", "vm", "delete", vmName, "--force")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Logf("Warning: Failed to delete VM %s: %s", vmName, output)
		}
	}()

	// Verify VM appears in list
	cmd = exec.Command("./ceso", "vm", "list", "--json")
	output, err = cmd.Output()
	require.NoError(t, err, "Failed to list VMs")

	var vms []VM
	err = json.Unmarshal(output, &vms)
	require.NoError(t, err, "Failed to parse VM list")

	found := false
	for _, vm := range vms {
		if vm.Name == vmName {
			found = true
			assert.Equal(t, "poweredOn", vm.Status, "VM should be powered on")
			assert.NotEmpty(t, vm.UUID, "VM should have UUID")
			break
		}
	}
	require.True(t, found, "VM %s not found in list", vmName)

	// Get detailed VM info
	cmd = exec.Command("./ceso", "vm", "info", vmName, "--json")
	output, err = cmd.Output()
	require.NoError(t, err, "Failed to get VM info")

	var vmInfo VM
	err = json.Unmarshal(output, &vmInfo)
	require.NoError(t, err, "Failed to parse VM info")
	
	// Verify VM properties
	assert.Equal(t, vmName, vmInfo.Name)
	assert.Equal(t, 2, vmInfo.CPU)
	assert.Equal(t, 4096, vmInfo.Memory)
	assert.Equal(t, 40, vmInfo.Disk)
	assert.Equal(t, testConfig.Template, vmInfo.Template)
	assert.NotEmpty(t, vmInfo.MACAddr, "VM should have MAC address")

	// Wait for IP assignment (cloud-init)
	time.Sleep(30 * time.Second)

	// Verify IP was assigned
	cmd = exec.Command("./ceso", "vm", "info", vmName, "--json")
	output, err = cmd.Output()
	require.NoError(t, err)

	err = json.Unmarshal(output, &vmInfo)
	require.NoError(t, err)
	if vmInfo.IP != "" {
		t.Logf("VM IP assigned: %s", vmInfo.IP)
		assert.Contains(t, vmInfo.IP, "10.0.1.24", "IP should be in test range")
	}

	// Test power operations
	t.Run("PowerCycle", func(t *testing.T) {
		// Power off
		cmd = exec.Command("./ceso", "vm", "power", "off", vmName)
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Power off failed: %s", output)
		
		time.Sleep(5 * time.Second)
		
		// Verify powered off
		cmd = exec.Command("./ceso", "vm", "info", vmName, "--json")
		output, err = cmd.Output()
		require.NoError(t, err)
		
		err = json.Unmarshal(output, &vmInfo)
		require.NoError(t, err)
		assert.Equal(t, "poweredOff", vmInfo.Status)
		
		// Power on
		cmd = exec.Command("./ceso", "vm", "power", "on", vmName)
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Power on failed: %s", output)
		
		time.Sleep(5 * time.Second)
		
		// Verify powered on
		cmd = exec.Command("./ceso", "vm", "info", vmName, "--json")
		output, err = cmd.Output()
		require.NoError(t, err)
		
		err = json.Unmarshal(output, &vmInfo)
		require.NoError(t, err)
		assert.Equal(t, "poweredOn", vmInfo.Status)
	})
}

func TestVMClone(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	skipIfNoESXi(t)

	sourceName := fmt.Sprintf("test-source-%d", time.Now().Unix())
	cloneName := fmt.Sprintf("test-clone-%d", time.Now().Unix())
	sshKey := getTestSSHKey()

	// Create source VM
	cmd := exec.Command("./ceso", "vm", "create", sourceName,
		"--template", testConfig.Template,
		"--ip", testConfig.TestIP2,
		"--gateway", testConfig.Gateway,
		"--dns", testConfig.DNS,
		"--ssh-key", sshKey,
		"--cpu", "1",
		"--memory", "2048")

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Source VM creation failed: %s", output)
	t.Logf("Source VM created: %s", sourceName)

	// Ensure cleanup
	defer func() {
		cmd = exec.Command("./ceso", "vm", "delete", sourceName, "--force")
		cmd.Run()
		cmd = exec.Command("./ceso", "vm", "delete", cloneName, "--force")
		cmd.Run()
	}()

	// Wait for source VM to be ready
	time.Sleep(10 * time.Second)

	// Clone VM with new IP
	cmd = exec.Command("./ceso", "vm", "clone", sourceName, cloneName,
		"--ip", testConfig.TestIP3,
		"--gateway", testConfig.Gateway,
		"--dns", testConfig.DNS)

	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "VM clone failed: %s", output)
	t.Logf("Clone VM created: %s", cloneName)

	// Verify clone properties
	cmd = exec.Command("./ceso", "vm", "info", cloneName, "--json")
	output, err = cmd.Output()
	require.NoError(t, err, "Failed to get clone info")

	var cloneInfo VM
	err = json.Unmarshal(output, &cloneInfo)
	require.NoError(t, err)
	assert.Equal(t, cloneName, cloneInfo.Name)
	assert.Equal(t, 1, cloneInfo.CPU, "Clone should have same CPU as source")
	assert.Equal(t, 2048, cloneInfo.Memory, "Clone should have same memory as source")

	// Verify source VM info for comparison
	cmd = exec.Command("./ceso", "vm", "info", sourceName, "--json")
	output, err = cmd.Output()
	require.NoError(t, err)

	var sourceInfo VM
	err = json.Unmarshal(output, &sourceInfo)
	require.NoError(t, err)

	// Verify clone has different UUID and MAC
	assert.NotEqual(t, sourceInfo.UUID, cloneInfo.UUID, "Clone should have different UUID")
	assert.NotEqual(t, sourceInfo.MACAddr, cloneInfo.MACAddr, "Clone should have different MAC address")
}

func TestVMCreateWithCloudInit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	skipIfNoESXi(t)

	vmName := fmt.Sprintf("test-cloudinit-%d", time.Now().Unix())
	sshKey := getTestSSHKey()

	// Create VM with full cloud-init configuration
	cmd := exec.Command("./ceso", "vm", "create", vmName,
		"--template", testConfig.Template,
		"--ip", testConfig.TestIP1,
		"--gateway", testConfig.Gateway,
		"--dns", testConfig.DNS,
		"--ssh-key", sshKey,
		"--packages", "nginx,htop,curl",
		"--run-cmd", "echo 'Hello from cloud-init' > /tmp/cloudinit.txt",
		"--cpu", "2",
		"--memory", "4096")

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "VM creation with cloud-init failed: %s", output)

	defer func() {
		cmd = exec.Command("./ceso", "vm", "delete", vmName, "--force")
		cmd.Run()
	}()

	// Wait for cloud-init to complete
	time.Sleep(60 * time.Second)

	// Verify VM has the configured IP
	cmd = exec.Command("./ceso", "vm", "info", vmName, "--json")
	output, err = cmd.Output()
	require.NoError(t, err)

	var vmInfo VM
	err = json.Unmarshal(output, &vmInfo)
	require.NoError(t, err)

	if vmInfo.IP != "" {
		t.Logf("VM configured with IP: %s", vmInfo.IP)
		assert.Contains(t, vmInfo.IP, "10.0.1.241", "Should have the configured static IP")
	}
}

func TestVMCreatePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	skipIfNoESXi(t)

	vmName := fmt.Sprintf("test-perf-%d", time.Now().Unix())
	sshKey := getTestSSHKey()

	start := time.Now()

	cmd := exec.Command("./ceso", "vm", "create", vmName,
		"--template", testConfig.Template,
		"--ip", "dhcp",
		"--ssh-key", sshKey,
		"--cpu", "2",
		"--memory", "4096")

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "VM creation failed: %s", output)

	duration := time.Since(start)
	t.Logf("VM creation took: %v", duration)

	defer func() {
		cmd = exec.Command("./ceso", "vm", "delete", vmName, "--force")
		cmd.Run()
	}()

	// Performance requirement: VM creation should complete within 90 seconds
	assert.Less(t, duration, 90*time.Second, "VM creation took %v, exceeding 90s limit", duration)
}

func TestVMDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	skipIfNoESXi(t)

	vmName := fmt.Sprintf("test-delete-%d", time.Now().Unix())
	sshKey := getTestSSHKey()

	// Create a VM to delete
	cmd := exec.Command("./ceso", "vm", "create", vmName,
		"--template", testConfig.Template,
		"--ip", "dhcp",
		"--ssh-key", sshKey)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "VM creation failed: %s", output)

	// Verify VM exists
	cmd = exec.Command("./ceso", "vm", "info", vmName)
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "VM should exist before deletion")

	// Delete the VM
	cmd = exec.Command("./ceso", "vm", "delete", vmName)
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "VM deletion failed: %s", output)

	// Verify VM no longer exists
	cmd = exec.Command("./ceso", "vm", "info", vmName)
	output, err = cmd.CombinedOutput()
	assert.Error(t, err, "VM should not exist after deletion")
	assert.Contains(t, string(output), "not found", "Should report VM not found")
}

func TestDryRun(t *testing.T) {
	skipIfNoESXi(t)

	vmName := fmt.Sprintf("test-dryrun-%d", time.Now().Unix())
	sshKey := getTestSSHKey()

	cmd := exec.Command("./ceso", "vm", "create", vmName,
		"--template", testConfig.Template,
		"--ip", "dhcp",
		"--ssh-key", sshKey,
		"--dry-run")

	output, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(output), "[DRY-RUN]")

	cmd = exec.Command("./ceso", "vm", "list", "--json")
	output, err = cmd.Output()
	require.NoError(t, err)

	var vms []VM
	err = json.Unmarshal(output, &vms)
	require.NoError(t, err)

	for _, vm := range vms {
		assert.NotEqual(t, vmName, vm.Name, "VM should not exist after dry-run")
	}
}

func TestConcurrentVMOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	skipIfNoESXi(t)

	const numVMs = 3
	vmNames := make([]string, numVMs)
	sshKey := getTestSSHKey()

	// Create VMs concurrently
	type result struct {
		name string
		err  error
	}

	results := make(chan result, numVMs)

	for i := 0; i < numVMs; i++ {
		vmNames[i] = fmt.Sprintf("test-concurrent-%d-%d", time.Now().Unix(), i)
		go func(vmName string, index int) {
			cmd := exec.Command("./ceso", "vm", "create", vmName,
				"--template", testConfig.Template,
				"--ip", "dhcp",
				"--ssh-key", sshKey,
				"--cpu", "1",
				"--memory", "1024")

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Logf("VM %s creation failed: %s", vmName, output)
			}
			results <- result{name: vmName, err: err}
		}(vmNames[i], i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numVMs; i++ {
		r := <-results
		if r.err == nil {
			successCount++
			t.Logf("Successfully created VM: %s", r.name)
		}
	}

	// Cleanup
	for _, vmName := range vmNames {
		cmd := exec.Command("./ceso", "vm", "delete", vmName, "--force")
		cmd.Run()
	}

	// At least some VMs should be created successfully
	assert.Greater(t, successCount, 0, "At least one concurrent VM creation should succeed")
}