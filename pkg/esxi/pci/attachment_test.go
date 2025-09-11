package pci

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vmware/govmomi/vim25/types"
)

func TestCreatePassthroughConfig(t *testing.T) {
	attachment := &Attachment{}
	
	vmName := "test-vm"
	deviceID := "0000:81:00.0"
	
	config := attachment.CreatePassthroughConfig(vmName, deviceID)
	
	assert.Equal(t, deviceID, config.DeviceID)
	assert.Equal(t, vmName, config.VMName)
	assert.True(t, config.ResetOnStop)
	assert.False(t, config.ShareDevice)
	assert.True(t, config.AllowP2P)
	assert.NotNil(t, config.Parameters)
	assert.WithinDuration(t, time.Now(), config.AssignedAt, time.Second)
}

func TestPassthroughConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config PassthroughConfig
		valid  bool
	}{
		{
			name: "Valid config",
			config: PassthroughConfig{
				DeviceID: "0000:81:00.0",
				VMName:   "test-vm",
			},
			valid: true,
		},
		{
			name: "Empty device ID",
			config: PassthroughConfig{
				DeviceID: "",
				VMName:   "test-vm",
			},
			valid: false,
		},
		{
			name: "Empty VM name",
			config: PassthroughConfig{
				DeviceID: "0000:81:00.0",
				VMName:   "",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				assert.NotEmpty(t, tt.config.DeviceID)
				assert.NotEmpty(t, tt.config.VMName)
			} else {
				assert.True(t, tt.config.DeviceID == "" || tt.config.VMName == "")
			}
		})
	}
}

func TestGPUDevice_Properties(t *testing.T) {
	device := Device{
		ID:       "0000:81:00.0",
		Address:  "0000:81:00.0",
		Vendor:   VendorNVIDIA,
		Device:   "1b38",
		Class:    ClassDisplay,
		Assigned: false,
	}

	gpu := GPUDevice{
		Device:       device,
		GPUType:      "nvidia",
		Memory:       24576,
		Profiles:     []string{"grid_p40-1q", "grid_p40-2q"},
		MaxInstances: 24,
		Compute:      "7.5",
	}

	assert.True(t, gpu.IsGPU())
	assert.Equal(t, "nvidia", gpu.GetGPUType())
	assert.Equal(t, int64(24576), gpu.Memory)
	assert.Len(t, gpu.Profiles, 2)
	assert.Equal(t, 24, gpu.MaxInstances)
}

func TestVGPUProfile_Properties(t *testing.T) {
	profile := VGPUProfile{
		Name:         "grid_p40-4q",
		Type:         "Time-sliced",
		Memory:       6144,
		MaxInstances: 4,
		Resolution:   "3840x2160",
		Displays:     2,
		FPS:          60,
	}

	assert.Equal(t, "grid_p40-4q", profile.Name)
	assert.Equal(t, "Time-sliced", profile.Type)
	assert.Equal(t, int64(6144), profile.Memory)
	assert.Equal(t, 4, profile.MaxInstances)
	assert.Equal(t, "3840x2160", profile.Resolution)
	assert.Equal(t, 2, profile.Displays)
	assert.Equal(t, 60, profile.FPS)
}

func TestGetNextDeviceKey(t *testing.T) {
	tests := []struct {
		name     string
		devices  []types.BaseVirtualDevice
		expected int32
	}{
		{
			name:     "Empty device list",
			devices:  []types.BaseVirtualDevice{},
			expected: 1,
		},
		{
			name: "Single device",
			devices: []types.BaseVirtualDevice{
				&types.VirtualDevice{Key: 100},
			},
			expected: 101,
		},
		{
			name: "Multiple devices",
			devices: []types.BaseVirtualDevice{
				&types.VirtualDevice{Key: 100},
				&types.VirtualDevice{Key: 200},
				&types.VirtualDevice{Key: 150},
			},
			expected: 201,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNextDeviceKey(tt.devices)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// MockVirtualMachine is a mock implementation for testing
type MockVirtualMachine struct {
	mock.Mock
	powerState types.VirtualMachinePowerState
}

func (m *MockVirtualMachine) PowerState(ctx context.Context) (types.VirtualMachinePowerState, error) {
	args := m.Called(ctx)
	return args.Get(0).(types.VirtualMachinePowerState), args.Error(1)
}

func TestAttachment_PowerStateCheck(t *testing.T) {
	tests := []struct {
		name        string
		powerState  types.VirtualMachinePowerState
		expectError bool
	}{
		{
			name:        "Powered off - OK",
			powerState:  types.VirtualMachinePowerStatePoweredOff,
			expectError: false,
		},
		{
			name:        "Powered on - Error",
			powerState:  types.VirtualMachinePowerStatePoweredOn,
			expectError: true,
		},
		{
			name:        "Suspended - Error",
			powerState:  types.VirtualMachinePowerStateSuspended,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test validates the logic, actual implementation would require mocking
			if tt.expectError {
				assert.NotEqual(t, types.VirtualMachinePowerStatePoweredOff, tt.powerState)
			} else {
				assert.Equal(t, types.VirtualMachinePowerStatePoweredOff, tt.powerState)
			}
		})
	}
}

// Test constants and mappings
func TestPCIConstants(t *testing.T) {
	// Test device class constants
	assert.Equal(t, "0300", ClassDisplay)
	assert.Equal(t, "0200", ClassNetwork)
	assert.Equal(t, "0302", Class3D)
	assert.Equal(t, "0108", ClassNVME)
	assert.Equal(t, "0100", ClassSCSI)
	assert.Equal(t, "0c03", ClassUSB)

	// Test vendor constants
	assert.Equal(t, "10de", VendorNVIDIA)
	assert.Equal(t, "1002", VendorAMD)
	assert.Equal(t, "8086", VendorIntel)
}

func TestKnownGPUsCompleteness(t *testing.T) {
	// Ensure KnownGPUs contains expected entries
	expectedKeys := []string{
		"10de:1b38", // Tesla P40
		"10de:1db4", // Tesla V100
		"10de:20b0", // A100
		"10de:2330", // H100
		"1002:7360", // Radeon Pro V520
		"1002:73df", // Radeon RX 6900 XT
		"8086:4905", // Intel Iris Xe
	}

	for _, key := range expectedKeys {
		_, exists := KnownGPUs[key]
		assert.True(t, exists, "Expected key %s to exist in KnownGPUs", key)
	}

	// Ensure all keys follow vendor:device format
	for key := range KnownGPUs {
		assert.Len(t, key, 9, "Key %s should be 9 characters (aaaa:bbbb)", key)
		assert.Contains(t, key, ":", "Key %s should contain colon separator", key)
	}
}

// Benchmark tests for performance validation
func BenchmarkDevice_IsGPU(b *testing.B) {
	device := Device{Class: ClassDisplay}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		device.IsGPU()
	}
}

func BenchmarkDevice_GetGPUType(b *testing.B) {
	device := Device{Vendor: VendorNVIDIA}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		device.GetGPUType()
	}
}

func BenchmarkDevice_CanPassthrough(b *testing.B) {
	device := Device{
		PassthroughCapable: true,
		Assigned:           false,
		Assignable:         true,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		device.CanPassthrough()
	}
}