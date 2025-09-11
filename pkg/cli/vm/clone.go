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
	"github.com/r11/esxi-commander/pkg/esxi/vm"
)

var (
	cloneIP      string
	cloneGateway string
	cloneDNS     []string
	cloneSSHKey  string
)

var cloneCmd = &cobra.Command{
	Use:   "clone <source> <dest>",
	Short: "Clone an existing VM",
	Long:  `Clone an existing VM with cold clone method and optional re-IP`,
	Args:  cobra.ExactArgs(2),
	RunE:  runClone,
}

func init() {
	cloneCmd.Flags().StringVar(&cloneIP, "ip", "", "New static IP in CIDR notation")
	cloneCmd.Flags().StringVar(&cloneGateway, "gateway", "", "Gateway IP address")
	cloneCmd.Flags().StringSliceVar(&cloneDNS, "dns", []string{"8.8.8.8", "8.8.4.4"}, "DNS servers")
	cloneCmd.Flags().StringVar(&cloneSSHKey, "ssh-key", "", "SSH public key for ubuntu user")
}

func runClone(cmd *cobra.Command, args []string) error {
	sourceName := args[0]
	destName := args[1]
	ctx := context.Background()

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		fmt.Printf("[DRY-RUN] Would clone VM '%s' to '%s'\n", sourceName, destName)
		if cloneIP != "" {
			fmt.Printf("[DRY-RUN]   New IP: %s\n", cloneIP)
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

	var guestinfo map[string]string
	if cloneIP != "" {
		cloudInitData := &cloudinit.CloudInitData{
			Hostname: destName,
			FQDN:     fmt.Sprintf("%s.local", destName),
			IP:       cloneIP,
			Gateway:  cloneGateway,
			DNS:      cloneDNS,
		}

		if cloneSSHKey != "" {
			if strings.HasPrefix(cloneSSHKey, "@") {
				keyFile := strings.TrimPrefix(cloneSSHKey, "@")
				keyBytes, err := os.ReadFile(keyFile)
				if err != nil {
					return fmt.Errorf("failed to read SSH key file: %w", err)
				}
				cloudInitData.SSHKeys = []string{string(keyBytes)}
			} else {
				cloudInitData.SSHKeys = []string{cloneSSHKey}
			}
		}

		guestinfo, err = cloudinit.BuildGuestinfo(cloudInitData)
		if err != nil {
			return fmt.Errorf("failed to build cloud-init: %w", err)
		}
	}

	start := time.Now()

	vmOps := vm.NewOperations(esxi)
	newVM, err := vmOps.CloneVM(ctx, sourceName, destName, guestinfo)
	if err != nil {
		return fmt.Errorf("failed to clone VM: %w", err)
	}

	duration := time.Since(start)

	if err := vmOps.PowerOn(ctx, newVM); err != nil {
		return fmt.Errorf("failed to power on VM: %w", err)
	}

	fmt.Printf("✅ VM '%s' cloned to '%s' successfully in %v\n", sourceName, destName, duration)
	if cloneIP != "" {
		fmt.Printf("   New IP: %s\n", cloneIP)
	}

	return nil
}