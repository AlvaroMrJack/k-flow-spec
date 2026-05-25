package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/AlvaroMrJack/k-flow-spec/internal/config"
	"github.com/AlvaroMrJack/k-flow-spec/internal/deploy"
	"github.com/AlvaroMrJack/k-flow-spec/internal/discovery"
)

var (
	deployDryRun bool
	deployFull   bool
	deployEnv    string
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Build + push + test pipeline",
	Long: `Un solo comando para compilar, desplegar y testear un workflow Kapso.

Pipeline: kapso build → kapso push → kfs generate → kfs run --ci

Usa --dry-run para build + test sin push.
Usa --full para deploy completo con broadcast test y webhook validation.`,
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

		workflows := cfg.Deploy.Workflows
		if len(workflows) == 0 && len(args) > 0 {
			workflows = args
		}

		pipeline := deploy.NewPipeline(root, deployEnv, deployDryRun, deployFull, workflows)
		ctx := context.Background()

		fmt.Println("🚀 Iniciando pipeline de deploy...")
		fmt.Printf("   Entorno: %s\n", deployEnv)
		fmt.Printf("   Workflows: %d\n", len(workflows))
		if deployDryRun {
			fmt.Println("   Modo: dry-run (sin push)")
		}

		result := pipeline.Run(ctx)

		fmt.Println()
		for _, step := range result.Steps {
			status := "✓"
			if !step.Passed {
				status = "✗"
			}
			fmt.Printf("   %s %s", status, step.Name)
			if step.Output != "" {
				fmt.Printf(" — %s", step.Output)
			}
			if step.Error != "" {
				fmt.Printf(" — %s", step.Error)
			}
			fmt.Println()
		}

		fmt.Printf("\n📊 Resultado: ")
		if result.Passed {
			fmt.Printf("✓ Deploy completado (%v)\n", result.Duration)
		} else {
			fmt.Printf("✗ Deploy falló (%v)\n", result.Duration)
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	deployCmd.Flags().BoolVar(&deployDryRun, "dry-run", false, "Build + test sin push")
	deployCmd.Flags().BoolVar(&deployFull, "full", false, "Deploy completo + broadcast + webhook")
	deployCmd.Flags().StringVarP(&deployEnv, "env", "e", "production", "Entorno de deploy")
	RootCmd.AddCommand(deployCmd)
}
