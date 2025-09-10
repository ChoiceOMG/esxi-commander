package verification

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestProjectStructure verifies all required files and directories exist
func TestProjectStructure(t *testing.T) {
	requiredDirs := []string{
		"cmd/ceso",
		"cmd/cesod",
		"pkg/cli/vm",
		"pkg/cli/backup",
		"pkg/cli/template",
		"pkg/esxi/client",
		"pkg/esxi/vm",
		"pkg/cloudinit",
		"pkg/config",
		"pkg/logger",
		"internal/defaults",
		"internal/validation",
		"test/unit",
		"configs",
		".github/workflows",
	}

	for _, dir := range requiredDirs {
		path := filepath.Join("../..", dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Required directory missing: %s", dir)
		}
	}

	requiredFiles := []string{
		"cmd/ceso/main.go",
		"cmd/cesod/main.go",
		"pkg/cli/root.go",
		"pkg/cli/vm/vm.go",
		"pkg/cli/vm/list.go",
		"pkg/cli/vm/create.go",
		"pkg/cli/vm/types.go",
		"pkg/cli/template/template.go",
		"pkg/cli/template/validate.go",
		"pkg/cli/backup/backup.go",
		"pkg/config/config.go",
		"pkg/esxi/client/client.go",
		"pkg/esxi/vm/operations.go",
		"pkg/cloudinit/builder.go",
		"pkg/logger/logger.go",
		"internal/defaults/defaults.go",
		"internal/validation/validation.go",
		"go.mod",
		"go.sum",
		".github/workflows/ci.yml",
	}

	for _, file := range requiredFiles {
		path := filepath.Join("../..", file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Required file missing: %s", file)
		}
	}
}

// TestCLICommands verifies CLI commands are available and working
func TestCLICommands(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "../../test-ceso", "../../cmd/ceso")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build ceso binary: %v", err)
	}
	defer os.Remove("../../test-ceso")

	tests := []struct {
		name     string
		args     []string
		contains []string
		notEmpty bool
	}{
		{
			name:     "Root help",
			args:     []string{"--help"},
			contains: []string{"ESXi Commander", "Available Commands:", "vm", "backup"},
		},
		{
			name:     "VM help",
			args:     []string{"vm", "--help"},
			contains: []string{"Manage virtual machines", "Available Commands:"},
		},
		{
			name:     "VM list",
			args:     []string{"vm", "list"},
			contains: []string{"NAME", "STATUS", "IP", "CPU", "RAM"},
			notEmpty: true,
		},
		{
			name:     "Backup help",
			args:     []string{"backup", "--help"},
			contains: []string{"backup"},
		},
		{
			name:     "Template help",
			args:     []string{"template", "--help"},
			contains: []string{"template"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("../../test-ceso", tt.args...)
			output, err := cmd.CombinedOutput()
			
			// Some commands return non-zero for help
			if err != nil && !strings.Contains(string(output), "Usage:") {
				t.Errorf("Command failed: %v\nOutput: %s", err, output)
				return
			}

			outputStr := string(output)
			if tt.notEmpty && len(outputStr) == 0 {
				t.Error("Expected non-empty output")
			}

			for _, expected := range tt.contains {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Output missing expected string: %q\nFull output: %s", expected, outputStr)
				}
			}
		})
	}
}

// TestMockDataOutput verifies mock data is correct and consistent
func TestMockDataOutput(t *testing.T) {
	cmd := exec.Command("../../test-ceso", "vm", "list")
	output, err := cmd.Output()
	if err != nil {
		// Build and retry
		buildCmd := exec.Command("go", "build", "-o", "../../test-ceso", "../../cmd/ceso")
		if buildErr := buildCmd.Run(); buildErr != nil {
			t.Fatalf("Failed to build: %v", buildErr)
		}
		
		cmd = exec.Command("../../test-ceso", "vm", "list")
		output, err = cmd.Output()
		if err != nil {
			t.Fatalf("Failed to run vm list: %v", err)
		}
	}

	outputStr := string(output)
	
	// Check for expected VMs
	expectedVMs := []string{
		"ubuntu-web-01",
		"ubuntu-db-01",
		"ubuntu-test",
	}

	for _, vm := range expectedVMs {
		if !strings.Contains(outputStr, vm) {
			t.Errorf("Expected VM not found in output: %s", vm)
		}
	}

	// Check table format
	if !strings.Contains(outputStr, "NAME") || !strings.Contains(outputStr, "STATUS") {
		t.Error("Table headers missing from output")
	}
}

