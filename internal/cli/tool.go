package cli

import (
	"github.com/spf13/cobra"
)

var toolCmd = &cobra.Command{
	Use:   "tool",
	Short: "Advanced tools (deploy, webhook, broadcast, flow, ui, mcp, mock)",
	Long: `Advanced tools for CI/CD, webhook testing, broadcast campaigns,
WhatsApp Flows, web dashboard, MCP server, and standalone mock.`,
}

func init() {
	RootCmd.AddCommand(toolCmd)
}
