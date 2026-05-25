package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/AlvaroMrJack/k-flow-spec/internal/config"
	"github.com/AlvaroMrJack/k-flow-spec/internal/discovery"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configura k-flow-spec interactivamente (API key, modo, etc.)",
	Long: `Asistente paso a paso para configurar k-flow-spec.

Configura tu API key de Kapso, teléfono por defecto,
modo de prueba (mock/real), y más opciones.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		root, err := discovery.FindWorkspaceRoot(cwd)
		if err != nil {
			return fmt.Errorf("no se encontró kfs.yaml. Usa 'kfs init' primero")
		}

		cfgPath := filepath.Join(root, "kfs.yaml")

		// Read raw file to preserve ${KAPSO_API_KEY} placeholder before env expansion
		rawData, _ := os.ReadFile(cfgPath)

		cfg, err := config.LoadConfig(cfgPath)
		if err != nil {
			return fmt.Errorf("error leyendo kfs.yaml: %v", err)
		}

		// Detect if api_key was originally "${KAPSO_API_KEY}" placeholder
		apiKeyDefault := cfg.APIKey
		if strings.Contains(string(rawData), "${KAPSO_API_KEY}") {
			apiKeyDefault = "${KAPSO_API_KEY}"
		}

		fmt.Println("╔══════════════════════════════════════╗")
		fmt.Println("║    Configuración de k-flow-spec      ║")
		fmt.Println("╚══════════════════════════════════════╝")
		fmt.Println()

		cfg.APIKey = prompt(
			"API Key de Kapso (deja vacío para usar ${KAPSO_API_KEY})",
			apiKeyDefault,
		)

		cfg.PhoneNumber = prompt(
			"Teléfono por defecto para pruebas",
			cfg.PhoneNumber,
		)

		cfg.BaseURL = prompt(
			"URL base de la API Kapso",
			cfg.BaseURL,
		)

		fmt.Println()
		fmt.Println("--- Modo de prueba ---")

		modeIdx := promptSelect("Modo por defecto", []string{"Mock (local, sin API real)", "Real (contra Kapso API)"}, 0)
		_ = modeIdx // stored in config for future use; currently used by --mock flag

		testFlows := promptBool("Probar WhatsApp Flows?", false)

		fmt.Println()
		fmt.Println("--- Timeouts y defaults ---")

		cfg.Defaults.Timeout = promptInt("Timeout por defecto (segundos)", cfg.Defaults.Timeout)
		cfg.Defaults.Snapshot = promptBool("Auto-snapshot (guardar resultados de referencia)", cfg.Defaults.Snapshot)
		cfg.Defaults.PollIntervalMs = promptInt("Intervalo de polling (ms)", cfg.Defaults.PollIntervalMs)
		cfg.Defaults.PollMaxRetries = promptInt("Máximo de reintentos de polling", cfg.Defaults.PollMaxRetries)

		fmt.Println()
		fmt.Println("--- Límites de rate ---")

		cfg.RateLimit.MaxBurst = promptInt("Máx requests simultáneos por workflow", cfg.RateLimit.MaxBurst)
		cfg.RateLimit.GeneralRPM = promptInt("Requests por minuto (global)", cfg.RateLimit.GeneralRPM)

		fmt.Println()
		fmt.Println("--- Directorios ---")

		cfg.SpecsDir = prompt("Directorio de specs", cfg.SpecsDir)
		cfg.SnapshotsDir = prompt("Directorio de snapshots", cfg.SnapshotsDir)
		cfg.ReportsDir = prompt("Directorio de reportes", cfg.ReportsDir)

		fmt.Println()
		fmt.Println("--- Notificaciones ---")

		slack := prompt(
			"Slack webhook URL (opcional, para CI)",
			"",
		)
		if slack != "" {
			cfg.Notifications = &config.NotificationsConfig{SlackWebhook: slack}
		}

		fmt.Println()
		fmt.Println("--- Deploy ---")

		env := prompt("Entorno de deploy (dev/staging/prod)", "")
		autoGen := promptBool("Auto-generar specs antes de deploy", false)
		autoRun := promptBool("Auto-ejecutar tests después de deploy", false)
		if env != "" || autoGen || autoRun {
			cfg.Deploy = &config.DeployConfig{
				Environment:  env,
				AutoGenerate: autoGen,
				AutoRun:      autoRun,
			}
		}

		_ = testFlows

		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("error generando YAML: %v", err)
		}

		if err := os.WriteFile(cfgPath, data, 0644); err != nil {
			return fmt.Errorf("error guardando kfs.yaml: %v", err)
		}

		fmt.Println()
		fmt.Println("✅ kfs.yaml actualizado exitosamente.")
		if cfg.APIKey != "" && cfg.APIKey != "${KAPSO_API_KEY}" {
			fmt.Println("⚠️  Tu API key está en kfs.yaml. No lo commits a git.")
			fmt.Println("   Mejor usa ${KAPSO_API_KEY} y define la variable de entorno.")
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(configureCmd)
}
