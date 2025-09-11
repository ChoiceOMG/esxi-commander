package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/r11/esxi-commander/pkg/backup"
	"github.com/r11/esxi-commander/pkg/config"
	"github.com/r11/esxi-commander/pkg/esxi/client"
	"github.com/spf13/cobra"
)

var createFlags struct {
	compress    bool
	powerOff    bool
	hot         bool
	target      string
	description string
}

// NewCreateCommand creates the backup create command
func NewCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <vm-name>",
		Short: "Create a backup of a virtual machine",
		Long: `Create a backup of a virtual machine.

This command creates a cold backup of the specified VM by exporting it
to OVF/OVA format and storing it in the configured backup target.`,
		Args: cobra.ExactArgs(1),
		RunE: runCreate,
	}

	cmd.Flags().BoolVar(&createFlags.compress, "compress", true, "Compress the backup")
	cmd.Flags().BoolVar(&createFlags.powerOff, "power-off", false, "Power off VM before backup (cold backup)")
	cmd.Flags().BoolVar(&createFlags.hot, "hot", false, "Create hot backup using snapshots (VM stays running)")
	cmd.Flags().StringVar(&createFlags.target, "target", "datastore", "Backup target (datastore, nfs, s3)")
	cmd.Flags().StringVar(&createFlags.description, "description", "", "Backup description")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	vmName := args[0]

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

	// Context for operations
	ctx := context.Background()

	// Get catalog path from config
	catalogPath := cfg.Backup.CatalogPath
	if catalogPath == "" {
		catalogPath = "/var/lib/ceso/backup.db"
	}

	// Ensure catalog directory exists
	if err := os.MkdirAll(filepath.Dir(catalogPath), 0755); err != nil {
		return fmt.Errorf("failed to create catalog directory: %w", err)
	}

	// Create backup manager
	backupManager, err := backup.NewBackupManager(esxiClient, catalogPath)
	if err != nil {
		return fmt.Errorf("failed to create backup manager: %w", err)
	}
	defer backupManager.Close()

	// Validate flags
	if createFlags.hot && createFlags.powerOff {
		return fmt.Errorf("cannot use both --hot and --power-off flags")
	}

	// Create backup options
	opts := backup.BackupOptions{
		VMName:      vmName,
		PowerOff:    createFlags.powerOff,
		Hot:         createFlags.hot,
		Compress:    createFlags.compress,
		Description: createFlags.description,
	}

	// Create backup target based on flag
	switch createFlags.target {
	case "datastore":
		// Default datastore target
		// opts.Target would be set here
	case "nfs":
		return fmt.Errorf("NFS target not yet implemented")
	case "s3":
		return fmt.Errorf("S3 target not yet implemented")
	default:
		return fmt.Errorf("unknown target: %s", createFlags.target)
	}

	// Create the backup
	fmt.Printf("Creating backup of VM '%s'...\n", vmName)
	if createFlags.hot {
		fmt.Println("Creating hot backup using snapshots (VM will stay running)")
	} else if createFlags.powerOff {
		fmt.Println("VM will be powered off for cold backup")
	} else {
		fmt.Println("Creating backup of VM in current state")
	}

	backupInfo, err := backupManager.CreateBackup(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Display result
	fmt.Printf("\nBackup created successfully:\n")
	fmt.Printf("  ID:       %s\n", backupInfo.ID)
	fmt.Printf("  VM:       %s\n", backupInfo.VMName)
	fmt.Printf("  Type:     %s backup\n", backupInfo.Type)
	fmt.Printf("  Size:     %.2f MB\n", float64(backupInfo.Size)/(1024*1024))
	fmt.Printf("  Created:  %s\n", backupInfo.Created.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Location: %s\n", backupInfo.Location)
	fmt.Printf("  Status:   %s\n", backupInfo.Status)

	return nil
}