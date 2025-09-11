package pci

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/r11/esxi-commander/pkg/audit"
	"github.com/r11/esxi-commander/pkg/esxi/client"
	"github.com/r11/esxi-commander/pkg/esxi/pci"
)

var (
	gpusOnly     bool
	assignableOnly bool
	jsonOutput   bool
)

// PciCmd is the root command for PCI operations
var PciCmd = &cobra.Command{
	Use:   "pci",
	Short: "Manage PCI device passthrough",
	Long: `Manage PCI device passthrough for ESXi VMs.
	
This command allows you to list, enable, and manage PCI devices
for GPU passthrough and other hardware acceleration.`,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List PCI devices on the ESXi host",
	Long:  "List all PCI devices, with optional filtering for GPUs or assignable devices only.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runListDevices(cmd.Context())
	},
}

var infoCmd = &cobra.Command{
	Use:   "info <device-id>",
	Short: "Show detailed information about a PCI device",
	Long:  "Display detailed information about a specific PCI device including passthrough capability and assignment status.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDeviceInfo(cmd.Context(), args[0])
	},
}

var enableCmd = &cobra.Command{
	Use:   "enable <device-id>",
	Short: "Enable passthrough for a PCI device",
	Long:  "Enable PCI passthrough for a specific device. This requires a host reboot to take effect.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEnablePassthrough(cmd.Context(), args[0])
	},
}

var disableCmd = &cobra.Command{
	Use:   "disable <device-id>",
	Short: "Disable passthrough for a PCI device",
	Long:  "Disable PCI passthrough for a specific device. This requires a host reboot to take effect.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDisablePassthrough(cmd.Context(), args[0])
	},
}

func init() {
	// Add subcommands
	PciCmd.AddCommand(listCmd)
	PciCmd.AddCommand(infoCmd)
	PciCmd.AddCommand(enableCmd)
	PciCmd.AddCommand(disableCmd)

	// List command flags
	listCmd.Flags().BoolVar(&gpusOnly, "gpus", false, "show GPU devices only")
	listCmd.Flags().BoolVar(&assignableOnly, "assignable", false, "show assignable devices only")
	listCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
}

func runListDevices(ctx context.Context) error {
	// Audit log
	auditCtx := audit.GetLogger().LogOperation(ctx, "pci.list", map[string]interface{}{
		"gpus_only":     gpusOnly,
		"assignable_only": assignableOnly,
	})
	defer func() {
		if err := recover(); err != nil {
			auditCtx.Failure(fmt.Errorf("panic: %v", err))
			panic(err)
		}
	}()

	// Create ESXi client
	esxiClient, err := createESXiClient()
	if err != nil {
		auditCtx.Failure(err)
		return fmt.Errorf("failed to create ESXi client: %w", err)
	}
	defer esxiClient.Close()

	// Create PCI discovery
	discovery := pci.NewDiscovery(esxiClient)

	var devices []*pci.Device
	var gpus []*pci.GPUDevice

	if gpusOnly {
		gpus, err = discovery.ListGPUs(ctx)
		if err != nil {
			auditCtx.Failure(err)
			return fmt.Errorf("failed to list GPUs: %w", err)
		}
		// Convert to devices for unified handling
		devices = make([]*pci.Device, len(gpus))
		for i, gpu := range gpus {
			devices[i] = &gpu.Device
		}
	} else if assignableOnly {
		devices, err = discovery.ListAssignableDevices(ctx)
		if err != nil {
			auditCtx.Failure(err)
			return fmt.Errorf("failed to list assignable devices: %w", err)
		}
	} else {
		devices, err = discovery.ListDevices(ctx)
		if err != nil {
			auditCtx.Failure(err)
			return fmt.Errorf("failed to list devices: %w", err)
		}
	}

	auditCtx.Success()

	// Output results
	if jsonOutput {
		if gpusOnly {
			return outputJSON(gpus)
		}
		return outputJSON(devices)
	}

	return outputTable(devices, gpusOnly)
}

