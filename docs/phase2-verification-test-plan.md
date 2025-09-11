# Phase 2 Verification Test Plan

## Executive Summary

This test plan ensures Phase 2 deliverables meet all requirements and establishes regression protection for the ESXi Commander project. It covers functional verification, performance validation, integration testing, and regression prevention strategies.

## Test Scope

### In Scope
- ESXi connection via govmomi
- VM operations (create, clone, delete, info, list)
- Cloud-init guestinfo injection
- SSH fallback mechanisms
- Error handling and recovery
- Performance metrics validation
- Regression prevention

### Out of Scope
- Backup/restore operations (Phase 4)
- Hot cloning/snapshots (Phase 3)
- vCenter integration
- Windows VM support

## Test Categories

### 1. Unit Tests (Target: 70% Coverage)

#### File: `test/unit/esxi_client_test.go`
```go
package unit

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/r11/esxi-commander/pkg/esxi/client"
)

func TestESXiConnection(t *testing.T) {
    tests := []struct {
        name    string
        config  *client.Config
        wantErr bool
    }{
        {
            name: "valid connection",
            config: &client.Config{
                Host:     "192.168.1.10",
                User:     "root",
                Password: "password",
            },
            wantErr: false,
        },
        {
            name: "invalid host",
            config: &client.Config{
                Host: "",
                User: "root",
            },
            wantErr: true,
        },
        {
            name: "missing credentials",
            config: &client.Config{
                Host: "192.168.1.10",
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := client.NewClient(tt.config)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

#### File: `test/unit/cloudinit_test.go`
```go
func TestCloudInitBuilder(t *testing.T) {
    tests := []struct {
        name string
        data *cloudinit.CloudInitData
        validate func(t *testing.T, result map[string]string)
    }{
        {
            name: "static IP configuration",
            data: &cloudinit.CloudInitData{
                Hostname: "test-vm",
                IP:       "192.168.1.100/24",
                Gateway:  "192.168.1.1",
                DNS:      []string{"8.8.8.8", "8.8.4.4"},
                SSHKeys:  []string{"ssh-rsa AAAAB3..."},
            },
            validate: func(t *testing.T, result map[string]string) {
                assert.Contains(t, result, "guestinfo.metadata")
                assert.Contains(t, result, "guestinfo.userdata")
                assert.Contains(t, result, "guestinfo.vendordata")
                
                // Verify encoding
                assert.Equal(t, "gzip+base64", result["guestinfo.metadata.encoding"])
                
                // Decode and verify content
                metadata := decodeGuestinfo(result["guestinfo.metadata"])
                assert.Contains(t, metadata, "test-vm")
            },
        },
        {
            name: "DHCP configuration",
            data: &cloudinit.CloudInitData{
                Hostname: "dhcp-vm",
                SSHKeys:  []string{"ssh-rsa AAAAB3..."},
            },
            validate: func(t *testing.T, result map[string]string) {
                assert.Contains(t, result, "guestinfo.metadata")
                assert.Empty(t, result["guestinfo.vendordata"]) // No network config for DHCP
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := cloudinit.BuildGuestinfo(tt.data)
            assert.NoError(t, err)
            tt.validate(t, result)
        })
    }
}
```

### 2. Integration Tests

#### File: `test/integration/vm_lifecycle_test.go`
```go
//go:build integration
package integration

import (
    "encoding/json"
    "fmt"
    "os/exec"
    "testing"
    "time"
)

