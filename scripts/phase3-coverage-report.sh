#!/bin/bash
# Phase 3 Coverage Report Generator
# Generates detailed test coverage analysis for Phase 3 verification

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "========================================="
echo "   Phase 3 Test Coverage Report         "
echo "========================================="
echo ""
echo "Generated: $(date)"
echo "Project: ESXi Commander"
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Create reports directory
mkdir -p reports
REPORT_DIR="reports/coverage-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$REPORT_DIR"

# Function to display colored percentage
show_coverage() {
    local coverage=$1
    local target=$2
    local name=$3
    
    if (( $(echo "$coverage >= $target" | bc -l 2>/dev/null || echo 0) )); then
        echo -e "${GREEN}✓${NC} $name: ${GREEN}${coverage}%${NC} (target: ${target}%)"
    elif (( $(echo "$coverage >= $target - 10" | bc -l 2>/dev/null || echo 0) )); then
        echo -e "${YELLOW}⚠${NC} $name: ${YELLOW}${coverage}%${NC} (target: ${target}%)"
    else
        echo -e "${RED}✗${NC} $name: ${RED}${coverage}%${NC} (target: ${target}%)"
    fi
}

echo "========================================="
echo "1. GENERATING COVERAGE DATA"
echo "========================================="
echo ""

# Run tests with coverage for each package
echo "Running tests with coverage analysis..."
echo ""

# Overall coverage
echo "Calculating overall coverage..."
if go test -coverprofile="$REPORT_DIR/coverage-all.out" ./... > "$REPORT_DIR/test-output.log" 2>&1; then
    echo -e "${GREEN}✓${NC} Tests executed successfully"
    
    # Generate overall coverage percentage
    OVERALL_COVERAGE=$(go tool cover -func="$REPORT_DIR/coverage-all.out" 2>/dev/null | grep total | awk '{print $3}' | sed 's/%//' || echo "0")
    
    if [ -z "$OVERALL_COVERAGE" ]; then
        OVERALL_COVERAGE=0
    fi
else
    echo -e "${RED}✗${NC} Some tests failed (see $REPORT_DIR/test-output.log)"
    OVERALL_COVERAGE=0
fi

echo ""
echo "========================================="
echo "2. PACKAGE-LEVEL COVERAGE ANALYSIS"
echo "========================================="
echo ""

# Critical packages with their target coverage
declare -A PACKAGE_TARGETS=(
    ["pkg/cloudinit"]=90
    ["pkg/esxi/vm"]=80
    ["pkg/backup"]=80
    ["pkg/esxi/client"]=70
    ["pkg/security"]=75
    ["pkg/audit"]=75
    ["pkg/metrics"]=70
    ["pkg/cli/vm"]=60
    ["pkg/cli/backup"]=60
    ["internal/storage"]=70
)

# Test each package individually
echo "Package Coverage Breakdown:"
echo "---------------------------"

TOTAL_PACKAGES=0
PACKAGES_MEETING_TARGET=0

for package in "${!PACKAGE_TARGETS[@]}"; do
    TARGET=${PACKAGE_TARGETS[$package]}
    TOTAL_PACKAGES=$((TOTAL_PACKAGES + 1))
    
    # Check if package exists
    if [ -d "$package" ]; then
        # Run coverage for this package
        if go test -coverprofile="$REPORT_DIR/coverage-$(basename $package).out" ./$package/... > /dev/null 2>&1; then
            PACKAGE_COV=$(go tool cover -func="$REPORT_DIR/coverage-$(basename $package).out" 2>/dev/null | grep total | awk '{print $3}' | sed 's/%//' || echo "0")
            
            if [ -z "$PACKAGE_COV" ] || [ "$PACKAGE_COV" = "" ]; then
                PACKAGE_COV=0
            fi
            
            show_coverage "$PACKAGE_COV" "$TARGET" "$package"
            
            if (( $(echo "$PACKAGE_COV >= $TARGET" | bc -l 2>/dev/null || echo 0) )); then
                PACKAGES_MEETING_TARGET=$((PACKAGES_MEETING_TARGET + 1))
            fi
        else
            echo -e "${YELLOW}⚠${NC} $package: No tests or package not found"
        fi
    else
        echo -e "${YELLOW}⚠${NC} $package: Package directory not found"
    fi
