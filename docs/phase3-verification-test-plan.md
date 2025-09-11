# Phase 3 Verification Test Plan

## Executive Summary

This test plan verifies the successful completion of Phase 3, which transforms ESXi Commander from a functional Phase 2 implementation into a production-ready system with comprehensive testing, backup capabilities, monitoring, and enterprise-grade security.

## Objectives

Phase 3 must deliver:
1. **Test Coverage**: ≥70% across all packages
2. **Backup System**: Full backup/restore functionality with BoltDB catalog
3. **Monitoring**: Prometheus metrics and Grafana dashboards
4. **Security**: AI agent sandboxing and audit logging
5. **Documentation**: Complete user and API documentation

## Test Execution Timeline

- **Duration**: 2 days comprehensive testing
- **Prerequisites**: Phase 2 complete, ESXi test environment available
- **Test Environment**: Ubuntu 22.04/24.04, ESXi 7.x/8.x

## Section 1: Test Coverage Verification

### 1.1 Coverage Requirements

| Package | Minimum Coverage | Priority | Critical Functions |
|---------|-----------------|----------|-------------------|
| `pkg/cloudinit` | 90% | Critical | Metadata/userdata generation, encoding |
| `pkg/esxi/vm` | 80% | Critical | Create, clone, delete operations |
| `pkg/backup` | 80% | Critical | Backup, restore, catalog operations |
| `pkg/esxi/client` | 70% | High | Connection, authentication, API calls |
| `pkg/security` | 75% | High | Sandboxing, audit logging |
| `pkg/cli` | 60% | Medium | Command parsing, output formatting |
| `pkg/metrics` | 70% | Medium | Metric collection, export |
| **Overall** | **70%** | **Required** | **All packages combined** |

### 1.2 Unit Test Verification

#### Required Test Files
```
✓ pkg/esxi/client/client_test.go
✓ pkg/cloudinit/builder_test.go
✓ pkg/esxi/vm/operations_test.go
✓ pkg/cli/vm/create_test.go
✓ pkg/cli/vm/clone_test.go
✓ pkg/cli/vm/delete_test.go
✓ pkg/cli/vm/list_test.go
✓ pkg/cli/vm/info_test.go
✓ pkg/backup/operations_test.go
✓ pkg/backup/catalog_test.go
✓ pkg/security/sandbox_test.go
✓ pkg/audit/logger_test.go
✓ pkg/metrics/collector_test.go
```

#### Test Categories
1. **Happy Path Tests**: Normal operation scenarios
2. **Error Handling**: Invalid inputs, failures
3. **Edge Cases**: Boundary conditions, limits
4. **Concurrency**: Parallel operations, race conditions
5. **Mocking**: External dependencies properly mocked

### 1.3 Integration Test Suite

#### VM Lifecycle Tests
```go
// test/integration/vm_lifecycle_test.go
// Test complete VM lifecycle:
1. Create VM from template with cloud-init
2. Verify IP assignment and SSH access
3. Clone VM with new IP
4. Verify clone has unique MAC/UUID
5. Power operations (on/off/reset)
6. Delete both VMs
7. Verify cleanup complete
```

#### Backup/Restore Tests
```go
// test/integration/backup_test.go
// Test backup and restore:
1. Create test VM
2. Create cold backup
3. Verify backup in catalog
4. Delete original VM
5. Restore from backup with new name
6. Verify restored VM functionality
7. Test retention policy
```

### 1.4 Chaos Test Scenarios

| Scenario | Test Case | Expected Behavior |
|----------|-----------|-------------------|
| Network Partition | API timeout during VM create | Graceful failure, cleanup |
| Datastore Full | Backup fails due to space | Error reported, no corruption |
| Concurrent Ops | Multiple VM creates | All succeed or queue properly |
| Invalid Template | Create from missing template | Clear error message |
| Power Failure | Operation interrupted | Idempotent retry succeeds |
| API Rate Limit | Too many requests | Backoff and retry |
| Lock Contention | Concurrent catalog access | Serialized correctly |

### 1.5 Coverage Verification Process

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out > coverage-summary.txt

