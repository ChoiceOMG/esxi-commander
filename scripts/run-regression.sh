#!/bin/bash
# Regression Test Runner
# Runs all regression tests to ensure no functionality has been broken

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "========================================="
echo "    ESXi Commander Regression Tests     "
echo "========================================="
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Build the binary first
echo "Building binary..."
go build -o ceso ./cmd/ceso
if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to build binary${NC}"
    exit 1
fi

echo ""
echo "Running regression tests..."
echo "============================"

# Run verification tests
echo ""
echo "1. Phase 1 Verification Tests"
echo "------------------------------"
go test ./test/verification/... -v 2>&1 | grep -E "^(=== RUN|--- PASS|--- FAIL|PASS|FAIL|ok)" || true

# Run regression tests
echo ""
echo "2. CLI Regression Tests"
echo "------------------------"
go test ./test/regression/... -v 2>&1 | grep -E "^(=== RUN|--- PASS|--- FAIL|PASS|FAIL|ok)" || true

# Run integration test helpers
echo ""
echo "3. Integration Test Setup"
echo "--------------------------"
go test ./test/integration/... -v 2>&1 | grep -E "^(=== RUN|--- PASS|--- FAIL|PASS|FAIL|ok)" || true

# Run benchmarks (quick mode)
echo ""
echo "4. Performance Benchmarks"
echo "--------------------------"
go test ./test/benchmark/... -bench=. -benchtime=1s -run=^$ 2>&1 | grep -E "^(Benchmark|PASS|FAIL|ok)" || true

# Run unit tests
echo ""
echo "5. Unit Tests"
echo "--------------"
go test ./test/unit/... -v 2>&1 | grep -E "^(=== RUN|--- PASS|--- FAIL|PASS|FAIL|ok)" || true

# Summary
echo ""
echo "========================================="
echo "           TEST SUMMARY                 "
echo "========================================="

# Count test results
TOTAL_PACKAGES=$(go list ./test/... | wc -l)
PASSED_PACKAGES=$(go test ./test/... 2>&1 | grep -c "ok" || true)
FAILED_PACKAGES=$((TOTAL_PACKAGES - PASSED_PACKAGES))

if [ $FAILED_PACKAGES -eq 0 ]; then
    echo -e "${GREEN}✓ All regression tests passed${NC}"
    echo "  Packages tested: $TOTAL_PACKAGES"
    echo "  All packages passed"
    
    # Run quick smoke test
    echo ""
    echo "Running smoke test..."
    ./ceso vm list >/dev/null 2>&1
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Smoke test passed${NC}"
    else
        echo -e "${RED}✗ Smoke test failed${NC}"
    fi
    
    exit 0
else
    echo -e "${RED}✗ Some regression tests failed${NC}"
    echo "  Packages tested: $TOTAL_PACKAGES"
    echo "  Passed: $PASSED_PACKAGES"
    echo "  Failed: $FAILED_PACKAGES"
    exit 1
fi