package unit

import (
	"testing"

	"github.com/r11/esxi-commander/internal/defaults"
)

func TestDefaults(t *testing.T) {
	if defaults.GetCPU() != 2 {
		t.Errorf("Expected default CPU to be 2, got %d", defaults.GetCPU())
	}

	if defaults.GetRAM() != 4 {
		t.Errorf("Expected default RAM to be 4, got %d", defaults.GetRAM())
	}

	if defaults.GetDisk() != 20 {
		t.Errorf("Expected default disk to be 20, got %d", defaults.GetDisk())
	}

	if defaults.GetTemplate() != "ubuntu-22.04-lts" {
		t.Errorf("Expected default template to be 'ubuntu-22.04-lts', got %s", defaults.GetTemplate())
	}

	if defaults.GetDatastore() != "datastore1" {
		t.Errorf("Expected default datastore to be 'datastore1', got %s", defaults.GetDatastore())
	}

	if defaults.GetNetwork() != "VM Network" {
		t.Errorf("Expected default network to be 'VM Network', got %s", defaults.GetNetwork())
	}
}
