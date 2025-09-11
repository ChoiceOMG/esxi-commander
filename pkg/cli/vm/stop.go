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

var stopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop (power off) a virtual machine",
	Long: `Stop a virtual machine. By default, attempts graceful shutdown via VMware Tools.
Use --force for immediate power off.`,
	Args: cobra.ExactArgs(1),
	RunE: runStop,
}

var (
	forceStop   bool
	stopAllFlag bool
)

func init() {
	stopCmd.Flags().BoolVar(&forceStop, "force", false, "Force power off instead of graceful shutdown")
	stopCmd.Flags().BoolVar(&stopAllFlag, "all", false, "Stop all powered-on VMs")
}

func runStop(cmd *cobra.Command, args []string) error {
	vmName := args[0]
	ctx := context.Background()
	
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		action := "shutdown"
		if forceStop {
			action = "force power off"
		}
		fmt.Printf("[DRY-RUN] Would %s VM '%s'\n", action, vmName)
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
	
	if stopAllFlag {
		return stopAllVMs(ctx, esxi, forceStop)
	}
	
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
		fmt.Printf("VM '%s' is already powered off\n", vmName)
		return nil
	}
	
	// Stop the VM
	start := time.Now()
	var stopErr error
	
	if forceStop {
		fmt.Printf("Force powering off VM '%s'...\n", vmName)
		stopErr = vmOps.PowerOff(ctx, vmObj)
	} else {
		fmt.Printf("Gracefully shutting down VM '%s'...\n", vmName)
		stopErr = vmOps.Shutdown(ctx, vmObj)
		if stopErr != nil {
			fmt.Printf("Graceful shutdown failed, attempting force power off...\n")
			stopErr = vmOps.PowerOff(ctx, vmObj)
		}
	}
	
	if stopErr != nil {
		return fmt.Errorf("failed to stop VM: %w", stopErr)
	}
	
	duration := time.Since(start)
	fmt.Printf("✅ VM '%s' stopped successfully in %v\n", vmName, duration)
	
	return nil
}

func stopAllVMs(ctx context.Context, esxi *client.ESXiClient, force bool) error {
	vms, err := esxi.ListVMs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list VMs: %w", err)
	}
	
	vmOps := vm.NewOperations(esxi)
	stopped := 0
	
	for _, vmInfo := range vms {
		if vmInfo.Status == "poweredOn" {
			vmObj, err := esxi.FindVM(ctx, vmInfo.Name)
			if err != nil {
				fmt.Printf("⚠️  Failed to find VM '%s': %v\n", vmInfo.Name, err)
				continue
			}
			
			var stopErr error
			if force {
				stopErr = vmOps.PowerOff(ctx, vmObj)
			} else {
				stopErr = vmOps.Shutdown(ctx, vmObj)
				if stopErr != nil {
					stopErr = vmOps.PowerOff(ctx, vmObj)
				}
			}
			
			if stopErr != nil {
				fmt.Printf("⚠️  Failed to stop VM '%s': %v\n", vmInfo.Name, stopErr)
				continue
			}
			
			fmt.Printf("✅ Stopped VM '%s'\n", vmInfo.Name)
			stopped++
		}
	}
	
	fmt.Printf("Stopped %d VMs\n", stopped)
	return nil
}