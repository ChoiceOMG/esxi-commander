# Phase 3 Implementation Brief for ESXi Commander

## Mission
Complete the production hardening of ESXi Commander by implementing comprehensive testing, backup systems, monitoring, and security features. Phase 3 transforms the working Phase 2 implementation into a production-ready system with enterprise-grade reliability.

## Context Documents
- **Phase 2 Completion**: `docs/phase-3-readiness.md` - Phase 2 implementation summary
- **Test Results**: `docs/phase2-test-results.md` - Current test coverage and gaps
- **Requirements**: `REQUIREMENTS.md` - Original project requirements
- **Implementation Plan**: `docs/implementation-plan.md` - Overall 6-week timeline

## Current State Summary
Phase 2 delivered a fully functional ESXi orchestrator with:
- ✅ Govmomi-based ESXi connection
- ✅ Cloud-init guestinfo injection for Ubuntu
- ✅ All core VM operations (create, clone, delete, info, list)
- ✅ SSH fallback mechanism
- ✅ Configuration management
- ⚠️ 0% test coverage (needs immediate attention)
- ⚠️ No backup functionality yet
- ⚠️ No monitoring/metrics
- ⚠️ Security features pending

## Phase 3 Objectives (Week 3-4)

### Week 3: Testing Infrastructure & Backup System
1. **Comprehensive test suite with 70%+ coverage**
2. **Integration tests for VM lifecycle**
3. **Chaos testing framework**
4. **Backup/restore functionality**
5. **BoltDB catalog implementation**

### Week 4: Monitoring, Security & Hardening
1. **Prometheus metrics integration**
2. **AI agent sandboxing**
3. **Audit logging with zerolog**
4. **Performance optimization**
5. **Production documentation**

## Specific Implementation Tasks

### Priority 1: Test Coverage (URGENT)
**Goal**: Achieve 70% test coverage across all packages

#### Task 1.1: Unit Tests for Core Components
Create comprehensive unit tests for:

**File**: `pkg/esxi/client/client_test.go`
```go
// Test cases needed:
// - Connection establishment with valid/invalid credentials
// - VM listing with mock responses
// - Resource pool/datastore/folder retrieval
// - Error handling for network failures
// - SSL certificate validation scenarios
```

**File**: `pkg/cloudinit/builder_test.go`
```go
// Test cases needed:
// - Metadata generation for different network configs
// - Userdata with various SSH key formats
// - Gzip+base64 encoding validation
// - DHCP vs static IP configurations
// - Invalid input handling
```

**File**: `pkg/esxi/vm/operations_test.go`
```go
// Test cases needed:
// - CreateFromTemplate with mock govmomi client
// - CloneVM with various configurations
// - Delete operations with powered on/off VMs
// - PowerOn/PowerOff state transitions
// - GetVMInfo data parsing
```

#### Task 1.2: Integration Tests
**File**: `test/integration/vm_lifecycle_test.go`
```go
// Implement full lifecycle tests:
// 1. Create VM from template
// 2. Verify cloud-init applied correctly
// 3. Clone VM with new IP
// 4. Power cycle operations
// 5. Delete both VMs
// Note: Use build tags for ESXi-dependent tests
```

#### Task 1.3: Chaos Tests
**File**: `test/chaos/failure_scenarios_test.go`
```go
// Implement chaos scenarios:
// - Network partition during VM creation
// - Datastore full during clone
// - ESXi API timeout handling
// - Concurrent operations safety
// - Invalid template handling
```

### Priority 2: Backup System Implementation

#### Task 2.1: BoltDB Catalog
**File**: `internal/storage/catalog.go`
```go
package storage

import (
    "go.etcd.io/bbolt"
    "time"
)

type BackupCatalog struct {
    db *bbolt.DB
}

type BackupEntry struct {
    ID          string
    VMName      string
    Timestamp   time.Time
    Size        int64
    Location    string
    Checksum    string
    Metadata    map[string]string
}

// Implement:
// - InitCatalog(path string) (*BackupCatalog, error)
// - AddBackup(entry *BackupEntry) error
// - GetBackup(id string) (*BackupEntry, error)
// - ListBackups(vmName string) ([]*BackupEntry, error)
// - DeleteBackup(id string) error
// - ApplyRetention(policy RetentionPolicy) error
```

