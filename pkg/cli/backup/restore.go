package backup

import (
	"context"
	"fmt"

	"github.com/r11/esxi-commander/pkg/backup"
	"github.com/r11/esxi-commander/pkg/cloudinit"
	"github.com/r11/esxi-commander/pkg/config"
	"github.com/r11/esxi-commander/pkg/esxi/client"
	"github.com/spf13/cobra"
)

var restoreFlags struct {
	asNew    string
	powerOn  bool
	ip       string
	gateway  string
	dns      []string
	sshKey   string
}

// NewRestoreCommand creates the backup restore command
func NewRestoreCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <backup-id>",
		Short: "Restore a virtual machine from backup",
		Long: `Restore a virtual machine from a backup.

This command restores a VM from a backup, optionally with a new name
and network configuration for re-IP.`,
		Args: cobra.ExactArgs(1),
		RunE: runRestore,
	}

	cmd.Flags().StringVar(&restoreFlags.asNew, "as-new", "", "Restore with a new VM name (required)")
	cmd.Flags().BoolVar(&restoreFlags.powerOn, "power-on", true, "Power on VM after restore")
	cmd.Flags().StringVar(&restoreFlags.ip, "ip", "", "IP address for restored VM (CIDR notation)")
	cmd.Flags().StringVar(&restoreFlags.gateway, "gateway", "", "Gateway for restored VM")
	cmd.Flags().StringSliceVar(&restoreFlags.dns, "dns", []string{}, "DNS servers for restored VM")
	cmd.Flags().StringVar(&restoreFlags.sshKey, "ssh-key", "", "SSH public key for restored VM")

	cmd.MarkFlagRequired("as-new")

	return cmd
}

func runRestore(cmd *cobra.Command, args []string) error {
	backupID := args[0]

	if restoreFlags.asNew == "" {
		return fmt.Errorf("--as-new flag is required")
	}

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

	// Create backup manager
	backupManager, err := backup.NewBackupManager(esxiClient, catalogPath)
	if err != nil {
		return fmt.Errorf("failed to create backup manager: %w", err)
	}
	defer backupManager.Close()

	// Build restore options
	opts := backup.RestoreOptions{
		BackupID: backupID,
		NewName:  restoreFlags.asNew,
		PowerOn:  restoreFlags.powerOn,
	}

	// Build guestinfo for re-IP if network options provided
	if restoreFlags.ip != "" {
		data := &cloudinit.CloudInitData{
			Hostname: restoreFlags.asNew,
			IP:       restoreFlags.ip,
			Gateway:  restoreFlags.gateway,
			DNS:      restoreFlags.dns,
		}

		if restoreFlags.sshKey != "" {
			data.SSHKeys = []string{restoreFlags.sshKey}
		}

		guestinfo, err := cloudinit.BuildGuestinfo(data)
		if err != nil {
			return fmt.Errorf("failed to build cloud-init data: %w", err)
		}

		opts.Guestinfo = guestinfo
	}

	// Restore the backup
	fmt.Printf("Restoring backup '%s' as VM '%s'...\n", backupID, restoreFlags.asNew)
	if restoreFlags.ip != "" {
		fmt.Printf("Configuring network: IP=%s, Gateway=%s\n", restoreFlags.ip, restoreFlags.gateway)
	}

	if err := backupManager.RestoreBackup(ctx, opts); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	// Display result
	fmt.Printf("\nBackup restored successfully:\n")
	fmt.Printf("  Backup ID: %s\n", backupID)
	fmt.Printf("  VM Name:   %s\n", restoreFlags.asNew)
	if restoreFlags.powerOn {
		fmt.Printf("  Status:    Powered On\n")
	} else {
		fmt.Printf("  Status:    Powered Off\n")
	}
	if restoreFlags.ip != "" {
		fmt.Printf("  IP:        %s\n", restoreFlags.ip)
	}

	return nil
}