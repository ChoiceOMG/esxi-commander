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
	"github.com/r11/esxi-commander/pkg/interactive"
	"github.com/r11/esxi-commander/pkg/validation"
)

var (
	force bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a VM",
	Long:  `Delete a virtual machine with safety checks`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

func init() {
	deleteCmd.Flags().BoolVar(&force, "force", false, "Force delete without confirmation")
}

func runDelete(cmd *cobra.Command, args []string) error {
	vmName := args[0]
	ctx := context.Background()

	// Validate VM name
	if err := validation.ValidateVMName(vmName); err != nil {
		return fmt.Errorf("invalid VM name: %w", err)
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		fmt.Printf("[DRY-RUN] Would delete VM '%s'\n", vmName)
		return nil
	}

	if !force {
		if !interactive.ConfirmDeletion("VM", vmName) {
			fmt.Println("Delete cancelled")
			return nil
		}
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

	start := time.Now()

	vmOps := vm.NewOperations(esxi)
	if err := vmOps.Delete(ctx, vmName); err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}

	duration := time.Since(start)

	fmt.Printf("✅ VM '%s' deleted successfully in %v\n", vmName, duration)

	return nil
}