func TestCompleteVMLifecycle(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    vmName := fmt.Sprintf("test-vm-%d", time.Now().Unix())
    vmIP := "192.168.1.200/24"
    
    // Phase 1: Create VM
    t.Run("CreateVM", func(t *testing.T) {
        start := time.Now()
        
        cmd := exec.Command("./ceso", "vm", "create", vmName,
            "--template", "ubuntu-22.04-template",
            "--ip", vmIP,
            "--cpu", "2",
            "--memory", "4",
            "--ssh-key", getTestSSHKey())
        
        output, err := cmd.CombinedOutput()
        assert.NoError(t, err, "VM creation failed: %s", output)
        
        duration := time.Since(start)
        assert.Less(t, duration, 90*time.Second, "VM creation exceeded 90s limit")
        
        t.Logf("VM created in %v", duration)
    })
    
    // Phase 2: Verify VM exists and is running
    t.Run("VerifyVM", func(t *testing.T) {
        cmd := exec.Command("./ceso", "vm", "info", vmName, "--json")
        output, err := cmd.Output()
        assert.NoError(t, err)
        
        var vmInfo VMInfo
        err = json.Unmarshal(output, &vmInfo)
        assert.NoError(t, err)
        
        assert.Equal(t, vmName, vmInfo.Name)
        assert.Equal(t, "poweredOn", vmInfo.PowerState)
        assert.Equal(t, 2, vmInfo.CPU)
        assert.Equal(t, 4096, vmInfo.Memory)
    })
    
    // Phase 3: Test Cloud-Init
    t.Run("VerifyCloudInit", func(t *testing.T) {
        // Wait for VM to boot and cloud-init to complete
        time.Sleep(45 * time.Second)
        
        // SSH to VM and verify configuration
        sshCmd := exec.Command("ssh", 
            "-o", "StrictHostKeyChecking=no",
            "-i", getTestSSHKeyPath(),
            "ubuntu@192.168.1.200",
            "hostname")
        
        output, err := sshCmd.Output()
        assert.NoError(t, err, "SSH connection failed")
        assert.Equal(t, vmName, strings.TrimSpace(string(output)))
        
        // Verify IP configuration
        sshCmd = exec.Command("ssh",
            "-o", "StrictHostKeyChecking=no",
            "-i", getTestSSHKeyPath(),
            "ubuntu@192.168.1.200",
            "ip addr show ens192 | grep inet | awk '{print $2}'")
        
        output, err = sshCmd.Output()
        assert.NoError(t, err)
        assert.Contains(t, string(output), "192.168.1.200/24")
    })
    
    // Phase 4: Clone VM
    t.Run("CloneVM", func(t *testing.T) {
        cloneName := fmt.Sprintf("%s-clone", vmName)
        
        start := time.Now()
        
        cmd := exec.Command("./ceso", "vm", "clone", vmName, cloneName,
            "--ip", "192.168.1.201/24")
        
        output, err := cmd.CombinedOutput()
        assert.NoError(t, err, "VM clone failed: %s", output)
        
        duration := time.Since(start)
        t.Logf("VM cloned in %v", duration)
        
        // Verify clone exists
        cmd = exec.Command("./ceso", "vm", "list", "--json")
        output, err = cmd.Output()
        assert.NoError(t, err)
        
        var vms []VMSummary
        json.Unmarshal(output, &vms)
        
        found := false
        for _, vm := range vms {
            if vm.Name == cloneName {
                found = true
                break
            }
        }
        assert.True(t, found, "Clone VM not found")
        
        // Cleanup clone
        defer exec.Command("./ceso", "vm", "delete", cloneName).Run()
    })
    
    // Phase 5: Delete VM
    t.Run("DeleteVM", func(t *testing.T) {
        cmd := exec.Command("./ceso", "vm", "delete", vmName)
        output, err := cmd.CombinedOutput()
        assert.NoError(t, err, "VM deletion failed: %s", output)
        
        // Verify VM is gone
        cmd = exec.Command("./ceso", "vm", "list", "--json")
        output, err = cmd.Output()
        assert.NoError(t, err)
        
        var vms []VMSummary
        json.Unmarshal(output, &vms)
        
        for _, vm := range vms {
            assert.NotEqual(t, vmName, vm.Name, "VM still exists after deletion")
        }
    })
}
```

### 3. Performance Tests

#### File: `test/performance/vm_operations_bench_test.go`
```go
//go:build performance
package performance

func BenchmarkVMCreate(b *testing.B) {
    setupESXiConnection(b)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        vmName := fmt.Sprintf("bench-vm-%d", i)
        
        start := time.Now()
        err := createVM(vmName, "ubuntu-22.04-template")
        duration := time.Since(start)
        
        if err != nil {
            b.Fatalf("VM creation failed: %v", err)
        }
        
        if duration > 90*time.Second {
            b.Fatalf("VM creation took %v, exceeding 90s limit", duration)
        }
        
        b.ReportMetric(duration.Seconds(), "create_seconds")
        
        // Cleanup
        deleteVM(vmName)
    }
}

func BenchmarkVMList(b *testing.B) {
    setupESXiConnection(b)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        start := time.Now()
        
        cmd := exec.Command("./ceso", "vm", "list")
        err := cmd.Run()
        
        duration := time.Since(start)
        
        if err != nil {
            b.Fatalf("VM list failed: %v", err)
        }
        
        if duration > 2*time.Second {
            b.Fatalf("VM list took %v, exceeding 2s limit", duration)
        }
        
        b.ReportMetric(duration.Milliseconds(), "list_ms")
    }
}
```

### 4. Regression Tests

#### File: `test/regression/phase1_compatibility_test.go`
```go
//go:build regression
package regression

