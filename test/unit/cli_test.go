package unit

import (
	"testing"

	"github.com/r11/esxi-commander/internal/validation"
	"github.com/r11/esxi-commander/pkg/cli/template"
	"github.com/r11/esxi-commander/pkg/cli/vm"
)

func TestVMTypes(t *testing.T) {
	vm := vm.VM{
		Name:   "test-vm",
		Status: "running",
		IP:     "192.168.1.100",
		CPU:    2,
		RAM:    4,
	}

	if vm.Name != "test-vm" {
		t.Errorf("Expected VM name to be 'test-vm', got %s", vm.Name)
	}

	if vm.CPU != 2 {
		t.Errorf("Expected VM CPU to be 2, got %d", vm.CPU)
	}
}

func TestTemplateValidator(t *testing.T) {
	validator := template.TemplateValidator{
		Name: "ubuntu-22.04-lts",
		Path: "/vmfs/volumes/datastore1/ubuntu-22.04-template",
	}

	result, err := validator.Validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !result.Valid {
		t.Error("Expected validation result to be valid")
	}

	if !result.CloudInitInstalled {
		t.Error("Expected CloudInit to be installed")
	}
}

func TestValidation(t *testing.T) {
	// Test VM name validation
	if err := validation.ValidateVMName("test-vm"); err != nil {
		t.Errorf("Valid VM name failed validation: %v", err)
	}

	if err := validation.ValidateVMName(""); err == nil {
		t.Error("Empty VM name should fail validation")
	}

	if err := validation.ValidateVMName("test@vm"); err == nil {
		t.Error("VM name with special characters should fail validation")
	}

	// Test IP validation
	if err := validation.ValidateIP("192.168.1.100"); err != nil {
		t.Errorf("Valid IP failed validation: %v", err)
	}

	if err := validation.ValidateIP("invalid-ip"); err == nil {
		t.Error("Invalid IP should fail validation")
	}

	// Test CIDR validation
	if err := validation.ValidateCIDR("192.168.1.0/24"); err != nil {
		t.Errorf("Valid CIDR failed validation: %v", err)
	}

	if err := validation.ValidateCIDR("invalid-cidr"); err == nil {
		t.Error("Invalid CIDR should fail validation")
	}
}
