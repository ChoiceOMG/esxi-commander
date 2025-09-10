package template

import (
	"github.com/spf13/cobra"
)

var TemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage templates",
	Long:  `Manage and validate VM templates`,
}
