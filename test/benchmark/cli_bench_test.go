package benchmark

import (
	"encoding/json"
	"os/exec"
	"testing"
)

// BenchmarkVMList benchmarks the vm list command
func BenchmarkVMList(b *testing.B) {
	// Build binary once
	buildCmd := exec.Command("go", "build", "-o", "../../bench-ceso", "../../cmd/ceso")
	if err := buildCmd.Run(); err != nil {
		b.Fatalf("Failed to build binary: %v", err)
	}
	defer exec.Command("rm", "../../bench-ceso").Run()

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("../../bench-ceso", "vm", "list")
		if _, err := cmd.Output(); err != nil {
			b.Errorf("Command failed: %v", err)
		}
	}
}

// BenchmarkJSONOutput benchmarks JSON parsing and output
func BenchmarkJSONOutput(b *testing.B) {
	// Build binary once
	buildCmd := exec.Command("go", "build", "-o", "../../bench-ceso", "../../cmd/ceso")
	if err := buildCmd.Run(); err != nil {
		b.Fatalf("Failed to build binary: %v", err)
	}
	defer exec.Command("rm", "../../bench-ceso").Run()

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("../../bench-ceso", "vm", "list", "--json")
		output, err := cmd.Output()
		if err != nil {
			b.Errorf("Command failed: %v", err)
			continue
		}
		
		// Also benchmark JSON parsing
		var data []interface{}
		if err := json.Unmarshal(output, &data); err != nil {
			b.Errorf("JSON parsing failed: %v", err)
		}
	}
}

// BenchmarkCommandParsing benchmarks command parsing speed
func BenchmarkCommandParsing(b *testing.B) {
	// Build binary once
	buildCmd := exec.Command("go", "build", "-o", "../../bench-ceso", "../../cmd/ceso")
	if err := buildCmd.Run(); err != nil {
		b.Fatalf("Failed to build binary: %v", err)
	}
	defer exec.Command("rm", "../../bench-ceso").Run()

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("../../bench-ceso", "--help")
		if _, err := cmd.CombinedOutput(); err != nil {
			// Help returns exit code 0 or 1, which is ok
			if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() > 1 {
				b.Errorf("Command failed: %v", err)
			}
		}
	}
}

// BenchmarkMemoryUsage benchmarks memory usage patterns
func BenchmarkMemoryUsage(b *testing.B) {
	// Build binary once
	buildCmd := exec.Command("go", "build", "-o", "../../bench-ceso", "../../cmd/ceso")
	if err := buildCmd.Run(); err != nil {
		b.Fatalf("Failed to build binary: %v", err)
	}
	defer exec.Command("rm", "../../bench-ceso").Run()

	b.ResetTimer()
	
	// Track memory allocations
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("../../bench-ceso", "vm", "list", "--json")
		output, err := cmd.Output()
		if err != nil {
			b.Errorf("Command failed: %v", err)
			continue
		}
		
		// Force some memory operations
		var data []map[string]interface{}
		if err := json.Unmarshal(output, &data); err != nil {
			b.Errorf("JSON parsing failed: %v", err)
		}
		
		// Access the data to prevent optimization
		if len(data) > 0 {
			_ = data[0]["name"]
		}
	}
}

// BenchmarkStartupTime benchmarks CLI startup time
func BenchmarkStartupTime(b *testing.B) {
	// Build binary once
	buildCmd := exec.Command("go", "build", "-o", "../../bench-ceso", "../../cmd/ceso")
	if err := buildCmd.Run(); err != nil {
		b.Fatalf("Failed to build binary: %v", err)
	}
	defer exec.Command("rm", "../../bench-ceso").Run()

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Measure minimal command that just starts and exits
		cmd := exec.Command("../../bench-ceso", "completion", "bash")
		if _, err := cmd.Output(); err != nil {
			b.Errorf("Command failed: %v", err)
		}
	}
}

// BenchmarkComplexCommand benchmarks a complex command with multiple flags
func BenchmarkComplexCommand(b *testing.B) {
	// Build binary once
	buildCmd := exec.Command("go", "build", "-o", "../../bench-ceso", "../../cmd/ceso")
	if err := buildCmd.Run(); err != nil {
		b.Fatalf("Failed to build binary: %v", err)
	}
	defer exec.Command("rm", "../../bench-ceso").Run()

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("../../bench-ceso", 
			"--config", "/tmp/test.yaml",
			"vm", "list",
			"--json")
		if _, err := cmd.Output(); err != nil {
			b.Errorf("Command failed: %v", err)
		}
	}
}