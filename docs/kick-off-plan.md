# ESXi Commander Kick-off Plan (Ubuntu-only)

## Executive Summary
A production-grade, Ubuntu-focused ESXi orchestrator with simplified CLI, robust testing, and comprehensive security. Built for both human operators and AI agents with measurable success criteria.

## Architecture Overview

```
┌─────────────────────────────────────────────────┐
│          Simplified CLI (Cobra)                 │
│     8 core commands, smart defaults             │
└─────────────────────────────────────────────────┘
                      │
┌─────────────────────────────────────────────────┐
│         JSON-RPC API (AI Agents)                │
│          Optional, off by default               │
└─────────────────────────────────────────────────┘
                      │
┌─────────────────────────────────────────────────┐
│          Orchestration Engine                   │
│    Workflows, dry-run, rollback                 │
└─────────────────────────────────────────────────┘
                      │
┌─────────────────────────────────────────────────┐
│             Core Services                       │
├────────────┬────────────┬───────────────────────┤
│ VM Manager │   Backup   │     Cloud-Init        │
│  Create    │   Engine   │  Ubuntu guestinfo     │
│  Clone     │  Compress  │   Hostname/IP/SSH     │
│  Configure │   Encrypt  │                       │
└────────────┴────────────┴───────────────────────┘
                      │
┌─────────────────────────────────────────────────┐
│           ESXi Adapter Layer                    │
│   govmomi (primary) | SSH fallback              │
└─────────────────────────────────────────────────┘
```

## Simplified CLI Interface

```bash
# Core VM Commands (minimal flags)
ceso vm create <name> --template <tmpl> --ip <cidr> --ssh-key <key> [--cpu N] [--ram N]
ceso vm clone <source> <dest> --ip <cidr>
ceso vm list [--json]
ceso vm info <name> [--json]
ceso vm delete <name>

# Backup Commands
ceso backup create <vm-name> [--compress zstd]
ceso backup restore <backup-id> --as-new <name>
ceso backup list [--json]

# Template Validation (NEW)
ceso template validate <template-name>
ceso template validate-all [--json]
```

## Technology Stack

**Core:**
- **Go** - Single binary deployment, excellent concurrency
- **govmomi** - VMware's mature vSphere SDK (primary interface)
- **Cobra + Viper** - CLI with rich tab completion + configuration
- **zerolog** - Structured JSON logging for audit trails
- **BoltDB** - Embedded DB for IPAM, backup catalog, operation history
- **golang.org/x/crypto/ssh** - SSH fallback when API unavailable

**Testing:**
- **testify** - Assertions and test suites
- **gomock** - Mocking for unit tests
- **ginkgo** - BDD-style integration tests

**Monitoring:**
- **Prometheus** - Metrics collection
- **Grafana** - Dashboards for operational visibility

## Project Structure (Focused)

```
esxi-commander/
├── cmd/
│   ├── ceso/                 # Main CLI binary
│   └── cesod/                # Optional daemon (JSON-RPC server)
├── pkg/
│   ├── cli/                  # Cobra commands
│   │   ├── vm/               # vm subcommands
│   │   ├── backup/           # backup subcommands
│   │   └── template/         # template subcommands
│   ├── esxi/                 # ESXi interaction
│   │   ├── client/           # Connection management
│   │   ├── vm/               # VM operations
│   │   └── datastore/        # Datastore operations
│   ├── cloudinit/            # Ubuntu cloud-init
│   ├── backup/               # Backup engine
│   │   ├── targets/          # Extensible backup targets
│   │   ├── catalog/          # Backup metadata
│   │   └── retention/        # Retention policies
│   ├── template/             # Template validation
│   ├── security/             # Security components
│   │   ├── sandbox/          # AI agent sandboxing
│   │   ├── allowlist/        # IP allowlisting
│   │   ├── certs/            # SSH certificates
│   │   └── secrets/          # Secret management
│   ├── metrics/              # Prometheus metrics
│   └── audit/                # Audit logging
├── internal/
│   ├── defaults/             # Smart defaults
│   ├── validation/           # Input validation
│   └── storage/              # Embedded database
├── test/
│   ├── unit/                 # Unit tests
│   ├── integration/          # Integration tests
│   └── chaos/                # Chaos tests
└── configs/
    └── templates.yaml        # Template definitions
```

