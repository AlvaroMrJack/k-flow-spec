package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
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

		envContent := `# k-flow-spec environment variables
# Copia este archivo como .env y completa los valores
KAPSO_API_KEY=tu-api-key-aqui
`
		os.WriteFile(".env.example", []byte(envContent), 0644)

		fmt.Println("✅ kfs.yaml creado exitosamente.")
		fmt.Println("✅ .env.example creado — copia a .env y agrega tu KAPSO_API_KEY")
		fmt.Println()
		fmt.Println("Siguientes pasos:")
		fmt.Println("  1. Edita kfs.yaml o crea un .env con tu KAPSO_API_KEY")
		fmt.Println("  2. kfs spec generate         — Genera specs desde tus flujos")
		fmt.Println("  3. kfs spec run --mock        — Ejecuta pruebas en modo simulado")
		return nil
	},
}

func init() {
	RootCmd.AddCommand(initCmd)
}