func createESXiClient() (*client.ESXiClient, error) {
	esxiCfg := &client.Config{
		Host:     viper.GetString("esxi.host"),
		User:     viper.GetString("esxi.user"),
		Password: os.Getenv("ESXI_PASSWORD"),
		Insecure: viper.GetBool("esxi.insecure"),
		Timeout:  30 * time.Second,
	}

	if esxiCfg.Password == "" {
		esxiCfg.Password = viper.GetString("esxi.password")
	}

	return client.NewClient(esxiCfg)
}

func runDeviceInfo(ctx context.Context, deviceID string) error {
	// Audit log
	auditCtx := audit.GetLogger().LogOperation(ctx, "pci.info", map[string]interface{}{
		"device_id": deviceID,
	})
	defer func() {
		if err := recover(); err != nil {
			auditCtx.Failure(fmt.Errorf("panic: %v", err))
			panic(err)
		}
	}()

	// Create ESXi client
	esxiClient, err := createESXiClient()
	if err != nil {
		auditCtx.Failure(err)
		return fmt.Errorf("failed to create ESXi client: %w", err)
	}
	defer esxiClient.Close()

	// Create PCI discovery
	discovery := pci.NewDiscovery(esxiClient)

	// Get device info
	device, err := discovery.GetDevice(ctx, deviceID)
	if err != nil {
		auditCtx.Failure(err)
		return fmt.Errorf("failed to get device info: %w", err)
	}

	auditCtx.Success()

	// If it's a GPU, get GPU-specific info
	if device.IsGPU() {
		gpus, err := discovery.ListGPUs(ctx)
		if err == nil {
			for _, gpu := range gpus {
				if gpu.ID == device.ID {
					return outputDeviceInfo(gpu)
				}
			}
		}
	}

	return outputDeviceInfo(device)
}

func runEnablePassthrough(ctx context.Context, deviceID string) error {
	// Audit log
	auditCtx := audit.GetLogger().LogOperation(ctx, "pci.enable", map[string]interface{}{
		"device_id": deviceID,
	})
	defer func() {
		if err := recover(); err != nil {
			auditCtx.Failure(fmt.Errorf("panic: %v", err))
			panic(err)
		}
	}()

	// Create ESXi client
	esxiClient, err := createESXiClient()
	if err != nil {
		auditCtx.Failure(err)
		return fmt.Errorf("failed to create ESXi client: %w", err)
	}
	defer esxiClient.Close()

	// Create PCI discovery
	discovery := pci.NewDiscovery(esxiClient)

	// Validate device exists and can be enabled
	device, err := discovery.GetDevice(ctx, deviceID)
	if err != nil {
		auditCtx.Failure(err)
		return fmt.Errorf("failed to get device: %w", err)
	}

	if !device.PassthroughCapable {
		err = fmt.Errorf("device %s does not support passthrough", deviceID)
		auditCtx.Failure(err)
		return err
	}

	// Enable passthrough
	err = discovery.EnablePassthrough(ctx, deviceID)
	if err != nil {
		auditCtx.Failure(err)
		return fmt.Errorf("failed to enable passthrough: %w", err)
	}

	auditCtx.Success()

	fmt.Printf("Passthrough enabled for device %s\n", deviceID)
	fmt.Println("NOTE: Host reboot required for changes to take effect")
	return nil
}

