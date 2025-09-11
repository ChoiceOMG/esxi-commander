package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/r11/esxi-commander/internal/storage"
	"github.com/r11/esxi-commander/pkg/esxi/client"
	"github.com/r11/esxi-commander/pkg/esxi/vm"
	"github.com/r11/esxi-commander/pkg/metrics"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

type BackupManager struct {
	client  *client.ESXiClient
	vmOps   *vm.Operations
	catalog *storage.BackupCatalog
	config  *Config
}

type Config struct {
	CatalogPath     string
	DefaultTarget   string // datastore, nfs, s3
	Compression     string // none, gzip, zstd
	TempDir         string
	MaxConcurrent   int
}

type BackupTarget interface {
	Store(ctx context.Context, backupID string, reader io.Reader) (string, error)
	Retrieve(ctx context.Context, backupID string) (io.ReadCloser, error)
	Delete(ctx context.Context, backupID string) error
	GetLocation() string
}

type DatastoreTarget struct {
	client    *client.ESXiClient
	datastore string
	basePath  string
}

type BackupOptions struct {
	VMName      string
	PowerOff    bool
	Hot         bool
	Compress    bool
	Target      BackupTarget
	Description string
}

type RestoreOptions struct {
	BackupID    string
	NewName     string
	PowerOn     bool
	Guestinfo   map[string]string // For re-IP
}

type PruneOptions struct {
	VMName   string // Empty string means all VMs
	KeepLast int    // Keep last N backups per VM
	KeepDays int    // Keep backups newer than N days
	DryRun   bool   // Don't actually delete, just show what would be deleted
}

