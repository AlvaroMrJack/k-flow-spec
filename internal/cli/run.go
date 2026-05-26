package cli

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/AlvaroMrJack/k-flow-spec/internal/config"
	"github.com/AlvaroMrJack/k-flow-spec/internal/discovery"
	"github.com/AlvaroMrJack/k-flow-spec/internal/mock"
	"github.com/AlvaroMrJack/k-flow-spec/internal/report"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/kapso"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/runner"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/spec"
)

var (
	runMock            bool
	runSpecFile        string
	runWatch           bool
	runCI              bool
	runFormat          string
	runUpdateSnapshots bool
	runInteractive     bool
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Ejecuta todos los specs (o uno específico)",
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

		if runWatch {
			return runWatchMode(root, cfg)
		}

		return runOnce(root, cfg)
	},
}

// runOnce executes all specs once and reports results.
func runOnce(root string, cfg *config.KfsConfig) error {
	ctx := context.Background()

	var client *kapso.Client

	if runMock {
		mockServer := mock.NewServer(":4172")
		go mockServer.Start()
		client = kapso.NewClient("http://localhost:4172/platform/v1", cfg.APIKey)
		fmt.Println("Usando mock server embebido en :4172")
		time.Sleep(100 * time.Millisecond) // let server start
	} else {
		client = kapso.NewClient(cfg.BaseURL, cfg.APIKey)
	}

	// Load specs
	var specs []*spec.Spec
	if runSpecFile != "" {
		s, err := spec.Load(runSpecFile)
		if err != nil {
			return fmt.Errorf("error cargando spec: %v", err)
		}
		specs = append(specs, s)
	} else {
		specsDir := filepath.Join(root, cfg.SpecsDir)
		entries, err := os.ReadDir(specsDir)
		if err != nil {
			return fmt.Errorf("directorio de specs '%s' no encontrado. Usa 'kfs generate' o 'kfs generate -i' para crear specs", cfg.SpecsDir)
		}
		for _, entry := range entries {
			ext := filepath.Ext(entry.Name())
			if ext != ".yaml" && ext != ".yml" {
				continue
			}
			s, err := spec.Load(filepath.Join(specsDir, entry.Name()))
			if err != nil {
				fmt.Printf("Aviso: error cargando %s: %v\n", entry.Name(), err)
				continue
			}
			specs = append(specs, s)
		}
	}

	if len(specs) == 0 {
		return fmt.Errorf("no se encontraron specs para ejecutar. Usa 'kfs generate' primero")
	}

	// Run specs
	scheduler := runner.NewScheduler(client, cfg)
	scheduler.SetInteractive(runInteractive)
	scheduler.SetProgress(func(format string, args ...interface{}) {
		fmt.Printf(format+"\n", args...)
	})
	results := scheduler.RunAll(ctx, specs)

	// Print summary
	passed, failed := 0, 0
	for _, res := range results {
		if res.Passed {
			passed++
		} else {
			failed++
			fmt.Printf("✗ %s (%v)", res.SpecName, res.Duration)
			if len(res.Errors) > 0 {
				fmt.Printf(" — %s", res.Errors[0].Message)
			}
			fmt.Println()
			if runInteractive {
				for _, e := range res.Errors {
					fmt.Printf("  ↓ %s: %s\n", e.Type, e.Message)
				}
			}
		}

		// Save snapshot if enabled
		if res.Snapshot != nil {
			snapshotsDir := filepath.Join(root, cfg.SnapshotsDir)
			os.MkdirAll(snapshotsDir, 0755)
			snapPath := filepath.Join(snapshotsDir, res.WorkflowID+".snap.yml")

			// Compare with existing snapshot if not updating
			if !runUpdateSnapshots {
				existing, err := spec.LoadSnapshot(snapPath)
				if err == nil && existing != nil {
					// Compare snapshots for diff
					if fmt.Sprintf("%v", existing.ExecutionContext) != fmt.Sprintf("%v", res.Snapshot.ExecutionContext) {
						fmt.Printf("  ⚠ snapshot diff detectado: %s\n", snapPath)
					}
				}
			}

			// Save new snapshot
			if err := spec.SaveSnapshot(snapPath, res.Snapshot); err != nil {
				fmt.Printf("  ⚠ error guardando snapshot: %v\n", err)
			}
		}
	}

	fmt.Printf("\nResultados: %d passed, %d failed, 0 skipped\n", passed, failed)

	// Generate report
	reportFormat := report.FormatJSON
	if runCI {
		reportFormat = report.FormatJUnit
	}
	if runFormat != "" {
		reportFormat = report.Format(runFormat)
	}

	if runCI || runFormat != "" {
		reporter := report.NewReporter(reportFormat)
		reportsDir := filepath.Join(root, cfg.ReportsDir)
		os.MkdirAll(reportsDir, 0755)

		ext := ".json"
		switch reportFormat {
		case report.FormatJUnit:
			ext = ".xml"
		case report.FormatTAP:
			ext = ".tap"
		case report.FormatMarkdown:
			ext = ".md"
		}

		f, err := os.Create(filepath.Join(reportsDir, "results"+ext))
		if err != nil {
			return fmt.Errorf("error creando reporte: %v", err)
		}
		defer f.Close()

		if err := reporter.Write(f, results); err != nil {
			return fmt.Errorf("error escribiendo reporte: %v", err)
		}
	}

	if failed > 0 {
		os.Exit(1)
	}
	return nil
}

// runWatchMode watches spec files for changes and re-runs.
func runWatchMode(root string, cfg *config.KfsConfig) error {
	specsDir := filepath.Join(root, cfg.SpecsDir)
	fmt.Printf("👀 Modo watch activo en %s (Ctrl+C para salir)\n", specsDir)

	// Track file hashes
	hashes := make(map[string]string)

	for {
		entries, err := os.ReadDir(specsDir)
		if err != nil {
			return err
		}

		changed := false
		for _, entry := range entries {
			ext := filepath.Ext(entry.Name())
			if ext != ".yaml" && ext != ".yml" {
				continue
			}

			path := filepath.Join(specsDir, entry.Name())
			hash, err := fileHash(path)
			if err != nil {
				continue
			}

			if hashes[path] != hash {
				hashes[path] = hash
				changed = true
			}
		}

		if changed {
			fmt.Println("\n🔄 Cambios detectados, re-ejecutando...")
			runOnce(root, cfg)
			fmt.Println("\n👀 Esperando cambios...")
		}

		time.Sleep(2 * time.Second)
	}
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func init() {
	runCmd.Flags().BoolVar(&runMock, "mock", false, "Correr contra mock server embebido")
	runCmd.Flags().StringVar(&runSpecFile, "spec", "", "Correr un spec específico")
	runCmd.Flags().BoolVar(&runWatch, "watch", false, "Modo watch: re-ejecutar al cambiar archivos")
	runCmd.Flags().BoolVar(&runCI, "ci", false, "Modo CI: JUnit XML + exit codes estrictos")
	runCmd.Flags().StringVar(&runFormat, "format", "", "Formato de reporte (json, junit, tap, markdown)")
	runCmd.Flags().BoolVar(&runUpdateSnapshots, "update-snapshots", false, "Actualizar snapshots existentes")
	runCmd.Flags().BoolVarP(&runInteractive, "interactive", "i", false, "Modo debug paso a paso")
	RootCmd.AddCommand(runCmd)
}
