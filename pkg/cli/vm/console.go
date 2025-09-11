package vm

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/r11/esxi-commander/pkg/esxi/client"
)

var consoleCmd = &cobra.Command{
	Use:   "console <name>",
	Short: "Get console URL for a virtual machine",
	Long: `Get the web console URL for a virtual machine. 
This provides browser-based access to the VM console.`,
	Args: cobra.ExactArgs(1),
	RunE: runConsole,
}

func runConsole(cmd *cobra.Command, args []string) error {
	vmName := args[0]
	ctx := context.Background()
	
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
	
	// Find the VM
	vmObj, err := esxi.FindVM(ctx, vmName)
	if err != nil {
		return fmt.Errorf("VM '%s' not found: %w", vmName, err)
	}

	// Get the VM's managed object reference
	vmRef := vmObj.Reference()
	
	// Build console URL
	scheme := "https"
	if esxiCfg.Insecure {
		scheme = "https" // ESXi web console is always HTTPS
	}
	
	consoleURL := fmt.Sprintf("%s://%s/ui/webconsole.html?vmId=%s&vmName=%s&serverGuid=", 
		scheme, esxiCfg.Host, vmRef.Value, vmName)
	
	fmt.Printf("Console URL for VM '%s':\n", vmName)
	fmt.Printf("%s\n", consoleURL)
	fmt.Printf("\n")
	fmt.Printf("Alternative console access:\n")
	fmt.Printf("1. Open ESXi web UI: %s://%s/ui/\n", scheme, esxiCfg.Host)
	fmt.Printf("2. Navigate to Virtual Machines\n")
	fmt.Printf("3. Click on '%s' and select 'Launch Web Console'\n", vmName)
	
	return nil
}