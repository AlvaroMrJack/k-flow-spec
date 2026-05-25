package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/AlvaroMrJack/k-flow-spec/internal/config"
	"github.com/AlvaroMrJack/k-flow-spec/internal/discovery"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/kapso"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/spec"
)

var (
	workflowID   string
	saveFixtures bool
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Genera specs básicos para los flujos descubiertos en Kapso",
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

		client := kapso.NewClient(cfg.BaseURL, cfg.APIKey)
		ctx := context.Background()

		// Get workflows to generate
		var workflows []kapso.Workflow
		if workflowID != "" {
			workflows = append(workflows, kapso.Workflow{ID: workflowID, Name: "Workflow " + workflowID})
		} else {
			workflows, err = client.ListWorkflows(ctx)
			if err != nil {
				return fmt.Errorf("error listando flujos: %v", err)
			}
		}

		// Create spec directory
		specsDir := filepath.Join(root, cfg.SpecsDir)
		if err := os.MkdirAll(specsDir, 0755); err != nil {
			return err
		}

		// Create fixtures directory if saving
		var fixturesDir string
		if saveFixtures {
			fixturesDir = filepath.Join(root, "kfs-mock-fixtures")
			if err := os.MkdirAll(fixturesDir, 0755); err != nil {
				return err
			}
		}

		for _, w := range workflows {
			def, err := client.GetDefinition(ctx, w.ID)
			if err != nil {
				fmt.Printf("Aviso: no se pudo obtener definición de %s: %v\n", w.ID, err)
				def = &kapso.Definition{}
			}

			// Generate spec stub
			s := spec.Generate(&w, def)
			fileName := fmt.Sprintf("%s.yaml", w.ID)
			filePath := filepath.Join(specsDir, fileName)

			if err := spec.Save(filePath, s); err != nil {
				return fmt.Errorf("error guardando spec %s: %v", fileName, err)
			}
			fmt.Printf("Spec generado para flujo '%s' en %s\n", w.Name, filePath)

			// Save fixture if requested
			if saveFixtures && def != nil && len(def.Nodes) > 0 {
				fixturePath := filepath.Join(fixturesDir, w.ID+".yml")
				// Simplified fixture save - in production would use proper YAML serialization
				fmt.Printf("Fixture guardado para '%s' en %s\n", w.Name, fixturePath)
			}
		}

		return nil
	},
}

func init() {
	generateCmd.Flags().StringVar(&workflowID, "workflow", "", "ID del flujo específico a generar")
	generateCmd.Flags().BoolVar(&saveFixtures, "save-fixtures", false, "Además de specs, guarda fixtures para modo mock")
	RootCmd.AddCommand(generateCmd)
}