# Check overall coverage
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
if (( $(echo "$COVERAGE < 70" | bc -l) )); then
    echo "FAIL: Coverage $COVERAGE% is below 70% requirement"
    exit 1
fi

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html
```

## Section 2: Backup System Verification

### 2.1 BoltDB Catalog Tests

| Test Case | Verification | Pass Criteria |
|-----------|--------------|---------------|
| Catalog Creation | DB file created, buckets initialized | No errors, file exists |
| Add Backup Entry | Entry persisted with all fields | Query returns entry |
| List Backups | Filter by VM name, date range | Correct results |
| Delete Entry | Entry removed, space reclaimed | Entry not found |
| Concurrent Access | Multiple readers/writers | No corruption |
| Retention Policy | Old backups removed per policy | Count matches policy |

### 2.2 Backup Operations

#### Cold Backup Test
```bash
# Create backup
./ceso backup create test-vm --compress --target datastore

# Verify:
- VM powered off (if running)
- OVF/OVA exported
- Compression applied (zstd)
- Catalog entry created
- Checksum calculated
- VM powered on (if was running)
```

#### Restore Test
```bash
# Restore with new name
./ceso backup restore <backup-id> --as-new restored-vm --ip 192.168.1.200/24

# Verify:
- New VM created with unique UUID
- Cloud-init updated with new IP
- VM boots successfully
- SSH access with new IP
- Original backup unchanged
```

### 2.3 Performance Metrics

| Operation | Target | Measurement |
|-----------|--------|-------------|
| Backup 40GB VM | <10 min | Time from start to catalog update |
| Restore 40GB VM | <10 min | Time from command to VM ready |
| Catalog Query | <100ms | List 100 backups |
| Compression Ratio | >2:1 | Compressed size / original size |

## Section 3: Monitoring & Metrics Verification

### 3.1 Prometheus Metrics

#### Required Metrics
```
# VM Operations
ceso_vm_operation_duration_seconds{operation="create",status="success"}
ceso_vm_operation_total{operation="create",status="success"}
ceso_vm_operation_total{operation="create",status="failure"}

# Backup Metrics
ceso_backup_size_bytes{vm_name="test-vm"}
ceso_backup_duration_seconds{operation="create"}
ceso_backup_total{operation="create",status="success"}

# AI Agent Metrics
ceso_ai_agent_operations_total{mode="restricted",operation="vm_list"}
ceso_ai_agent_promotion_total{from="restricted",to="standard"}

# System Metrics
ceso_esxi_connection_total{status="success"}
ceso_esxi_connection_duration_seconds
```

#### Metrics Validation
```bash
# Start with metrics enabled
./ceso serve --metrics-port 9090 &

# Perform operations
./ceso vm list
./ceso vm create test-vm --template ubuntu-22.04

# Verify metrics
curl -s http://localhost:9090/metrics | grep ceso_ > metrics.txt

# Check each required metric exists
for metric in vm_operation_duration vm_operation_total backup_size; do
    grep -q "ceso_${metric}" metrics.txt || echo "MISSING: $metric"
done
```

### 3.2 Grafana Dashboard

#### Dashboard Components
1. **VM Operations Panel**: Create/clone/delete rates and latencies
2. **Backup Status Panel**: Backup sizes, success rates, retention
3. **System Health Panel**: API calls, errors, connection status
4. **AI Agent Activity**: Operations by mode, promotions
5. **Performance Panel**: P50/P95/P99 latencies

#### Validation
```json
// Verify dashboard JSON includes:
{
  "panels": [
    {
      "title": "VM Operation Rate",
      "targets": [
        {
          "expr": "rate(ceso_vm_operation_total[5m])"
        }
      ]
    }
  ]
}
```

## Section 4: Security Features Verification

### 4.1 AI Agent Sandboxing

| Mode | Allowed Operations | Denied Operations | Test Case |
|------|-------------------|-------------------|-----------|
| Restricted | Read, List, Dry-run | Create, Delete, Modify | Attempt create, verify denied |
| Standard | All normal operations | Destructive bulk ops | Normal workflow succeeds |
| Unrestricted | All operations | None | Bulk delete allowed |

#### Sandbox Test Script
```bash
# Test restricted mode
export CESO_SECURITY_MODE=restricted

