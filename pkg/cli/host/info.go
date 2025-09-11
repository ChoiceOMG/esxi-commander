package host

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/r11/esxi-commander/pkg/config"
	"github.com/r11/esxi-commander/pkg/esxi/client"
	"github.com/spf13/cobra"
	"github.com/vmware/govmomi/vim25/mo"
)

// NewInfoCommand creates the host info command
func NewInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Display ESXi host information",
		Long:  `Display detailed information about the ESXi host including hardware, software, and configuration.`,
		RunE:  runInfo,
	}

	cmd.Flags().Bool("json", false, "Output in JSON format")

	return cmd
}

type HostInfo struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	Build           string `json:"build"`
	Vendor          string `json:"vendor"`
	Model           string `json:"model"`
	CPUSockets      int    `json:"cpu_sockets"`
	CPUCores        int    `json:"cpu_cores"`
	CPUThreads      int    `json:"cpu_threads"`
	CPUModel        string `json:"cpu_model"`
	MemoryTotal     int64  `json:"memory_total_gb"`
	PowerState      string `json:"power_state"`
	ConnectionState string `json:"connection_state"`
	InMaintenanceMode bool `json:"in_maintenance_mode"`
}

func runInfo(cmd *cobra.Command, args []string) error {
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
		"runtime",
		"config",
	}, &hostObj)
	if err != nil {
		return fmt.Errorf("failed to get host properties: %w", err)
	}

	hostInfo := &HostInfo{
		Name:              hostObj.Summary.Config.Name,
		Version:           hostObj.Summary.Config.Product.Version,
		Build:             hostObj.Summary.Config.Product.Build,
		Vendor:            hostObj.Hardware.SystemInfo.Vendor,
		Model:             hostObj.Hardware.SystemInfo.Model,
		CPUSockets:        int(hostObj.Hardware.CpuInfo.NumCpuPackages),
		CPUCores:          int(hostObj.Hardware.CpuInfo.NumCpuCores),
		CPUThreads:        int(hostObj.Hardware.CpuInfo.NumCpuThreads),
		CPUModel:          hostObj.Hardware.CpuPkg[0].Description,
		MemoryTotal:       hostObj.Hardware.MemorySize / (1024 * 1024 * 1024),
		PowerState:        string(hostObj.Runtime.PowerState),
		ConnectionState:   string(hostObj.Runtime.ConnectionState),
		InMaintenanceMode: hostObj.Runtime.InMaintenanceMode,
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		output, err := json.MarshalIndent(hostInfo, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		fmt.Printf("ESXi Host Information\n")
		fmt.Printf("====================\n\n")
		fmt.Printf("Name:               %s\n", hostInfo.Name)
		fmt.Printf("Version:            %s (Build %s)\n", hostInfo.Version, hostInfo.Build)
		fmt.Printf("Hardware:\n")
		fmt.Printf("  Vendor:           %s\n", hostInfo.Vendor)
		fmt.Printf("  Model:            %s\n", hostInfo.Model)
		fmt.Printf("  CPU Sockets:      %d\n", hostInfo.CPUSockets)
		fmt.Printf("  CPU Cores:        %d\n", hostInfo.CPUCores)
		fmt.Printf("  CPU Threads:      %d\n", hostInfo.CPUThreads)
		fmt.Printf("  CPU Model:        %s\n", hostInfo.CPUModel)
		fmt.Printf("  Memory:           %d GB\n", hostInfo.MemoryTotal)
		fmt.Printf("Status:\n")
		fmt.Printf("  Power State:      %s\n", hostInfo.PowerState)
		fmt.Printf("  Connection:       %s\n", hostInfo.ConnectionState)
		fmt.Printf("  Maintenance Mode: %t\n", hostInfo.InMaintenanceMode)
	}

	return nil
}