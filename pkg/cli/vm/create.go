package vm

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/r11/esxi-commander/pkg/cloudinit"
	"github.com/r11/esxi-commander/pkg/esxi/client"
	"github.com/r11/esxi-commander/pkg/esxi/pci"
	"github.com/r11/esxi-commander/pkg/esxi/vm"
)

var (
	template string
	ip       string
	gateway  string
	dns      []string
	sshKey   string
	cpu      int
	memory   int
	disk     int
	gpu      string
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new VM from template",
	Args:  cobra.ExactArgs(1),
	RunE:  runCreate,
}

func init() {
	createCmd.Flags().StringVar(&template, "template", "", "Template to clone from (required)")
	createCmd.Flags().StringVar(&ip, "ip", "", "Static IP in CIDR notation (e.g., 192.168.1.100/24)")
	createCmd.Flags().StringVar(&gateway, "gateway", "", "Gateway IP address")
	createCmd.Flags().StringSliceVar(&dns, "dns", []string{"8.8.8.8", "8.8.4.4"}, "DNS servers")
	createCmd.Flags().StringVar(&sshKey, "ssh-key", "", "SSH public key for ubuntu user")
	createCmd.Flags().IntVar(&cpu, "cpu", 2, "Number of vCPUs")
	createCmd.Flags().IntVar(&memory, "memory", 4, "Memory in GB")
	createCmd.Flags().IntVar(&disk, "disk", 40, "Disk size in GB")
	createCmd.Flags().StringVar(&gpu, "gpu", "", "PCI device ID for GPU passthrough (e.g., 0000:81:00.0)")
	
	createCmd.MarkFlagRequired("template")
}

func runCreate(cmd *cobra.Command, args []string) error {
	vmName := args[0]
	ctx := context.Background()
	
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		fmt.Printf("[DRY-RUN] Would create VM '%s' from template '%s'\n", vmName, template)
		if ip != "" {
			fmt.Printf("[DRY-RUN]   IP: %s\n", ip)
		}
		fmt.Printf("[DRY-RUN]   Resources: %d vCPU, %d GB RAM, %d GB disk\n", cpu, memory, disk)
		if gpu != "" {
			fmt.Printf("[DRY-RUN]   GPU: %s\n", gpu)
		}
		return nil
	}
	
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
	
	esxi, err := client.NewClient(esxiCfg)
	if err != nil {
		return fmt.Errorf("failed to connect to ESXi: %w", err)
	}
	defer esxi.Close()
	
	cloudInitData := &cloudinit.CloudInitData{
		Hostname: vmName,
		FQDN:     fmt.Sprintf("%s.local", vmName),
		IP:       ip,
		Gateway:  gateway,
		DNS:      dns,
	}
	
	if sshKey != "" {
		if strings.HasPrefix(sshKey, "@") {
			keyFile := strings.TrimPrefix(sshKey, "@")
			keyBytes, err := os.ReadFile(keyFile)
			if err != nil {
				return fmt.Errorf("failed to read SSH key file: %w", err)
			}
			cloudInitData.SSHKeys = []string{string(keyBytes)}
		} else {
			cloudInitData.SSHKeys = []string{sshKey}
		}
	}
	
	guestinfo, err := cloudinit.BuildGuestinfo(cloudInitData)
	if err != nil {
		return fmt.Errorf("failed to build cloud-init: %w", err)
	}
	
	start := time.Now()
	
	vmOps := vm.NewOperations(esxi)
	newVM, err := vmOps.CreateFromTemplate(ctx, &vm.CreateOptions{
		Name:      vmName,
		Template:  template,
		CPU:       cpu,
		Memory:    memory * 1024, // Convert to MB
		Disk:      disk,
		Guestinfo: guestinfo,
	})
	
	if err != nil {
		return fmt.Errorf("failed to create VM: %w", err)
	}
	
	// Attach GPU if specified (VM must be powered off for PCI attachment)
	if gpu != "" {
		fmt.Printf("Attaching GPU device %s...\n", gpu)
		pciAttachment := pci.NewAttachment(esxi)
		
		// Validate GPU device first
		if err := pciAttachment.ValidateAttachment(ctx, vmName, gpu); err != nil {
			return fmt.Errorf("GPU validation failed: %w", err)
		}
		
		// Attach the GPU device
		if err := pciAttachment.AttachDevice(ctx, vmName, gpu); err != nil {
			return fmt.Errorf("failed to attach GPU: %w", err)
		}
		
		fmt.Printf("✅ GPU %s attached successfully\n", gpu)
	}
	
	duration := time.Since(start)
	
	if err := vmOps.PowerOn(ctx, newVM); err != nil {
		return fmt.Errorf("failed to power on VM: %w", err)
	}
	
	fmt.Printf("✅ VM '%s' created successfully in %v\n", vmName, duration)
	fmt.Printf("   Template: %s\n", template)
	fmt.Printf("   Resources: %d vCPU, %d GB RAM, %d GB disk\n", cpu, memory, disk)
	if gpu != "" {
		fmt.Printf("   GPU: %s\n", gpu)
	}
	if ip != "" {
		fmt.Printf("   IP: %s\n", ip)
	}
	
	return nil
}