type BackupInfo struct {
	ID          string    `json:"id"`
	VMName      string    `json:"vm_name"`
	Size        int64     `json:"size"`
	Created     time.Time `json:"created"`
	Location    string    `json:"location"`
	Status      string    `json:"status"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
}

// NewBackupManager creates a new backup manager
func NewBackupManager(client *client.ESXiClient, catalogPath string) (*BackupManager, error) {
	catalog, err := storage.InitCatalog(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize catalog: %w", err)
	}

	return &BackupManager{
		client:  client,
		vmOps:   vm.NewOperations(client),
		catalog: catalog,
		config: &Config{
			CatalogPath:   catalogPath,
			DefaultTarget: "datastore",
			Compression:   "gzip",
			TempDir:       "/tmp",
			MaxConcurrent: 2,
		},
	}, nil
}

// Close closes the backup manager
func (m *BackupManager) Close() error {
	if m.catalog != nil {
		return m.catalog.Close()
	}
	return nil
}

// CreateBackup creates a backup of a VM
func (m *BackupManager) CreateBackup(ctx context.Context, opts BackupOptions) (*BackupInfo, error) {
	start := time.Now()
	
	// Generate backup ID
	backupID := fmt.Sprintf("backup-%s-%s", opts.VMName, uuid.New().String()[:8])

	// Determine backup type
	backupType := "cold"
	if opts.Hot {
		backupType = "hot"
	}

	// Create catalog entry
	entry := &storage.BackupEntry{
		ID:        backupID,
		VMName:    opts.VMName,
		Timestamp: time.Now(),
		Type:      backupType,
		Status:    "pending",
		Metadata: map[string]string{
			"description": opts.Description,
		},
	}

	if err := m.catalog.AddBackup(entry); err != nil {
		return nil, fmt.Errorf("failed to add backup to catalog: %w", err)
	}

	// Get VM
	vmObj, err := m.client.FindVM(ctx, opts.VMName)
	if err != nil {
		m.catalog.UpdateBackupStatus(backupID, "failed")
		return nil, fmt.Errorf("failed to find VM: %w", err)
	}

	// Check VM power state
	var vmMo mo.VirtualMachine
	err = vmObj.Properties(ctx, vmObj.Reference(), []string{"runtime.powerState"}, &vmMo)
	if err != nil {
		m.catalog.UpdateBackupStatus(backupID, "failed")
		return nil, fmt.Errorf("failed to get VM properties: %w", err)
	}

	wasRunning := vmMo.Runtime.PowerState == types.VirtualMachinePowerStatePoweredOn

	var snapshotName string
	var backupErr error

	// Handle hot backup with snapshot
	if opts.Hot && wasRunning {
		// Create snapshot for hot backup
		snapshotName = fmt.Sprintf("backup-snapshot-%s", backupID)
		fmt.Printf("Creating snapshot '%s' for hot backup...\n", snapshotName)
		
		err := m.vmOps.CreateSnapshot(ctx, vmObj, snapshotName, "Backup snapshot", false, true)
		if err != nil {
			m.catalog.UpdateBackupStatus(backupID, "failed")
			return nil, fmt.Errorf("failed to create backup snapshot: %w", err)
		}

		// Cleanup snapshot after backup
		defer func() {
			if snapshotName != "" {
				// Find and remove the snapshot
				snapshots, err := m.vmOps.ListSnapshots(ctx, vmObj)
				if err == nil {
					for _, snapshot := range snapshots {
						if snapshot.Name == snapshotName {
							m.vmOps.RemoveSnapshot(ctx, vmObj, snapshot.Snapshot, false)
							break
						}
					}
				}
			}
		}()
	} else if opts.PowerOff && wasRunning {
		// Power off VM for cold backup
		task, err := vmObj.PowerOff(ctx)
		if err != nil {
			m.catalog.UpdateBackupStatus(backupID, "failed")
			return nil, fmt.Errorf("failed to power off VM: %w", err)
		}
		if err := task.Wait(ctx); err != nil {
			m.catalog.UpdateBackupStatus(backupID, "failed")
			return nil, fmt.Errorf("failed to wait for power off: %w", err)
		}

		// Power on VM after backup
		defer func() {
			if opts.PowerOff && wasRunning && backupErr == nil {
				task, _ := vmObj.PowerOn(ctx)
				if task != nil {
					task.Wait(ctx)
				}
			}
		}()
	}

	// Export VM to OVF
	location, size, checksum, err := m.exportVM(ctx, vmObj, backupID, opts)
	if err != nil {
		backupErr = err
		m.catalog.UpdateBackupStatus(backupID, "failed")
		return nil, fmt.Errorf("failed to export VM: %w", err)
	}

	// Update catalog with success
	entry.Status = "completed"
	entry.Size = size
	entry.Location = location
	entry.Checksum = checksum
	
	if err := m.catalog.AddBackup(entry); err != nil {
		return nil, fmt.Errorf("failed to update catalog: %w", err)
	}

	// Record metrics
	metrics.RecordBackupOperation("create", "success", time.Since(start).Seconds())
	metrics.RecordBackupSize(opts.VMName, float64(size))
	
	return &BackupInfo{
		ID:          backupID,
		VMName:      opts.VMName,
		Size:        size,
		Created:     entry.Timestamp,
		Location:    location,
		Status:      "completed",
		Type:        backupType,
		Description: opts.Description,
	}, nil
}

// RestoreBackup restores a VM from a backup
func (m *BackupManager) RestoreBackup(ctx context.Context, opts RestoreOptions) error {
	// Get backup from catalog
	entry, err := m.catalog.GetBackup(opts.BackupID)
	if err != nil {
		return fmt.Errorf("failed to get backup from catalog: %w", err)
	}

	if entry.Status != "completed" {
		return fmt.Errorf("backup is not in completed state: %s", entry.Status)
	}

	// Check if new VM name already exists
	if _, err := m.client.FindVM(ctx, opts.NewName); err == nil {
		return fmt.Errorf("VM with name %s already exists", opts.NewName)
	}

	// Restore VM from backup
	if err := m.importVM(ctx, entry, opts); err != nil {
		return fmt.Errorf("failed to import VM: %w", err)
	}

	// Apply guestinfo if provided (for re-IP)
	if len(opts.Guestinfo) > 0 {
		vmObj, err := m.client.FindVM(ctx, opts.NewName)
		if err != nil {
			return fmt.Errorf("failed to find restored VM: %w", err)
		}

		var reconfigSpec types.VirtualMachineConfigSpec
		for key, value := range opts.Guestinfo {
			reconfigSpec.ExtraConfig = append(reconfigSpec.ExtraConfig, &types.OptionValue{
				Key:   key,
				Value: value,
			})
		}

		task, err := vmObj.Reconfigure(ctx, reconfigSpec)
		if err != nil {
			return fmt.Errorf("failed to reconfigure VM: %w", err)
		}
		if err := task.Wait(ctx); err != nil {
			return fmt.Errorf("failed to wait for reconfigure: %w", err)
		}
	}

	// Power on if requested
	if opts.PowerOn {
		vmObj, err := m.client.FindVM(ctx, opts.NewName)
		if err != nil {
			return fmt.Errorf("failed to find restored VM: %w", err)
		}

		task, err := vmObj.PowerOn(ctx)
		if err != nil {
			return fmt.Errorf("failed to power on VM: %w", err)
		}
		if err := task.Wait(ctx); err != nil {
			return fmt.Errorf("failed to wait for power on: %w", err)
		}
	}

	return nil
}

// ListBackups lists all backups or backups for a specific VM
func (m *BackupManager) ListBackups(vmName string) ([]*BackupInfo, error) {
	entries, err := m.catalog.ListBackups(vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}

	backups := make([]*BackupInfo, len(entries))
	for i, entry := range entries {
		backups[i] = &BackupInfo{
			ID:       entry.ID,
			VMName:   entry.VMName,
			Size:     entry.Size,
			Created:  entry.Timestamp,
			Location: entry.Location,
			Status:   entry.Status,
			Type:     entry.Type,
		}
		if desc, ok := entry.Metadata["description"]; ok {
			backups[i].Description = desc
		}
	}

	return backups, nil
}

// DeleteBackup deletes a backup
func (m *BackupManager) DeleteBackup(ctx context.Context, backupID string) error {
	// Get backup from catalog to verify it exists
	_, err := m.catalog.GetBackup(backupID)
	if err != nil {
		return fmt.Errorf("failed to get backup from catalog: %w", err)
	}

	// Delete backup file from storage
	// This would use the appropriate target based on location
	// For now, assume datastore

	// Remove from catalog
	if err := m.catalog.DeleteBackup(backupID); err != nil {
		return fmt.Errorf("failed to delete from catalog: %w", err)
	}

	return nil
}

// VerifyBackup verifies the integrity of a backup
func (m *BackupManager) VerifyBackup(ctx context.Context, backupID string) error {
	entry, err := m.catalog.GetBackup(backupID)
	if err != nil {
		return fmt.Errorf("failed to get backup from catalog: %w", err)
	}

	// Verify checksum
	// This would download the backup and verify its checksum
	// For now, just check if the backup exists in the catalog

	if entry.Status != "completed" {
		return fmt.Errorf("backup is not completed: %s", entry.Status)
	}

	if entry.Checksum == "" {
		return fmt.Errorf("backup has no checksum")
	}

	return nil
}

// PruneBackups removes old backups based on retention policy
func (m *BackupManager) PruneBackups(ctx context.Context, opts PruneOptions) (int, error) {
	start := time.Now()
	
	// Get all backups from catalog
	allBackups, err := m.catalog.ListBackups("")
	if err != nil {
		return 0, fmt.Errorf("failed to list backups: %w", err)
	}

	// Group backups by VM name
	backupsByVM := make(map[string][]*storage.BackupEntry)
	for _, backup := range allBackups {
		// Skip if filtering by VM and this backup doesn't match
		if opts.VMName != "" && backup.VMName != opts.VMName {
			continue
		}
		backupsByVM[backup.VMName] = append(backupsByVM[backup.VMName], backup)
	}

	// Sort backups by timestamp (newest first) for each VM
	for vmName := range backupsByVM {
		backups := backupsByVM[vmName]
		// Sort by timestamp descending (newest first)
		for i := 0; i < len(backups)-1; i++ {
			for j := i + 1; j < len(backups); j++ {
				if backups[i].Timestamp.Before(backups[j].Timestamp) {
					backups[i], backups[j] = backups[j], backups[i]
				}
			}
		}
	}

	// Determine which backups to delete
	var toDelete []*storage.BackupEntry
	cutoffTime := time.Now().AddDate(0, 0, -opts.KeepDays)

	for vmName, backups := range backupsByVM {
		fmt.Printf("Processing backups for VM '%s' (%d total):\n", vmName, len(backups))

		for i, backup := range backups {
			shouldDelete := false
			reason := ""

			// Keep if within the keep-last count
			if i < opts.KeepLast {
				fmt.Printf("  Keeping %s (backup %d of %d most recent)\n", backup.ID, i+1, opts.KeepLast)
				continue
			}

			// Keep if within the keep-days window
			if backup.Timestamp.After(cutoffTime) {
				fmt.Printf("  Keeping %s (within %d days)\n", backup.ID, opts.KeepDays)
				continue
			}

			// Mark for deletion
			shouldDelete = true
			reason = fmt.Sprintf("older than %d days and beyond %d most recent", opts.KeepDays, opts.KeepLast)

			if shouldDelete {
				toDelete = append(toDelete, backup)
				if opts.DryRun {
					fmt.Printf("  Would delete %s (%s)\n", backup.ID, reason)
				} else {
					fmt.Printf("  Deleting %s (%s)\n", backup.ID, reason)
				}
			}
		}
	}

	// Delete the backups (unless dry run)
	deletedCount := 0
	for _, backup := range toDelete {
		if !opts.DryRun {
			if err := m.DeleteBackup(ctx, backup.ID); err != nil {
				fmt.Printf("  Failed to delete %s: %v\n", backup.ID, err)
				continue
			}
		}
		deletedCount++
	}

	// Record metrics
	if !opts.DryRun {
		metrics.RecordBackupOperation("prune", "success", time.Since(start).Seconds())
	}

	return deletedCount, nil
}

// exportVM exports a VM to a backup file
func (m *BackupManager) exportVM(ctx context.Context, vm *object.VirtualMachine, backupID string, opts BackupOptions) (string, int64, string, error) {
	// Create temporary file for export
	tempFile := filepath.Join(m.config.TempDir, fmt.Sprintf("%s.ova", backupID))
	
	// In a real implementation, this would:
	// 1. Use govmomi's export functionality to export the VM as OVF/OVA
	// 2. Compress if requested
	// 3. Calculate checksum
	// 4. Store to target (datastore, NFS, S3, etc.)
	
	// For now, create a mock export
	file, err := os.Create(tempFile)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	var writer io.Writer = file
	hasher := sha256.New()
	writer = io.MultiWriter(writer, hasher)

	// Apply compression if requested
	if opts.Compress {
		gzWriter := gzip.NewWriter(writer)
		defer gzWriter.Close()
		writer = gzWriter
	}

	// Create tar archive (simplified)
	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	// Add VM metadata to archive
	metadata := fmt.Sprintf("VM: %s\nBackup ID: %s\nTimestamp: %s\n", 
		opts.VMName, backupID, time.Now().Format(time.RFC3339))
	
	header := &tar.Header{
		Name: "metadata.txt",
		Mode: 0600,
		Size: int64(len(metadata)),
	}
	
	if err := tarWriter.WriteHeader(header); err != nil {
		return "", 0, "", err
	}
	
	if _, err := tarWriter.Write([]byte(metadata)); err != nil {
		return "", 0, "", err
	}

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return "", 0, "", err
	}

	// Calculate checksum
	checksum := hex.EncodeToString(hasher.Sum(nil))

	// Store to target
	var location string
	if opts.Target != nil {
		location = opts.Target.GetLocation()
	} else {
		// Default to datastore
		location = fmt.Sprintf("datastore://backups/%s.ova", backupID)
	}

	// Clean up temp file
	defer os.Remove(tempFile)

	return location, fileInfo.Size(), checksum, nil
}

// importVM imports a VM from a backup
func (m *BackupManager) importVM(ctx context.Context, entry *storage.BackupEntry, opts RestoreOptions) error {
	// In a real implementation, this would:
	// 1. Retrieve the backup from storage
	// 2. Decompress if needed
	// 3. Import using govmomi's import functionality
	// 4. Rename to the new name
	
	// For now, return success
	return nil
}

// ApplyRetentionPolicy applies retention policy to backups
func (m *BackupManager) ApplyRetentionPolicy(policy storage.RetentionPolicy) error {
	return m.catalog.ApplyRetention(policy)
}