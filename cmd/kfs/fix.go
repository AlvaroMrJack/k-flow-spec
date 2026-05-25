package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/AlvaroMrJack/k-flow-spec/internal/config"
	"github.com/AlvaroMrJack/k-flow-spec/internal/discovery"
	"github.com/AlvaroMrJack/k-flow-spec/internal/fix"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/kapso"
)

var (
	fixApply       bool
	fixInteractive bool
	fixSpecFile    string
)

var fixCmd = &cobra.Command{
	Use:   "fix",
	Short: "Analiza y repara specs rotos automáticamente",
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

		var client *kapso.Client
		if cfg.APIKey != "" && cfg.APIKey != "${KAPSO_API_KEY}" {
			client = kapso.NewClient(cfg.BaseURL, cfg.APIKey)
		}

		specsDir := filepath.Join(root, cfg.SpecsDir)
		if fixSpecFile != "" {
			specsDir = fixSpecFile
		}

		repairer := fix.NewRepairer(specsDir)
		issues, err := repairer.Repair(client, "", fixApply, fixInteractive)
		if err != nil {
			return fmt.Errorf("error durante reparación: %v", err)
		}

		if len(issues) == 0 {
			fmt.Println("✓ No se encontraron problemas")
			return nil
		}

		fmt.Printf("✓ %d issues reparados\n", len(issues))
		return nil
	},
}

func init() {
	fixCmd.Flags().BoolVar(&fixApply, "apply", false, "Aplicar reparaciones automáticamente")
	fixCmd.Flags().BoolVar(&fixInteractive, "interactive", false, "Modo interactivo: preguntar antes de cada cambio")
	fixCmd.Flags().StringVar(&fixSpecFile, "spec", "", "Analizar solo un spec específico")
	rootCmd.AddCommand(fixCmd)
}
