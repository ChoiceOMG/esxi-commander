package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/r11/esxi-commander/pkg/config"
	"github.com/r11/esxi-commander/pkg/esxi/client"
	"github.com/spf13/cobra"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

var statsCmd = &cobra.Command{
	Use:   "stats <vm-name>",
	Short: "Display VM resource statistics",
	Long:  `Display current resource usage statistics for a virtual machine including CPU, memory, and storage.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runVMStats,
}

type VMResourceStats struct {
	VMName      string      `json:"vm_name"`
	Timestamp   time.Time   `json:"timestamp"`
	PowerState  string      `json:"power_state"`
	CPU         VMCPUStats  `json:"cpu"`
	Memory      VMMemStats  `json:"memory"`
	Storage     VMStorage   `json:"storage"`
	Network     VMNetwork   `json:"network"`
	GuestOS     string      `json:"guest_os"`
	VMwareTools string      `json:"vmware_tools"`
}

type VMCPUStats struct {
	UsageMHz      int32   `json:"usage_mhz"`
	AllocatedMHz  int32   `json:"allocated_mhz"`
	UsagePercent  float64 `json:"usage_percent"`
	CPUs          int32   `json:"cpus"`
}

type VMMemStats struct {
	UsageMB       int32   `json:"usage_mb"`
	AllocatedMB   int64   `json:"allocated_mb"`
	UsagePercent  float64 `json:"usage_percent"`
	SharedMB      int32   `json:"shared_mb"`
	BalloonedMB   int32   `json:"ballooned_mb"`
	CompressedMB  int32   `json:"compressed_mb"`
	SwappedMB     int32   `json:"swapped_mb"`
}

type VMStorage struct {
	TotalGB       int64 `json:"total_gb"`
	UsedGB        int64 `json:"used_gb"`
	ProvisionedGB int64 `json:"provisioned_gb"`
	UnsharedGB    int64 `json:"unshared_gb"`
}

type VMNetwork struct {
	BytesRx    int64 `json:"bytes_rx"`
	BytesTx    int64 `json:"bytes_tx"`
	PacketsRx  int64 `json:"packets_rx"`
	PacketsTx  int64 `json:"packets_tx"`
}

func runVMStats(cmd *cobra.Command, args []string) error {
	vmName := args[0]

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	clientConfig := &client.Config{
		Host:     cfg.ESXi.Host,
		User:     cfg.ESXi.User,
		Password: cfg.ESXi.Password,
		Insecure: cfg.ESXi.Insecure,
	}

	esxiClient, err := client.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create ESXi client: %w", err)
	}
	defer esxiClient.Close()

	ctx := context.Background()

	// Find the VM
	vmObj, err := esxiClient.FindVM(ctx, vmName)
	if err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	// Get VM properties
	var vm mo.VirtualMachine
	err = vmObj.Properties(ctx, vmObj.Reference(), []string{
		"summary",
		"runtime",
		"config",
		"guest",
		"storage",
	}, &vm)
	if err != nil {
		return fmt.Errorf("failed to get VM properties: %w", err)
	}

	stats := &VMResourceStats{
		VMName:    vmName,
		Timestamp: time.Now(),
		PowerState: string(vm.Runtime.PowerState),
		GuestOS:   vm.Config.GuestFullName,
	}

	// VMware Tools status
	if vm.Guest.ToolsStatus != "" {
		stats.VMwareTools = string(vm.Guest.ToolsStatus)
	} else {
		stats.VMwareTools = "unknown"
	}

	// CPU stats
	cpuUsage := vm.Summary.QuickStats.OverallCpuUsage
	cpuAllocation := vm.Config.Hardware.NumCPU * 1000 // Rough estimate
	var cpuPercent float64
	if cpuAllocation > 0 {
		cpuPercent = float64(cpuUsage) / float64(cpuAllocation) * 100
	}

	stats.CPU = VMCPUStats{
		UsageMHz:     cpuUsage,
		AllocatedMHz: cpuAllocation,
		UsagePercent: cpuPercent,
		CPUs:         vm.Config.Hardware.NumCPU,
	}

	// Memory stats
	memUsage := vm.Summary.QuickStats.GuestMemoryUsage
	memAllocated := vm.Config.Hardware.MemoryMB
	var memPercent float64
	if memAllocated > 0 {
		memPercent = float64(memUsage) / float64(memAllocated) * 100
	}

	stats.Memory = VMMemStats{
		UsageMB:      memUsage,
		AllocatedMB:  int64(memAllocated),
		UsagePercent: memPercent,
		SharedMB:     vm.Summary.QuickStats.SharedMemory,
		BalloonedMB:  vm.Summary.QuickStats.BalloonedMemory,
		CompressedMB: int32(vm.Summary.QuickStats.CompressedMemory),
		SwappedMB:    vm.Summary.QuickStats.SwappedMemory,
	}

	// Storage stats
	committedStorage := vm.Summary.Storage.Committed / (1024 * 1024 * 1024)  // Convert to GB
	uncommittedStorage := vm.Summary.Storage.Uncommitted / (1024 * 1024 * 1024)
	unsharedStorage := vm.Summary.Storage.Unshared / (1024 * 1024 * 1024)

	stats.Storage = VMStorage{
		TotalGB:       (committedStorage + uncommittedStorage),
		UsedGB:        committedStorage,
		ProvisionedGB: committedStorage + uncommittedStorage,
		UnsharedGB:    unsharedStorage,
	}

	// Network stats (basic - would need performance manager for detailed stats)
	stats.Network = VMNetwork{
		BytesRx:   0, // Would need performance counters
		BytesTx:   0,
		PacketsRx: 0,
		PacketsTx: 0,
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		output, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		fmt.Printf("VM Resource Statistics: %s\n", vmName)
		fmt.Printf("==============================\n")
		fmt.Printf("Timestamp:      %s\n", stats.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("Power State:    %s\n", stats.PowerState)
		fmt.Printf("Guest OS:       %s\n", stats.GuestOS)
		fmt.Printf("VMware Tools:   %s\n", stats.VMwareTools)
		
		fmt.Printf("\nCPU:\n")
		fmt.Printf("  vCPUs:        %d\n", stats.CPU.CPUs)
		fmt.Printf("  Usage:        %d MHz (%.1f%%)\n", stats.CPU.UsageMHz, stats.CPU.UsagePercent)
		
		fmt.Printf("\nMemory:\n")
		fmt.Printf("  Allocated:    %d MB\n", stats.Memory.AllocatedMB)
		fmt.Printf("  Usage:        %d MB (%.1f%%)\n", stats.Memory.UsageMB, stats.Memory.UsagePercent)
		if stats.Memory.SharedMB > 0 {
			fmt.Printf("  Shared:       %d MB\n", stats.Memory.SharedMB)
		}
		if stats.Memory.BalloonedMB > 0 {
			fmt.Printf("  Ballooned:    %d MB\n", stats.Memory.BalloonedMB)
		}
		if stats.Memory.CompressedMB > 0 {
			fmt.Printf("  Compressed:   %d MB\n", stats.Memory.CompressedMB)
		}
		if stats.Memory.SwappedMB > 0 {
			fmt.Printf("  Swapped:      %d MB\n", stats.Memory.SwappedMB)
		}

		fmt.Printf("\nStorage:\n")
		fmt.Printf("  Provisioned:  %d GB\n", stats.Storage.ProvisionedGB)
		fmt.Printf("  Used:         %d GB\n", stats.Storage.UsedGB)
		fmt.Printf("  Unshared:     %d GB\n", stats.Storage.UnsharedGB)

		if stats.PowerState != string(types.VirtualMachinePowerStatePoweredOn) {
			fmt.Printf("\nNote: Some statistics may not be available when VM is not running.\n")
		}
	}

	return nil
}