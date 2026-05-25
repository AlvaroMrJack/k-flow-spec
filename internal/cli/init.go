package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	runConfigure bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Inicializa un proyecto k-flow-spec en el directorio actual",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		projName := filepath.Base(cwd)

		content := fmt.Sprintf(`project: "%s"
base_url: "https://api.kapso.ai/platform/v1"
api_key: "${KAPSO_API_KEY}"
phone_number: "+56912345678"
specs_dir: "kfs-specs"
snapshots_dir: "kfs-snapshots"
reports_dir: "kfs-reports"
rate_limit:
  max_burst: 5
  general_rpm: 100
defaults:
  timeout: 60
  snapshot: true
  poll_interval_ms: 500
  poll_max_retries: 60
`, projName)
		if err := os.WriteFile("kfs.yaml", []byte(content), 0644); err != nil {
			return err
		}

		fmt.Println("✅ kfs.yaml creado exitosamente.")

		if runConfigure {
			if err := configureCmd.RunE(cmd, args); err != nil {
				return err
			}
		} else {
			fmt.Println()
			fmt.Println("Siguientes pasos:")
			fmt.Println("  1. kfs configure   — Configura tu API key y preferencias")
			fmt.Println("  2. kfs generate    — Genera specs desde tus flujos")
			fmt.Println("  3. kfs run --mock  — Ejecuta pruebas en modo simulado")
		}
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&runConfigure, "configure", false, "Ejecutar el asistente de configuración después de init")
	RootCmd.AddCommand(initCmd)
}