## Cloud-Init Integration (Centerpiece)

```go
// Simplified but powerful cloud-init config
type SimpleVMConfig struct {
    // Required
    Name     string
    Template string
    IP       string  // CIDR notation
    SSHKey   string
    
    // Optional with defaults
    CPU      int     // Default: 2
    RAM      int     // in GB, default: 4
    Disk     int     // in GB, default: 40
    
    // Auto-computed
    Gateway  string  // From IP subnet if not specified
    Hostname string  // From Name if not specified
}

// VMware guestinfo injection
func BuildGuestInfo(cfg SimpleVMConfig) map[string]string {
    return map[string]string{
        "guestinfo.metadata": base64(metadata),
        "guestinfo.metadata.encoding": "gzip+base64",
        "guestinfo.userdata": base64(userdata),
        "guestinfo.userdata.encoding": "gzip+base64",
    }
}
```

## AI Agent Sandboxing (NEW)

```go
type OperationMode int

const (
    ModeRestricted OperationMode = iota  // Default for AI agents
    ModeStandard                          // Human operators
    ModeUnrestricted                      // Full admin
)

// Restricted mode allows only:
// - vm.list, vm.info
// - backup.list
// - template.validate
// - All commands with --dry-run flag
```

**Promotion workflow:**
```bash
# Claude starts in restricted mode
export CESO_AGENT=claude
ceso vm create web-01 --template ubuntu-22.04 --ip 192.168.1.100/24 --dry-run  # OK
ceso vm create web-01 --template ubuntu-22.04 --ip 192.168.1.100/24            # BLOCKED

# Human operator promotes Claude
ceso admin promote-agent claude --duration 1h
```

## Template Validation (NEW)

Validates Ubuntu templates before use:
- Cloud-init installed and configured
- VMware Tools (open-vm-tools) present
- Guestinfo datasource enabled
- Network configuration compatible

```bash
ceso template validate ubuntu-22.04-golden
# Output:
# ✓ Cloud-init installed (version 23.4)
# ✓ VMware Tools installed (open-vm-tools 12.3.0)
# ✓ Guestinfo datasource enabled
# Template is ready for use
```

## Backup Architecture (Future-Proof)

```go
// Extensible backup target interface
type BackupTarget interface {
    Type() string
    Upload(backup io.Reader, metadata BackupMetadata) error
    Download(backupID string) (io.ReadCloser, error)
    List(filter BackupFilter) ([]BackupMetadata, error)
    Delete(backupID string) error
}

// Phase 1: Datastore only
// Phase 3: Add S3, NFS, SFTP via rclone
```

## Testing Strategy

### Unit Tests (70% coverage target)
- Command builders
- Cloud-init generators
- Input validators
- Secret redactors
- Template validators

### Integration Tests
- VM lifecycle (create → clone → backup → restore)
- Cloud-init verification (hostname, IP, SSH access)
- Backup integrity checks
- Performance benchmarks
- Template validation

### Chaos Tests (Automated in CI/CD)
- Datastore full scenarios
- Network partitions
- Snapshot failures
- Concurrent operations

```yaml
# .github/workflows/chaos-tests.yaml
name: Chaos Tests
on:
  schedule:
    - cron: '0 2 * * *'  # Daily
  pull_request:
    paths:
      - 'pkg/**'

jobs:
  chaos-datastore-full:
    runs-on: self-hosted
    steps:
      - name: Run datastore full test
        run: go test -tags=chaos ./test/chaos -run TestDatastoreFull
```

## Success Metrics (CI/CD Gates)

```yaml
acceptance_criteria:
  performance:
    vm_create_time: "< 90s"
    vm_clone_80gb: "< 5m"
    backup_100gb: "< 10m"
    
  reliability:
    backup_success_rate: "> 99%"
    restore_success_rate: "> 99%"
    vm_create_success: "> 99.5%"
    
  operations:
    zero_manual_steps: true
    audit_log_completeness: "100%"
    secret_leakage: "0"
```

## Prometheus Metrics Integration

```go
var (
    vmCreateDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "ceso_vm_create_duration_seconds",
            Help: "Time to create VM from template",
            Buckets: []float64{30, 60, 90, 120, 180},
        },
        []string{"template", "result"},
    )
    
    aiOperations = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ceso_ai_operations_total",
            Help: "Operations executed by AI agents",
        },
        []string{"agent", "operation", "result"},
    )
)
```

