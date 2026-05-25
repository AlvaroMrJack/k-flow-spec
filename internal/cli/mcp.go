package cli

import (
	"context"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/AlvaroMrJack/k-flow-spec/internal/config"
	"github.com/AlvaroMrJack/k-flow-spec/internal/discovery"
	"github.com/AlvaroMrJack/k-flow-spec/internal/mcp"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/kapso"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Inicia MCP server para asistentes IA (JSON-RPC sobre stdio)",
	Long: `Inicia un servidor MCP (Model Context Protocol) que permite a asistentes
IA como Claude interactuar con k-flow-spec. Implementa JSON-RPC sobre stdio.

Tools expuestas:
- generate_spec: Genera specs desde workflows en Kapso
- run_specs: Ejecuta tests y devuelve resultados
- get_status: Consulta estado del proyecto
- update_snapshots: Actualiza snapshots existentes
- fix_specs: Analiza y repara specs rotos

Uso en opencode.json o claude_desktop_config.json:
{
  "mcpServers": {
    "k-flow-spec": {
      "command": "kfs",
      "args": ["mcp"]
    }
  }
}`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		root, err := discovery.FindWorkspaceRoot(cwd)
		if err != nil {
			root = cwd
		}

		cfg, err := config.LoadConfig(filepath.Join(root, "kfs.yaml"))
		if err != nil {
			cfg = &config.KfsConfig{
				BaseURL: "https://api.kapso.ai/platform/v1",
			}
		}

		client := kapso.NewClient(cfg.BaseURL, cfg.APIKey)
		server := mcp.NewServer(cfg, client)
		return server.Start(context.Background())
	},
}

func init() {
	RootCmd.AddCommand(mcpCmd)
}
