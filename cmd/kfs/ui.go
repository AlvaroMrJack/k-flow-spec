package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/AlvaroMrJack/k-flow-spec/internal/config"
	"github.com/AlvaroMrJack/k-flow-spec/internal/discovery"
	"github.com/AlvaroMrJack/k-flow-spec/internal/ui"
)

var (
	uiPort   int
	uiMock   bool
	uiExport string
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Inicia dashboard web local",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		root, err := discovery.FindWorkspaceRoot(cwd)
		if err != nil {
			return fmt.Errorf("no se encontró kfs.yaml. Usa 'kfs init' primero")
		}

		cfg, err := config.LoadConfig(filepath.Join(root, "kfs.yaml"))
		if err != nil {
			return err
		}

		if uiExport != "" {
			return fmt.Errorf("export no implementado todavía")
		}

		server := ui.NewServer(uiPort, cfg)
		if uiMock {
			server.SetMockMode(":4172")
		}

		fmt.Printf("Dashboard disponible en http://localhost:%d\n", uiPort)
		return server.Start()
	},
}

func init() {
	uiCmd.Flags().IntVarP(&uiPort, "port", "p", 4173, "Puerto para el dashboard")
	uiCmd.Flags().BoolVar(&uiMock, "mock", false, "Dashboard + mock server integrado")
	uiCmd.Flags().StringVar(&uiExport, "export", "", "Exportar HTML estático a un directorio")
	rootCmd.AddCommand(uiCmd)
}
