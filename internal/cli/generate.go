package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/AlvaroMrJack/k-flow-spec/internal/config"
	"github.com/AlvaroMrJack/k-flow-spec/internal/discovery"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/kapso"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/spec"
)

var (
	workflowID      string
	saveFixtures    bool
	interactiveGen  bool
)

func runGenerate(cfg *config.KfsConfig, root string, workflows []kapso.Workflow) error {
	client := kapso.NewClient(cfg.BaseURL, cfg.APIKey)
	ctx := context.Background()

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
			fmt.Printf("⚠️  Aviso: no se pudo obtener definición de %s: %v\n", w.ID, err)
			def = &kapso.Definition{}
		}

		s := spec.Generate(&w, def)
		fileName := fmt.Sprintf("%s.yaml", w.ID)
		filePath := filepath.Join(specsDir, fileName)

		if err := spec.Save(filePath, s); err != nil {
			return fmt.Errorf("error guardando spec %s: %v", fileName, err)
		}
		fmt.Printf("✅ Spec generado para flujo '%s' en %s\n", w.Name, filePath)

		if saveFixtures && def != nil && len(def.Nodes) > 0 {
			fixturePath := filepath.Join(fixturesDir, w.ID+".yml")
			fmt.Printf("✅ Fixture guardado para '%s' en %s\n", w.Name, fixturePath)
		}
	}
	return nil
}

func generateInteractive(cfg *config.KfsConfig, root string) error {
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║     Generación interactiva specs     ║")
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()

	modeIdx := promptSelect(
		"¿Cómo quieres probar?",
		[]string{"Mock (local, sin API real)", "Real (contra Kapso API)"},
		0,
	)
	isMock := modeIdx == 0

	testFlows := promptBool("¿Incluir tests de componentes Flow?", false)

	// Decide workflow selection
	workflowChoice := promptSelect(
		"¿Para qué flujos generar specs?",
		[]string{"Todos los flujos", "Seleccionar manualmente"},
		0,
	)

	if testFlows {
		fmt.Println("ℹ️  Los tests Flow se generarán como specs adicionales.")
	}

	var workflows []kapso.Workflow

	if workflowChoice == 1 {
		// Manual selection - ask for workflow IDs
		ids := prompt("IDs de flujos separados por coma", "")
		if ids == "" {
			return fmt.Errorf("debes especificar al menos un ID")
		}
		for _, id := range splitAndTrim(ids, ",") {
			workflows = append(workflows, kapso.Workflow{ID: id, Name: "Workflow " + id})
		}
	} else {
		client := kapso.NewClient(cfg.BaseURL, cfg.APIKey)
		ctx := context.Background()
		var err error
		workflows, err = client.ListWorkflows(ctx)
		if err != nil {
			return fmt.Errorf("error listando flujos: %v", err)
		}
	}

	if isMock {
		saveFixtures = promptBool("¿Guardar fixtures para modo mock?", true)
	}

	_ = testFlows
	return runGenerate(cfg, root, workflows)
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Genera specs básicos para los flujos descubiertos en Kapso",
	Long: `Genera specs YAML a partir de los flujos de Kapso.

Sin flags, ejecuta en modo silencioso (no interactivo).
Con --interactive (-i) muestra un asistente paso a paso.`,
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

		if interactiveGen {
			return generateInteractive(cfg, root)
		}

		client := kapso.NewClient(cfg.BaseURL, cfg.APIKey)
		ctx := context.Background()

		var workflows []kapso.Workflow
		if workflowID != "" {
			workflows = append(workflows, kapso.Workflow{ID: workflowID, Name: "Workflow " + workflowID})
		} else {
			workflows, err = client.ListWorkflows(ctx)
			if err != nil {
				return fmt.Errorf("error listando flujos: %v", err)
			}
		}

		return runGenerate(cfg, root, workflows)
	},
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range strings.Split(s, sep) {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func init() {
	generateCmd.Flags().StringVar(&workflowID, "workflow", "", "ID del flujo específico a generar")
	generateCmd.Flags().BoolVar(&saveFixtures, "save-fixtures", false, "Además de specs, guarda fixtures para modo mock")
	generateCmd.Flags().BoolVarP(&interactiveGen, "interactive", "i", false, "Modo interactivo paso a paso")
	RootCmd.AddCommand(generateCmd)
}
