# ESXi Commander (ceso)

[![CI](https://github.com/ChoiceOMG/esxi-commander/actions/workflows/ci.yml/badge.svg)](https://github.com/ChoiceOMG/esxi-commander/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/r11/esxi-commander)](https://goreportcard.com/report/github.com/r11/esxi-commander)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A production-grade, Ubuntu-focused VMware ESXi orchestrator designed for standalone ESXi hosts without vCenter. ESXi Commander provides a simplified CLI for managing Ubuntu LTS virtual machines with emphasis on security, auditability, and AI agent compatibility.

## Features

### Core Operations
- **VM Management**: Complete lifecycle management with power operations, snapshots, and GPU passthrough
- **Backup System**: Hot/cold backups with retention policies and restore capabilities
- **Host Monitoring**: Resource statistics, health checks, and system information
- **PCI Device Management**: GPU passthrough support for NVIDIA, AMD, and Intel devices

### Production Features
- **Ubuntu LTS Focus**: Optimized for Ubuntu virtual machines with cloud-init integration
- **No vCenter Required**: Direct ESXi host management for standalone environments
- **Security First**: Built-in audit logging, AI agent sandboxing, and access controls
- **Quality of Life**: Input validation, interactive confirmations, progress indicators, and batch operations

## Quick Start

### Installation

```bash
# Download the latest release
wget https://github.com/ChoiceOMG/esxi-commander/releases/latest/download/ceso-linux-amd64
chmod +x ceso-linux-amd64
sudo mv ceso-linux-amd64 /usr/local/bin/ceso

# Or build from source
git clone https://github.com/ChoiceOMG/esxi-commander.git
cd esxi-commander
go build -o ceso cmd/ceso/main.go
```

### First-Time Setup

Run the interactive setup wizard:
```bash
ceso setup
```

Or create `~/.ceso/config.yaml`:
```yaml
esxi:
  host: "192.168.1.100"
  user: "root"
  password: ""  # Use ESXI_PASSWORD env var
  insecure: true
```

### VM Operations
```bash
# Create a new VM with cloud-init configuration
ceso vm create myvm --template ubuntu-22.04 --ip 192.168.1.100/24 --ssh-key ~/.ssh/id_rsa.pub --cpu 4 --memory 8

# Create VM with GPU passthrough
ceso vm create gpu-workstation --template ubuntu-22.04 --gpu 0000:81:00.0 --cpu 8 --memory 32

# VM power management
ceso vm start myvm
ceso vm stop myvm --graceful
ceso vm restart myvm
ceso vm start --all  # Start all powered-off VMs

# VM information and monitoring
ceso vm list --json
ceso vm info myvm
ceso vm stats myvm
ceso vm console myvm

# VM snapshots
ceso vm snapshot create myvm "pre-update" --memory --quiesce
ceso vm snapshot list myvm
ceso vm snapshot revert myvm "pre-update"
ceso vm snapshot delete myvm "pre-update"

# Clone and delete operations
ceso vm clone myvm myvm-clone --ip 192.168.1.101/24
ceso vm delete myvm --force  # Skip confirmation
```

### Backup Operations
```bash
# Cold backup (VM powered off)
ceso backup create myvm --power-off --compress

# Hot backup (VM stays running using snapshots)
ceso backup create myvm --hot --compress --description "Weekly backup"

# Backup management
ceso backup list --json
ceso backup verify backup-uuid-123
ceso backup delete backup-uuid-123

# Restore with network reconfiguration
ceso backup restore backup-uuid-123 --as-new restored-vm \
  --ip 192.168.1.200/24 --gateway 192.168.1.1 --power-on

# Backup retention and pruning
ceso backup prune --keep-last 5 --vm myvm
ceso backup prune --keep-days 30 --dry-run  # Preview deletions
```

### Host Monitoring
```bash
# Host information and health
ceso host info --json
ceso host health
ceso host stats

# PCI device management
ceso pci list --type gpu
ceso pci info 0000:81:00.0
```

### Utility Commands
```bash
# Show practical examples
ceso examples

# Validate operations without executing
ceso vm create test-vm --template ubuntu-22.04 --dry-run

# Interactive setup
ceso setup wizard
```

## Architecture

ESXi Commander is built with Go and uses:

- **govmomi**: Primary VMware vSphere SDK interface
- **SSH fallback**: Direct ESXi command execution when needed
- **BoltDB**: Embedded database for IPAM, backup catalog, and audit logs
- **Cloud-init**: Ubuntu VM configuration via VMware guestinfo injection
- **Cobra + Viper**: CLI framework and configuration management

## Key Components

### Cloud-Init Integration
- Network configuration (static IP/CIDR)
- SSH key injection and hostname setup
- Automated Ubuntu post-boot configuration
- VMware guestinfo metadata injection

### AI Agent Sandboxing
- Restricted mode for AI agents (read-only + dry-run)
- Standard mode for human operators
- Time-limited promotion workflow with human approval

### Backup System
- Hot backups using VMware snapshots (zero downtime)
- Cold backups with optional VM power-off
- Compression support (gzip) for space efficiency
- Automated retention policies with pruning
- Cross-platform restore with network reconfiguration
- BoltDB-based backup catalog with metadata tracking

## Security Features

- **Audit Logging**: Structured, append-only logs with secret redaction
- **Access Control**: IP allowlists and short-lived SSH certificates  
- **Secret Management**: OS keyring with encrypted file fallback
- **AI Safety**: Restricted command execution with promotion workflow

## Command Reference

### VM Commands
| Command | Description | Key Flags |
|---------|-------------|-----------|
| `ceso vm create <name>` | Create new VM from template | `--template`, `--ip`, `--cpu`, `--memory`, `--gpu` |
| `ceso vm clone <source> <dest>` | Clone existing VM | `--ip`, `--gateway`, `--dns` |
| `ceso vm list` | List all VMs | `--json` |
| `ceso vm info <name>` | Get VM details | `--json` |
| `ceso vm stats <name>` | Show resource usage | `--json` |
| `ceso vm delete <name>` | Delete VM | `--force` |
| `ceso vm start <name>` | Power on VM | `--all` |
| `ceso vm stop <name>` | Power off VM | `--graceful`, `--all` |
| `ceso vm restart <name>` | Restart VM | `--graceful` |
| `ceso vm suspend <name>` | Suspend VM | |
| `ceso vm resume <name>` | Resume suspended VM | |
| `ceso vm console <name>` | Get console access info | |

### VM Snapshot Commands
| Command | Description | Key Flags |
|---------|-------------|-----------|
| `ceso vm snapshot create <vm> <name>` | Create snapshot | `--memory`, `--quiesce`, `--description` |
| `ceso vm snapshot list <vm>` | List snapshots | `--json` |
| `ceso vm snapshot revert <vm> <name>` | Revert to snapshot | |
| `ceso vm snapshot delete <vm> <name>` | Delete snapshot | `--children` |

### Backup Commands
| Command | Description | Key Flags |
|---------|-------------|-----------|
| `ceso backup create <vm>` | Create backup | `--hot`, `--power-off`, `--compress`, `--description` |
| `ceso backup list` | List backups | `--json` |
| `ceso backup restore <id>` | Restore backup | `--as-new`, `--ip`, `--gateway`, `--power-on` |
| `ceso backup delete <id>` | Delete backup | |
| `ceso backup verify <id>` | Verify backup integrity | |
| `ceso backup prune` | Clean old backups | `--keep-last`, `--keep-days`, `--vm`, `--dry-run` |

### Host Commands
| Command | Description | Key Flags |
|---------|-------------|-----------|
| `ceso host info` | Host system information | `--json` |
| `ceso host health` | Health status check | `--json` |
| `ceso host stats` | Resource statistics | `--json` |

### PCI Commands
| Command | Description | Key Flags |
|---------|-------------|-----------|
| `ceso pci list` | List PCI devices | `--type`, `--available`, `--json` |
| `ceso pci info <device>` | Device details | `--json` |

### Template Commands
| Command | Description | Key Flags |
|---------|-------------|-----------|
| `ceso template list` | List available templates | `--json` |
| `ceso template validate <name>` | Validate template | |

### Utility Commands
| Command | Description | Key Flags |
|---------|-------------|-----------|
| `ceso examples` | Show usage examples | |
| `ceso setup wizard` | Interactive setup | |

### Global Flags
- `--config <file>`: Specify config file
- `--json`: Output in JSON format
- `--dry-run`: Preview operations without execution
- `--help`: Show command help

## Development

### Requirements
- Go 1.21+
- Access to ESXi 7.x/8.x host
- Ubuntu LTS VM templates

### Building
```bash
go build ./cmd/ceso
```

### Testing
```bash
# Unit tests
go test ./pkg/...

# Integration tests
go test -tags=integration ./test/integration/...

# All tests
go test ./...
```

## Performance Targets

- VM creation: <90 seconds
- VM cloning (80GB): <5 minutes  
- Backup/restore success rate: >99%
- Zero manual ESXi UI steps required

## Contributing

ESXi Commander emphasizes security and production readiness. All contributions should maintain:

- Comprehensive test coverage
- Structured audit logging
- Security best practices
- Clear documentation

## License

[License TBD]

## Support

For issues and feature requests, please use the GitHub issue tracker.