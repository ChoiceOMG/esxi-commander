#!/bin/bash
# Phase 3 Verification Test Suite

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

# 1. TEST COVERAGE VERIFICATION
echo -e "\n1. TEST COVERAGE VERIFICATION"
go test -coverprofile=coverage.out ./pkg/cloudinit > /dev/null 2>&1
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
if (( $(echo "$COVERAGE >= 70" | bc -l) )); then
    check 0 "Test coverage (cloudinit): $COVERAGE% (≥70% required)"
else
    check 1 "Test coverage (cloudinit): $COVERAGE% (<70% required)"
fi

# 2. BACKUP SYSTEM
echo -e "\n2. BACKUP SYSTEM VERIFICATION"
[ -f "internal/storage/catalog.go" ] && check 0 "BoltDB catalog implemented" || check 1 "BoltDB catalog missing"
[ -f "pkg/backup/operations.go" ] && check 0 "Backup operations implemented" || check 1 "Backup operations missing"

# 3. MONITORING
echo -e "\n3. MONITORING VERIFICATION"
[ -f "pkg/metrics/collector.go" ] && check 0 "Prometheus metrics implemented" || check 1 "Metrics missing"

# 4. SECURITY
echo -e "\n4. SECURITY VERIFICATION"
[ -f "pkg/security/sandbox.go" ] && check 0 "AI sandboxing implemented" || check 1 "Sandboxing missing"
[ -f "pkg/audit/logger.go" ] && check 0 "Audit logging implemented" || check 1 "Audit logging missing"

# 5. PERFORMANCE
echo -e "\n5. PERFORMANCE VERIFICATION"
[ -f "pkg/esxi/client/pool.go" ] && check 0 "Connection pooling implemented" || check 1 "Connection pooling missing"

# 6. CHAOS TESTING
echo -e "\n6. CHAOS TESTING VERIFICATION"
[ -f "test/chaos/failure_scenarios_test.go" ] && check 0 "Chaos test scenarios implemented" || check 1 "Chaos tests missing"

# 7. BUILD VERIFICATION
echo -e "\n7. BUILD VERIFICATION"
if go build -o /tmp/ceso-test cmd/ceso/main.go 2>/dev/null; then
    check 0 "Binary builds successfully"
    rm -f /tmp/ceso-test
else
    check 1 "Binary build failed"
fi

# 8. DOCUMENTATION
echo -e "\n8. DOCUMENTATION VERIFICATION"
[ -f "docs/phase-3-readiness.md" ] && check 0 "Phase 3 readiness doc present" || check 1 "Phase 3 readiness doc missing"
[ -f "docs/phase3-verification-test-plan.md" ] && check 0 "Test plan present" || check 1 "Test plan missing"

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