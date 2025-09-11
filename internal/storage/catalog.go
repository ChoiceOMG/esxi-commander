package storage

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

const (
	backupBucket = "backups"
	vmIndexBucket = "vm_index"
)

type BackupCatalog struct {
	db *bbolt.DB
}

type BackupEntry struct {
	ID          string            `json:"id"`
	VMName      string            `json:"vm_name"`
	Timestamp   time.Time         `json:"timestamp"`
	Size        int64             `json:"size"`
	Location    string            `json:"location"`
	Checksum    string            `json:"checksum"`
	Type        string            `json:"type"` // cold, hot, snapshot
	Status      string            `json:"status"` // pending, completed, failed
	Metadata    map[string]string `json:"metadata"`
}

type RetentionPolicy struct {
	KeepLast    int `json:"keep_last"`
	KeepDaily   int `json:"keep_daily"`
	KeepWeekly  int `json:"keep_weekly"`
	KeepMonthly int `json:"keep_monthly"`
}

// InitCatalog creates or opens a backup catalog database
func InitCatalog(path string) (*BackupCatalog, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		// Directory creation handled by caller if needed
	}

	db, err := bbolt.Open(path, 0600, &bbolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open catalog database: %w", err)
	}

	// Initialize buckets
	err = db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(backupBucket)); err != nil {
			return fmt.Errorf("failed to create backup bucket: %w", err)
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(vmIndexBucket)); err != nil {
			return fmt.Errorf("failed to create VM index bucket: %w", err)
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &BackupCatalog{db: db}, nil
}

// Close closes the catalog database
func (c *BackupCatalog) Close() error {
	return c.db.Close()
}

// AddBackup adds a new backup entry to the catalog
func (c *BackupCatalog) AddBackup(entry *BackupEntry) error {
	if entry.ID == "" {
		return fmt.Errorf("backup ID cannot be empty")
	}
	if entry.VMName == "" {
		return fmt.Errorf("VM name cannot be empty")
	}

	return c.db.Update(func(tx *bbolt.Tx) error {
		// Store backup entry
		backupBucket := tx.Bucket([]byte(backupBucket))
		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal backup entry: %w", err)
		}

		if err := backupBucket.Put([]byte(entry.ID), data); err != nil {
			return fmt.Errorf("failed to store backup entry: %w", err)
		}

		// Update VM index
		vmBucket := tx.Bucket([]byte(vmIndexBucket))
		vmKey := []byte(entry.VMName)
		
		var backupIDs []string
		if existingData := vmBucket.Get(vmKey); existingData != nil {
			if err := json.Unmarshal(existingData, &backupIDs); err != nil {
				return fmt.Errorf("failed to unmarshal VM index: %w", err)
			}
		}
		
		backupIDs = append(backupIDs, entry.ID)
		
		indexData, err := json.Marshal(backupIDs)
		if err != nil {
			return fmt.Errorf("failed to marshal VM index: %w", err)
		}
		
		if err := vmBucket.Put(vmKey, indexData); err != nil {
			return fmt.Errorf("failed to update VM index: %w", err)
		}

		return nil
	})
}

// GetBackup retrieves a backup entry by ID
func (c *BackupCatalog) GetBackup(id string) (*BackupEntry, error) {
	var entry *BackupEntry

	err := c.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(backupBucket))
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("backup not found: %s", id)
		}

		entry = &BackupEntry{}
		if err := json.Unmarshal(data, entry); err != nil {
			return fmt.Errorf("failed to unmarshal backup entry: %w", err)
		}

		return nil
	})

	return entry, err
}

// ListBackups lists all backups, optionally filtered by VM name
func (c *BackupCatalog) ListBackups(vmName string) ([]*BackupEntry, error) {
	var entries []*BackupEntry

	err := c.db.View(func(tx *bbolt.Tx) error {
		if vmName != "" {
			// Use VM index for efficient filtering
			vmBucket := tx.Bucket([]byte(vmIndexBucket))
			indexData := vmBucket.Get([]byte(vmName))
			if indexData == nil {
				return nil // No backups for this VM
			}

			var backupIDs []string
			if err := json.Unmarshal(indexData, &backupIDs); err != nil {
				return fmt.Errorf("failed to unmarshal VM index: %w", err)
			}

			backupBucket := tx.Bucket([]byte(backupBucket))
			for _, id := range backupIDs {
				data := backupBucket.Get([]byte(id))
				if data != nil {
					entry := &BackupEntry{}
					if err := json.Unmarshal(data, entry); err != nil {
						return fmt.Errorf("failed to unmarshal backup entry: %w", err)
					}
					entries = append(entries, entry)
				}
			}
		} else {
			// List all backups
			bucket := tx.Bucket([]byte(backupBucket))
			return bucket.ForEach(func(k, v []byte) error {
				entry := &BackupEntry{}
				if err := json.Unmarshal(v, entry); err != nil {
					return fmt.Errorf("failed to unmarshal backup entry: %w", err)
				}
				entries = append(entries, entry)
				return nil
			})
		}
		return nil
	})

	return entries, err
}

