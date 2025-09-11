package pci

import (
	"context"
	"fmt"
	"strings"

	"github.com/r11/esxi-commander/pkg/esxi/client"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// Discovery handles PCI device discovery on ESXi hosts
type Discovery struct {
	client *client.ESXiClient
}

// NewDiscovery creates a new PCI discovery instance
func NewDiscovery(c *client.ESXiClient) *Discovery {
	return &Discovery{client: c}
}

// ListDevices lists all PCI devices on the host
func (d *Discovery) ListDevices(ctx context.Context) ([]*Device, error) {
	// Get host system
	host, err := d.client.DefaultHost(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	var hostSystem mo.HostSystem
	err = host.Properties(ctx, host.Reference(), []string{"hardware.pciDevice", "config.pciPassthruInfo"}, &hostSystem)
	if err != nil {
		return nil, fmt.Errorf("failed to get host properties: %w", err)
	}

	// Map passthrough info by ID
	passthroughInfo := make(map[string]*types.HostPciPassthruInfo)
	if hostSystem.Config != nil && hostSystem.Config.PciPassthruInfo != nil {
		for i := range hostSystem.Config.PciPassthruInfo {
			info := &hostSystem.Config.PciPassthruInfo[i]
			passthroughInfo[info.Id] = info
		}
	}

	// Convert to our Device type
	devices := make([]*Device, 0)
	if hostSystem.Hardware != nil && hostSystem.Hardware.PciDevice != nil {
		for _, pciDevice := range hostSystem.Hardware.PciDevice {
			device := &Device{
				ID:         pciDevice.Id,
				Vendor:     fmt.Sprintf("%04x", pciDevice.VendorId),
				VendorName: pciDevice.VendorName,
				Device:     fmt.Sprintf("%04x", pciDevice.DeviceId),
				DeviceName: pciDevice.DeviceName,
				Class:      fmt.Sprintf("%04x", pciDevice.ClassId),
				SubVendor:  fmt.Sprintf("%04x", pciDevice.SubVendorId),
				SubDevice:  fmt.Sprintf("%04x", pciDevice.SubDeviceId),
				Address:    pciDevice.Id, // PCI address
			}

			// Add passthrough info if available
			if ptInfo, ok := passthroughInfo[pciDevice.Id]; ok {
				device.PassthroughCapable = ptInfo.PassthruCapable
				device.Assignable = ptInfo.PassthruEnabled
				device.Assigned = ptInfo.PassthruActive
			}

			// Determine class name
			device.ClassName = getClassName(device.Class)
			
			devices = append(devices, device)
		}
	}

	return devices, nil
}

// ListGPUs lists only GPU devices
func (d *Discovery) ListGPUs(ctx context.Context) ([]*GPUDevice, error) {
	devices, err := d.ListDevices(ctx)
	if err != nil {
		return nil, err
	}

	gpus := make([]*GPUDevice, 0)
	for _, device := range devices {
		if device.IsGPU() {
			gpu := &GPUDevice{
				Device:  *device,
				GPUType: device.GetGPUType(),
			}
			
			// Add GPU-specific properties
			gpu.Memory = d.getGPUMemory(device)
			gpu.Profiles = d.getVGPUProfiles(device)
			gpu.MaxInstances = d.getMaxInstances(device)
			
			gpus = append(gpus, gpu)
		}
	}

	return gpus, nil
}

// ListAssignableDevices lists devices that can be assigned
func (d *Discovery) ListAssignableDevices(ctx context.Context) ([]*Device, error) {
	devices, err := d.ListDevices(ctx)
	if err != nil {
		return nil, err
	}

	assignable := make([]*Device, 0)
	for _, device := range devices {
		if device.CanPassthrough() {
			assignable = append(assignable, device)
		}
	}

	return assignable, nil
}

// GetDevice gets a specific PCI device by ID
func (d *Discovery) GetDevice(ctx context.Context, deviceID string) (*Device, error) {
	devices, err := d.ListDevices(ctx)
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		if device.ID == deviceID || device.Address == deviceID {
			return device, nil
		}
	}

	return nil, fmt.Errorf("device not found: %s", deviceID)
}

// EnablePassthrough enables passthrough for a device
func (d *Discovery) EnablePassthrough(ctx context.Context, deviceID string) error {
	// Get host system
	host, err := d.client.DefaultHost(ctx)
	if err != nil {
		return fmt.Errorf("failed to get host: %w", err)
	}

	// Create PCI passthrough config
	config := types.HostPciPassthruConfig{
		Id:              deviceID,
		PassthruEnabled: true,
	}

	// Update host PCI passthrough configuration
	task, err := host.ConfigManager().PciPassthruSystem(ctx).UpdatePassthruConfig(ctx, []types.HostPciPassthruConfig{config})
	if err != nil {
		return fmt.Errorf("failed to enable passthrough: %w", err)
	}

	// Wait for task completion
	err = task.Wait(ctx)
	if err != nil {
		return fmt.Errorf("failed to enable passthrough: %w", err)
	}

	return nil
}

// DisablePassthrough disables passthrough for a device
func (d *Discovery) DisablePassthrough(ctx context.Context, deviceID string) error {
	// Get host system
	host, err := d.client.DefaultHost(ctx)
	if err != nil {
		return fmt.Errorf("failed to get host: %w", err)
	}

	// Create PCI passthrough config
	config := types.HostPciPassthruConfig{
		Id:              deviceID,
		PassthruEnabled: false,
	}

	// Update host PCI passthrough configuration
	task, err := host.ConfigManager().PciPassthruSystem(ctx).UpdatePassthruConfig(ctx, []types.HostPciPassthruConfig{config})
	if err != nil {
		return fmt.Errorf("failed to disable passthrough: %w", err)
	}

	// Wait for task completion
	err = task.Wait(ctx)
	if err != nil {
		return fmt.Errorf("failed to disable passthrough: %w", err)
	}

	return nil
}

// Helper functions

func getClassName(class string) string {
	switch class {
	case ClassDisplay:
		return "VGA Controller"
	case Class3D:
		return "3D Controller"
	case ClassNetwork:
		return "Network Controller"
	case ClassNVME:
		return "NVMe Controller"
	case ClassSCSI:
		return "SCSI Controller"
	case ClassUSB:
		return "USB Controller"
	default:
		return "Unknown"
	}
}

func (d *Discovery) getGPUMemory(device *Device) int64 {
	// This would query actual GPU memory
	// For now, return estimates based on known models
	deviceKey := device.Vendor + ":" + device.Device
	switch deviceKey {
	case "10de:1b38": // Tesla P40
		return 24576 // 24GB
	case "10de:1db4": // Tesla V100
		return 32768 // 32GB
	case "10de:20b0": // A100
		return 40960 // 40GB
	case "10de:2330": // H100
		return 81920 // 80GB
	default:
		return 8192 // Default 8GB
	}
}

func (d *Discovery) getVGPUProfiles(device *Device) []string {
	if device.GetGPUType() != "nvidia" {
		return []string{}
	}
	
	// Return common vGPU profiles for NVIDIA GPUs
	// In production, query actual profiles from host
	return []string{
		"grid_p40-1q",
		"grid_p40-2q",
		"grid_p40-4q",
		"grid_p40-8q",
	}
}

func (d *Discovery) getMaxInstances(device *Device) int {
	// Based on GPU type and memory
	memory := d.getGPUMemory(device)
	if memory >= 24576 {
		return 24 // High-end GPU
	} else if memory >= 16384 {
		return 16 // Mid-range GPU
	}
	return 8 // Entry-level GPU
}

// GetAssignedVMs returns VMs that have PCI devices assigned
func (d *Discovery) GetAssignedVMs(ctx context.Context) (map[string][]string, error) {
	// This would query VMs and their PCI assignments
	// Returns map of VM name -> list of device IDs
	assigned := make(map[string][]string)
	
	// Implementation would iterate through VMs and check their PCI devices
	// For now, return empty map
	return assigned, nil
}

// ValidateDeviceForVM validates if a device can be assigned to a VM
func (d *Discovery) ValidateDeviceForVM(ctx context.Context, deviceID string, vmName string) error {
	device, err := d.GetDevice(ctx, deviceID)
	if err != nil {
		return err
	}

	if !device.PassthroughCapable {
		return fmt.Errorf("device %s is not passthrough capable", deviceID)
	}

	if device.Assigned {
		return fmt.Errorf("device %s is already assigned to %s", deviceID, device.AssignedTo)
	}

	if !device.Assignable {
		return fmt.Errorf("device %s is not assignable (passthrough not enabled)", deviceID)
	}

	// Additional validation could check:
	// - VM compatibility
	// - Resource requirements
	// - IOMMU groups
	// - Device dependencies

	return nil
}