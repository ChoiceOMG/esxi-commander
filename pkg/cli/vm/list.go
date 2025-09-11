package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/r11/esxi-commander/pkg/esxi/client"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List virtual machines",
	Long:  `List all virtual machines on the ESXi host`,
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
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

	vms, err := esxi.ListVMs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list VMs: %w", err)
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		output, err := json.MarshalIndent(vms, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATUS\tIP\tCPU\tRAM(GB)")
		for _, vm := range vms {
			ip := vm.IP
			if ip == "" {
				ip = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\n", vm.Name, vm.Status, ip, vm.CPU, vm.Memory)
		}
		w.Flush()
	}

	return nil
}
