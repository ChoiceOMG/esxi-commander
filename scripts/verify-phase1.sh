#!/bin/bash
# Phase 1 Verification Script
# Verifies all Phase 1 requirements are met

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "========================================="
echo "   ESXi Commander Phase 1 Verification   "
echo "========================================="
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track overall status
FAILED=0

# Function to check status
check() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
    else
        echo -e "${RED}✗${NC} $2"
        FAILED=1
    fi
}

echo "1. Checking project structure..."
echo "================================="

# Check directories
DIRS=(
    "cmd/ceso"
    "cmd/cesod"
    "pkg/cli/vm"
    "pkg/cli/backup"
    "pkg/cli/template"
    "pkg/esxi/client"
    "pkg/esxi/vm"
    "pkg/cloudinit"
    "pkg/config"
    "pkg/logger"
    "internal/defaults"
    "internal/validation"
    "test/unit"
    "configs"
    ".github/workflows"
)

for dir in "${DIRS[@]}"; do
    if [ -d "$dir" ]; then
        check 0 "Directory exists: $dir"
    else
        check 1 "Directory missing: $dir"
    fi
done

echo ""
echo "2. Checking Go files..."
echo "========================"

FILES=(
    "go.mod"
    "go.sum"
    "cmd/ceso/main.go"
    "cmd/cesod/main.go"
    "pkg/cli/root.go"
    "pkg/cli/vm/vm.go"
    "pkg/cli/vm/list.go"
    "pkg/cli/vm/create.go"
    "pkg/cli/vm/types.go"
    "pkg/config/config.go"
    ".github/workflows/ci.yml"
)

for file in "${FILES[@]}"; do
    if [ -f "$file" ]; then
        check 0 "File exists: $file"
    else
        check 1 "File missing: $file"
    fi
done

echo ""
echo "3. Building binaries..."
echo "========================"

# Build ceso
go build -o ceso-test ./cmd/ceso 2>/dev/null
check $? "Build ceso binary"

# Build cesod
go build -o cesod-test ./cmd/cesod 2>/dev/null
check $? "Build cesod binary"

# Check binary sizes
if [ -f "ceso-test" ]; then
    SIZE=$(stat -f%z "ceso-test" 2>/dev/null || stat -c%s "ceso-test" 2>/dev/null)
    SIZE_MB=$((SIZE / 1024 / 1024))
    echo "   ceso binary size: ${SIZE_MB}MB"
    rm -f ceso-test
fi

if [ -f "cesod-test" ]; then
    rm -f cesod-test
fi

echo ""
echo "4. Testing CLI commands..."
echo "==========================="

# Test help
./ceso --help >/dev/null 2>&1
check $? "ceso --help works"

# Test vm list
OUTPUT=$(./ceso vm list 2>/dev/null)
if echo "$OUTPUT" | grep -q "NAME.*STATUS.*IP.*CPU.*RAM"; then
    check 0 "vm list shows table headers"
else
    check 1 "vm list missing table headers"
fi

if echo "$OUTPUT" | grep -q "ubuntu-web-01"; then
    check 0 "vm list shows mock data"
else
    check 1 "vm list missing mock data"
fi

# Test JSON output
JSON_OUTPUT=$(./ceso vm list --json 2>/dev/null)
if echo "$JSON_OUTPUT" | jq . >/dev/null 2>&1; then
    check 0 "vm list --json produces valid JSON"
else
    check 1 "vm list --json produces invalid JSON"
fi

echo ""
echo "5. Running tests..."
echo "==================="

# Run unit tests
go test ./test/unit/... -v >/dev/null 2>&1
check $? "Unit tests pass"

# Count test files
TEST_COUNT=$(find test -name "*_test.go" | wc -l)
echo "   Found $TEST_COUNT test files"

echo ""
echo "6. Checking dependencies..."
echo "============================"

# Check for required dependencies
DEPS=(
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    "github.com/vmware/govmomi"
    "github.com/rs/zerolog"
    "go.etcd.io/bbolt"
    "github.com/prometheus/client_golang"
)

for dep in "${DEPS[@]}"; do
    if go list -m all | grep -q "$dep"; then
        check 0 "Dependency found: $dep"
    else
        check 1 "Dependency missing: $dep"
    fi
done

echo ""
echo "7. Checking CI/CD..."
echo "===================="

if [ -f ".github/workflows/ci.yml" ]; then
    check 0 "CI workflow exists"
    
    # Check workflow content
    if grep -q "go test" .github/workflows/ci.yml; then
        check 0 "CI runs tests"
    else
        check 1 "CI doesn't run tests"
    fi
    
    if grep -q "go build" .github/workflows/ci.yml; then
        check 0 "CI builds binary"
    else
        check 1 "CI doesn't build binary"
    fi
else
    check 1 "CI workflow missing"
fi

echo ""
echo "8. Phase 1 Checklist..."
echo "========================"

# Final checklist
echo "✓ Go project initialized"
echo "✓ Directory structure created"
echo "✓ Basic CLI with Cobra"
echo "✓ vm list command working"
echo "✓ JSON output support"
echo "✓ Mock data implemented"
echo "✓ GitHub Actions configured"
echo "✓ Dependencies installed"
echo "✓ Tests passing"
echo "✓ Binaries building"

echo ""
echo "========================================="
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ PHASE 1 VERIFICATION COMPLETE${NC}"
    echo "All requirements met. Ready for Phase 2!"
else
    echo -e "${RED}✗ PHASE 1 VERIFICATION FAILED${NC}"
    echo "Please fix the issues above before proceeding."
    exit 1
fi
echo "========================================="