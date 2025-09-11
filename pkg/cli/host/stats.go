package host

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

// NewStatsCommand creates the host stats command
func NewStatsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Display ESXi host resource statistics",
		Long:  `Display current resource usage statistics for the ESXi host including CPU, memory, and storage.`,
		RunE:  runStats,
	}

	cmd.Flags().Bool("json", false, "Output in JSON format")

	return cmd
}

type HostStats struct {
	Timestamp    time.Time    `json:"timestamp"`
	CPU          CPUStats     `json:"cpu"`
	Memory       MemoryStats  `json:"memory"`
	Network      NetworkStats `json:"network"`
	Datastores   []DSStats    `json:"datastores"`
	VMs          VMStats      `json:"vms"`
}

type CPUStats struct {
	UsageMHz       int32   `json:"usage_mhz"`
	TotalMHz       int32   `json:"total_mhz"`
	UsagePercent   float64 `json:"usage_percent"`
}

type MemoryStats struct {
	UsageMB      int32   `json:"usage_mb"`
	TotalMB      int64   `json:"total_mb"`
	UsagePercent float64 `json:"usage_percent"`
}

type NetworkStats struct {
	PacketsRx   int64 `json:"packets_rx"`
	PacketsTx   int64 `json:"packets_tx"`
	BytesRx     int64 `json:"bytes_rx"`
	BytesTx     int64 `json:"bytes_tx"`
}

type DSStats struct {
	Name         string  `json:"name"`
	CapacityGB   int64   `json:"capacity_gb"`
	FreeSpaceGB  int64   `json:"free_space_gb"`
	UsagePercent float64 `json:"usage_percent"`
}

type VMStats struct {
	Total    int `json:"total"`
	Running  int `json:"running"`
	Stopped  int `json:"stopped"`
	Suspended int `json:"suspended"`
}

func runStats(cmd *cobra.Command, args []string) error {
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

	// Get host information
	host, err := esxiClient.GetHostSystem(ctx)
	if err != nil {
		return fmt.Errorf("failed to get host system: %w", err)
	}

	var hostObj mo.HostSystem
	err = host.Properties(ctx, host.Reference(), []string{
		"summary",
		"hardware",
		"vm",
		"datastore",
	}, &hostObj)
	if err != nil {
		return fmt.Errorf("failed to get host properties: %w", err)
	}

	stats := &HostStats{
		Timestamp:  time.Now(),
		Datastores: []DSStats{},
	}

	// CPU stats
	cpuUsage := hostObj.Summary.QuickStats.OverallCpuUsage
	cpuTotal := int32(hostObj.Hardware.CpuInfo.Hz / 1000000 * int64(hostObj.Hardware.CpuInfo.NumCpuCores))
	
	stats.CPU = CPUStats{
		UsageMHz:     cpuUsage,
		TotalMHz:     cpuTotal,
		UsagePercent: float64(cpuUsage) / float64(cpuTotal) * 100,
	}

	// Memory stats  
	memUsage := hostObj.Summary.QuickStats.OverallMemoryUsage
	memTotal := hostObj.Hardware.MemorySize / (1024 * 1024) // Convert to MB

	stats.Memory = MemoryStats{
		UsageMB:      memUsage,
		TotalMB:      memTotal,
		UsagePercent: float64(memUsage) / float64(memTotal) * 100,
	}

	// Network stats (basic - would need performance manager for detailed stats)
	stats.Network = NetworkStats{
		PacketsRx: 0,
		PacketsTx: 0,
		BytesRx:   0,
		BytesTx:   0,
	}

	// Datastore stats
	for _, dsRef := range hostObj.Datastore {
		ds := types.ManagedObjectReference{Type: "Datastore", Value: dsRef.Value}
		var dsObj mo.Datastore
		err := esxiClient.RetrieveOne(ctx, ds, []string{"summary"}, &dsObj)
		if err != nil {
			continue
		}

		capacityGB := dsObj.Summary.Capacity / (1024 * 1024 * 1024)
		freeSpaceGB := dsObj.Summary.FreeSpace / (1024 * 1024 * 1024)
		usagePercent := float64(dsObj.Summary.Capacity-dsObj.Summary.FreeSpace) / float64(dsObj.Summary.Capacity) * 100

		stats.Datastores = append(stats.Datastores, DSStats{
			Name:         dsObj.Summary.Name,
			CapacityGB:   capacityGB,
			FreeSpaceGB:  freeSpaceGB,
			UsagePercent: usagePercent,
		})
	}

	// VM stats
	var running, stopped, suspended int
	for _, vmRef := range hostObj.Vm {
		vm := types.ManagedObjectReference{Type: "VirtualMachine", Value: vmRef.Value}
		var vmObj mo.VirtualMachine
		err := esxiClient.RetrieveOne(ctx, vm, []string{"runtime.powerState"}, &vmObj)
		if err != nil {
			continue
		}

		switch vmObj.Runtime.PowerState {
		case types.VirtualMachinePowerStatePoweredOn:
			running++
		case types.VirtualMachinePowerStatePoweredOff:
			stopped++
		case types.VirtualMachinePowerStateSuspended:
			suspended++
		}
	}

	stats.VMs = VMStats{
		Total:     len(hostObj.Vm),
		Running:   running,
		Stopped:   stopped,
		Suspended: suspended,
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		output, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		fmt.Printf("ESXi Host Resource Statistics\n")
		fmt.Printf("============================\n")
		fmt.Printf("Timestamp: %s\n\n", stats.Timestamp.Format("2006-01-02 15:04:05"))
		
		fmt.Printf("CPU:\n")
		fmt.Printf("  Usage:      %d MHz / %d MHz (%.1f%%)\n", 
			stats.CPU.UsageMHz, stats.CPU.TotalMHz, stats.CPU.UsagePercent)
		
		fmt.Printf("\nMemory:\n")
		fmt.Printf("  Usage:      %d MB / %d MB (%.1f%%)\n", 
			stats.Memory.UsageMB, stats.Memory.TotalMB, stats.Memory.UsagePercent)
		
		fmt.Printf("\nVirtual Machines:\n")
		fmt.Printf("  Total:      %d\n", stats.VMs.Total)
		fmt.Printf("  Running:    %d\n", stats.VMs.Running)
		fmt.Printf("  Stopped:    %d\n", stats.VMs.Stopped)
		fmt.Printf("  Suspended:  %d\n", stats.VMs.Suspended)

		fmt.Printf("\nDatastores:\n")
		for _, ds := range stats.Datastores {
			fmt.Printf("  %s:\n", ds.Name)
			fmt.Printf("    Capacity:   %d GB\n", ds.CapacityGB)
			fmt.Printf("    Free:       %d GB\n", ds.FreeSpaceGB)
			fmt.Printf("    Usage:      %.1f%%\n", ds.UsagePercent)
		}
	}

	return nil
}