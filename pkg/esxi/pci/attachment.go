package pci

import (
	"context"
	"fmt"
	"time"

	"github.com/r11/esxi-commander/pkg/esxi/client"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// Attachment handles PCI device attachment to VMs
type Attachment struct {
	client *client.ESXiClient
}

// NewAttachment creates a new PCI attachment manager
func NewAttachment(c *client.ESXiClient) *Attachment {
	return &Attachment{client: c}
}

// AttachDevice attaches a PCI device to a VM
func (a *Attachment) AttachDevice(ctx context.Context, vmName string, deviceID string) error {
	// Get VM reference
	vm, err := a.client.FindVM(ctx, vmName)
	if err != nil {
		return fmt.Errorf("failed to find VM %s: %w", vmName, err)
	}

	// Get device information
	discovery := NewDiscovery(a.client)
	device, err := discovery.GetDevice(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("failed to get device %s: %w", deviceID, err)
	}

	// Validate device can be attached
	err = discovery.ValidateDeviceForVM(ctx, deviceID, vmName)
	if err != nil {
		return fmt.Errorf("device validation failed: %w", err)
	}

	// Check if VM is powered off
	powerState, err := vm.PowerState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get VM power state: %w", err)
	}
	if powerState != types.VirtualMachinePowerStatePoweredOff {
		return fmt.Errorf("VM must be powered off to attach PCI devices")
	}

	// Get VM config
	var vmObj mo.VirtualMachine
	err = vm.Properties(ctx, vm.Reference(), []string{"config"}, &vmObj)
	if err != nil {
		return fmt.Errorf("failed to get VM properties: %w", err)
	}

	// Create PCI device config
	pciDevice := &types.VirtualPCIPassthrough{
		VirtualDevice: types.VirtualDevice{
			Key: getNextDeviceKey(vmObj.Config.Hardware.Device),
			DeviceInfo: &types.Description{
				Label:   fmt.Sprintf("PCI device %s", device.Address),
				Summary: fmt.Sprintf("%s %s", device.VendorName, device.DeviceName),
			},
			Backing: &types.VirtualPCIPassthroughVmiopBackingInfo{
				Vgpu: device.ID,
			},
		},
	}

	// Create device config spec
	deviceChange := &types.VirtualDeviceConfigSpec{
		Operation: types.VirtualDeviceConfigSpecOperationAdd,
		Device:    pciDevice,
	}

	// Create VM config spec
	configSpec := &types.VirtualMachineConfigSpec{
		DeviceChange: []types.BaseVirtualDeviceConfigSpec{deviceChange},
	}

	// Apply configuration
	task, err := vm.Reconfigure(ctx, *configSpec)
	if err != nil {
		return fmt.Errorf("failed to reconfigure VM: %w", err)
	}

	err = task.Wait(ctx)
	if err != nil {
		return fmt.Errorf("failed to attach PCI device: %w", err)
	}

	// Mark device as assigned (this would ideally be tracked in our database)
	device.Assigned = true
	device.AssignedTo = vmName

	return nil
}

// DetachDevice detaches a PCI device from a VM
func (a *Attachment) DetachDevice(ctx context.Context, vmName string, deviceID string) error {
	// Get VM reference
	vm, err := a.client.FindVM(ctx, vmName)
	if err != nil {
		return fmt.Errorf("failed to find VM %s: %w", vmName, err)
	}

	// Check if VM is powered off
	powerState, err := vm.PowerState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get VM power state: %w", err)
	}
	if powerState != types.VirtualMachinePowerStatePoweredOff {
		return fmt.Errorf("VM must be powered off to detach PCI devices")
	}

	// Get VM config
	var vmObj mo.VirtualMachine
	err = vm.Properties(ctx, vm.Reference(), []string{"config"}, &vmObj)
	if err != nil {
		return fmt.Errorf("failed to get VM properties: %w", err)
	}

	// Find the PCI device in VM config
	var deviceToRemove types.BaseVirtualDevice
	for _, device := range vmObj.Config.Hardware.Device {
		if pciDevice, ok := device.(*types.VirtualPCIPassthrough); ok {
			if backing, ok := pciDevice.Backing.(*types.VirtualPCIPassthroughVmiopBackingInfo); ok {
				if backing.Vgpu == deviceID {
					deviceToRemove = device
					break
				}
			}
		}
	}

	if deviceToRemove == nil {
		return fmt.Errorf("PCI device %s not found in VM %s", deviceID, vmName)
	}

	// Create device config spec for removal
	deviceChange := &types.VirtualDeviceConfigSpec{
		Operation: types.VirtualDeviceConfigSpecOperationRemove,
		Device:    deviceToRemove,
	}

	// Create VM config spec
	configSpec := &types.VirtualMachineConfigSpec{
		DeviceChange: []types.BaseVirtualDeviceConfigSpec{deviceChange},
	}

	// Apply configuration
	task, err := vm.Reconfigure(ctx, *configSpec)
	if err != nil {
		return fmt.Errorf("failed to reconfigure VM: %w", err)
	}

	err = task.Wait(ctx)
	if err != nil {
		return fmt.Errorf("failed to detach PCI device: %w", err)
	}

	return nil
}

