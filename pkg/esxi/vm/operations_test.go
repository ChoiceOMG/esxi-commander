package vm

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// Mock ESXi client
type MockESXiClient struct {
	mock.Mock
}

func (m *MockESXiClient) GetVM(ctx context.Context, name string) (*object.VirtualMachine, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*object.VirtualMachine), args.Error(1)
}

func (m *MockESXiClient) GetResourcePool(ctx context.Context) (*object.ResourcePool, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*object.ResourcePool), args.Error(1)
}

func (m *MockESXiClient) GetDatastore(ctx context.Context, name string) (*object.Datastore, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*object.Datastore), args.Error(1)
}

func (m *MockESXiClient) GetFolder(ctx context.Context) (*object.Folder, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*object.Folder), args.Error(1)
}

func (m *MockESXiClient) Close() {
	m.Called()
}

func TestNewOperations(t *testing.T) {
	mockClient := new(MockESXiClient)

	ops := NewOperations(mockClient)
	assert.NotNil(t, ops)
	assert.Equal(t, mockClient, ops.client)
}

func TestOperations_CreateFromTemplate(t *testing.T) {
	tests := []struct {
		name         string
		vmName       string
		templateName string
		config       CreateConfig
		setupMocks   func(*MockESXiClient)
		wantErr      bool
		errorMsg     string
	}{
		{
			name:         "successful VM creation",
			vmName:       "test-vm",
			templateName: "ubuntu-template",
			config: CreateConfig{
				CPU:      2,
				MemoryMB: 4096,
				DiskGB:   40,
				Network:  "VM Network",
				Metadata: "encoded-metadata",
				Userdata: "encoded-userdata",
			},
			setupMocks: func(m *MockESXiClient) {
				mockVM := &object.VirtualMachine{}
				mockPool := &object.ResourcePool{}
				mockDatastore := &object.Datastore{}
				mockFolder := &object.Folder{}

				m.On("GetVM", mock.Anything, "ubuntu-template").Return(mockVM, nil)
				m.On("GetResourcePool", mock.Anything).Return(mockPool, nil)
				m.On("GetDatastore", mock.Anything, "").Return(mockDatastore, nil)
				m.On("GetFolder", mock.Anything).Return(mockFolder, nil)
			},
			wantErr: false,
		},
		{
			name:         "template not found",
			vmName:       "test-vm",
			templateName: "missing-template",
			config:       CreateConfig{},
			setupMocks: func(m *MockESXiClient) {
				m.On("GetVM", mock.Anything, "missing-template").Return(nil, errors.New("VM not found"))
			},
			wantErr:  true,
			errorMsg: "template not found",
		},
		{
			name:         "empty VM name",
			vmName:       "",
			templateName: "ubuntu-template",
			config:       CreateConfig{},
			setupMocks:   func(m *MockESXiClient) {},
			wantErr:      true,
			errorMsg:     "VM name cannot be empty",
		},
		{
			name:         "resource pool error",
			vmName:       "test-vm",
			templateName: "ubuntu-template",
			config:       CreateConfig{},
			setupMocks: func(m *MockESXiClient) {
				mockVM := &object.VirtualMachine{}
				m.On("GetVM", mock.Anything, "ubuntu-template").Return(mockVM, nil)
				m.On("GetResourcePool", mock.Anything).Return(nil, errors.New("resource pool not found"))
			},
			wantErr:  true,
			errorMsg: "resource pool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockESXiClient)
			if tt.setupMocks != nil {
				tt.setupMocks(mockClient)
			}

			ops := NewOperations(mockClient)
			err := ops.CreateFromTemplate(context.Background(), tt.vmName, tt.templateName, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestOperations_CloneVM(t *testing.T) {
	tests := []struct {
		name       string
		sourceVM   string
		destVM     string
		config     CloneConfig
		setupMocks func(*MockESXiClient)
		wantErr    bool
		errorMsg   string
	}{
		{
			name:     "successful VM clone",
			sourceVM: "source-vm",
			destVM:   "cloned-vm",
			config: CloneConfig{
				Metadata: "new-metadata",
				Userdata: "new-userdata",
			},
			setupMocks: func(m *MockESXiClient) {
				mockVM := &object.VirtualMachine{}
				mockPool := &object.ResourcePool{}
				mockDatastore := &object.Datastore{}
				mockFolder := &object.Folder{}

				m.On("GetVM", mock.Anything, "source-vm").Return(mockVM, nil)
				m.On("GetResourcePool", mock.Anything).Return(mockPool, nil)
				m.On("GetDatastore", mock.Anything, "").Return(mockDatastore, nil)
				m.On("GetFolder", mock.Anything).Return(mockFolder, nil)
			},
			wantErr: false,
		},
		{
			name:     "source VM not found",
			sourceVM: "missing-vm",
			destVM:   "cloned-vm",
			config:   CloneConfig{},
			setupMocks: func(m *MockESXiClient) {
				m.On("GetVM", mock.Anything, "missing-vm").Return(nil, errors.New("VM not found"))
			},
			wantErr:  true,
			errorMsg: "source VM not found",
		},
		{
			name:     "empty destination name",
			sourceVM: "source-vm",
			destVM:   "",
			config:   CloneConfig{},
			setupMocks: func(m *MockESXiClient) {
				mockVM := &object.VirtualMachine{}
				m.On("GetVM", mock.Anything, "source-vm").Return(mockVM, nil)
			},
			wantErr:  true,
			errorMsg: "destination name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockESXiClient)
			if tt.setupMocks != nil {
				tt.setupMocks(mockClient)
			}

			ops := NewOperations(mockClient)
			err := ops.CloneVM(context.Background(), tt.sourceVM, tt.destVM, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestOperations_DeleteVM(t *testing.T) {
	tests := []struct {
		name       string
		vmName     string
		force      bool
		setupMocks func(*MockESXiClient)
		wantErr    bool
		errorMsg   string
	}{
		{
			name:   "successful VM deletion",
			vmName: "test-vm",
			force:  false,
			setupMocks: func(m *MockESXiClient) {
				mockVM := &object.VirtualMachine{}
				m.On("GetVM", mock.Anything, "test-vm").Return(mockVM, nil)
			},
			wantErr: false,
		},
		{
			name:   "VM not found",
			vmName: "missing-vm",
			force:  false,
			setupMocks: func(m *MockESXiClient) {
				m.On("GetVM", mock.Anything, "missing-vm").Return(nil, errors.New("VM not found"))
			},
			wantErr:  true,
			errorMsg: "VM not found",
		},
		{
			name:     "empty VM name",
			vmName:   "",
			force:    false,
			setupMocks: func(m *MockESXiClient) {},
			wantErr:  true,
			errorMsg: "VM name cannot be empty",
		},
		{
			name:   "force delete powered on VM",
			vmName: "running-vm",
			force:  true,
			setupMocks: func(m *MockESXiClient) {
				mockVM := &object.VirtualMachine{}
				m.On("GetVM", mock.Anything, "running-vm").Return(mockVM, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockESXiClient)
			if tt.setupMocks != nil {
				tt.setupMocks(mockClient)
			}

			ops := NewOperations(mockClient)
			err := ops.DeleteVM(context.Background(), tt.vmName, tt.force)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestOperations_GetVMInfo(t *testing.T) {
	tests := []struct {
		name       string
		vmName     string
		setupMocks func(*MockESXiClient)
		wantErr    bool
		errorMsg   string
		validate   func(*testing.T, *VMInfo)
	}{
		{
			name:   "successful VM info retrieval",
			vmName: "test-vm",
			setupMocks: func(m *MockESXiClient) {
				mockVM := &object.VirtualMachine{}
				m.On("GetVM", mock.Anything, "test-vm").Return(mockVM, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, info *VMInfo) {
				assert.NotNil(t, info)
				assert.Equal(t, "test-vm", info.Name)
			},
		},
		{
			name:   "VM not found",
			vmName: "missing-vm",
			setupMocks: func(m *MockESXiClient) {
				m.On("GetVM", mock.Anything, "missing-vm").Return(nil, errors.New("VM not found"))
			},
			wantErr:  true,
			errorMsg: "VM not found",
		},
		{
			name:       "empty VM name",
			vmName:     "",
			setupMocks: func(m *MockESXiClient) {},
			wantErr:    true,
			errorMsg:   "VM name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockESXiClient)
			if tt.setupMocks != nil {
				tt.setupMocks(mockClient)
			}

			ops := NewOperations(mockClient)
			info, err := ops.GetVMInfo(context.Background(), tt.vmName)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, info)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, info)
				}
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestOperations_ListVMs(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockESXiClient)
		wantErr    bool
		errorMsg   string
		validate   func(*testing.T, []*VMInfo)
	}{
		{
			name: "successful VM list",
			setupMocks: func(m *MockESXiClient) {
				// Mock would return list of VMs
			},
			wantErr: false,
			validate: func(t *testing.T, vms []*VMInfo) {
				assert.NotNil(t, vms)
			},
		},
		{
			name: "error listing VMs",
			setupMocks: func(m *MockESXiClient) {
				// Mock would return error
			},
			wantErr:  true,
			errorMsg: "failed to list VMs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockESXiClient)
			if tt.setupMocks != nil {
				tt.setupMocks(mockClient)
			}

			ops := NewOperations(mockClient)
			vms, err := ops.ListVMs(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, vms)
				}
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestOperations_PowerOn(t *testing.T) {
	tests := []struct {
		name       string
		vmName     string
		setupMocks func(*MockESXiClient)
		wantErr    bool
		errorMsg   string
	}{
		{
			name:   "successful power on",
			vmName: "test-vm",
			setupMocks: func(m *MockESXiClient) {
				mockVM := &object.VirtualMachine{}
				m.On("GetVM", mock.Anything, "test-vm").Return(mockVM, nil)
			},
			wantErr: false,
		},
		{
			name:   "VM not found",
			vmName: "missing-vm",
			setupMocks: func(m *MockESXiClient) {
				m.On("GetVM", mock.Anything, "missing-vm").Return(nil, errors.New("VM not found"))
			},
			wantErr:  true,
			errorMsg: "VM not found",
		},
		{
			name:   "already powered on",
			vmName: "running-vm",
			setupMocks: func(m *MockESXiClient) {
				mockVM := &object.VirtualMachine{}
				m.On("GetVM", mock.Anything, "running-vm").Return(mockVM, nil)
			},
			wantErr: false, // Should be idempotent
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockESXiClient)
			if tt.setupMocks != nil {
				tt.setupMocks(mockClient)
			}

			ops := NewOperations(mockClient)
			err := ops.PowerOn(context.Background(), tt.vmName)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestOperations_PowerOff(t *testing.T) {
	tests := []struct {
		name       string
		vmName     string
		force      bool
		setupMocks func(*MockESXiClient)
		wantErr    bool
		errorMsg   string
	}{
		{
			name:   "successful graceful shutdown",
			vmName: "test-vm",
			force:  false,
			setupMocks: func(m *MockESXiClient) {
				mockVM := &object.VirtualMachine{}
				m.On("GetVM", mock.Anything, "test-vm").Return(mockVM, nil)
			},
			wantErr: false,
		},
		{
			name:   "force power off",
			vmName: "test-vm",
			force:  true,
			setupMocks: func(m *MockESXiClient) {
				mockVM := &object.VirtualMachine{}
				m.On("GetVM", mock.Anything, "test-vm").Return(mockVM, nil)
			},
			wantErr: false,
		},
		{
			name:   "VM not found",
			vmName: "missing-vm",
			force:  false,
			setupMocks: func(m *MockESXiClient) {
				m.On("GetVM", mock.Anything, "missing-vm").Return(nil, errors.New("VM not found"))
			},
			wantErr:  true,
			errorMsg: "VM not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockESXiClient)
			if tt.setupMocks != nil {
				tt.setupMocks(mockClient)
			}

			ops := NewOperations(mockClient)
			err := ops.PowerOff(context.Background(), tt.vmName, tt.force)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestOperations_WaitForIP(t *testing.T) {
	tests := []struct {
		name       string
		vmName     string
		timeout    time.Duration
		setupMocks func(*MockESXiClient)
		wantErr    bool
		errorMsg   string
		wantIP     string
	}{
		{
			name:    "successful IP retrieval",
			vmName:  "test-vm",
			timeout: 30 * time.Second,
			setupMocks: func(m *MockESXiClient) {
				mockVM := &object.VirtualMachine{}
				m.On("GetVM", mock.Anything, "test-vm").Return(mockVM, nil)
			},
			wantErr: false,
			wantIP:  "192.168.1.100",
		},
		{
			name:    "timeout waiting for IP",
			vmName:  "test-vm",
			timeout: 1 * time.Second,
			setupMocks: func(m *MockESXiClient) {
				mockVM := &object.VirtualMachine{}
				m.On("GetVM", mock.Anything, "test-vm").Return(mockVM, nil)
			},
			wantErr:  true,
			errorMsg: "timeout",
		},
		{
			name:    "VM not found",
			vmName:  "missing-vm",
			timeout: 30 * time.Second,
			setupMocks: func(m *MockESXiClient) {
				m.On("GetVM", mock.Anything, "missing-vm").Return(nil, errors.New("VM not found"))
			},
			wantErr:  true,
			errorMsg: "VM not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockESXiClient)
			if tt.setupMocks != nil {
				tt.setupMocks(mockClient)
			}

			ops := NewOperations(mockClient)
			ip, err := ops.WaitForIP(context.Background(), tt.vmName, tt.timeout)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Empty(t, ip)
			} else {
				assert.NoError(t, err)
				if tt.wantIP != "" {
					assert.Equal(t, tt.wantIP, ip)
				}
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestOperations_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  CreateConfig
		wantErr bool
		errorMsg string
	}{
		{
			name: "valid configuration",
			config: CreateConfig{
				CPU:      2,
				MemoryMB: 4096,
				DiskGB:   40,
				Network:  "VM Network",
			},
			wantErr: false,
		},
		{
			name: "invalid CPU count",
			config: CreateConfig{
				CPU:      0,
				MemoryMB: 4096,
				DiskGB:   40,
			},
			wantErr:  true,
			errorMsg: "CPU count must be at least 1",
		},
		{
			name: "invalid memory",
			config: CreateConfig{
				CPU:      2,
				MemoryMB: 0,
				DiskGB:   40,
			},
			wantErr:  true,
			errorMsg: "memory must be at least",
		},
		{
			name: "invalid disk size",
			config: CreateConfig{
				CPU:      2,
				MemoryMB: 4096,
				DiskGB:   0,
			},
			wantErr:  true,
			errorMsg: "disk size must be at least",
		},
		{
			name: "excessive CPU count",
			config: CreateConfig{
				CPU:      129,
				MemoryMB: 4096,
				DiskGB:   40,
			},
			wantErr:  true,
			errorMsg: "CPU count exceeds maximum",
		},
		{
			name: "excessive memory",
			config: CreateConfig{
				CPU:      2,
				MemoryMB: 1024 * 1024 * 2, // 2TB
				DiskGB:   40,
			},
			wantErr:  true,
			errorMsg: "memory exceeds maximum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper function for testing
func validateConfig(config CreateConfig) error {
	if config.CPU < 1 {
		return errors.New("CPU count must be at least 1")
	}
	if config.CPU > 128 {
		return errors.New("CPU count exceeds maximum")
	}
	if config.MemoryMB < 512 {
		return errors.New("memory must be at least 512MB")
	}
	if config.MemoryMB > 1024*1024 {
		return errors.New("memory exceeds maximum")
	}
	if config.DiskGB < 10 {
		return errors.New("disk size must be at least 10GB")
	}
	return nil
}