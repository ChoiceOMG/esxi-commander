package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List virtual machines",
	Long:  `List all virtual machines on the ESXi host`,
	Run:   runList,
}

var jsonOutput bool

func init() {
	listCmd.Flags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
}

func runList(cmd *cobra.Command, args []string) {
	// Mock data for Phase 1
	vms := []VM{
		{Name: "ubuntu-web-01", Status: "running", IP: "192.168.1.100", CPU: 2, RAM: 4},
		{Name: "ubuntu-db-01", Status: "running", IP: "192.168.1.101", CPU: 4, RAM: 8},
		{Name: "ubuntu-test", Status: "stopped", IP: "", CPU: 1, RAM: 2},
	}

	if jsonOutput {
		output, err := json.MarshalIndent(vms, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
			os.Exit(1)
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
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\n", vm.Name, vm.Status, ip, vm.CPU, vm.RAM)
		}
		w.Flush()
	}
}
