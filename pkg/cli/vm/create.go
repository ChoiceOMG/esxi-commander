package vm

import (
	"fmt"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a virtual machine",
	Long:  `Create a new virtual machine from a template`,
	Run:   runCreate,
}

func runCreate(cmd *cobra.Command, args []string) {
	fmt.Println("VM create command - stub implementation for Phase 1")
}