done

echo ""
echo "========================================="
echo "3. TEST FILE ANALYSIS"
echo "========================================="
echo ""

# Count test files
echo "Test File Statistics:"
echo "--------------------"

UNIT_TESTS=$(find . -path ./vendor -prune -o -name "*_test.go" -type f | grep -v vendor | wc -l)
INTEGRATION_TESTS=$(find test/integration -name "*_test.go" 2>/dev/null | wc -l || echo 0)
CHAOS_TESTS=$(find test/chaos -name "*_test.go" 2>/dev/null | wc -l || echo 0)
BENCHMARK_TESTS=$(find . -name "*_test.go" -type f 2>/dev/null | xargs grep -l "^func Benchmark" 2>/dev/null | wc -l || echo 0)

echo "Unit test files:        $UNIT_TESTS"
echo "Integration test files: $INTEGRATION_TESTS"
echo "Chaos test files:       $CHAOS_TESTS"
echo "Benchmark test files:   $BENCHMARK_TESTS"
echo -e "${BOLD}Total test files:       $((UNIT_TESTS + INTEGRATION_TESTS + CHAOS_TESTS))${NC}"

# Count test functions
echo ""
echo "Test Function Count:"
echo "-------------------"

TEST_FUNCTIONS=$(find . -name "*_test.go" -type f 2>/dev/null | xargs grep -h "^func Test" 2>/dev/null | wc -l || echo 0)
BENCHMARK_FUNCTIONS=$(find . -name "*_test.go" -type f 2>/dev/null | xargs grep -h "^func Benchmark" 2>/dev/null | wc -l || echo 0)
TABLE_TESTS=$(find . -name "*_test.go" -type f 2>/dev/null | xargs grep -h "t.Run" 2>/dev/null | wc -l || echo 0)

echo "Test functions:         $TEST_FUNCTIONS"
echo "Benchmark functions:    $BENCHMARK_FUNCTIONS"
echo "Table-driven tests:     $TABLE_TESTS"
echo -e "${BOLD}Total test cases:       $((TEST_FUNCTIONS + TABLE_TESTS))${NC}"

echo ""
echo "========================================="
echo "4. CRITICAL PATH COVERAGE"
echo "========================================="
echo ""

# Check coverage for critical functions
echo "Critical Function Coverage:"
echo "--------------------------"

CRITICAL_FUNCTIONS=(
    "CreateFromTemplate"
    "CloneVM"
    "DeleteVM"
    "CreateBackup"
    "RestoreBackup"
    "BuildCloudInit"
    "EnforceSandbox"
    "LogAuditEvent"
)

echo "Checking coverage for critical functions..."
for func in "${CRITICAL_FUNCTIONS[@]}"; do
    if grep -r "func.*$func" --include="*.go" pkg/ internal/ 2>/dev/null | head -1 > /dev/null; then
        # Check if function has tests
        if grep -r "$func" --include="*_test.go" pkg/ internal/ test/ 2>/dev/null | head -1 > /dev/null; then
            echo -e "${GREEN}✓${NC} $func: Has test coverage"
        else
            echo -e "${RED}✗${NC} $func: No test coverage found"
        fi
    else
        echo -e "${YELLOW}⚠${NC} $func: Function not implemented yet"
    fi
done

echo ""
echo "========================================="
echo "5. UNTESTED CODE DETECTION"
echo "========================================="
echo ""

# Find files with no corresponding test files
echo "Files without tests:"
echo "-------------------"