// ListVMDevices lists all PCI devices attached to a VM
func (a *Attachment) ListVMDevices(ctx context.Context, vmName string) ([]*Device, error) {
	// Get VM reference
	vm, err := a.client.FindVM(ctx, vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to find VM %s: %w", vmName, err)
	}

	// Get VM config
	var vmObj mo.VirtualMachine
	err = vm.Properties(ctx, vm.Reference(), []string{"config"}, &vmObj)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM properties: %w", err)
	}

	// Get all host devices for reference
	discovery := NewDiscovery(a.client)
	hostDevices, err := discovery.ListDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list host devices: %w", err)
	}

	// Create map for quick lookup
	deviceMap := make(map[string]*Device)
	for _, device := range hostDevices {
		deviceMap[device.ID] = device
	}

	// Find PCI devices in VM config
	vmDevices := make([]*Device, 0)
	for _, device := range vmObj.Config.Hardware.Device {
		if pciDevice, ok := device.(*types.VirtualPCIPassthrough); ok {
			if backing, ok := pciDevice.Backing.(*types.VirtualPCIPassthroughVmiopBackingInfo); ok {
				if hostDevice, exists := deviceMap[backing.Vgpu]; exists {
					// Clone the device and mark as assigned
					vmDevice := *hostDevice
					vmDevice.Assigned = true
					vmDevice.AssignedTo = vmName
					vmDevices = append(vmDevices, &vmDevice)
				}
			}
		}
	}

	return vmDevices, nil
}

// GetVMGPUs gets all GPU devices attached to a VM
func (a *Attachment) GetVMGPUs(ctx context.Context, vmName string) ([]*GPUDevice, error) {
	devices, err := a.ListVMDevices(ctx, vmName)
	if err != nil {
		return nil, err
	}

	gpus := make([]*GPUDevice, 0)
	discovery := NewDiscovery(a.client)

	for _, device := range devices {
		if device.IsGPU() {
			gpu := &GPUDevice{
				Device:  *device,
				GPUType: device.GetGPUType(),
			}
			
			// Add GPU-specific properties
			gpu.Memory = discovery.getGPUMemory(device)
			gpu.Profiles = discovery.getVGPUProfiles(device)
			gpu.MaxInstances = discovery.getMaxInstances(device)
			
			gpus = append(gpus, gpu)
		}
	}

	return gpus, nil
}

// CreatePassthroughConfig creates a passthrough configuration
func (a *Attachment) CreatePassthroughConfig(vmName, deviceID string) *PassthroughConfig {
	return &PassthroughConfig{
		DeviceID:     deviceID,
		VMName:       vmName,
		ResetOnStop:  true,
		ShareDevice:  false,
		AllowP2P:     true,
		Parameters:   make(map[string]string),
		AssignedAt:   time.Now(),
	}
}

// ValidateAttachment validates that a device can be attached to a VM
func (a *Attachment) ValidateAttachment(ctx context.Context, vmName, deviceID string) error {
	// Check if VM exists
	_, err := a.client.FindVM(ctx, vmName)
	if err != nil {
		return fmt.Errorf("VM %s not found: %w", vmName, err)
	}

	// Check if device exists and is available
	discovery := NewDiscovery(a.client)
	return discovery.ValidateDeviceForVM(ctx, deviceID, vmName)
}

// GetAttachedVMs returns a map of device ID to VM name for all attached devices
func (a *Attachment) GetAttachedVMs(ctx context.Context) (map[string]string, error) {
	attached := make(map[string]string)

	// Get all VMs
	vms, err := a.client.ListVMs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}

	// Check each VM for PCI devices
	for _, vm := range vms {
		vmName := vm.Name
		devices, err := a.ListVMDevices(ctx, vmName)
		if err != nil {
			continue // Skip VMs we can't query
		}

		for _, device := range devices {
			attached[device.ID] = vmName
		}
	}

	return attached, nil
}

// Helper functions

func getNextDeviceKey(devices []types.BaseVirtualDevice) int32 {
	maxKey := int32(0)
	for _, device := range devices {
		if device.GetVirtualDevice().Key > maxKey {
			maxKey = device.GetVirtualDevice().Key
		}
	}
	return maxKey + 1
}

// IsGPUCompatible checks if a GPU is compatible with a VM configuration
func (a *Attachment) IsGPUCompatible(ctx context.Context, vmName, deviceID string) (bool, error) {
	// Get VM info
	vm, err := a.client.FindVM(ctx, vmName)
	if err != nil {
		return false, err
	}

	var vmObj mo.VirtualMachine
	err = vm.Properties(ctx, vm.Reference(), []string{"config"}, &vmObj)
	if err != nil {
		return false, err
	}

	// Check VM version (GPU passthrough requires recent VM hardware)
	if vmObj.Config.Version == "" {
		return false, fmt.Errorf("unable to determine VM hardware version")
	}

	// Get device info
	discovery := NewDiscovery(a.client)
	device, err := discovery.GetDevice(ctx, deviceID)
	if err != nil {
		return false, err
	}

	if !device.IsGPU() {
		return false, fmt.Errorf("device %s is not a GPU", deviceID)
	}

	// Additional compatibility checks could include:
	// - VM memory requirements
	// - Host IOMMU configuration
	// - GPU driver compatibility
	// - vGPU profile requirements

	return true, nil
}