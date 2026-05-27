package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/AlvaroMrJack/k-flow-spec/internal/config"
	"github.com/AlvaroMrJack/k-flow-spec/internal/discovery"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/spec"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "Lista los specs disponibles",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listSpecs()
	},
}

var listSpecCmd = &cobra.Command{
	Use:   "spec",
	Short: "Lista los specs disponibles",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listSpecs()
	},
}

func listSpecs() error {
	cwd, _ := os.Getwd()
	root, err := discovery.FindWorkspaceRoot(cwd)
	if err != nil {
		return fmt.Errorf("no se encontró kfs.yaml. Usa 'kfs init' primero")
	}

	cfg, err := config.LoadConfig(filepath.Join(root, "kfs.yaml"))
	if err != nil {
		return err
	}

	specsDir := filepath.Join(root, cfg.SpecsDir)
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return fmt.Errorf("directorio de specs '%s' no encontrado", cfg.SpecsDir)
	}

	type specInfo struct {
		File           string
		Name           string
		WorkflowID     string
		Messages       int
		PathLength     int
		TerminalStatus string
	}

	var specs []specInfo
	for _, entry := range entries {
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(specsDir, entry.Name())
		s, err := spec.Load(path)
		if err != nil {
			fmt.Printf("  ⚠ %s: error al cargar: %v\n", entry.Name(), err)
			continue
		}

		specs = append(specs, specInfo{
			File:           entry.Name(),
			Name:           s.Name,
			WorkflowID:     s.Workflow,
			Messages:       len(s.When.Messages),
			PathLength:     len(s.Then.Path),
			TerminalStatus: s.Then.TerminalStatus,
		})
	}

	if len(specs) == 0 {
		fmt.Println("No hay specs en " + cfg.SpecsDir + "/")
		fmt.Println("Usa 'kfs generate' o 'kfs learn' para crear uno.")
		return nil
	}

	fmt.Printf("Specs en %s/ (%d):\n\n", cfg.SpecsDir, len(specs))

	for _, sp := range specs {
		fmt.Printf("  %s\n", sp.File)
		fmt.Printf("    nombre:      %s\n", sp.Name)
		fmt.Printf("    workflow:    %s\n", sp.WorkflowID)
		fmt.Printf("    mensajes:    %d\n", sp.Messages)
		fmt.Printf("    path:        %d nodos\n", sp.PathLength)
		fmt.Printf("    termina en:  %s\n", sp.TerminalStatus)
		fmt.Println()
	}

	fmt.Println("Para ejecutar un spec:")
	fmt.Println("  kfs run")
	fmt.Println("  kfs run --spec <archivo>")

	return nil
}

func init() {
	listCmd.AddCommand(listSpecCmd)
	RootCmd.AddCommand(listCmd)
}
