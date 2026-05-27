package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/AlvaroMrJack/k-flow-spec/internal/config"
	"github.com/AlvaroMrJack/k-flow-spec/internal/discovery"
	"github.com/AlvaroMrJack/k-flow-spec/internal/webhook"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/kapso"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/runner"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/spec"
)

var (
	webhookPort    int
	webhookRun     bool
	webhookVerbose bool
	webhookTunnel  bool
)

var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Webhook receiver + validador en tiempo real",
	Long: `Levanta un servidor HTTP temporal, lo registra como webhook en Kapso,
captura eventos en tiempo real, y los valida contra el spec.

Si ngrok está instalado, usa --tunnel para crear un tunnel automático.`,
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

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println("\nDeteniendo webhook receiver...")
			cancel()
		}()

		receiver := webhook.NewReceiver(webhook.WebhookConfig{
			Port:       webhookPort,
			KapsoAPI:   cfg.BaseURL,
			APIKey:     cfg.APIKey,
			UseTunnel:  webhookTunnel,
			CaptureDir: filepath.Join(root, cfg.ReportsDir),
		})

		webhookURL := fmt.Sprintf("http://localhost:%d/webhook", webhookPort)
		if webhookTunnel {
			tunnelURL, err := webhook.StartTunnel(webhookPort)
			if err != nil {
				fmt.Printf("Aviso: no se pudo iniciar ngrok: %v\n", err)
				fmt.Println("Usando localhost (solo funcionará si Kapso puede alcanzarlo)")
			} else {
				webhookURL = tunnelURL + "/webhook"
				fmt.Printf("Túnel ngrok activo: %s\n", tunnelURL)
			}
		} else {
			fmt.Printf("Webhook escuchando en %s\n", webhookURL)
			if webhookTunnel == false {
				fmt.Println("Usa --tunnel si Kapso no puede alcanzar localhost")
			}
		}

		go receiver.Start(ctx)

		if webhookRun && len(args) > 0 {
			specPath := args[0]
			s, err := spec.Load(specPath)
			if err != nil {
				return fmt.Errorf("error cargando spec: %v", err)
			}

			client := kapso.NewClient(cfg.BaseURL, cfg.APIKey)
			engine := runner.NewEngine(client, cfg)
			result := engine.Run(ctx, s)

			if result.Passed {
				fmt.Printf("✓ %s (%v)\n", s.Name, result.Duration)
			} else {
				fmt.Printf("✗ %s (%v)\n", s.Name, result.Duration)
				for _, e := range result.Errors {
					fmt.Printf("  - %s: %s\n", e.Type, e.Message)
				}
			}
		}

		fmt.Println("Esperando webhooks... (Ctrl+C para salir)")
		eventCh := receiver.EventChannel()
		for {
			select {
			case <-ctx.Done():
				return nil
			case evt, ok := <-eventCh:
				if !ok {
					return nil
				}
				if webhookVerbose {
					fmt.Printf("📩 Webhook recibido: %s\n", evt.EventType)
					if len(evt.Payload) > 0 {
						for k, v := range evt.Payload {
							fmt.Printf("  %s: %v\n", k, v)
						}
					}
				}
			}
		}
	},
}

func init() {
	webhookCmd.Flags().IntVarP(&webhookPort, "port", "p", 9000, "Puerto del webhook receiver")
	webhookCmd.Flags().BoolVar(&webhookRun, "run", false, "Ejecutar workflow spec después de iniciar webhook")
	webhookCmd.Flags().BoolVar(&webhookVerbose, "verbose", false, "Ver eventos en tiempo real por consola")
	webhookCmd.Flags().BoolVarP(&webhookTunnel, "tunnel", "t", false, "Auto-túnel con ngrok")
	toolCmd.AddCommand(webhookCmd)
}
