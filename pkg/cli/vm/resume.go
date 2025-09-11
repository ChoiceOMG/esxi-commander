package vm

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/r11/esxi-commander/pkg/esxi/client"
	"github.com/r11/esxi-commander/pkg/esxi/vm"
)

var resumeCmd = &cobra.Command{
	Use:   "resume <name>",
	Short: "Resume a suspended virtual machine",
	Long: `Resume a suspended virtual machine from its saved state.
The VM must be in suspended state to resume.`,
	Args: cobra.ExactArgs(1),
	RunE: runResume,
}

func runResume(cmd *cobra.Command, args []string) error {
	vmName := args[0]
	ctx := context.Background()
	
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		fmt.Printf("[DRY-RUN] Would resume VM '%s'\n", vmName)
		return nil
	}
	
	esxiCfg := &client.Config{
		Host:     viper.GetString("esxi.host"),
		User:     viper.GetString("esxi.user"),
		Password: os.Getenv("ESXI_PASSWORD"),
		Insecure: viper.GetBool("esxi.insecure"),
		Timeout:  30 * time.Second,
	}
	
	if esxiCfg.Password == "" {
		esxiCfg.Password = viper.GetString("esxi.password")
	}
	
	esxi, err := client.NewClient(esxiCfg)
	if err != nil {
		return fmt.Errorf("failed to connect to ESXi: %w", err)
	}
	defer esxi.Close()
	
	// Find the VM
	vmObj, err := esxi.FindVM(ctx, vmName)
	if err != nil {
		return fmt.Errorf("VM '%s' not found: %w", vmName, err)
	}
	
	// Check current power state
	vmOps := vm.NewOperations(esxi)
	powerState, err := vmOps.GetPowerState(ctx, vmObj)
	if err != nil {
		return fmt.Errorf("failed to get power state: %w", err)
	}
	
	if powerState != "suspended" {
		return fmt.Errorf("VM '%s' must be suspended to resume (current state: %s)", vmName, powerState)
	}
	
	// Resume the VM
	start := time.Now()
	fmt.Printf("Resuming VM '%s'...\n", vmName)
	
	err = vmOps.Resume(ctx, vmObj)
	if err != nil {
		return fmt.Errorf("failed to resume VM: %w", err)
	}
	
	duration := time.Since(start)
	fmt.Printf("✅ VM '%s' resumed successfully in %v\n", vmName, duration)
	
	return nil
}