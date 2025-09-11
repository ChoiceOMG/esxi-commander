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

var suspendCmd = &cobra.Command{
	Use:   "suspend <name>",
	Short: "Suspend a virtual machine",
	Long: `Suspend a virtual machine, saving its state to disk.
The VM must be powered on to suspend.`,
	Args: cobra.ExactArgs(1),
	RunE: runSuspend,
}

func runSuspend(cmd *cobra.Command, args []string) error {
	vmName := args[0]
	ctx := context.Background()
	
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		fmt.Printf("[DRY-RUN] Would suspend VM '%s'\n", vmName)
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
	
	if powerState != "poweredOn" {
		return fmt.Errorf("VM '%s' must be powered on to suspend (current state: %s)", vmName, powerState)
	}
	
	// Suspend the VM
	start := time.Now()
	fmt.Printf("Suspending VM '%s'...\n", vmName)
	
	err = vmOps.Suspend(ctx, vmObj)
	if err != nil {
		return fmt.Errorf("failed to suspend VM: %w", err)
	}
	
	duration := time.Since(start)
	fmt.Printf("✅ VM '%s' suspended successfully in %v\n", vmName, duration)
	
	return nil
}