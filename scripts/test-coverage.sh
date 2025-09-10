#!/bin/bash
# Test Coverage Reporter
# Generates test coverage reports for the project

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "========================================="
echo "   ESXi Commander Test Coverage Report  "
echo "========================================="
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Create coverage directory
mkdir -p coverage

echo "Generating coverage reports..."
echo "==============================="
echo ""

# Run tests with coverage for each package
echo "1. CLI Package Coverage"
go test -coverprofile=coverage/cli.out -covermode=atomic ./pkg/cli/... 2>/dev/null
go tool cover -func=coverage/cli.out | tail -1

echo ""
echo "2. VM Package Coverage"
go test -coverprofile=coverage/vm.out -covermode=atomic ./pkg/cli/vm/... 2>/dev/null
go tool cover -func=coverage/vm.out | tail -1

echo ""
echo "3. Config Package Coverage"
go test -coverprofile=coverage/config.out -covermode=atomic ./pkg/config/... 2>/dev/null
go tool cover -func=coverage/config.out | tail -1 || echo "No tests yet"

echo ""
echo "4. Internal Package Coverage"
go test -coverprofile=coverage/internal.out -covermode=atomic ./internal/... 2>/dev/null
go tool cover -func=coverage/internal.out | tail -1

echo ""
echo "5. Unit Tests Coverage"
go test -coverprofile=coverage/unit.out -covermode=atomic ./test/unit/... 2>/dev/null
go tool cover -func=coverage/unit.out | tail -1

# Combine coverage files
echo ""
echo "Combining coverage reports..."
echo "gocovmerge" coverage/*.out > coverage/combined.out 2>/dev/null || {
    # If gocovmerge is not installed, use a simple approach
    echo "mode: atomic" > coverage/combined.out
    for file in coverage/*.out; do
        if [ "$file" != "coverage/combined.out" ]; then
            tail -n +2 "$file" >> coverage/combined.out 2>/dev/null || true
        fi
    done
}

# Generate HTML report
echo "Generating HTML report..."
go tool cover -html=coverage/combined.out -o coverage/coverage.html 2>/dev/null || {
    # Fallback to first coverage file if combined fails
    go tool cover -html=coverage/cli.out -o coverage/coverage.html 2>/dev/null || true
}

# Calculate overall coverage
echo ""
echo "========================================="
echo "         OVERALL COVERAGE                "
echo "========================================="

# Function to extract coverage percentage
get_coverage() {
    local file=$1
    if [ -f "$file" ]; then
        go tool cover -func="$file" 2>/dev/null | tail -1 | awk '{print $3}' | sed 's/%//'
    else
        echo "0.0"
    fi
}

# Get coverage for main packages
CLI_COV=$(get_coverage "coverage/cli.out")
VM_COV=$(get_coverage "coverage/vm.out")
CONFIG_COV=$(get_coverage "coverage/config.out")
INTERNAL_COV=$(get_coverage "coverage/internal.out")

# Display coverage summary
echo ""
echo "Package Coverage Summary:"
echo "------------------------"
printf "CLI Package:      %6s%%\n" "$CLI_COV"
printf "VM Package:       %6s%%\n" "$VM_COV"
printf "Config Package:   %6s%%\n" "$CONFIG_COV"
printf "Internal Package: %6s%%\n" "$INTERNAL_COV"

# Calculate average (simplified)
if command -v bc >/dev/null 2>&1; then
    AVG=$(echo "scale=1; ($CLI_COV + $VM_COV + $CONFIG_COV + $INTERNAL_COV) / 4" | bc)
    echo "------------------------"
    printf "Average:          %6s%%\n" "$AVG"
    
    # Check against target
    TARGET=70
    if (( $(echo "$AVG >= $TARGET" | bc -l) )); then
        echo ""
        echo -e "${GREEN}✓ Coverage meets target (>=${TARGET}%)${NC}"
    else
        echo ""
        echo -e "${YELLOW}⚠ Coverage below target (<${TARGET}%)${NC}"
    fi
fi

echo ""
echo "========================================="
echo "         COVERAGE BY FILE                "
echo "========================================="
echo ""

# Show detailed coverage for key files
echo "Key Files Coverage:"
echo "-------------------"

KEY_FILES=(
    "pkg/cli/root.go"
    "pkg/cli/vm/list.go"
    "pkg/cli/vm/types.go"
    "pkg/cli/template/validate.go"
    "internal/defaults/defaults.go"
    "internal/validation/validation.go"
)

for file in "${KEY_FILES[@]}"; do
    if [ -f "$file" ]; then
        # Try to get coverage for specific file
        COV=$(go test -cover ./$(dirname "$file") 2>/dev/null | grep -oE '[0-9]+\.[0-9]+%' | head -1 || echo "N/A")
        printf "%-40s %s\n" "$file:" "$COV"
    fi
done

echo ""
echo "========================================="
echo "           UNCOVERED CODE                "
echo "========================================="
echo ""

# Show uncovered lines for critical files
echo "Files with low coverage that need attention:"
echo "--------------------------------------------"

go tool cover -func=coverage/combined.out 2>/dev/null | awk '$3 != "100.0%" && $1 != "total:" {print $1, $3}' | head -10 || {
    echo "Unable to determine uncovered code"
}

echo ""
echo "========================================="
echo "              REPORT FILES               "
echo "========================================="
echo ""

if [ -f "coverage/coverage.html" ]; then
    echo -e "${GREEN}✓${NC} HTML coverage report generated: coverage/coverage.html"
    echo "   Open in browser: file://$PROJECT_ROOT/coverage/coverage.html"
else
    echo -e "${YELLOW}⚠${NC} HTML report generation failed"
fi

if [ -f "coverage/combined.out" ]; then
    echo -e "${GREEN}✓${NC} Combined coverage data: coverage/combined.out"
else
    echo -e "${YELLOW}⚠${NC} Combined coverage data not available"
fi

echo ""
echo "To view detailed coverage for a specific package:"
echo "  go tool cover -html=coverage/<package>.out"
echo ""
echo "To view overall coverage in terminal:"
echo "  go tool cover -func=coverage/combined.out"