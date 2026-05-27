package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/AlvaroMrJack/k-flow-spec/internal/config"
	"github.com/AlvaroMrJack/k-flow-spec/internal/discovery"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/kapso"
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

type specEntry struct {
	File           string
	Name           string
	WorkflowID     string
	WorkflowName   string
	Messages       int
	PathLength     int
	TerminalStatus string
	SnapshotAt     string
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

	// Load workflow names from API
	client := kapso.NewClient(cfg.BaseURL, cfg.APIKey)
	workflowNames := loadWorkflowNames(client)

	snapshotsDir := filepath.Join(root, cfg.SnapshotsDir)

	var specs []specEntry
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

		snapshotAt := ""
		snapPath := filepath.Join(snapshotsDir, s.Workflow+".snap.yml")
		if snap, err := spec.LoadSnapshot(snapPath); err == nil && snap != nil {
			snapshotAt = snap.RunAt
		}

		specs = append(specs, specEntry{
			File:           entry.Name(),
			Name:           s.Name,
			WorkflowID:     s.Workflow,
			WorkflowName:   workflowNames[s.Workflow],
			Messages:       len(s.When.Messages),
			PathLength:     len(s.Then.Path),
			TerminalStatus: s.Then.TerminalStatus,
			SnapshotAt:     snapshotAt,
		})
	}

	if len(specs) == 0 {
		fmt.Println("No hay specs en " + cfg.SpecsDir + "/")
		fmt.Println("Usa 'kfs generate' o 'kfs learn' para crear uno.")
		return nil
	}

	fmt.Printf("Specs en %s/ (%d):\n\n", cfg.SpecsDir, len(specs))

	for _, sp := range specs {
		snapshot := ""
		if sp.SnapshotAt != "" {
			if t, err := time.Parse(time.RFC3339, sp.SnapshotAt); err == nil {
				snapshot = fmt.Sprintf(" 🖼  %s", t.Format("2006-01-02 15:04"))
			} else {
				snapshot = " 🖼"
			}
		}

		wfName := sp.WorkflowID
		if sp.WorkflowName != "" {
			wfName = sp.WorkflowName
		}

		fmt.Printf("  %s%s\n", sp.File, snapshot)
		fmt.Printf("    nombre:      %s\n", sp.Name)
		fmt.Printf("    workflow:    %s\n", wfName)
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

func loadWorkflowNames(client *kapso.Client) map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	names := map[string]string{}
	workflows, err := client.ListWorkflows(ctx)
	if err != nil {
		return names
	}
	for _, w := range workflows {
		names[w.ID] = w.Name
	}
	return names
}

func init() {
	listCmd.AddCommand(listSpecCmd)
	RootCmd.AddCommand(listCmd)
}
