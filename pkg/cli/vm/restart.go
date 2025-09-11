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

var restartCmd = &cobra.Command{
	Use:   "restart <name>",
	Short: "Restart a virtual machine",
	Long: `Restart a virtual machine. Attempts graceful reboot first, falls back to hard reset if needed.
The VM must be powered on to restart.`,
	Args: cobra.ExactArgs(1),
	RunE: runRestart,
}

func runRestart(cmd *cobra.Command, args []string) error {
	vmName := args[0]
	ctx := context.Background()
	
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		fmt.Printf("[DRY-RUN] Would restart VM '%s'\n", vmName)
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
		return fmt.Errorf("VM '%s' must be powered on to restart (current state: %s)", vmName, powerState)
	}
	
	// Restart the VM
	start := time.Now()
	fmt.Printf("Restarting VM '%s'...\n", vmName)
	
	err = vmOps.Restart(ctx, vmObj)
	if err != nil {
		return fmt.Errorf("failed to restart VM: %w", err)
	}
	
	duration := time.Since(start)
	fmt.Printf("✅ VM '%s' restarted successfully in %v\n", vmName, duration)
	
	return nil
}