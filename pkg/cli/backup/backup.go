package backup

import (
	"fmt"

	"github.com/spf13/cobra"
)

var BackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage backups",
	Long:  `Manage VM backups and restore operations`,
	Run:   runBackup,
}

func runBackup(cmd *cobra.Command, args []string) {
	fmt.Println("Backup command - stub implementation for Phase 1")
}
