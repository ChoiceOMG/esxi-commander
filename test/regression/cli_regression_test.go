package regression

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestCommandStructureRegression ensures command structure hasn't changed
func TestCommandStructureRegression(t *testing.T) {
	// Expected command structure
	expectedCommands := map[string][]string{
		"root": {"vm", "backup", "template", "completion", "help"},
		"vm":   {"list", "create", "clone", "delete", "info"},
		"backup": {"create", "restore", "list"},
		"template": {"validate"},
	}

	for parent, commands := range expectedCommands {
		t.Run(parent+" commands", func(t *testing.T) {
			var cmd *exec.Cmd
			if parent == "root" {
				cmd = exec.Command("../../ceso", "--help")
			} else {
				cmd = exec.Command("../../ceso", parent, "--help")
			}
			
			output, _ := cmd.CombinedOutput()
			outputStr := string(output)
			
			for _, expectedCmd := range commands {
				if !strings.Contains(outputStr, expectedCmd) {
					t.Errorf("Command %q not found under %s", expectedCmd, parent)
				}
			}
		})
	}
}

// TestFlagRegression ensures flags haven't changed
func TestFlagRegression(t *testing.T) {
	globalFlags := []string{
		"--config",
		"--dry-run",
		"--json",
		"--help",
	}

	// Test global flags
	cmd := exec.Command("../../ceso", "--help")
	output, _ := cmd.CombinedOutput()
	outputStr := string(output)
	
	for _, flag := range globalFlags {
		if !strings.Contains(outputStr, flag) {
			t.Errorf("Global flag %q not found", flag)
		}
	}

	// Test command-specific flags
	commandFlags := map[string][]string{
		"vm create": {"--template", "--ip", "--ssh-key"},
		"vm clone": {"--ip"},
		"backup restore": {"--as-new"},
	}

	for cmdStr, flags := range commandFlags {
		t.Run(cmdStr+" flags", func(t *testing.T) {
			parts := strings.Split(cmdStr, " ")
			args := append(parts, "--help")
			cmd := exec.Command("../../ceso", args...)
			output, _ := cmd.CombinedOutput()
			outputStr := string(output)
			
			for _, flag := range flags {
				if !strings.Contains(outputStr, flag) {
					t.Errorf("Flag %q not found for command %s", flag, cmdStr)
				}
			}
		})
	}
}

// TestOutputFormatRegression ensures output formats remain consistent
func TestOutputFormatRegression(t *testing.T) {
	t.Run("Table format", func(t *testing.T) {
		cmd := exec.Command("../../ceso", "vm", "list")
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("Failed to run vm list: %v", err)
		}
		
		outputStr := string(output)
		
		// Check table headers
		expectedHeaders := []string{"NAME", "STATUS", "IP", "CPU", "RAM"}
		for _, header := range expectedHeaders {
			if !strings.Contains(outputStr, header) {
				t.Errorf("Table header %q not found", header)
			}
		}
		
		// Check table alignment (headers should be properly spaced)
		lines := strings.Split(outputStr, "\n")
		if len(lines) > 0 {
			headerLine := lines[0]
			// Headers should have consistent spacing
			if !strings.Contains(headerLine, "  ") {
				t.Error("Table headers not properly spaced")
			}
		}
	})

	t.Run("JSON format", func(t *testing.T) {
		cmd := exec.Command("../../ceso", "vm", "list", "--json")
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("Failed to run vm list --json: %v", err)
		}
		
		// Should be valid JSON array
		var data []interface{}
		if err := json.Unmarshal(output, &data); err != nil {
			t.Errorf("Invalid JSON format: %v", err)
		}
		
		// Should have consistent structure
		if len(data) > 0 {
			if vm, ok := data[0].(map[string]interface{}); ok {
				requiredKeys := []string{"name", "status", "ip", "cpu", "ram"}
				for _, key := range requiredKeys {
					if _, exists := vm[key]; !exists {
						t.Errorf("JSON missing required key: %s", key)
					}
				}
			}
		}
	})
}

