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

var startCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start (power on) a virtual machine",
	Args:  cobra.ExactArgs(1),
	RunE:  runStart,
}

var startAllFlag bool

func init() {
	startCmd.Flags().BoolVar(&startAllFlag, "all", false, "Start all powered-off VMs")
}

func runStart(cmd *cobra.Command, args []string) error {
	vmName := args[0]
	ctx := context.Background()
	
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		fmt.Printf("[DRY-RUN] Would start VM '%s'\n", vmName)
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
	
	if startAllFlag {
		return startAllVMs(ctx, esxi)
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
	
	if powerState == "poweredOn" {
		fmt.Printf("VM '%s' is already powered on\n", vmName)
		return nil
	}
	
	// Start the VM
	start := time.Now()
	err = vmOps.PowerOn(ctx, vmObj)
	if err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}
	
	duration := time.Since(start)
	fmt.Printf("✅ VM '%s' started successfully in %v\n", vmName, duration)
	
	return nil
}

func startAllVMs(ctx context.Context, esxi *client.ESXiClient) error {
	vms, err := esxi.ListVMs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list VMs: %w", err)
	}
	
	vmOps := vm.NewOperations(esxi)
	started := 0
	
	for _, vmInfo := range vms {
		if vmInfo.Status == "poweredOff" {
			vmObj, err := esxi.FindVM(ctx, vmInfo.Name)
			if err != nil {
				fmt.Printf("⚠️  Failed to find VM '%s': %v\n", vmInfo.Name, err)
				continue
			}
			
			err = vmOps.PowerOn(ctx, vmObj)
			if err != nil {
				fmt.Printf("⚠️  Failed to start VM '%s': %v\n", vmInfo.Name, err)
				continue
			}
			
			fmt.Printf("✅ Started VM '%s'\n", vmInfo.Name)
			started++
		}
	}
	
	fmt.Printf("Started %d VMs\n", started)
	return nil
}