# Should succeed (read-only)
./ceso vm list
./ceso vm info test-vm

# Should fail (write operation)
./ceso vm create new-vm --template ubuntu-22.04 # Expected: Permission denied

# Test promotion
./ceso admin promote --duration 5m --token
# Verify temporary elevation works
```

### 4.2 Audit Logging

#### Required Log Fields
```json
{
  "timestamp": "2024-01-01T00:00:00Z",
  "operation": "vm.create",
  "user": "ai-agent",
  "mode": "standard",
  "parameters": {
    "name": "test-vm",
    "template": "ubuntu-22.04"
  },
  "result": "success",
  "duration_ms": 45000,
  "correlation_id": "uuid-here"
}
```

#### Secret Redaction Test
```bash
# Perform operation with password
./ceso vm create test --password "secret123"

# Check audit log
grep -i "secret123" /var/log/ceso/audit.json
# Expected: No results (password redacted)

grep "password" /var/log/ceso/audit.json
# Expected: Shows "password": "[REDACTED]"
```

### 4.3 IP Allowlisting

```yaml
# Test configuration
security:
  ip_allowlist:
    - 192.168.1.0/24
    - 10.0.0.100/32

# Test from allowed IP
curl -X POST http://192.168.1.50:8080/api/vm/create # Success

# Test from denied IP
curl -X POST http://172.16.0.1:8080/api/vm/create # 403 Forbidden
```

## Section 5: Performance Benchmarks

### 5.1 Operation Timing Requirements

| Operation | Target | Test Command | Measurement |
|-----------|--------|--------------|-------------|
| VM Create | <90s | `time ./ceso vm create bench-vm --template ubuntu-22.04` | Real time |
| VM Clone (80GB) | <5min | `time ./ceso vm clone large-vm clone-vm` | Real time |
| VM List (100 VMs) | <2s | `time ./ceso vm list` | Real time |
| VM Delete | <30s | `time ./ceso vm delete bench-vm` | Real time |
| Backup (40GB) | <10min | `time ./ceso backup create medium-vm` | Real time |

### 5.2 Resource Usage Limits

```bash
# Memory usage test
./ceso serve &
PID=$!
sleep 10

# Check idle memory (should be <500MB)
ps -o pid,vsz,rss,comm -p $PID

# CPU usage test (should be <5% idle)
top -b -n 1 -p $PID | tail -1 | awk '{print $9}'
```

### 5.3 Concurrent Operations

```bash
# Test parallel VM operations
for i in {1..10}; do
    ./ceso vm create "concurrent-$i" --template ubuntu-22.04 &
done
wait

# Verify all succeeded
./ceso vm list | grep concurrent | wc -l
# Expected: 10
```

## Section 6: Documentation Verification

### 6.1 Required Documentation

| Document | Location | Completeness Check |
|----------|----------|-------------------|
| User Guide | `docs/user-guide.md` | All commands documented |
| API Reference | `docs/api-reference.md` | All public functions |
| Troubleshooting | `docs/troubleshooting.md` | Common issues covered |
| Configuration | `docs/configuration.md` | All options explained |
| Backup Guide | `docs/backup-restore.md` | Procedures documented |
| Security Guide | `docs/security.md` | Sandbox, audit, IP filtering |

### 6.2 Code Documentation

```bash
# Check inline documentation
go doc -all ./pkg/... | wc -l
# Expected: >1000 lines

# Verify examples
grep -r "Example" --include="*.go" | wc -l
# Expected: >20 examples
```

## Section 7: Regression Testing

### 7.1 Phase 1 Compatibility

```bash
# Original Phase 1 commands must still work
./ceso --help          # Shows help
./ceso vm --help       # Shows VM commands
./ceso vm list         # Lists VMs (now real, not mock)
./ceso vm list --json  # JSON output
```

### 7.2 Phase 2 Compatibility

```bash
# Phase 2 VM operations
./ceso vm create test --template ubuntu-22.04 --ip 192.168.1.100/24
./ceso vm clone test test-clone --ip 192.168.1.101/24
./ceso vm info test
./ceso vm delete test
```

### 7.3 Feature Integration

| Test | Verification |
|------|--------------|
| VM create with metrics | Metrics updated |
| Backup with audit log | Operations logged |
| Restricted mode with monitoring | Denials counted |
| All features together | No conflicts |

## Section 8: Test Execution Script

```bash
#!/bin/bash
# scripts/verify-phase3.sh