// TestErrorHandlingRegression ensures error messages remain consistent
func TestErrorHandlingRegression(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		shouldError bool
		errorCheck  func(string) bool
	}{
		{
			name:        "Invalid command",
			args:        []string{"invalid-command"},
			shouldError: true,
			errorCheck: func(output string) bool {
				return strings.Contains(output, "unknown command") || 
				       strings.Contains(output, "Error:")
			},
		},
		{
			name:        "Missing required args",
			args:        []string{"vm", "create"},
			shouldError: true,
			errorCheck: func(output string) bool {
				return strings.Contains(output, "requires") || 
				       strings.Contains(output, "Usage:")
			},
		},
		{
			name:        "Invalid flag",
			args:        []string{"vm", "list", "--invalid-flag"},
			shouldError: true,
			errorCheck: func(output string) bool {
				return strings.Contains(output, "unknown flag") ||
				       strings.Contains(output, "Error:")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("../../ceso", tt.args...)
			output, err := cmd.CombinedOutput()
			
			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			
			if tt.shouldError && tt.errorCheck != nil {
				if !tt.errorCheck(string(output)) {
					t.Errorf("Error message doesn't match expected format\nOutput: %s", output)
				}
			}
		})
	}
}

// TestBackwardCompatibility ensures old command formats still work
func TestBackwardCompatibility(t *testing.T) {
	// These commands should continue to work
	compatibilityTests := []struct {
		name string
		args []string
	}{
		{
			name: "Basic vm list",
			args: []string{"vm", "list"},
		},
		{
			name: "VM list with JSON",
			args: []string{"vm", "list", "--json"},
		},
		{
			name: "Help command",
			args: []string{"help"},
		},
		{
			name: "Help flag",
			args: []string{"--help"},
		},
	}

	for _, tt := range compatibilityTests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("../../ceso", tt.args...)
			if _, err := cmd.CombinedOutput(); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					// Help commands return exit code 0 or 1
					if exitErr.ExitCode() > 1 {
						t.Errorf("Command failed: %v", err)
					}
				} else {
					t.Errorf("Command failed: %v", err)
				}
			}
		})
	}
}

// TestPerformanceRegression ensures commands complete within expected time
func TestPerformanceRegression(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		maxLatency time.Duration
	}{
		{
			name:       "VM list",
			args:       []string{"vm", "list"},
			maxLatency: 100 * time.Millisecond,
		},
		{
			name:       "VM list JSON",
			args:       []string{"vm", "list", "--json"},
			maxLatency: 100 * time.Millisecond,
		},
		{
			name:       "Help",
			args:       []string{"--help"},
			maxLatency: 50 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			cmd := exec.Command("../../ceso", tt.args...)
			cmd.CombinedOutput()
			elapsed := time.Since(start)
			
			if elapsed > tt.maxLatency {
				t.Errorf("Command took %v, expected < %v", elapsed, tt.maxLatency)
			}
		})
	}
}

// TestDataConsistency ensures mock data remains consistent
func TestDataConsistency(t *testing.T) {
	// Run vm list multiple times and ensure output is consistent
	var outputs []string
	
	for i := 0; i < 3; i++ {
		cmd := exec.Command("../../ceso", "vm", "list", "--json")
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("Failed to run vm list: %v", err)
		}
		outputs = append(outputs, string(output))
	}
	
	// All outputs should be identical
	for i := 1; i < len(outputs); i++ {
		if outputs[i] != outputs[0] {
			t.Error("Inconsistent output between runs")
		}
	}
	
	// Parse and verify data structure
	var vms []map[string]interface{}
	if err := json.Unmarshal([]byte(outputs[0]), &vms); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	
	// Should always have 3 VMs
	if len(vms) != 3 {
		t.Errorf("Expected 3 VMs, got %d", len(vms))
	}
	
	// Specific VMs should always exist
	expectedVMs := map[string]bool{
		"ubuntu-web-01": false,
		"ubuntu-db-01":  false,
		"ubuntu-test":   false,
	}
	
	for _, vm := range vms {
		name, ok := vm["name"].(string)
		if !ok {
			t.Error("VM name is not a string")
			continue
		}
		if _, expected := expectedVMs[name]; expected {
			expectedVMs[name] = true
		}
	}
	
	for name, found := range expectedVMs {
		if !found {
			t.Errorf("Expected VM %q not found", name)
		}
	}
}