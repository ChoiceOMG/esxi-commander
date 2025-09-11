package setup

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

type SetupConfig struct {
	ESXi struct {
		Host     string `yaml:"host"`
		User     string `yaml:"user"`
		Password string `yaml:"password,omitempty"`
		Insecure bool   `yaml:"insecure"`
	} `yaml:"esxi"`
	VM struct {
		Defaults struct {
			Template string `yaml:"template"`
			CPU      int    `yaml:"cpu"`
			Memory   int    `yaml:"memory"`
			Disk     int    `yaml:"disk"`
		} `yaml:"defaults"`
	} `yaml:"vm"`
	Security struct {
		Mode string `yaml:"mode"`
	} `yaml:"security"`
}

var SetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard for first-time configuration",
	Long:  `Run an interactive wizard to create your ESXi Commander configuration file.`,
	RunE:  runSetupWizard,
}

func runSetupWizard(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)
	config := &SetupConfig{}

	fmt.Println("🚀 ESXi Commander Setup Wizard")
	fmt.Println("==============================")
	fmt.Println("This wizard will help you create a configuration file.")
	fmt.Println()

	// ESXi Connection
	fmt.Println("📡 ESXi Connection Settings")
	fmt.Println("---------------------------")
	
	config.ESXi.Host = promptString(reader, "ESXi Host IP/Hostname", "192.168.1.100")
	config.ESXi.User = promptString(reader, "ESXi Username", "root")
	
	// Password handling
	fmt.Print("ESXi Password (will be hidden): ")
	password, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println()
	
	savePassword := promptBool(reader, "Save password in config file? (not recommended)", false)
	if savePassword {
		config.ESXi.Password = string(password)
	}
	
	config.ESXi.Insecure = promptBool(reader, "Accept self-signed certificates?", true)

	fmt.Println()
	fmt.Println("🖥️  VM Default Settings")
	fmt.Println("----------------------")
	
	config.VM.Defaults.Template = promptString(reader, "Default VM Template", "ubuntu-22.04")
	config.VM.Defaults.CPU = promptInt(reader, "Default vCPUs", 2)
	config.VM.Defaults.Memory = promptInt(reader, "Default Memory (MB)", 4096)
	config.VM.Defaults.Disk = promptInt(reader, "Default Disk Size (GB)", 20)

	fmt.Println()
	fmt.Println("🔒 Security Settings")
	fmt.Println("-------------------")
	
	fmt.Println("Operation modes:")
	fmt.Println("  - restricted: Read-only operations (recommended for AI agents)")
	fmt.Println("  - standard: Normal operations (default)")
	fmt.Println("  - unrestricted: All operations allowed")
	
	mode := promptString(reader, "Security Mode", "standard")
	if mode == "restricted" || mode == "standard" || mode == "unrestricted" {
		config.Security.Mode = mode
	} else {
		config.Security.Mode = "standard"
	}

	// Determine config file path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	
	configDir := filepath.Join(homeDir, ".ceso")
	configFile := filepath.Join(configDir, "config.yaml")
	
	// Create directory if needed
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(configFile); err == nil {
		overwrite := promptBool(reader, fmt.Sprintf("Config file exists at %s. Overwrite?", configFile), false)
		if !overwrite {
			altPath := promptString(reader, "Alternative config path", configFile)
			configFile = altPath
		}
	}

	// Write config file
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ Configuration saved to:", configFile)
	
	if !savePassword {
		fmt.Println()
		fmt.Println("⚠️  Password was not saved. Set it using:")
		fmt.Printf("   export ESXI_PASSWORD='%s'\n", string(password))
	}
	
	fmt.Println()
	fmt.Println("🎉 Setup complete! You can now use ESXi Commander:")
	fmt.Println("   ceso vm list")
	fmt.Println("   ceso vm create my-vm --template", config.VM.Defaults.Template)
	fmt.Println()
	fmt.Println("For more information, run: ceso --help")

	return nil
}

func promptString(reader *bufio.Reader, prompt, defaultValue string) string {
	fmt.Printf("%s [%s]: ", prompt, defaultValue)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

func promptInt(reader *bufio.Reader, prompt string, defaultValue int) int {
	fmt.Printf("%s [%d]: ", prompt, defaultValue)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	
	var value int
	if _, err := fmt.Sscanf(input, "%d", &value); err != nil {
		fmt.Printf("Invalid input, using default: %d\n", defaultValue)
		return defaultValue
	}
	return value
}

func promptBool(reader *bufio.Reader, prompt string, defaultValue bool) bool {
	defaultStr := "n"
	if defaultValue {
		defaultStr = "y"
	}
	
	fmt.Printf("%s [%s]: ", prompt, defaultStr)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	
	if input == "" {
		return defaultValue
	}
	
	return input == "y" || input == "yes" || input == "true"
}