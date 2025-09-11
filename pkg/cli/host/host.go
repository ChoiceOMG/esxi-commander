package host

import (
	"github.com/spf13/cobra"
)

var HostCmd = &cobra.Command{
	Use:   "host",
	Short: "ESXi host management and monitoring",
	Long:  `Manage and monitor ESXi hosts.`,
}

func init() {
	HostCmd.AddCommand(NewInfoCommand())
	HostCmd.AddCommand(NewHealthCommand())
	HostCmd.AddCommand(NewStatsCommand())
}