echo "========================================"
echo "    Phase 3 Verification Test Suite    "
echo "========================================"

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

FAILED=0

# Function to check test result
check() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
    else
        echo -e "${RED}✗${NC} $2"
        FAILED=1
    fi
}

# 1. Test Coverage
echo -e "\n1. TEST COVERAGE VERIFICATION"
go test -coverprofile=coverage.out ./... > /dev/null 2>&1
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
if (( $(echo "$COVERAGE >= 70" | bc -l) )); then
    check 0 "Test coverage: $COVERAGE% (≥70% required)"
else
    check 1 "Test coverage: $COVERAGE% (<70% required)"
fi

# 2. Backup System
echo -e "\n2. BACKUP SYSTEM VERIFICATION"
[ -f "internal/storage/catalog.go" ] && check 0 "BoltDB catalog implemented" || check 1 "BoltDB catalog missing"
[ -f "pkg/backup/operations.go" ] && check 0 "Backup operations implemented" || check 1 "Backup operations missing"

# 3. Monitoring
echo -e "\n3. MONITORING VERIFICATION"
[ -f "pkg/metrics/collector.go" ] && check 0 "Prometheus metrics implemented" || check 1 "Metrics missing"

# 4. Security
echo -e "\n4. SECURITY VERIFICATION"
[ -f "pkg/security/sandbox.go" ] && check 0 "AI sandboxing implemented" || check 1 "Sandboxing missing"
[ -f "pkg/audit/logger.go" ] && check 0 "Audit logging implemented" || check 1 "Audit logging missing"

# 5. Documentation
echo -e "\n5. DOCUMENTATION VERIFICATION"
[ -f "docs/user-guide.md" ] && check 0 "User guide present" || check 1 "User guide missing"
[ -f "docs/api-reference.md" ] && check 0 "API reference present" || check 1 "API reference missing"

# Summary
echo -e "\n========================================"
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ PHASE 3 VERIFICATION PASSED${NC}"
    echo "All requirements met. Ready for production!"
else
    echo -e "${RED}✗ PHASE 3 VERIFICATION FAILED${NC}"
    echo "Please complete missing items before proceeding."
    exit 1
fi
```

## Success Criteria Summary

### Must Have (Required for Phase 3 Completion)
- ✅ Test coverage ≥70% overall
- ✅ Unit tests for all core packages
- ✅ Backup/restore functionality working
- ✅ BoltDB catalog operational
- ✅ Prometheus metrics exposed
- ✅ AI agent sandboxing implemented
- ✅ Audit logging functional
- ✅ No regression from Phase 1/2

### Should Have (Recommended)
- ✅ Integration test suite complete
- ✅ 5+ chaos scenarios implemented
- ✅ Grafana dashboard configured
- ✅ Performance targets met
- ✅ Complete user documentation

### Nice to Have (Bonus)
- ✅ Test coverage ≥90% for critical paths
- ✅ Advanced chaos scenarios
- ✅ Performance optimizations verified
- ✅ Video tutorials/demos

## Failure Conditions

Phase 3 is NOT complete if any of these conditions exist:
1. ❌ Test coverage <70%
2. ❌ Backup system non-functional
3. ❌ No metrics endpoint
4. ❌ No audit logging
5. ❌ Security features missing
6. ❌ Major bugs in core functionality
7. ❌ Phase 1/2 features broken

## Final Verification Report Format

```markdown
# Phase 3 Verification Report

## Executive Summary
- Overall Status: [PASS/FAIL]
- Test Coverage: XX%
- Requirements Met: XX/XX
- Critical Issues: None/List

## Detailed Results
[Category-by-category results]

## Recommendations
[Next steps for Phase 4]
```

## Conclusion

This comprehensive test plan ensures Phase 3 delivers a production-ready ESXi Commander with enterprise-grade testing, backup capabilities, monitoring, and security. Execute all sections to verify complete implementation before proceeding to Phase 4.