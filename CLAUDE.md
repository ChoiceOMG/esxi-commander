# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ESXi Commander is a production-grade, Ubuntu-focused VMware ESXi orchestrator. It provides a simplified CLI for managing Ubuntu LTS virtual machines on standalone ESXi 7.x/8.x hosts without vCenter. The project emphasizes security, auditability, and AI agent compatibility.

## Technology Stack

- **Language**: Go (targeting single binary deployment)
- **ESXi Integration**: govmomi (VMware vSphere SDK) as primary interface, SSH fallback
- **CLI Framework**: Cobra + Viper for commands and configuration
- **Database**: BoltDB (embedded) for IPAM, backup catalog, operation history
- **Logging**: zerolog for structured JSON audit trails
- **Testing**: testify, gomock, ginkgo for unit/integration/BDD tests
- **Monitoring**: Prometheus metrics, Grafana dashboards

## Core CLI Commands

The project implements 8 simplified commands:

```bash
# VM Operations
ceso vm create <name> --template <tmpl> --ip <cidr> --ssh-key <key>
ceso vm clone <source> <dest> --ip <cidr>
ceso vm list [--json]
ceso vm info <name> [--json]
ceso vm delete <name>

# Backup Operations  
ceso backup create <vm-name>
ceso backup restore <backup-id> --as-new <name>
ceso backup list [--json]
```

## Project Structure

```
esxi-commander/
├── cmd/
│   ├── ceso/                 # Main CLI binary
│   └── cesod/                # Optional JSON-RPC daemon
├── pkg/
│   ├── cli/                  # Cobra commands (vm/, backup/, template/)
│   ├── esxi/                 # ESXi interaction (client/, vm/, datastore/)
│   ├── cloudinit/            # Ubuntu cloud-init guestinfo injection
│   ├── backup/               # Backup engine with extensible targets
│   ├── security/             # AI sandboxing, allowlists, certs, secrets
│   └── audit/                # Audit logging
├── internal/
│   ├── defaults/             # Smart defaults and validation
│   └── storage/              # Embedded BoltDB operations
└── test/
    ├── unit/                 # Unit tests
    ├── integration/          # Full VM lifecycle tests
    └── chaos/                # Chaos engineering tests
```

## Development Workflow

### Initialize Project (if not already done)
```bash
go mod init github.com/r11/esxi-commander
go mod tidy
```

### Common Commands
- **Build**: `go build ./cmd/ceso`
- **Test**: `go test ./...`
- **Unit tests**: `go test ./pkg/...`
- **Integration tests**: `go test -tags=integration ./test/integration/...`
- **Chaos tests**: `go test -tags=chaos ./test/chaos/...`
- **Lint**: `golangci-lint run` (if configured)
- **Format**: `go fmt ./...`

## Key Architectural Components

### Cloud-Init Integration
Core feature using VMware guestinfo injection for Ubuntu VMs:
- Network configuration (static IP/CIDR)
- Hostname and SSH key injection
- Automated Ubuntu post-boot configuration
- Base64 + gzip encoding for guestinfo metadata/userdata

### AI Agent Sandboxing
- **Restricted Mode**: Default for AI agents (read-only + dry-run)
- **Standard Mode**: Human operators
- **Promotion Workflow**: Time-limited elevation with human approval

### Backup Architecture
Extensible target interface supporting:
- Phase 1: Local datastore storage
- Future: S3, NFS, SFTP via pluggable targets

## Testing Strategy

- **Unit Tests**: 70% coverage target, focus on command builders, validators
- **Integration Tests**: Full VM lifecycle with cloud-init verification
- **Chaos Tests**: Automated failure scenarios (datastore full, network partitions)
- **Performance Gates**: VM create <90s, clone 80GB <5min, backup success >99%

## Security Considerations

- **Secret Management**: OS keyring with encrypted file fallback
- **Audit Logging**: Structured, append-only with secret redaction
- **Access Control**: IP allowlists, short-lived SSH certificates
- **AI Safety**: Restricted command execution with promotion workflow

## Agent Configuration

This repository includes specialized Claude agents in `.claude/agents/`:
- **esxi-ubuntu-specialist**: ESXi expertise and command planning
- **safety-reviewer**: Security and production readiness review
- **implementation-assistant**: Active development and testing
- **testing-architect**: Test strategy, verification plans, and quality assurance

Use these agents for focused expertise in their respective domains.

## Success Metrics

The project targets measurable outcomes:
- VM operations: <90s create, <5min clone, >99% success rate
- Backup reliability: >99% success rate for backup/restore
- Zero manual ESXi UI steps required
- Complete audit trail for all operations