// Ensure Phase 1 functionality still works
func TestPhase1Compatibility(t *testing.T) {
    t.Run("CLI Structure", func(t *testing.T) {
        // Verify all Phase 1 commands still exist
        commands := [][]string{
            {"./ceso", "--help"},
            {"./ceso", "vm", "--help"},
            {"./ceso", "vm", "list", "--help"},
            {"./ceso", "backup", "--help"},
            {"./ceso", "template", "--help"},
        }
        
        for _, cmd := range commands {
            c := exec.Command(cmd[0], cmd[1:]...)
            err := c.Run()
            assert.NoError(t, err, "Command failed: %v", cmd)
        }
    })
    
    t.Run("JSON Output", func(t *testing.T) {
        cmd := exec.Command("./ceso", "vm", "list", "--json")
        output, err := cmd.Output()
        assert.NoError(t, err)
        
        // Verify valid JSON
        var data interface{}
        err = json.Unmarshal(output, &data)
        assert.NoError(t, err, "Invalid JSON output")
    })
    
    t.Run("Configuration Loading", func(t *testing.T) {
        // Create test config
        config := `
esxi:
  host: test.local
  user: testuser
defaults:
  template: test-template
`
        configPath := "/tmp/test-config.yaml"
        ioutil.WriteFile(configPath, []byte(config), 0644)
        defer os.Remove(configPath)
        
        cmd := exec.Command("./ceso", "--config", configPath, "vm", "list")
        err := cmd.Run()
        // Should not crash even with test config
        assert.NotNil(t, cmd.ProcessState)
    })
}
```

#### File: `test/regression/api_stability_test.go`
```go
//go:build regression
package regression

// Ensure API contracts remain stable
func TestAPIStability(t *testing.T) {
    t.Run("VMListOutput", func(t *testing.T) {
        cmd := exec.Command("./ceso", "vm", "list", "--json")
        output, err := cmd.Output()
        assert.NoError(t, err)
        
        var vms []map[string]interface{}
        err = json.Unmarshal(output, &vms)
        assert.NoError(t, err)
        
        // Verify expected fields exist
        if len(vms) > 0 {
            vm := vms[0]
            requiredFields := []string{"name", "status", "cpu", "memory"}
            for _, field := range requiredFields {
                assert.Contains(t, vm, field, "Missing required field: %s", field)
            }
        }
    })
    
    t.Run("ErrorFormats", func(t *testing.T) {
        // Test error output format
        cmd := exec.Command("./ceso", "vm", "delete", "non-existent-vm")
        output, err := cmd.CombinedOutput()
        assert.Error(t, err)
        assert.Contains(t, string(output), "not found")
    })
}
```

### 5. Failure Scenario Tests

#### File: `test/integration/failure_scenarios_test.go`
```go
//go:build integration
package integration

func TestFailureScenarios(t *testing.T) {
    t.Run("InvalidTemplate", func(t *testing.T) {
        cmd := exec.Command("./ceso", "vm", "create", "fail-vm",
            "--template", "non-existent-template")
        
        output, err := cmd.CombinedOutput()
        assert.Error(t, err)
        assert.Contains(t, string(output), "template not found")
    })
    
    t.Run("DuplicateVMName", func(t *testing.T) {
        vmName := "duplicate-test"
        
        // Create first VM
        cmd := exec.Command("./ceso", "vm", "create", vmName,
            "--template", "ubuntu-22.04-template")
        cmd.Run()
        defer exec.Command("./ceso", "vm", "delete", vmName).Run()
        
        // Try to create duplicate
        cmd = exec.Command("./ceso", "vm", "create", vmName,
            "--template", "ubuntu-22.04-template")
        output, err := cmd.CombinedOutput()
        assert.Error(t, err)
        assert.Contains(t, string(output), "already exists")
    })
    
    t.Run("InvalidIPFormat", func(t *testing.T) {
        cmd := exec.Command("./ceso", "vm", "create", "bad-ip-vm",
            "--template", "ubuntu-22.04-template",
            "--ip", "not-an-ip")
        
        output, err := cmd.CombinedOutput()
        assert.Error(t, err)
        assert.Contains(t, string(output), "invalid IP")
    })
    
    t.Run("ConnectionFailure", func(t *testing.T) {
        // Temporarily break connection
        os.Setenv("ESXI_HOST", "invalid.host.local")
        defer os.Unsetenv("ESXI_HOST")
        
        cmd := exec.Command("./ceso", "vm", "list")
        output, err := cmd.CombinedOutput()
        assert.Error(t, err)
        assert.Contains(t, string(output), "connection failed")
    })
}
```

### 6. SSH Fallback Tests

#### File: `test/integration/ssh_fallback_test.go`
```go
//go:build integration
package integration

