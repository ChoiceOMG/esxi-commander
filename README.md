# ESXi Commander

A production-grade, Ubuntu-focused VMware ESXi orchestrator designed for standalone ESXi hosts without vCenter. ESXi Commander provides a simplified CLI for managing Ubuntu LTS virtual machines with emphasis on security, auditability, and AI agent compatibility.

## Features

- **Simple CLI Interface**: 8 core commands for VM and backup operations
- **Ubuntu LTS Focus**: Optimized for Ubuntu virtual machines with cloud-init integration
- **No vCenter Required**: Direct ESXi host management for standalone environments
- **Security First**: Built-in audit logging, AI agent sandboxing, and access controls
- **Production Ready**: Comprehensive testing, monitoring, and backup capabilities

## Quick Start

### VM Operations
```bash
# Create a new VM with cloud-init configuration
ceso vm create myvm --template ubuntu-22.04 --ip 192.168.1.100/24 --ssh-key ~/.ssh/id_rsa.pub

# List all VMs
ceso vm list

# Get VM information
ceso vm info myvm

# Clone an existing VM
ceso vm clone myvm myvm-clone --ip 192.168.1.101/24

# Delete a VM
ceso vm delete myvm
```

### Backup Operations
```bash
# Create a backup
ceso backup create myvm

# List backups
ceso backup list

# Restore from backup
ceso backup restore backup-123 --as-new restored-vm
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

### Extensible Backup System
- Local datastore storage (Phase 1)
- Pluggable architecture for future S3/NFS/SFTP support
- Incremental backup capabilities

## Security Features

- **Audit Logging**: Structured, append-only logs with secret redaction
- **Access Control**: IP allowlists and short-lived SSH certificates  
- **Secret Management**: OS keyring with encrypted file fallback
- **AI Safety**: Restricted command execution with promotion workflow

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