#### Task 2.2: Backup Operations
**File**: `pkg/backup/operations.go`
```go
package backup

// Implement backup operations:
// - CreateBackup(vm *VM, target BackupTarget) (*BackupID, error)
// - RestoreBackup(backupID string, newName string) error
// - ListBackups() ([]*BackupInfo, error)
// - VerifyBackup(backupID string) error

// Support cold backup initially:
// 1. Power off VM (if requested)
// 2. Export VM to OVF/OVA
// 3. Compress with zstd
// 4. Store to target (datastore initially)
// 5. Update catalog
// 6. Power on VM (if was running)
```

#### Task 2.3: CLI Backup Commands
**File**: `pkg/cli/backup/create.go`
```go
// Implement: ceso backup create <vm-name> [flags]
// Flags:
//   --compress (default: true)
//   --power-off (default: false for cold backup)
//   --target (default: datastore)
```

**File**: `pkg/cli/backup/restore.go`
```go
// Implement: ceso backup restore <backup-id> --as-new <name>
// With re-IP support via cloud-init
```

**File**: `pkg/cli/backup/list.go`
```go
// Implement: ceso backup list [--vm <name>] [--json]
// Show backup catalog with sizes, dates, locations
```

### Priority 3: Monitoring & Metrics

#### Task 3.1: Prometheus Metrics
**File**: `pkg/metrics/collector.go`
```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
)

// Implement metrics:
var (
    VMOperationDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "ceso_vm_operation_duration_seconds",
            Help: "Duration of VM operations",
        },
        []string{"operation", "status"},
    )
    
    VMOperationTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ceso_vm_operation_total",
            Help: "Total number of VM operations",
        },
        []string{"operation", "status"},
    )
    
    BackupSizeBytes = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "ceso_backup_size_bytes",
            Help: "Size of backups in bytes",
        },
        []string{"vm_name"},
    )
)
```

#### Task 3.2: Metrics Integration
Update all operations to record metrics:
```go
// Example in vm/operations.go:
func (o *Operations) CreateFromTemplate(...) error {
    timer := prometheus.NewTimer(metrics.VMOperationDuration.WithLabelValues("create", "pending"))
    defer func() {
        timer.ObserveDuration()
        metrics.VMOperationTotal.WithLabelValues("create", "success").Inc()
    }()
    // ... existing code
}
```

### Priority 4: Security Implementation

#### Task 4.1: AI Agent Sandboxing
**File**: `pkg/security/sandbox.go`
```go
package security

type OperationMode string

const (
    ModeRestricted   OperationMode = "restricted"   // Read-only + dry-run
    ModeStandard     OperationMode = "standard"     // Normal operations
    ModeUnrestricted OperationMode = "unrestricted" // Full access
)

type Sandbox struct {
    Mode           OperationMode
    AllowedOps     []string
    DryRun         bool
    PromotionToken string
    ExpiresAt      time.Time
}

// Implement:
// - CheckOperation(op string) error
// - PromoteTemporary(duration time.Duration) (*PromotionToken, error)
// - EnforceRestrictions() error
```

#### Task 4.2: Audit Logging
**File**: `pkg/audit/logger.go`
```go
package audit

import "github.com/rs/zerolog"

// Implement structured audit logging:
// - All operations logged with user, timestamp, parameters
// - Secret redaction (passwords, keys)
// - Append-only log file
// - JSON format for analysis
// - Operation correlation IDs
```

### Priority 5: Performance Optimization

#### Task 5.1: Connection Pooling
**File**: `pkg/esxi/client/pool.go`
```go
// Implement connection pooling for govmomi clients:
// - Reuse authenticated sessions
// - Handle session expiry/renewal
// - Concurrent operation support
// - Connection limits
```

#### Task 5.2: Parallel Operations
Update commands to support parallel execution where safe:
```go
// Example: Parallel VM info gathering
func ListVMsDetailed(vms []string) ([]*VMInfo, error) {
    var wg sync.WaitGroup
    results := make([]*VMInfo, len(vms))
    errors := make([]error, len(vms))
    
    for i, vm := range vms {
        wg.Add(1)
        go func(idx int, name string) {
            defer wg.Done()
            results[idx], errors[idx] = GetVMInfo(name)
        }(i, vm)
    }
    
    wg.Wait()
    // ... handle results and errors
}
```

## Test Execution Plan

### Unit Test Implementation Order
1. **Day 1**: Cloud-init builder tests (critical path)
2. **Day 1**: VM operations tests (core functionality)
3. **Day 2**: ESXi client tests (connection layer)
4. **Day 2**: CLI command tests (user interface)
5. **Day 3**: Backup operations tests
6. **Day 3**: Integration test suite

