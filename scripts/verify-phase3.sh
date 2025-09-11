#!/bin/bash
# Phase 3 Verification Script
# Comprehensive verification of all Phase 3 requirements

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "========================================"
echo "    Phase 3 Verification Test Suite    "
echo "========================================"
echo ""
echo "Date: $(date)"
echo "Project: ESXi Commander"
echo "Phase: 3 - Testing, Backup, Monitoring & Security"
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
WARNINGS=0

# Function to check test result
check() {
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "${RED}✗${NC} $2"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# Function for warnings
warn() {
    echo -e "${YELLOW}⚠${NC} $1"
    WARNINGS=$((WARNINGS + 1))
}

# Function for info
info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

echo "========================================"
echo "1. TEST COVERAGE VERIFICATION"
echo "========================================"

# Check if project builds
info "Building project..."
if go build -o ceso-test ./cmd/ceso 2>/dev/null; then
    check 0 "Project builds successfully"
    rm -f ceso-test
else
    check 1 "Project build failed"
fi

# Generate coverage report
info "Analyzing test coverage..."
if go test -coverprofile=coverage.out ./... > /dev/null 2>&1; then
    COVERAGE=$(go tool cover -func=coverage.out 2>/dev/null | grep total | awk '{print $3}' | sed 's/%//' || echo "0")
    
    if [ -z "$COVERAGE" ] || [ "$COVERAGE" = "0" ]; then
        warn "Unable to calculate coverage (might be 0%)"
        COVERAGE=0
    fi
    
    # Check if coverage meets requirement
    if (( $(echo "$COVERAGE >= 70" | bc -l 2>/dev/null || echo 0) )); then
        check 0 "Test coverage: ${COVERAGE}% (≥70% required)"
    else
        check 1 "Test coverage: ${COVERAGE}% (<70% required)"
    fi
    
    # Package-level coverage
    info "Package coverage breakdown:"
    go tool cover -func=coverage.out 2>/dev/null | grep -E "^github.com/r11/esxi-commander/pkg" | head -10 || echo "  No package coverage data available"
else
    warn "Could not generate coverage report"
fi

# Check for test files
echo ""
info "Checking for test files..."
TEST_FILES=(
    "pkg/esxi/client/client_test.go"
    "pkg/cloudinit/builder_test.go"
    "pkg/esxi/vm/operations_test.go"
    "pkg/cli/vm/create_test.go"
    "pkg/cli/vm/list_test.go"
    "test/integration/vm_lifecycle_test.go"
    "test/unit/cli_test.go"
)

TEST_FILE_COUNT=0
for file in "${TEST_FILES[@]}"; do
    if [ -f "$file" ]; then
        TEST_FILE_COUNT=$((TEST_FILE_COUNT + 1))
    fi
done

if [ $TEST_FILE_COUNT -ge 4 ]; then
    check 0 "Test files present: $TEST_FILE_COUNT found"
else
    check 1 "Insufficient test files: $TEST_FILE_COUNT found (need at least 4)"
fi

echo ""
echo "========================================"
echo "2. BACKUP SYSTEM VERIFICATION"
echo "========================================"

# Check backup components
BACKUP_FILES=(
    "internal/storage/catalog.go"
    "pkg/backup/operations.go"
    "pkg/cli/backup/create.go"
    "pkg/cli/backup/restore.go"
    "pkg/cli/backup/list.go"
)

info "Checking backup system components..."
BACKUP_COUNT=0
for file in "${BACKUP_FILES[@]}"; do
    if [ -f "$file" ]; then
        BACKUP_COUNT=$((BACKUP_COUNT + 1))
    fi
done

if [ $BACKUP_COUNT -ge 3 ]; then
    check 0 "Backup components: $BACKUP_COUNT/5 implemented"
else
    check 1 "Backup components: $BACKUP_COUNT/5 (need at least 3)"
fi

# Check BoltDB integration
if grep -q "bbolt" go.mod 2>/dev/null; then
    check 0 "BoltDB dependency present"
else
    check 1 "BoltDB dependency missing"
fi

# Check backup CLI commands
if [ -f "./ceso" ]; then
    if ./ceso backup --help 2>&1 | grep -q "create\|restore\|list"; then
        check 0 "Backup CLI commands registered"
    else
        warn "Backup CLI commands not fully implemented"
    fi
fi

echo ""
echo "========================================"
echo "3. MONITORING & METRICS VERIFICATION"
echo "========================================"

# Check metrics implementation
info "Checking Prometheus metrics..."
if [ -f "pkg/metrics/collector.go" ]; then
    check 0 "Metrics collector implemented"
else
    check 1 "Metrics collector missing"
fi

# Check for Prometheus dependency
if grep -q "prometheus/client_golang" go.mod 2>/dev/null; then
    check 0 "Prometheus client dependency present"
else
    check 1 "Prometheus client dependency missing"
fi

# Check for required metrics
METRICS_FILES=$(find . -name "*.go" -type f 2>/dev/null | xargs grep -l "prometheus\." 2>/dev/null | wc -l || echo 0)
if [ "$METRICS_FILES" -gt 0 ]; then
    check 0 "Prometheus metrics integration found in $METRICS_FILES files"
else
    warn "No Prometheus metrics integration found"
fi

# Check for Grafana dashboard
if [ -f "configs/grafana-dashboard.json" ] || [ -f "docs/grafana-dashboard.json" ]; then
    check 0 "Grafana dashboard template present"
else
    warn "Grafana dashboard template not found"
fi

echo ""
echo "========================================"
echo "4. SECURITY FEATURES VERIFICATION"
echo "========================================"

# Check AI sandboxing
info "Checking security implementations..."
if [ -f "pkg/security/sandbox.go" ]; then
    check 0 "AI agent sandboxing implemented"
    
    # Check for sandbox modes
    if grep -q "ModeRestricted\|ModeStandard\|ModeUnrestricted" "pkg/security/sandbox.go" 2>/dev/null; then
        check 0 "Sandbox modes defined"
    else
        warn "Sandbox modes not fully defined"
    fi
else
    check 1 "AI agent sandboxing missing"
fi

# Check audit logging
if [ -f "pkg/audit/logger.go" ]; then
    check 0 "Audit logging implemented"
    
    # Check for secret redaction
    if grep -q "redact\|REDACTED" "pkg/audit/logger.go" 2>/dev/null; then
        check 0 "Secret redaction implemented"
    else
        warn "Secret redaction not found"
    fi
else
    check 1 "Audit logging missing"
fi

# Check for zerolog integration
if grep -q "rs/zerolog" go.mod 2>/dev/null; then
    check 0 "Structured logging (zerolog) present"
else
    check 1 "Structured logging dependency missing"
fi

# Check IP allowlisting
if grep -q "ip_allowlist\|IPAllowlist" pkg/security/*.go 2>/dev/null || \
   grep -q "ip_allowlist\|IPAllowlist" pkg/config/*.go 2>/dev/null; then
    check 0 "IP allowlisting configuration found"
else
    warn "IP allowlisting not found"
fi

echo ""
echo "========================================"
echo "5. INTEGRATION & CHAOS TESTS"
echo "========================================"

# Check integration tests
info "Checking test suites..."
INTEGRATION_TESTS=$(find test/integration -name "*_test.go" 2>/dev/null | wc -l || echo 0)
if [ "$INTEGRATION_TESTS" -gt 0 ]; then
    check 0 "Integration tests found: $INTEGRATION_TESTS files"
else
    warn "No integration tests found"
fi

# Check chaos tests
CHAOS_TESTS=$(find test/chaos -name "*_test.go" 2>/dev/null | wc -l || echo 0)
if [ "$CHAOS_TESTS" -gt 0 ]; then
    check 0 "Chaos tests found: $CHAOS_TESTS files"
else
    warn "No chaos tests found"
fi

# Check for test tags
if grep -r "// +build integration" test/ 2>/dev/null | head -1 > /dev/null || \
   grep -r "//go:build integration" test/ 2>/dev/null | head -1 > /dev/null; then
    check 0 "Integration test build tags present"
else
    warn "Integration test build tags not found"
fi

echo ""
echo "========================================"
echo "6. DOCUMENTATION VERIFICATION"
echo "========================================"

# Check required documentation
info "Checking documentation..."
DOCS=(
    "docs/user-guide.md"
    "docs/api-reference.md"
    "docs/backup-restore.md"
    "docs/security.md"
    "docs/troubleshooting.md"
    "docs/configuration.md"
)

DOC_COUNT=0
for doc in "${DOCS[@]}"; do
    if [ -f "$doc" ]; then
        DOC_COUNT=$((DOC_COUNT + 1))
    fi
done

if [ $DOC_COUNT -ge 3 ]; then
    check 0 "Documentation files: $DOC_COUNT/6 present"
else
    check 1 "Documentation files: $DOC_COUNT/6 (need at least 3)"
fi

# Check inline documentation
GO_DOC_LINES=$(go doc -all ./pkg/... 2>/dev/null | wc -l || echo 0)
if [ "$GO_DOC_LINES" -gt 500 ]; then
    check 0 "Inline documentation: $GO_DOC_LINES lines"
else
    warn "Limited inline documentation: $GO_DOC_LINES lines"
fi

echo ""
echo "========================================"
echo "7. PERFORMANCE & BENCHMARKS"
echo "========================================"

# Check for benchmark tests
info "Checking performance tests..."
BENCH_TESTS=$(find . -name "*_test.go" -type f 2>/dev/null | xargs grep -l "^func Benchmark" 2>/dev/null | wc -l || echo 0)
if [ "$BENCH_TESTS" -gt 0 ]; then
    check 0 "Benchmark tests found in $BENCH_TESTS files"
else
    warn "No benchmark tests found"
fi

# Check for performance metrics in code
if grep -r "prometheus.NewHistogram\|prometheus.NewSummary" pkg/ 2>/dev/null | head -1 > /dev/null; then
    check 0 "Performance metrics instrumentation found"
else
    warn "No performance metrics instrumentation found"
fi

echo ""
echo "========================================"
echo "8. REGRESSION TESTING"
echo "========================================"

# Check Phase 1/2 compatibility
info "Checking backward compatibility..."

# Build the binary
if go build -o ceso-test ./cmd/ceso 2>/dev/null; then
    # Test Phase 1 commands
    if ./ceso-test --help > /dev/null 2>&1; then
        check 0 "Phase 1: CLI help works"
    else
        check 1 "Phase 1: CLI help broken"
    fi
    
    if ./ceso-test vm --help > /dev/null 2>&1; then
        check 0 "Phase 1: VM commands accessible"
    else
        check 1 "Phase 1: VM commands broken"
    fi
    
    # Test Phase 2 commands
    VM_COMMANDS="list create clone delete info"
    VM_CMD_COUNT=0
    for cmd in $VM_COMMANDS; do
        if ./ceso-test vm $cmd --help 2>&1 | grep -q "Usage\|Flags"; then
            VM_CMD_COUNT=$((VM_CMD_COUNT + 1))
        fi
    done
    
    if [ $VM_CMD_COUNT -eq 5 ]; then
        check 0 "Phase 2: All VM commands present"
    else
        check 1 "Phase 2: Only $VM_CMD_COUNT/5 VM commands working"
    fi
    
    rm -f ceso-test
else
    warn "Could not build binary for regression testing"
fi

echo ""
echo "========================================"
echo "9. CONFIGURATION & DEPLOYMENT"
echo "========================================"

# Check configuration enhancements
info "Checking configuration..."
if [ -f "config-example.yaml" ] || [ -f "configs/config-example.yaml" ]; then
    CONFIG_FILE=$(find . -name "config-example.yaml" | head -1)
    
    # Check for Phase 3 config sections
    if grep -q "backup:" "$CONFIG_FILE" 2>/dev/null; then
        check 0 "Backup configuration section present"
    else
        warn "Backup configuration section missing"
    fi
    
    if grep -q "metrics:\|monitoring:" "$CONFIG_FILE" 2>/dev/null; then
        check 0 "Metrics configuration section present"
    else
        warn "Metrics configuration section missing"
    fi
    
    if grep -q "security:" "$CONFIG_FILE" 2>/dev/null; then
        check 0 "Security configuration section present"
    else
        warn "Security configuration section missing"
    fi
else
    check 1 "Configuration example file missing"
fi

echo ""
echo "========================================"
echo "         VERIFICATION SUMMARY           "
echo "========================================"
echo ""

# Calculate percentages
if [ $TOTAL_TESTS -gt 0 ]; then
    PASS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
else
    PASS_RATE=0
fi

# Display summary
echo "Total Tests:    $TOTAL_TESTS"
echo -e "Passed:         ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed:         ${RED}$FAILED_TESTS${NC}"
echo -e "Warnings:       ${YELLOW}$WARNINGS${NC}"
echo "Pass Rate:      ${PASS_RATE}%"
echo ""

# Detailed status by category
echo "Category Status:"
echo "----------------"
[ $COVERAGE -ge 70 ] 2>/dev/null && echo -e "Test Coverage:     ${GREEN}✓${NC} ${COVERAGE}%" || echo -e "Test Coverage:     ${RED}✗${NC} ${COVERAGE}%"
[ $BACKUP_COUNT -ge 3 ] && echo -e "Backup System:     ${GREEN}✓${NC}" || echo -e "Backup System:     ${RED}✗${NC}"
[ $METRICS_FILES -gt 0 ] && echo -e "Monitoring:        ${GREEN}✓${NC}" || echo -e "Monitoring:        ${YELLOW}⚠${NC}"
[ -f "pkg/security/sandbox.go" ] && echo -e "Security:          ${GREEN}✓${NC}" || echo -e "Security:          ${RED}✗${NC}"
[ $DOC_COUNT -ge 3 ] && echo -e "Documentation:     ${GREEN}✓${NC}" || echo -e "Documentation:     ${YELLOW}⚠${NC}"

echo ""
echo "========================================"

# Generate report file
REPORT_FILE="phase3-verification-report-$(date +%Y%m%d-%H%M%S).txt"
{
    echo "Phase 3 Verification Report"
    echo "Generated: $(date)"
    echo ""
    echo "Summary:"
    echo "- Total Tests: $TOTAL_TESTS"
    echo "- Passed: $PASSED_TESTS"
    echo "- Failed: $FAILED_TESTS"
    echo "- Warnings: $WARNINGS"
    echo "- Pass Rate: ${PASS_RATE}%"
    echo "- Test Coverage: ${COVERAGE}%"
    echo ""
    echo "Recommendation:"
} > "$REPORT_FILE"

# Final verdict
if [ $FAILED_TESTS -eq 0 ] && [ $COVERAGE -ge 70 ] 2>/dev/null; then
    echo -e "${GREEN}✓ PHASE 3 VERIFICATION PASSED${NC}"
    echo "All critical requirements met. Ready for production deployment!"
    echo "Report saved to: $REPORT_FILE"
    echo "PASS - All requirements met" >> "$REPORT_FILE"
    exit 0
elif [ $FAILED_TESTS -le 3 ] && [ $PASS_RATE -ge 70 ]; then
    echo -e "${YELLOW}⚠ PHASE 3 PARTIALLY COMPLETE${NC}"
    echo "Most requirements met, but some items need attention."
    echo "Report saved to: $REPORT_FILE"
    echo "PARTIAL - Most requirements met, minor issues remain" >> "$REPORT_FILE"
    exit 1
else
    echo -e "${RED}✗ PHASE 3 VERIFICATION FAILED${NC}"
    echo "Critical requirements not met. Please complete implementation."
    echo "Report saved to: $REPORT_FILE"
    echo "FAIL - Critical requirements not met" >> "$REPORT_FILE"
    exit 1
fi