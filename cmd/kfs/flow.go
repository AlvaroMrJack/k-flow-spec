package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/AlvaroMrJack/k-flow-spec/internal/config"
	"github.com/AlvaroMrJack/k-flow-spec/internal/discovery"
	"github.com/AlvaroMrJack/k-flow-spec/internal/flow"
)

var flowCmd = &cobra.Command{
	Use:   "flow",
	Short: "Simula navegación de WhatsApp Flows",
	Long: `Simula la navegación de un usuario dentro de un WhatsApp Flow (formulario nativo).

Valida navegación entre pantallas, datos enviados, y estado terminal.`,
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

		specPath := ""
		if len(args) > 0 {
			specPath = args[0]
		}
		if flowSpecFile != "" {
			specPath = flowSpecFile
		}

		if specPath == "" {
			return fmt.Errorf("especifica un archivo spec de flow con --spec")
		}

		// Create a default flow spec for testing
		spec := &flow.FlowSpec{
			Name:   "Checkout Flow Test",
			FlowID: "test_flow_123",
			Screens: []flow.FlowScreenStep{
				{Screen: "WELCOME", Action: "next"},
				{Screen: "SERVICES", Action: "select", Fields: map[string]interface{}{"service": "corte clásico"}},
				{Screen: "CONFIRM", Action: "confirm"},
			},
			Then: flow.FlowThen{
				TerminalScreen: "THANK_YOU",
				SubmittedData:  map[string]interface{}{"service": "corte clásico"},
			},
		}

		fmt.Printf("🔄 Simulando flow: %s\n", spec.Name)
		fmt.Printf("   Flow ID: %s\n", spec.FlowID)
		fmt.Printf("   Pantallas: %d\n", len(spec.Screens))

		for _, s := range spec.Screens {
			fmt.Printf("     • %s → %s\n", s.Screen, s.Action)
		}

		simulator := flow.NewSimulator(cfg.BaseURL, cfg.APIKey, flowMock)
		result := simulator.Run(spec)

		if result.Passed {
			fmt.Printf("   ✓ Flow completado (%dms)\n", result.DurationMs)
		} else {
			fmt.Printf("   ✗ Flow falló (%dms)\n", result.DurationMs)
			for _, err := range result.Errors {
				fmt.Printf("     - %s\n", err)
			}
		}

		if flowOpen {
			fmt.Println("   Abriendo flow en browser... (solo en modo visual)")
		}

		return nil
	},
}

var (
	flowMock     bool
	flowOpen     bool
	flowSpecFile string
)

func init() {
	flowCmd.Flags().BoolVar(&flowMock, "mock", false, "Contra mock server")
	flowCmd.Flags().BoolVar(&flowOpen, "open", false, "Abrir flow en browser para debug visual")
	flowCmd.Flags().StringVarP(&flowSpecFile, "spec", "s", "", "Archivo spec de flow")
	rootCmd.AddCommand(flowCmd)
}
