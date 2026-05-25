package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Inicializa un proyecto k-flow-spec en el directorio actual",
	RunE: func(cmd *cobra.Command, args []string) error {
		content := `project: "my-kapso-project"
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
`
		if err := os.WriteFile("kfs.yaml", []byte(content), 0644); err != nil {
			return err
		}
		
		fmt.Println("kfs.yaml creado exitosamente. Ahora puedes editarlo y usar kfs generate.")
		return nil
	},
}

func init() {
	RootCmd.AddCommand(initCmd)
}
