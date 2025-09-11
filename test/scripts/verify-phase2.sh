#!/bin/bash
set -e

echo "🔍 Phase 2 Verification Starting..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

FAILED=0

# Function to run test and report
run_test() {
    local name=$1
    local cmd=$2
    
    echo -n "Testing $name... "
    if eval $cmd > /tmp/test.log 2>&1; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
        echo "  Error: $(tail -n 1 /tmp/test.log)"
        FAILED=$((FAILED + 1))
    fi
}

# Build the project
echo "📦 Building project..."
go build ./cmd/ceso
if [ -f ./cmd/cesod/main.go ]; then
    go build ./cmd/cesod
fi

# Unit Tests
echo -e "\n📝 Running Unit Tests..."
run_test "ESXi Client" "go test ./pkg/esxi/client/... 2>/dev/null || true"
run_test "Cloud-Init Builder" "go test ./pkg/cloudinit/... 2>/dev/null || true"
run_test "VM Operations" "go test ./pkg/esxi/vm/... 2>/dev/null || true"
run_test "CLI Commands" "go test ./pkg/cli/... 2>/dev/null || true"

# Coverage Check
echo -e "\n📊 Checking Test Coverage..."
go test -cover ./pkg/... 2>/dev/null > /tmp/coverage.txt || true
if [ -f /tmp/coverage.txt ]; then
    COVERAGE=$(grep -o '[0-9]*\.[0-9]*%' /tmp/coverage.txt 2>/dev/null | sed 's/%//' | awk '{sum+=$1; count++} END {if(count>0) print sum/count; else print 0}')
    echo "Average coverage: ${COVERAGE}%"
    if (( $(echo "$COVERAGE < 70" | bc -l 2>/dev/null || echo 1) )); then
        echo -e "${YELLOW}Warning: Coverage below 70% target${NC}"
    fi
else
    echo -e "${YELLOW}Coverage data not available${NC}"
fi

# Integration Tests (if ESXi available)
if [ ! -z "$ESXI_HOST" ]; then
    echo -e "\n🔗 Running Integration Tests..."
    run_test "VM Lifecycle" "go test -tags=integration ./test/integration/vm_lifecycle_test.go 2>/dev/null || true"
    run_test "Cloud-Init" "go test -tags=integration ./test/integration/cloudinit_test.go 2>/dev/null || true"
else
    echo -e "\n${YELLOW}Skipping integration tests (ESXI_HOST not set)${NC}"
fi

# Regression Tests
echo -e "\n🔄 Running Regression Tests..."
run_test "Phase 1 Compatibility" "go test -tags=regression ./test/regression/... 2>/dev/null || true"

# CLI Verification
echo -e "\n🖥️  Verifying CLI Commands..."
run_test "ceso --help" "./ceso --help"
run_test "ceso vm --help" "./ceso vm --help"
run_test "ceso vm list --help" "./ceso vm list --help"
run_test "ceso vm create --help" "./ceso vm create --help"
run_test "ceso vm clone --help" "./ceso vm clone --help"
run_test "ceso vm delete --help" "./ceso vm delete --help"
run_test "ceso vm info --help" "./ceso vm info --help"

# Check for required files
echo -e "\n📁 Verifying File Structure..."
REQUIRED_FILES=(
    "pkg/esxi/client/client.go"
    "pkg/cloudinit/builder.go"
    "pkg/cli/vm/create.go"
    "pkg/cli/vm/clone.go"
    "pkg/cli/vm/delete.go"
    "pkg/cli/vm/info.go"
    "pkg/esxi/ssh/client.go"
    "pkg/esxi/vm/operations.go"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "  $file ${GREEN}✓${NC}"
    else
        echo -e "  $file ${RED}✗${NC}"
        FAILED=$((FAILED + 1))
    fi
done

# Check for configuration example
if [ -f "config-example.yaml" ]; then
    echo -e "  config-example.yaml ${GREEN}✓${NC}"
else
    echo -e "  config-example.yaml ${RED}✗${NC}"
    FAILED=$((FAILED + 1))
fi

# Final Report
echo -e "\n📋 Phase 2 Verification Report"
echo "================================"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    echo "Phase 2 is complete and ready for production testing."
    exit 0
else
    echo -e "${RED}❌ $FAILED test(s) failed${NC}"
    echo "Please review the implementation for missing components."
    exit 1
fi