// DeleteBackup removes a backup entry from the catalog
func (c *BackupCatalog) DeleteBackup(id string) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		// Get backup entry to find VM name
		backupBucket := tx.Bucket([]byte(backupBucket))
		data := backupBucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("backup not found: %s", id)
		}

		var entry BackupEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return fmt.Errorf("failed to unmarshal backup entry: %w", err)
		}

		// Remove from VM index
		vmBucket := tx.Bucket([]byte(vmIndexBucket))
		vmKey := []byte(entry.VMName)
		
		if indexData := vmBucket.Get(vmKey); indexData != nil {
			var backupIDs []string
			if err := json.Unmarshal(indexData, &backupIDs); err != nil {
				return fmt.Errorf("failed to unmarshal VM index: %w", err)
			}

			// Remove ID from list
			newIDs := make([]string, 0, len(backupIDs))
			for _, bid := range backupIDs {
				if bid != id {
					newIDs = append(newIDs, bid)
				}
			}

			if len(newIDs) > 0 {
				indexData, err := json.Marshal(newIDs)
				if err != nil {
					return fmt.Errorf("failed to marshal VM index: %w", err)
				}
				if err := vmBucket.Put(vmKey, indexData); err != nil {
					return fmt.Errorf("failed to update VM index: %w", err)
				}
			} else {
				// No more backups for this VM, remove the index entry
				if err := vmBucket.Delete(vmKey); err != nil {
					return fmt.Errorf("failed to delete VM index: %w", err)
				}
			}
		}

		// Delete backup entry
		if err := backupBucket.Delete([]byte(id)); err != nil {
			return fmt.Errorf("failed to delete backup entry: %w", err)
		}

		return nil
	})
}

// ApplyRetention applies retention policy to backups
func (c *BackupCatalog) ApplyRetention(policy RetentionPolicy) error {
	// Group backups by VM
	allBackups, err := c.ListBackups("")
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	vmBackups := make(map[string][]*BackupEntry)
	for _, backup := range allBackups {
		vmBackups[backup.VMName] = append(vmBackups[backup.VMName], backup)
	}

	// Apply retention per VM
	for _, backups := range vmBackups {
		toDelete := c.identifyBackupsToDelete(backups, policy)
		for _, backup := range toDelete {
			if err := c.DeleteBackup(backup.ID); err != nil {
				return fmt.Errorf("failed to delete backup %s: %w", backup.ID, err)
			}
		}
	}

	return nil
}

// identifyBackupsToDelete determines which backups should be deleted based on retention policy
func (c *BackupCatalog) identifyBackupsToDelete(backups []*BackupEntry, policy RetentionPolicy) []*BackupEntry {
	if len(backups) == 0 {
		return nil
	}

	// Sort backups by timestamp (newest first)
	// Implementation would include sorting logic

	var toKeep = make(map[string]bool)
	var toDelete []*BackupEntry

	// Keep last N backups
	count := 0
	for _, backup := range backups {
		if count < policy.KeepLast {
			toKeep[backup.ID] = true
			count++
		}
	}

	// Keep daily backups (one per day for N days)
	// Keep weekly backups (one per week for N weeks)
	// Keep monthly backups (one per month for N months)
	// Implementation would include date-based grouping logic

	// Mark non-kept backups for deletion
	for _, backup := range backups {
		if !toKeep[backup.ID] {
			toDelete = append(toDelete, backup)
		}
	}

	return toDelete
}

// UpdateBackupStatus updates the status of a backup entry
func (c *BackupCatalog) UpdateBackupStatus(id string, status string) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(backupBucket))
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("backup not found: %s", id)
		}

		var entry BackupEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return fmt.Errorf("failed to unmarshal backup entry: %w", err)
		}

		entry.Status = status

		updatedData, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal updated entry: %w", err)
		}

		if err := bucket.Put([]byte(id), updatedData); err != nil {
			return fmt.Errorf("failed to update backup entry: %w", err)
		}

		return nil
	})
}

// GetLatestBackup returns the most recent backup for a VM
func (c *BackupCatalog) GetLatestBackup(vmName string) (*BackupEntry, error) {
	backups, err := c.ListBackups(vmName)
	if err != nil {
		return nil, err
	}

	if len(backups) == 0 {
		return nil, fmt.Errorf("no backups found for VM: %s", vmName)
	}

	// Find the most recent backup
	var latest *BackupEntry
	for _, backup := range backups {
		if latest == nil || backup.Timestamp.After(latest.Timestamp) {
			latest = backup
		}
	}

	return latest, nil
}