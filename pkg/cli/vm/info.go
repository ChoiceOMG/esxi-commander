package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/r11/esxi-commander/pkg/esxi/client"
	"github.com/r11/esxi-commander/pkg/esxi/vm"
)

var infoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show VM information",
	Long:  `Show detailed information about a virtual machine`,
	Args:  cobra.ExactArgs(1),
	RunE:  runInfo,
}

func runInfo(cmd *cobra.Command, args []string) error {
	vmName := args[0]
	ctx := context.Background()

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

	vmOps := vm.NewOperations(esxi)
	vmInfo, err := vmOps.GetVMInfo(ctx, vmName)
	if err != nil {
		return fmt.Errorf("failed to get VM info: %w", err)
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		data, err := json.MarshalIndent(vmInfo, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
	} else {
		fmt.Printf("VM Information:\n")
		fmt.Printf("  Name:   %s\n", vmInfo.Name)
		fmt.Printf("  UUID:   %s\n", vmInfo.UUID)
		fmt.Printf("  Status: %s\n", vmInfo.Status)
		if vmInfo.IP != "" {
			fmt.Printf("  IP:     %s\n", vmInfo.IP)
		}
		fmt.Printf("  CPU:    %d vCPUs\n", vmInfo.CPU)
		fmt.Printf("  Memory: %d GB\n", vmInfo.Memory)
	}

	return nil
}