package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/r11/esxi-commander/pkg/backup"
	"github.com/r11/esxi-commander/pkg/config"
	"github.com/r11/esxi-commander/pkg/esxi/client"
	"github.com/spf13/cobra"
)

var listFlags struct {
	vmName string
	json   bool
}

// NewListCommand creates the backup list command
func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List backups",
		Long: `List all backups or backups for a specific VM.

This command shows all backups in the catalog with their details
including size, creation time, and status.`,
		RunE: runList,
	}

	cmd.Flags().StringVar(&listFlags.vmName, "vm", "", "Filter by VM name")
	cmd.Flags().BoolVar(&listFlags.json, "json", false, "Output in JSON format")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create ESXi client
	clientConfig := &client.Config{
		Host:     cfg.ESXi.Host,
		User:     cfg.ESXi.User,
		Password: cfg.ESXi.Password,
		Insecure: cfg.ESXi.Insecure,
	}

	esxiClient, err := client.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create ESXi client: %w", err)
	}
	defer esxiClient.Close()

	// Get catalog path from config
	catalogPath := cfg.Backup.CatalogPath
	if catalogPath == "" {
		catalogPath = "/var/lib/ceso/backup.db"
	}

	// Create backup manager
	backupManager, err := backup.NewBackupManager(esxiClient, catalogPath)
	if err != nil {
		return fmt.Errorf("failed to create backup manager: %w", err)
	}
	defer backupManager.Close()

	// List backups
	backups, err := backupManager.ListBackups(listFlags.vmName)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	// Output results
	if listFlags.json {
		return outputJSON(backups)
	}

	return outputTable(backups)
}

func outputJSON(backups []*backup.BackupInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(backups)
}

func outputTable(backups []*backup.BackupInfo) error {
	if len(backups) == 0 {
		if listFlags.vmName != "" {
			fmt.Printf("No backups found for VM '%s'\n", listFlags.vmName)
		} else {
			fmt.Println("No backups found")
		}
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "ID\tVM NAME\tSIZE\tCREATED\tSTATUS\tTYPE\tDESCRIPTION")
	fmt.Fprintln(w, "---\t---\t---\t---\t---\t---\t---")

	// Data rows
	for _, b := range backups {
		sizeStr := formatSize(b.Size)
		timeStr := formatTime(b.Created)
		desc := b.Description
		if desc == "" {
			desc = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			b.ID,
			b.VMName,
			sizeStr,
			timeStr,
			b.Status,
			b.Type,
			desc,
		)
	}

	// Summary
	fmt.Printf("\nTotal backups: %d\n", len(backups))

	// Calculate total size
	var totalSize int64
	for _, b := range backups {
		totalSize += b.Size
	}
	fmt.Printf("Total size: %s\n", formatSize(totalSize))

	return nil
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatTime(t time.Time) string {
	// If today, show time only
	now := time.Now()
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04:05")
	}
	
	// If this year, show month and day
	if t.Year() == now.Year() {
		return t.Format("Jan 02 15:04")
	}
	
	// Otherwise show full date
	return t.Format("2006-01-02 15:04")
}