NO_TEST_COUNT=0
for gofile in $(find pkg internal -name "*.go" -not -name "*_test.go" 2>/dev/null | head -20); do
    basefile=$(basename "$gofile" .go)
    dirname=$(dirname "$gofile")
    testfile="${dirname}/${basefile}_test.go"
    
    if [ ! -f "$testfile" ]; then
        echo "  - $gofile"
        NO_TEST_COUNT=$((NO_TEST_COUNT + 1))
    fi
done

if [ $NO_TEST_COUNT -eq 0 ]; then
    echo -e "${GREEN}  All code files have corresponding test files${NC}"
else
    echo -e "${YELLOW}  Total: $NO_TEST_COUNT files without tests${NC}"
fi

echo ""
echo "========================================="
echo "6. HTML COVERAGE REPORT"
echo "========================================="
echo ""

# Generate HTML coverage report
if [ -f "$REPORT_DIR/coverage-all.out" ]; then
    echo "Generating HTML coverage report..."
    if go tool cover -html="$REPORT_DIR/coverage-all.out" -o "$REPORT_DIR/coverage.html" 2>/dev/null; then
        echo -e "${GREEN}✓${NC} HTML report generated: $REPORT_DIR/coverage.html"
        echo "  Open in browser: file://$PROJECT_ROOT/$REPORT_DIR/coverage.html"
    else
        echo -e "${YELLOW}⚠${NC} Could not generate HTML report"
    fi
else
    echo -e "${RED}✗${NC} No coverage data available for HTML report"
fi

echo ""
echo "========================================="
echo "7. COVERAGE TRENDS"
echo "========================================="
echo ""

# Save current coverage for trending
COVERAGE_HISTORY_FILE="reports/coverage-history.csv"

# Create header if file doesn't exist
if [ ! -f "$COVERAGE_HISTORY_FILE" ]; then
    echo "Date,Overall,CloudInit,ESXi,Backup,Security,CLI" > "$COVERAGE_HISTORY_FILE"
fi

# Get individual package coverage for history
CLOUDINIT_COV=$(go test -cover ./pkg/cloudinit/... 2>/dev/null | grep -oE '[0-9]+\.[0-9]+%' | sed 's/%//' || echo "0")
ESXI_COV=$(go test -cover ./pkg/esxi/... 2>/dev/null | grep -oE '[0-9]+\.[0-9]+%' | sed 's/%//' || echo "0")
BACKUP_COV=$(go test -cover ./pkg/backup/... 2>/dev/null | grep -oE '[0-9]+\.[0-9]+%' | sed 's/%//' || echo "0")
SECURITY_COV=$(go test -cover ./pkg/security/... 2>/dev/null | grep -oE '[0-9]+\.[0-9]+%' | sed 's/%//' || echo "0")
CLI_COV=$(go test -cover ./pkg/cli/... 2>/dev/null | grep -oE '[0-9]+\.[0-9]+%' | sed 's/%//' || echo "0")

# Append to history
echo "$(date +%Y-%m-%d),$OVERALL_COVERAGE,$CLOUDINIT_COV,$ESXI_COV,$BACKUP_COV,$SECURITY_COV,$CLI_COV" >> "$COVERAGE_HISTORY_FILE"

echo "Coverage history updated: $COVERAGE_HISTORY_FILE"

# Show recent trend
echo ""
echo "Recent Coverage Trend (last 5 entries):"
echo "---------------------------------------"
tail -5 "$COVERAGE_HISTORY_FILE" | column -t -s ','

echo ""
echo "========================================="
echo "8. SUMMARY & RECOMMENDATIONS"
echo "========================================="
echo ""

# Calculate summary statistics
echo -e "${BOLD}Coverage Summary:${NC}"
echo "-----------------"
show_coverage "$OVERALL_COVERAGE" 70 "Overall Coverage"
echo "Packages meeting target: $PACKAGES_MEETING_TARGET/$TOTAL_PACKAGES"
echo ""

# Generate recommendations
echo -e "${BOLD}Recommendations:${NC}"
echo "----------------"

