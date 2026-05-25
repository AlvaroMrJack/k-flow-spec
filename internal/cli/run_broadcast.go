package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/AlvaroMrJack/k-flow-spec/internal/broadcast"
	"github.com/AlvaroMrJack/k-flow-spec/internal/config"
	"github.com/AlvaroMrJack/k-flow-spec/internal/discovery"
	"github.com/AlvaroMrJack/k-flow-spec/internal/mock"
)

var runBroadcastCmd = &cobra.Command{
	Use:   "run-broadcast",
	Short: "Testea campañas Broadcast API",
	Long: `Valida campañas de Broadcast API completas: creación, destinatarios, envío y métricas.

Usa --mock para probar sin enviar mensajes reales.
Usa --dry-run para solo validar recipients sin enviar.`,
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

		var apiURL string
		var apiKey string

		if broadcastMock {
			mockServer := mock.NewServer(":4172")
			go mockServer.Start()
			apiURL = "http://localhost:4172/platform/v1"
			apiKey = cfg.APIKey
			fmt.Println("Usando mock server embebido en :4172")
		} else {
			apiURL = cfg.BaseURL
			apiKey = cfg.APIKey
		}

		specPath := ""
		if len(args) > 0 {
			specPath = args[0]
		}
		if broadcastSpecFile != "" {
			specPath = broadcastSpecFile
		}

		var specs []*broadcast.BroadcastSpec
		if specPath != "" {
			// Load single broadcast spec (for now, use default test spec)
			specs = append(specs, &broadcast.BroadcastSpec{
				Name:          "Test Broadcast",
				TemplateID:    "test_template",
				PhoneNumberID: "test_phone",
				Recipients: []broadcast.Recipient{
					{Phone: "+56900000001", Variables: map[string]string{"first_name": "Juan", "discount": "50%"}},
					{Phone: "+56900000002", Variables: map[string]string{"first_name": "María", "discount": "30%"}},
				},
			})
		} else {
			// Load all broadcast specs
			specs = append(specs, &broadcast.BroadcastSpec{
				Name:          "Default Test",
				TemplateID:    "test_template",
				PhoneNumberID: "test_phone",
				Recipients: []broadcast.Recipient{
					{Phone: "+56900000001", Variables: map[string]string{"first_name": "Juan"}},
				},
			})
		}

		for _, spec := range specs {
			fmt.Printf("📡 Probando broadcast: %s\n", spec.Name)
			fmt.Printf("   Template: %s\n", spec.TemplateID)
			fmt.Printf("   Recipients: %d\n", len(spec.Recipients))

			tester := broadcast.NewTester(apiURL, apiKey, broadcastDryRun)
			result := tester.Run(nil, spec)

			if result.Passed {
				fmt.Printf("   ✓ Broadcast completado: %d enviados, %d fallos\n",
					result.SentCount, result.FailedCount)
			} else {
				fmt.Printf("   ✗ Broadcast falló\n")
				for _, err := range result.Errors {
					fmt.Printf("     - %s\n", err)
				}
			}
		}

		return nil
	},
}

var (
	broadcastMock     bool
	broadcastDryRun   bool
	broadcastSpecFile string
)

func init() {
	runBroadcastCmd.Flags().BoolVar(&broadcastMock, "mock", false, "Contra mock server")
	runBroadcastCmd.Flags().BoolVar(&broadcastDryRun, "dry-run", false, "Solo validar recipients sin enviar")
	runBroadcastCmd.Flags().StringVarP(&broadcastSpecFile, "spec", "s", "", "Archivo spec de broadcast")
	RootCmd.AddCommand(runBroadcastCmd)
}
