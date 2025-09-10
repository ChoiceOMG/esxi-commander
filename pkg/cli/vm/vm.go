package vm

import (
	"github.com/spf13/cobra"
)

var VmCmd = &cobra.Command{
	Use:   "vm",
	Short: "Manage virtual machines",
	Long:  `Manage virtual machines on ESXi hosts`,
}

func init() {
	VmCmd.AddCommand(listCmd)
	VmCmd.AddCommand(createCmd)
}
