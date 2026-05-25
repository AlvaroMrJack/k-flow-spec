package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var completionsCmd = &cobra.Command{
	Use:   "completions [bash|zsh|fish|powershell]",
	Short: "Genera autocompletado para tu shell",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return RootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return RootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return RootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return RootCmd.GenPowerShellCompletion(os.Stdout)
		default:
			return cmd.Help()
		}
	},
}

func init() {
	RootCmd.AddCommand(completionsCmd)
}
