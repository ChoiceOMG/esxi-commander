package host

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/r11/esxi-commander/pkg/config"
	"github.com/r11/esxi-commander/pkg/esxi/client"
	"github.com/spf13/cobra"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// NewHealthCommand creates the host health command
func NewHealthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check ESXi host health status",
		Long:  `Check the health status of the ESXi host including overall status and sensor information.`,
		RunE:  runHealth,
	}

	cmd.Flags().Bool("json", false, "Output in JSON format")

	return cmd
}

type HealthStatus struct {
	OverallStatus string             `json:"overall_status"`
	Alarms        []AlarmInfo        `json:"alarms,omitempty"`
	Sensors       []SensorInfo       `json:"sensors,omitempty"`
	Issues        []string           `json:"issues,omitempty"`
}

type AlarmInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type SensorInfo struct {
	Name   string  `json:"name"`
	Type   string  `json:"type"`
	Status string  `json:"status"`
	Value  string  `json:"value,omitempty"`
}

func runHealth(cmd *cobra.Command, args []string) error {
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
		"overallStatus",
		"runtime",
		"hardware.systemInfo",
		"triggeredAlarmState",
	}, &hostObj)
	if err != nil {
		return fmt.Errorf("failed to get host properties: %w", err)
	}

	health := &HealthStatus{
		OverallStatus: string(hostObj.OverallStatus),
		Alarms:        []AlarmInfo{},
		Sensors:       []SensorInfo{},
		Issues:        []string{},
	}

	// Check for triggered alarms
	if len(hostObj.TriggeredAlarmState) > 0 {
		for _, alarm := range hostObj.TriggeredAlarmState {
			health.Alarms = append(health.Alarms, AlarmInfo{
				Name:        alarm.Key,
				Description: alarm.Key, // Would need alarm manager to get description
				Status:      string(alarm.OverallStatus),
			})
		}
	}

	// Check runtime status issues
	if hostObj.Runtime.InMaintenanceMode {
		health.Issues = append(health.Issues, "Host is in maintenance mode")
	}

	if hostObj.Runtime.ConnectionState != types.HostSystemConnectionStateConnected {
		health.Issues = append(health.Issues, fmt.Sprintf("Host connection state: %s", hostObj.Runtime.ConnectionState))
	}

	if hostObj.Runtime.PowerState != types.HostSystemPowerStatePoweredOn {
		health.Issues = append(health.Issues, fmt.Sprintf("Host power state: %s", hostObj.Runtime.PowerState))
	}

	// Try to get hardware health sensors (may not be available on all systems)
	var sensors []SensorInfo
	if hostObj.Runtime.HealthSystemRuntime != nil {
		// Add basic health status
		sensors = append(sensors, SensorInfo{
			Name:   "Overall Health",
			Type:   "system",
			Status: string(hostObj.OverallStatus),
		})
	}
	health.Sensors = sensors

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		output, err := json.MarshalIndent(health, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		fmt.Printf("ESXi Host Health Status\n")
		fmt.Printf("=======================\n\n")
		fmt.Printf("Overall Status: %s\n", health.OverallStatus)

		if len(health.Issues) > 0 {
			fmt.Printf("\nIssues:\n")
			for _, issue := range health.Issues {
				fmt.Printf("  ! %s\n", issue)
			}
		}

		if len(health.Alarms) > 0 {
			fmt.Printf("\nActive Alarms:\n")
			for _, alarm := range health.Alarms {
				fmt.Printf("  - %s: %s\n", alarm.Name, alarm.Status)
			}
		}

		if len(health.Sensors) > 0 {
			fmt.Printf("\nSensors:\n")
			for _, sensor := range health.Sensors {
				fmt.Printf("  - %s (%s): %s", sensor.Name, sensor.Type, sensor.Status)
				if sensor.Value != "" {
					fmt.Printf(" - %s", sensor.Value)
				}
				fmt.Printf("\n")
			}
		}

		if len(health.Issues) == 0 && len(health.Alarms) == 0 {
			fmt.Printf("\n✓ No health issues detected\n")
		}
	}

	return nil
}