if (( $(echo "$OVERALL_COVERAGE < 70" | bc -l 2>/dev/null || echo 0) )); then
    echo -e "${RED}Priority 1: Increase overall coverage to meet 70% target${NC}"
    echo "  Focus on:"
    
    # Find packages with lowest coverage
    for package in "${!PACKAGE_TARGETS[@]}"; do
        if [ -d "$package" ]; then
            PACKAGE_COV=$(go test -cover ./$package/... 2>/dev/null | grep -oE '[0-9]+\.[0-9]+%' | sed 's/%//' || echo "0")
            TARGET=${PACKAGE_TARGETS[$package]}
            if (( $(echo "$PACKAGE_COV < $TARGET - 20" | bc -l 2>/dev/null || echo 0) )); then
                echo "  - $package (currently ${PACKAGE_COV}%, need ${TARGET}%)"
            fi
        fi
    done
fi

if [ $INTEGRATION_TESTS -eq 0 ]; then
    echo -e "${YELLOW}Priority 2: Add integration tests${NC}"
    echo "  - Create test/integration/vm_lifecycle_test.go"
    echo "  - Create test/integration/backup_test.go"
fi

if [ $CHAOS_TESTS -eq 0 ]; then
    echo -e "${YELLOW}Priority 3: Add chaos tests${NC}"
    echo "  - Network failure scenarios"
    echo "  - Resource exhaustion tests"
    echo "  - Concurrent operation tests"
fi

if [ $BENCHMARK_TESTS -lt 3 ]; then
    echo -e "${BLUE}Priority 4: Add performance benchmarks${NC}"
    echo "  - VM operation benchmarks"
    echo "  - Backup/restore benchmarks"
    echo "  - API call benchmarks"
fi

echo ""
echo "========================================="
echo "9. DETAILED REPORT FILES"
echo "========================================="
echo ""

# Generate detailed text report
{
    echo "ESXi Commander - Phase 3 Coverage Report"
    echo "========================================"
    echo ""
    echo "Generated: $(date)"
    echo ""
    echo "Overall Coverage: ${OVERALL_COVERAGE}%"
    echo "Target: 70%"
    echo "Status: $([ $(echo "$OVERALL_COVERAGE >= 70" | bc -l 2>/dev/null || echo 0) -eq 1 ] && echo "PASS" || echo "FAIL")"
    echo ""
    echo "Package Coverage:"
    for package in "${!PACKAGE_TARGETS[@]}"; do
        echo "  $package: Target ${PACKAGE_TARGETS[$package]}%"
    done
    echo ""
    echo "Test Statistics:"
    echo "  Unit Tests: $UNIT_TESTS files"
    echo "  Integration Tests: $INTEGRATION_TESTS files"
    echo "  Chaos Tests: $CHAOS_TESTS files"
    echo "  Total Test Functions: $TEST_FUNCTIONS"
    echo "  Total Test Cases: $((TEST_FUNCTIONS + TABLE_TESTS))"
} > "$REPORT_DIR/coverage-report.txt"

echo "Report files generated:"
echo "  - Coverage Report: $REPORT_DIR/coverage-report.txt"
echo "  - HTML Report: $REPORT_DIR/coverage.html"
echo "  - Test Output: $REPORT_DIR/test-output.log"
echo "  - Coverage Data: $REPORT_DIR/coverage-all.out"

echo ""
echo "========================================="

# Final status
if (( $(echo "$OVERALL_COVERAGE >= 70" | bc -l 2>/dev/null || echo 0) )); then
    echo -e "${GREEN}✓ PHASE 3 COVERAGE REQUIREMENT MET${NC}"
    echo "Coverage of ${OVERALL_COVERAGE}% exceeds 70% requirement"
    exit 0
else
    echo -e "${RED}✗ PHASE 3 COVERAGE REQUIREMENT NOT MET${NC}"
    echo "Coverage of ${OVERALL_COVERAGE}% is below 70% requirement"
    echo "Run this script after adding more tests to track progress"
    exit 1
fi