func runDisablePassthrough(ctx context.Context, deviceID string) error {
	// Audit log
	auditCtx := audit.GetLogger().LogOperation(ctx, "pci.disable", map[string]interface{}{
		"device_id": deviceID,
	})
	defer func() {
		if err := recover(); err != nil {
			auditCtx.Failure(fmt.Errorf("panic: %v", err))
			panic(err)
		}
	}()

	// Create ESXi client
	esxiClient, err := createESXiClient()
	if err != nil {
		auditCtx.Failure(err)
		return fmt.Errorf("failed to create ESXi client: %w", err)
	}
	defer esxiClient.Close()

	// Create PCI discovery
	discovery := pci.NewDiscovery(esxiClient)

	// Validate device exists
	_, err = discovery.GetDevice(ctx, deviceID)
	if err != nil {
		auditCtx.Failure(err)
		return fmt.Errorf("failed to get device: %w", err)
	}

	// Disable passthrough
	err = discovery.DisablePassthrough(ctx, deviceID)
	if err != nil {
		auditCtx.Failure(err)
		return fmt.Errorf("failed to disable passthrough: %w", err)
	}

	auditCtx.Success()

	fmt.Printf("Passthrough disabled for device %s\n", deviceID)
	fmt.Println("NOTE: Host reboot required for changes to take effect")
	return nil
}

func outputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func outputTable(devices []*pci.Device, gpuDetails bool) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	defer w.Flush()

	if gpuDetails {
		fmt.Fprintln(w, "ADDRESS\tTYPE\tVENDOR\tDEVICE\tMEMORY\tPASSTHROUGH\tASSIGNED")
	} else {
		fmt.Fprintln(w, "ADDRESS\tCLASS\tVENDOR\tDEVICE\tPASSTHROUGH\tASSIGNED")
	}

	for _, device := range devices {
		assigned := "No"
		if device.Assigned {
			assigned = "Yes"
		}
		
		passthrough := "No"
		if device.PassthroughCapable {
			if device.Assignable {
				passthrough = "Enabled"
			} else {
				passthrough = "Capable"
			}
		}

		if gpuDetails && device.IsGPU() {
			gpuType := device.GetGPUType()
			memory := "Unknown"
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", 
				device.Address, gpuType, device.VendorName, device.DeviceName, 
				memory, passthrough, assigned)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", 
				device.Address, device.ClassName, device.VendorName, device.DeviceName, 
				passthrough, assigned)
		}
	}

	return nil
}

func outputDeviceInfo(device interface{}) error {
	if jsonOutput {
		return outputJSON(device)
	}

	switch d := device.(type) {
	case *pci.GPUDevice:
		fmt.Printf("GPU Device Information:\n")
		fmt.Printf("  Address: %s\n", d.Address)
		fmt.Printf("  ID: %s\n", d.ID)
		fmt.Printf("  Vendor: %s (%s)\n", d.VendorName, d.Vendor)
		fmt.Printf("  Device: %s (%s)\n", d.DeviceName, d.Device.Device)
		fmt.Printf("  Class: %s (%s)\n", d.ClassName, d.Class)
		fmt.Printf("  GPU Type: %s\n", d.GPUType)
		fmt.Printf("  Memory: %d MB\n", d.Memory)
		fmt.Printf("  Passthrough Capable: %t\n", d.PassthroughCapable)
		fmt.Printf("  Assignable: %t\n", d.Assignable)
		fmt.Printf("  Assigned: %t\n", d.Assigned)
		if d.Assigned && d.AssignedTo != "" {
			fmt.Printf("  Assigned To: %s\n", d.AssignedTo)
		}
		if len(d.Profiles) > 0 {
			fmt.Printf("  vGPU Profiles: %v\n", d.Profiles)
		}
	case *pci.Device:
		fmt.Printf("PCI Device Information:\n")
		fmt.Printf("  Address: %s\n", d.Address)
		fmt.Printf("  ID: %s\n", d.ID)
		fmt.Printf("  Vendor: %s (%s)\n", d.VendorName, d.Vendor)
		fmt.Printf("  Device: %s (%s)\n", d.DeviceName, d.Device)
		fmt.Printf("  Class: %s (%s)\n", d.ClassName, d.Class)
		fmt.Printf("  Passthrough Capable: %t\n", d.PassthroughCapable)
		fmt.Printf("  Assignable: %t\n", d.Assignable)
		fmt.Printf("  Assigned: %t\n", d.Assigned)
		if d.Assigned && d.AssignedTo != "" {
			fmt.Printf("  Assigned To: %s\n", d.AssignedTo)
		}
	}

	return nil
}