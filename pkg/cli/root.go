package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/r11/esxi-commander/pkg/cli/backup"
	"github.com/r11/esxi-commander/pkg/cli/template"
	"github.com/r11/esxi-commander/pkg/cli/vm"
	"github.com/r11/esxi-commander/pkg/security"
)

var (
	cfgFile    string
	jsonOutput bool
	dryRun     bool
)

var rootCmd = &cobra.Command{
	Use:   "ceso",
	Short: "ESXi Commander - Ubuntu-focused VMware ESXi orchestrator",
	Long: `ESXi Commander (ceso) is a production-grade tool for managing
Ubuntu LTS virtual machines on standalone ESXi hosts.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ceso/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would be done without making changes")

	rootCmd.AddCommand(vm.VmCmd)
	rootCmd.AddCommand(backup.BackupCmd)
	rootCmd.AddCommand(template.TemplateCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home + "/.ceso")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
	
	// Initialize security sandbox
	mode := security.ModeStandard // Default mode
	if viper.IsSet("security.mode") {
		modeStr := viper.GetString("security.mode")
		switch modeStr {
		case "restricted":
			mode = security.ModeRestricted
		case "unrestricted":
			mode = security.ModeUnrestricted
		default:
			mode = security.ModeStandard
		}
	}
	security.Initialize(mode)
}
