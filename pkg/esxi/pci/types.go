package pci

import (
	"time"
)

// Device represents a PCI device on an ESXi host
type Device struct {
	ID           string            `json:"id"`
	Address      string            `json:"address"`      // PCI address (e.g., "0000:81:00.0")
	Vendor       string            `json:"vendor"`
	VendorName   string            `json:"vendor_name"`
	Device       string            `json:"device"`
	DeviceName   string            `json:"device_name"`
	Class        string            `json:"class"`
	ClassName    string            `json:"class_name"`
	SubVendor    string            `json:"sub_vendor,omitempty"`
	SubDevice    string            `json:"sub_device,omitempty"`
	Assignable   bool              `json:"assignable"`
	Assigned     bool              `json:"assigned"`
	AssignedTo   string            `json:"assigned_to,omitempty"`
	PassthroughCapable bool       `json:"passthrough_capable"`
	ResetCapable bool              `json:"reset_capable"`
	Properties   map[string]string `json:"properties,omitempty"`
}

// GPUDevice represents a GPU-specific PCI device
type GPUDevice struct {
	Device
	GPUType      string   `json:"gpu_type"`      // nvidia, amd, intel
	Memory       int64    `json:"memory"`        // Video memory in MB
	Profiles     []string `json:"profiles"`      // vGPU profiles available
	MaxInstances int      `json:"max_instances"` // Max vGPU instances
	Compute      string   `json:"compute"`       // Compute capability (e.g., "7.5" for CUDA)
}

// PassthroughConfig defines configuration for PCI passthrough
type PassthroughConfig struct {
	DeviceID     string            `json:"device_id"`
	DeviceAddress string           `json:"device_address"`
	VMName       string            `json:"vm_name"`
	VMID         string            `json:"vm_id"`
	ResetOnStop  bool              `json:"reset_on_stop"`
	ShareDevice  bool              `json:"share_device"`
	AllowP2P     bool              `json:"allow_p2p"`      // Allow peer-to-peer for multi-GPU
	Parameters   map[string]string `json:"parameters"`
	AssignedAt   time.Time         `json:"assigned_at"`
}

// VGPUProfile represents a virtual GPU profile
type VGPUProfile struct {
	Name         string `json:"name"`
	Type         string `json:"type"`         // Time-sliced, MIG, SR-IOV
	Memory       int64  `json:"memory"`       // MB
	MaxInstances int    `json:"max_instances"`
	Resolution   string `json:"resolution"`   // Max resolution
	Displays     int    `json:"displays"`     // Number of displays
	FPS          int    `json:"fps"`          // Target FPS
}

// DeviceClass constants
const (
	ClassDisplay    = "0300" // VGA compatible controller
	ClassNetwork    = "0200" // Ethernet controller
	Class3D         = "0302" // 3D controller
	ClassNVME       = "0108" // NVMe controller
	ClassSCSI       = "0100" // SCSI controller
	ClassUSB        = "0c03" // USB controller
)

// GPU vendor constants
const (
	VendorNVIDIA = "10de"
	VendorAMD    = "1002"
	VendorIntel  = "8086"
)

// Common GPU device IDs
var KnownGPUs = map[string]string{
	"10de:1b38": "Tesla P40",
	"10de:1db4": "Tesla V100",
	"10de:20b0": "A100-SXM4-40GB",
	"10de:2330": "H100-SXM5-80GB",
	"1002:7360": "Radeon Pro V520",
	"1002:73df": "Radeon RX 6900 XT",
	"8086:4905": "Intel Iris Xe",
}

// IsGPU checks if the device is a GPU
func (d *Device) IsGPU() bool {
	return d.Class == ClassDisplay || d.Class == Class3D
}

// IsNetwork checks if the device is a network adapter
func (d *Device) IsNetwork() bool {
	return d.Class == ClassNetwork
}

// GetGPUType returns the GPU vendor type
func (d *Device) GetGPUType() string {
	switch d.Vendor {
	case VendorNVIDIA:
		return "nvidia"
	case VendorAMD:
		return "amd"
	case VendorIntel:
		return "intel"
	default:
		return "unknown"
	}
}

// CanPassthrough checks if device can be passed through
func (d *Device) CanPassthrough() bool {
	return d.PassthroughCapable && !d.Assigned && d.Assignable
}