func TestSSHFallback(t *testing.T) {
    t.Run("FallbackOnAPIFailure", func(t *testing.T) {
        // Force API failure by using wrong port
        os.Setenv("ESXI_API_PORT", "99999")
        defer os.Unsetenv("ESXI_API_PORT")
        
        // Should fallback to SSH
        cmd := exec.Command("./ceso", "vm", "list")
        output, err := cmd.CombinedOutput()
        
        // Should succeed via SSH
        assert.NoError(t, err)
        assert.Contains(t, string(output), "NAME")
    })
    
    t.Run("SSHCommands", func(t *testing.T) {
        // Test direct SSH command execution
        sshClient := setupSSHClient(t)
        
        // List VMs via vim-cmd
        output, err := sshClient.RunCommand("vim-cmd vmsvc/getallvms")
        assert.NoError(t, err)
        assert.Contains(t, output, "Vmid")
    })
}
```

## Verification Script

### File: `test/scripts/verify-phase2.sh`
```bash
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
go build ./cmd/cesod

# Unit Tests
echo -e "\n📝 Running Unit Tests..."
run_test "ESXi Client" "go test ./pkg/esxi/client/..."
run_test "Cloud-Init Builder" "go test ./pkg/cloudinit/..."
run_test "VM Operations" "go test ./pkg/esxi/vm/..."
run_test "CLI Commands" "go test ./pkg/cli/..."

# Coverage Check
echo -e "\n📊 Checking Test Coverage..."
go test -cover ./pkg/... > /tmp/coverage.txt
COVERAGE=$(grep -o '[0-9]*\.[0-9]*%' /tmp/coverage.txt | sed 's/%//' | awk '{sum+=$1; count++} END {print sum/count}')
echo "Average coverage: ${COVERAGE}%"
if (( $(echo "$COVERAGE < 70" | bc -l) )); then
    echo -e "${YELLOW}Warning: Coverage below 70% target${NC}"
fi

# Integration Tests (if ESXi available)
if [ ! -z "$ESXI_HOST" ]; then
    echo -e "\n🔗 Running Integration Tests..."
    run_test "VM Lifecycle" "go test -tags=integration ./test/integration/vm_lifecycle_test.go"
    run_test "Cloud-Init" "go test -tags=integration ./test/integration/cloudinit_test.go"
    run_test "Failure Scenarios" "go test -tags=integration ./test/integration/failure_scenarios_test.go"
else
    echo -e "\n${YELLOW}Skipping integration tests (ESXI_HOST not set)${NC}"
fi

# Regression Tests
echo -e "\n🔄 Running Regression Tests..."
run_test "Phase 1 Compatibility" "go test -tags=regression ./test/regression/phase1_compatibility_test.go"
run_test "API Stability" "go test -tags=regression ./test/regression/api_stability_test.go"

# Performance Tests
echo -e "\n⚡ Running Performance Benchmarks..."
if [ ! -z "$ESXI_HOST" ]; then
    go test -bench=. -tags=performance ./test/performance/... > /tmp/bench.txt
    
    # Check VM create time
    CREATE_TIME=$(grep "BenchmarkVMCreate" /tmp/bench.txt | awk '{print $3}')
    if [ ! -z "$CREATE_TIME" ]; then
        echo "VM Create Time: ${CREATE_TIME}s"
        if (( $(echo "$CREATE_TIME > 90" | bc -l) )); then
            echo -e "${RED}Failed: VM creation exceeds 90s limit${NC}"
            FAILED=$((FAILED + 1))
        fi
    fi
fi

# CLI Verification
echo -e "\n🖥️  Verifying CLI Commands..."
run_test "ceso --help" "./ceso --help"
run_test "ceso vm list" "./ceso vm list --dry-run"
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
    "pkg/esxi/ssh/client.go"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "  $file ${GREEN}✓${NC}"
    else
        echo -e "  $file ${RED}✗${NC}"
        FAILED=$((FAILED + 1))
    fi
done

# Final Report
echo -e "\n📋 Phase 2 Verification Report"
echo "================================"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    echo "Phase 2 is complete and ready for review."
    exit 0
