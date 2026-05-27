package cli

import (
	"github.com/spf13/cobra"
)

var specCmd = &cobra.Command{
	Use:   "spec",
	Short: "Manage and run specs (generate, learn, run, ls, fix)",
	Long: `Spec lifecycle commands for k-flow-spec.

Generate, learn, run, list, and fix specs — the core of the
spec-driven testing workflow.`,
}

func init() {
	RootCmd.AddCommand(specCmd)
}
