package backup

import (
	"context"
	"fmt"

	"github.com/r11/esxi-commander/pkg/backup"
	"github.com/r11/esxi-commander/pkg/config"
	"github.com/r11/esxi-commander/pkg/esxi/client"
	"github.com/spf13/cobra"
)

var BackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "VM backup operations",
	Long: `Manage virtual machine backups.

The backup command provides functionality to create, restore, and manage
VM backups. Backups are stored as OVF/OVA files and tracked in a local
BoltDB catalog.

Features:
  - Cold backups with optional VM power-off
  - Compression support (gzip, zstd)
  - Multiple storage targets (datastore, NFS, S3)
  - Backup catalog with metadata tracking
  - Restore with re-IP capability using cloud-init
  - Retention policy management`,
}

func init() {
	// Add subcommands
	BackupCmd.AddCommand(
		NewCreateCommand(),
		NewRestoreCommand(),
		NewListCommand(),
		NewDeleteCommand(),
		NewVerifyCommand(),
		NewPruneCommand(),
	)
}

// NewDeleteCommand creates the backup delete command
func NewDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <backup-id>",
		Short: "Delete a backup",
		Long:  `Delete a backup from storage and remove it from the catalog.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runDelete,
	}

	return cmd
}

// NewVerifyCommand creates the backup verify command
func NewVerifyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify <backup-id>",
		Short: "Verify backup integrity",
		Long:  `Verify the integrity of a backup by checking its checksum.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runVerify,
	}

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	backupID := args[0]

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

	// Delete the backup
	ctx := context.Background()
	fmt.Printf("Deleting backup '%s'...\n", backupID)
	
	if err := backupManager.DeleteBackup(ctx, backupID); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	fmt.Printf("Backup '%s' deleted successfully\n", backupID)
	return nil
}

func runVerify(cmd *cobra.Command, args []string) error {
	backupID := args[0]

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

	// Verify the backup
	ctx := context.Background()
	fmt.Printf("Verifying backup '%s'...\n", backupID)
	
	if err := backupManager.VerifyBackup(ctx, backupID); err != nil {
		return fmt.Errorf("backup verification failed: %w", err)
	}

	fmt.Printf("Backup '%s' verified successfully\n", backupID)
	return nil
}

// NewPruneCommand creates the backup prune command
func NewPruneCommand() *cobra.Command {
	var (
		keepLast  int
		vmName    string
		dryRun    bool
		keepDays  int
	)

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Prune old backups based on retention policy",
		Long: `Prune old backups based on retention policy.
		
This command removes old backups to free up storage space while
keeping the specified number of recent backups or backups within
a time window.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrune(cmd, keepLast, vmName, dryRun, keepDays)
		},
	}

	cmd.Flags().IntVar(&keepLast, "keep-last", 5, "Keep last N backups per VM")
	cmd.Flags().StringVar(&vmName, "vm", "", "Prune backups for specific VM (default: all VMs)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be deleted without deleting")
	cmd.Flags().IntVar(&keepDays, "keep-days", 30, "Keep backups newer than N days")

	return cmd
}

func runPrune(cmd *cobra.Command, keepLast int, vmName string, dryRun bool, keepDays int) error {
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

	// Run pruning
	ctx := context.Background()
	prunedCount, err := backupManager.PruneBackups(ctx, backup.PruneOptions{
		VMName:   vmName,
		KeepLast: keepLast,
		KeepDays: keepDays,
		DryRun:   dryRun,
	})
	if err != nil {
		return fmt.Errorf("failed to prune backups: %w", err)
	}

	if dryRun {
		fmt.Printf("Would delete %d backups\n", prunedCount)
	} else {
		fmt.Printf("Deleted %d backups\n", prunedCount)
	}

	return nil
}