else
    echo -e "${RED}❌ $FAILED test(s) failed${NC}"
    echo "Please fix the issues before marking Phase 2 complete."
    exit 1
fi
```

## Continuous Regression Prevention

### GitHub Actions Workflow: `.github/workflows/regression-tests.yml`
```yaml
name: Regression Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]
  schedule:
    # Run nightly at 2 AM UTC
    - cron: '0 2 * * *'

jobs:
  regression:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Install dependencies
      run: go mod download
    
    - name: Run Phase 1 regression tests
      run: go test -tags=regression ./test/regression/phase1_compatibility_test.go
    
    - name: Run API stability tests
      run: go test -tags=regression ./test/regression/api_stability_test.go
    
    - name: Check backwards compatibility
      run: |
        # Build current version
        go build -o ceso-new ./cmd/ceso
        
        # Checkout previous version
        git checkout HEAD~1
        go build -o ceso-old ./cmd/ceso
        
        # Compare outputs
        ./ceso-old vm list --json > old-output.json
        git checkout -
        ./ceso-new vm list --json > new-output.json
        
        # Ensure JSON structure is compatible
        go run test/tools/json-compat-check.go old-output.json new-output.json
    
    - name: Performance regression check
      run: |
        go test -bench=. -tags=performance ./test/performance/... > current-bench.txt
        
        # Compare with baseline
        if [ -f benchmark-baseline.txt ]; then
          go run test/tools/bench-compare.go benchmark-baseline.txt current-bench.txt
        fi
        
        # Update baseline if on main
        if [ "${{ github.ref }}" == "refs/heads/main" ]; then
          cp current-bench.txt benchmark-baseline.txt
        fi
```

## Test Execution Plan

### Daily Testing
```bash
# Quick smoke tests
./test/scripts/smoke-tests.sh

# Unit tests
go test ./pkg/...
```

### Per-Commit Testing
```bash
# Pre-commit hook
go test ./pkg/...
go fmt ./...
golangci-lint run
```

### Per-PR Testing
- All unit tests
- Regression tests
- API compatibility checks
- Performance benchmarks

### Nightly Testing
- Full integration suite
- Performance regression checks
- Chaos testing scenarios

## Success Metrics Summary

| Metric | Target | Test Method |
|--------|--------|-------------|
| VM Create Time | <90s | Performance benchmark |
| VM Clone Time (80GB) | <5min | Integration test |
| VM List Response | <2s | Performance benchmark |
| Cloud-Init Success | >95% | Integration test |
| Test Coverage | >70% | Coverage report |
| API Compatibility | 100% | Regression test |
| Error Handling | 100% | Failure scenario tests |
| SSH Fallback | Works | Integration test |

## Test Data Management

### Test VM Templates
```yaml
# test/fixtures/templates.yaml
templates:
  ubuntu-22.04-template:
    path: "[datastore1] templates/ubuntu-22.04-cloud-init.vmx"
    validated: true
    cloud_init: true
    vmware_tools: true
    
  ubuntu-24.04-template:
    path: "[datastore1] templates/ubuntu-24.04-cloud-init.vmx"
    validated: true
    cloud_init: true
    vmware_tools: true
```

### Test SSH Keys
```bash
# Generate test keys
ssh-keygen -t rsa -b 2048 -f test/fixtures/test_key -N ""
```

### Test Configuration
```yaml
# test/fixtures/test-config.yaml
esxi:
  host: ${TEST_ESXI_HOST}
  user: ${TEST_ESXI_USER}
  password: ${TEST_ESXI_PASSWORD}
  insecure: true

test:
  vm_prefix: "test-"
  cleanup: true
  network: "VM Network"
  datastore: "datastore1"
  ip_range: "192.168.100.0/24"
```

## Reporting

### Test Report Generation
```bash
# Generate HTML report
go test -v ./... -json | go-test-report -o test-report.html

# Generate coverage report
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Metrics Dashboard
Track these metrics over time:
- Test pass rate
- Coverage percentage
- Performance benchmarks
- Regression detection rate
- Mean time to detection (MTTD)

## Conclusion

This comprehensive test plan ensures Phase 2 meets all requirements while preventing regression. The multi-layered approach includes:

1. **Unit tests** for component validation
2. **Integration tests** for end-to-end verification
3. **Performance tests** for metric validation
4. **Regression tests** for backward compatibility
5. **Failure tests** for error handling
6. **Continuous testing** via CI/CD

Execute `test/scripts/verify-phase2.sh` to validate Phase 2 completion.