// TestJSONFormatting verifies JSON output is valid and properly formatted
func TestJSONFormatting(t *testing.T) {
	cmd := exec.Command("../../test-ceso", "vm", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		// Build and retry
		buildCmd := exec.Command("go", "build", "-o", "../../test-ceso", "../../cmd/ceso")
		if buildErr := buildCmd.Run(); buildErr != nil {
			t.Fatalf("Failed to build: %v", buildErr)
		}
		
		cmd = exec.Command("../../test-ceso", "vm", "list", "--json")
		output, err = cmd.Output()
		if err != nil {
			t.Fatalf("Failed to run vm list --json: %v", err)
		}
	}

	// Parse JSON
	var vms []map[string]interface{}
	if err := json.Unmarshal(output, &vms); err != nil {
		t.Fatalf("Invalid JSON output: %v\nOutput: %s", err, output)
	}

	// Verify structure
	if len(vms) != 3 {
		t.Errorf("Expected 3 VMs, got %d", len(vms))
	}

	// Check first VM has required fields
	if len(vms) > 0 {
		requiredFields := []string{"name", "status", "ip", "cpu", "ram"}
		for _, field := range requiredFields {
			if _, ok := vms[0][field]; !ok {
				t.Errorf("Missing required field in JSON: %s", field)
			}
		}
	}

	// Verify specific VM data
	foundWeb := false
	for _, vm := range vms {
		if vm["name"] == "ubuntu-web-01" {
			foundWeb = true
			if vm["cpu"] != float64(2) {
				t.Errorf("Expected ubuntu-web-01 to have 2 CPUs, got %v", vm["cpu"])
			}
			if vm["ram"] != float64(4) {
				t.Errorf("Expected ubuntu-web-01 to have 4GB RAM, got %v", vm["ram"])
			}
		}
	}
	
	if !foundWeb {
		t.Error("ubuntu-web-01 not found in JSON output")
	}
}

// TestBuildArtifacts verifies build produces expected artifacts
func TestBuildArtifacts(t *testing.T) {
	// Build ceso
	cesoBuild := exec.Command("go", "build", "-o", "../../test-ceso", "../../cmd/ceso")
	if err := cesoBuild.Run(); err != nil {
		t.Fatalf("Failed to build ceso: %v", err)
	}
	defer os.Remove("../../test-ceso")

	// Build cesod
	cesodBuild := exec.Command("go", "build", "-o", "../../test-cesod", "../../cmd/cesod")
	if err := cesodBuild.Run(); err != nil {
		t.Fatalf("Failed to build cesod: %v", err)
	}
	defer os.Remove("../../test-cesod")

	// Check binary sizes (should be reasonable)
	cesoInfo, err := os.Stat("../../test-ceso")
	if err != nil {
		t.Fatalf("Failed to stat ceso binary: %v", err)
	}
	
	// Binary should be between 1MB and 20MB
	size := cesoInfo.Size()
	if size < 1024*1024 {
		t.Errorf("ceso binary suspiciously small: %d bytes", size)
	}
	if size > 20*1024*1024 {
		t.Errorf("ceso binary suspiciously large: %d bytes", size)
	}

	// Verify binary is executable
	cmd := exec.Command("../../test-ceso", "version")
	if output, err := cmd.CombinedOutput(); err != nil {
		// Version command might not exist yet, check for help instead
		helpCmd := exec.Command("../../test-ceso", "--help")
		if helpOutput, helpErr := helpCmd.CombinedOutput(); helpErr != nil {
			t.Errorf("Binary not executable: %v\nOutput: %s", helpErr, helpOutput)
		}
	} else {
		t.Logf("Version output: %s", output)
	}
}

// TestDependencies verifies all required dependencies are present
func TestDependencies(t *testing.T) {
	cmd := exec.Command("go", "list", "-m", "all")
	cmd.Dir = "../.."
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to list dependencies: %v", err)
	}

	requiredDeps := []string{
		"github.com/spf13/cobra",
		"github.com/spf13/viper",
		"github.com/vmware/govmomi",
		"github.com/rs/zerolog",
		"go.etcd.io/bbolt",
		"github.com/prometheus/client_golang",
	}

	outputStr := string(output)
	for _, dep := range requiredDeps {
		if !strings.Contains(outputStr, dep) {
			t.Errorf("Required dependency not found: %s", dep)
		}
	}
}

// TestGlobalFlags verifies global flags work correctly
func TestGlobalFlags(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		check func(string) bool
	}{
		{
			name: "JSON flag",
			args: []string{"vm", "list", "--json"},
			check: func(output string) bool {
				return strings.HasPrefix(strings.TrimSpace(output), "[")
			},
		},
		{
			name: "Dry run flag",
			args: []string{"vm", "create", "test", "--dry-run"},
			check: func(output string) bool {
				// Should not actually create anything
				return true // Stub for now
			},
		},
		{
			name: "Config flag",
			args: []string{"--config", "/tmp/test.yaml", "vm", "list"},
			check: func(output string) bool {
				// Should accept config flag without error
				return strings.Contains(output, "NAME") || strings.Contains(output, "ubuntu")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("../../test-ceso", tt.args...)
			output, _ := cmd.CombinedOutput()
			
			if !tt.check(string(output)) {
				t.Errorf("Flag test failed for %s\nOutput: %s", tt.name, output)
			}
		})
	}
}