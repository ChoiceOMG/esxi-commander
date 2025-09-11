package examples

import (
	"fmt"
	
	"github.com/spf13/cobra"
)

var ExamplesCmd = &cobra.Command{
	Use:   "examples",
	Short: "Show practical usage examples",
	Long:  `Display common usage examples for ESXi Commander operations.`,
	RunE:  runExamples,
}

func runExamples(cmd *cobra.Command, args []string) error {
	fmt.Println(`ESXi Commander - Practical Examples
===================================

VM Management:
--------------
# Create a new Ubuntu VM with static IP
ceso vm create web-server-01 --template ubuntu-22.04-template \
  --ip 192.168.1.100/24 --gateway 192.168.1.1 --dns 8.8.8.8,8.8.4.4 \
  --ssh-key "ssh-rsa AAAAB3NzaC1yc..." --cpu 4 --memory 8

# Clone an existing VM with new network settings
ceso vm clone web-server-01 web-server-02 --ip 192.168.1.101/24

# Power management
ceso vm start web-server-01
ceso vm stop web-server-01 --graceful
ceso vm restart web-server-01
ceso vm start --all  # Start all powered-off VMs

# Get VM information and stats
ceso vm info web-server-01
ceso vm stats web-server-01
ceso vm list --json

# VM snapshots
ceso vm snapshot create web-server-01 "pre-update" --description "Before system updates"
ceso vm snapshot list web-server-01
ceso vm snapshot revert web-server-01 "pre-update" 
ceso vm snapshot delete web-server-01 "pre-update"

Backup Operations:
------------------
# Create a cold backup (VM powered off)
ceso backup create web-server-01 --power-off --compress

# Create a hot backup (VM stays running using snapshots)
ceso backup create web-server-01 --hot --compress --description "Weekly backup"

# List backups
ceso backup list --json

# Restore backup with new VM name and IP
ceso backup restore backup-uuid-123 --as-new web-server-restored \
  --ip 192.168.1.200/24 --gateway 192.168.1.1

# Backup maintenance
ceso backup prune --keep-last 5 --vm web-server-01
ceso backup prune --keep-days 30 --dry-run  # See what would be deleted

Host Monitoring:
----------------
# Check ESXi host information
ceso host info --json
ceso host health  
ceso host stats

# Monitor resources
ceso host stats --json | jq '.cpu.usage_percent'

PCI Device Management:
---------------------
# List available PCI devices (GPU passthrough)
ceso pci list --type gpu

# Get detailed device information
ceso pci info 0000:81:00.0

# Create VM with GPU passthrough
ceso vm create gpu-workstation --template ubuntu-22.04-template \
  --cpu 8 --memory 32 --gpu 0000:81:00.0

Template Management:
-------------------
# Validate templates
ceso template validate ubuntu-22.04-template

# List available templates
ceso template list

Dry-Run and Safety:
------------------
# Preview operations without executing them
ceso vm create test-vm --template ubuntu-22.04-template --dry-run
ceso backup prune --keep-last 3 --dry-run

# Force operations without confirmation (use carefully)
ceso vm delete old-vm --force

Batch Operations:
----------------
# Start all powered-off VMs
ceso vm start --all

# List all VMs in JSON format for scripting
ceso vm list --json | jq '.[].name'

Configuration:
--------------
# Run initial setup wizard
ceso setup wizard

# Configuration files are stored in:
# ~/.ceso/config.yaml (user config)
# /etc/ceso/config.yaml (system config)

Advanced Usage:
---------------
# Combine with other tools
ceso vm list --json | jq -r '.[] | select(.status=="poweredOn") | .name' | \
  xargs -I {} ceso vm stats {}

# Monitor multiple VMs
for vm in $(ceso vm list --json | jq -r '.[].name'); do
  echo "=== $vm ==="
  ceso vm stats $vm
done

# Automated backup routine
#!/bin/bash
for vm in $(ceso vm list --json | jq -r '.[] | select(.status=="poweredOn") | .name'); do
  echo "Backing up $vm..."
  ceso backup create "$vm" --hot --compress --description "Automated backup $(date)"
done

# Clean up old backups
ceso backup prune --keep-days 30 --keep-last 5

Tips:
-----
1. Always test with --dry-run first for destructive operations
2. Use --json output for scripting and automation
3. Set up SSH keys for passwordless VM access
4. Regular backup pruning prevents storage bloat
5. Monitor host resources before creating new VMs
6. Use hot backups for production VMs to avoid downtime
7. Validate templates before creating VMs
8. Keep backups of critical VMs with longer retention`)

	return nil
}