### Test Coverage Goals
```bash
# Run coverage after each component:
go test -cover ./pkg/...

# Target coverage by package:
# - pkg/cloudinit: 90% (critical)
# - pkg/esxi/vm: 80% (core ops)
# - pkg/esxi/client: 70% (external dep)
# - pkg/cli: 60% (UI layer)
# - pkg/backup: 80% (data critical)
```

## Configuration Updates

Add to `config-example.yaml`:
```yaml
# Backup settings
backup:
  default_target: datastore
  compression: zstd
  retention:
    keep_last: 5
    keep_daily: 7
    keep_weekly: 4
    keep_monthly: 12

# Monitoring
metrics:
  enabled: true
  port: 9090
  path: /metrics

# Security
security:
  mode: standard  # restricted, standard, unrestricted
  audit_log: /var/log/ceso/audit.json
  ip_allowlist:
    - 192.168.1.0/24
    - 10.0.0.0/8
```

## Success Criteria for Phase 3

### Week 3 Deliverables
- [ ] Unit test coverage ≥70%
- [ ] Integration tests passing
- [ ] At least 3 chaos scenarios
- [ ] Backup create/restore working
- [ ] BoltDB catalog operational
- [ ] Retention policies implemented

### Week 4 Deliverables
- [ ] Prometheus metrics exposed
- [ ] Grafana dashboard template
- [ ] AI sandbox implemented
- [ ] Audit logging functional
- [ ] Performance targets met
- [ ] Production documentation

## Testing Commands

```bash
# Build and test
go build ./cmd/ceso
go test ./... -v

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Integration tests (requires ESXi)
export ESXI_HOST=192.168.1.100
export ESXI_USER=root
export ESXI_PASSWORD=password
go test -tags=integration ./test/integration/...

# Chaos tests
go test -tags=chaos ./test/chaos/...

# Benchmark performance
go test -bench=. ./test/benchmark/...

# Run with metrics
./ceso --metrics-enabled vm create test-vm --template ubuntu-22.04

# Check metrics endpoint
curl http://localhost:9090/metrics | grep ceso_
```

## Common Issues & Solutions

### Issue: Test Coverage Low
**Solution**: Focus on happy path first, then error cases. Use table-driven tests for comprehensive coverage.

### Issue: ESXi Connection in Tests
**Solution**: Use interface-based design with mock implementations. Only integration tests should require real ESXi.

### Issue: Backup Performance
**Solution**: Implement streaming compression, parallel disk export, and progress reporting.

### Issue: Metric Cardinality
**Solution**: Limit labels to essential dimensions (operation, status). Avoid high-cardinality labels like VM names in histograms.

## Deliverables Checklist

### Testing (Week 3)
- [ ] Unit tests with 70%+ coverage
- [ ] Integration test suite
- [ ] Chaos test scenarios
- [ ] Test documentation
- [ ] CI/CD integration

### Backup System (Week 3)
- [ ] BoltDB catalog
- [ ] Backup operations
- [ ] Restore with re-IP
- [ ] CLI commands
- [ ] Retention policies

### Monitoring (Week 4)
- [ ] Prometheus metrics
- [ ] Grafana dashboard
- [ ] Performance tracking
- [ ] SLI/SLO definitions

### Security (Week 4)
- [ ] AI agent sandbox
- [ ] Audit logging
- [ ] Secret management
- [ ] IP allowlisting

### Documentation (Week 4)
- [ ] User guide
- [ ] API documentation
- [ ] Troubleshooting guide
- [ ] Performance tuning

## Important Notes

1. **Test First**: Write tests before implementing features
2. **Mock External Deps**: Don't require ESXi for unit tests
3. **Incremental Coverage**: Build coverage gradually, focus on critical paths
4. **Document Failures**: When tests fail, document why and how to fix
5. **Performance Baseline**: Establish benchmarks before optimization

## Next Steps After Phase 3

Once Phase 3 is complete, the project will be ready for:
1. Production deployment
2. Phase 4 advanced features (PCI passthrough, vCenter support)
3. Community release and documentation
4. Performance optimization based on real-world usage

## Summary

Phase 3 is critical for production readiness. The primary focus is:
1. **Testing**: Achieve 70% coverage minimum
2. **Backup**: Implement reliable backup/restore
3. **Monitoring**: Add observability
4. **Security**: Implement sandboxing and audit
5. **Documentation**: Complete user and API docs

This brief provides everything needed to complete Phase 3. The test coverage gap is the highest priority, followed by the backup system implementation.