## Security Implementation

### Access Control
- IP allowlists for API/SSH access
- Short-lived SSH certificates (8-hour default)
- Role-based command restrictions
- AI agent sandboxing with promotion

### Secret Management
- OS keyring integration (Linux)
- Encrypted file fallback
- No secrets in logs or command history
- Automatic redaction in audit logs

### Audit Logging
- Append-only structured logs
- Secret redaction
- Optional remote shipping
- Every ESXi command logged
- AI agent operations tracked separately

## Implementation Timeline

**Week 1: Foundation + Template Validation**
- Project setup with sandbox modes
- Template validator implementation
- Basic CLI with restricted mode
- Prometheus metrics from day 1

**Week 2: Core VM Operations**
- VM create/clone with cloud-init
- Future-proof backup catalog schema
- First automated chaos test

**Week 3: Testing Infrastructure**
- Integration test suite
- Template validation in CI/CD
- Performance benchmarks

**Week 4: Backup System**
- Datastore target implementation
- Extensible target interface
- Backup catalog with forward compatibility

**Week 5: Security & Monitoring**
- AI agent sandbox with promotion
- Prometheus/Grafana dashboards
- Audit log shipping

**Week 6: Production Hardening**
- Additional chaos tests
- Performance optimization
- Documentation for AI agents

## AI Agent Integration

```bash
# JSON output for AI parsing (add --json to any command)
ceso vm list --json
ceso vm info web-01 --json

# Dry-run for AI validation
ceso vm create web-02 --template ubuntu-22.04 --ip 192.168.1.102/24 --dry-run --json

# Simple JSON-RPC endpoint (optional daemon mode)
cesod --port 8080  # Exposes same commands via JSON-RPC
```

### JSON-RPC Example for AI
```json
{
  "jsonrpc": "2.0",
  "method": "vm.create",
  "params": {
    "name": "web-02",
    "template": "ubuntu-22.04",
    "ip": "192.168.1.102/24",
    "ssh_key": "ssh-rsa AAAAB3...",
    "cpu": 4,
    "ram": 8
  },
  "id": 1
}
```

## Configuration File

```yaml
# ~/.ceso/config.yaml
esxi:
  host: 192.168.1.10
  user: root
  ssh_key: ~/.ssh/esxi_key

security:
  mode: restricted  # For Claude/AI agents
  allowed_networks:
    - 192.168.1.0/24
  agent_promotion:
    require_human: true
    max_duration: 4h

defaults:
  template: ubuntu-22.04-golden
  datastore: datastore1
  network: "VM Network"
  cpu: 2
  ram: 4  # GB
  disk: 40 # GB
  
templates:
  ubuntu-22.04-golden:
    path: /vmfs/volumes/datastore1/templates/ubuntu-22.04
    validated: true
    validation_date: 2024-01-15
  ubuntu-24.04-golden:
    path: /vmfs/volumes/datastore1/templates/ubuntu-24.04

backup:
  compress: zstd
  location: /vmfs/volumes/backup-datastore
  catalog:
    version: 2  # Future-proof schema

monitoring:
  prometheus:
    enabled: true
    port: 9090
  grafana:
    dashboards:
      - esxi-commander-ops
      - ai-agent-activity

chaos:
  enabled: true
  scenarios:
    - datastore_full
    - network_partition
```

## Why This Approach Succeeds

1. **Measurable Success**: Clear metrics with CI/CD gates
2. **Security-First**: Multiple layers including AI sandboxing
3. **Test-Driven**: Unit, integration, and chaos tests from day 1
4. **Simple Interface**: 8 commands, minimal flags
5. **Production-Ready**: Monitoring, logging, deployment scripts
6. **AI-Compatible**: JSON output, sandboxing, predictable patterns
7. **Ubuntu-Optimized**: Single OS path, deep cloud-init integration
8. **Future-Proof**: Extensible backup targets, forward-compatible schemas

## Next Immediate Steps

1. Initialize Go project: `go mod init github.com/r11/esxi-commander`
2. Set up GitHub Actions with acceptance tests
3. Implement basic CLI with Cobra
4. Create first working command: `ceso vm list`
5. Add template validator
6. Implement cloud-init guestinfo builder
7. Create VM with full test coverage
8. Add first chaos test to CI/CD