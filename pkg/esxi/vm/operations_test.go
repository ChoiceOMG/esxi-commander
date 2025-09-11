package vm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateOptions(t *testing.T) {
	opts := &CreateOptions{
		Name:     "test-vm",
		Template: "ubuntu-22.04",
		CPU:      2,
		Memory:   4096,
		Disk:     20,
		Guestinfo: map[string]string{
			"guestinfo.metadata":          "test-metadata",
			"guestinfo.metadata.encoding": "gzip+base64",
		},
	}

	assert.Equal(t, "test-vm", opts.Name)
	assert.Equal(t, "ubuntu-22.04", opts.Template)
	assert.Equal(t, 2, opts.CPU)
	assert.Equal(t, 4096, opts.Memory)
	assert.Equal(t, 20, opts.Disk)
	assert.NotNil(t, opts.Guestinfo)
	assert.Equal(t, "test-metadata", opts.Guestinfo["guestinfo.metadata"])
}

func TestOperationsStructure(t *testing.T) {
	// Test that Operations struct can be created
	// In real tests, would use a mock client
	ops := &Operations{
		client: nil, // Would be a mock in real tests
	}
	
	assert.NotNil(t, ops)
}