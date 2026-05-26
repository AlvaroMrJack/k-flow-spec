package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/AlvaroMrJack/k-flow-spec/internal/config"
	"github.com/AlvaroMrJack/k-flow-spec/internal/discovery"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/kapso"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/runner"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/spec"
)

var learnWorkflow string

var learnCmd = &cobra.Command{
	Use:   "learn",
	Short: "Graba un spec interactivamente ejecutando el flujo real",
	Long: `Graba un spec ejecutando el flujo real contra la API de Kapso.

Para cada paso del flujo, kfs te muestra dónde está esperando el workflow
y tú escribes lo que responderías como usuario. Al final, kfs genera
el spec con la ruta real, las decisiones tomadas y tus mensajes.

Ejemplo:
  kfs learn --workflow f1704737-...
  kfs learn  # usa el workflow de la config o te pide elegir

Escribe "done" o presiona Ctrl+C para terminar la grabación.`,
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

		return learnRun(cfg, root)
	},
}

func learnRun(cfg *config.KfsConfig, root string) error {
	client := kapso.NewClient(cfg.BaseURL, cfg.APIKey)
	ctx := context.Background()

	workflowID := learnWorkflow
	if workflowID == "" {
		workflows, err := client.ListWorkflows(ctx)
		if err != nil {
			return fmt.Errorf("error listando flujos: %v", err)
		}
		if len(workflows) == 0 {
			return fmt.Errorf("no hay flujos disponibles en tu proyecto")
		}

		names := make([]string, len(workflows))
		for i, w := range workflows {
			names[i] = fmt.Sprintf("%s (%s)", w.Name, w.ID)
		}
		idx := promptSelect("¿Qué flujo quieres grabar?", names, 0)
		workflowID = workflows[idx].ID
	}

	fmt.Println("\n  Conectando con Kapso API...")

	// Get workflow info for the spec
	workflows, err := client.ListWorkflows(ctx)
	if err != nil {
		return fmt.Errorf("error obteniendo flujos: %v", err)
	}
	var workflowName string
	for _, w := range workflows {
		if w.ID == workflowID {
			workflowName = w.Name
			break
		}
	}
	if workflowName == "" {
		workflowName = "Workflow " + workflowID
	}

	phone := cfg.PhoneNumber
	if phone == "" {
		phone = prompt("Número de teléfono para la ejecución", "")
		if phone == "" {
			return fmt.Errorf("número de teléfono requerido")
		}
	}

	fmt.Printf("  Iniciando ejecución del flujo %q...\n", workflowName)

	execResp, err := client.StartExecution(ctx, workflowID, phone, nil)
	if err != nil {
		return fmt.Errorf("error iniciando ejecución: %v", err)
	}
	fmt.Printf("  ✓ Ejecución iniciada (id: %s)\n", execResp.ExecutionID)

	// Cleanup: end execution when done
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.UpdateExecutionStatus(cleanupCtx, workflowID, execResp.ExecutionID, "ended")
	}()

	timeout := cfg.Defaults.Timeout
	if timeout == 0 {
		timeout = 120
	}

	var messages []string

	fmt.Println("\n  ╔══════════════════════════════════════╗")
	fmt.Println("  ║  Grabando flujo — escribe 'done'     ║")
	fmt.Println("  ║  para terminar, Ctrl+C para salir    ║")
	fmt.Println("  ╚══════════════════════════════════════╝")
	fmt.Println()

	for {
		status, err := runner.PollUntil(ctx, client, workflowID, execResp.ExecutionID,
			time.Duration(timeout)*time.Second, "waiting", "ended", "failed")
		if err != nil {
			return fmt.Errorf("error esperando al workflow: %v", err)
		}

		if status.Status == "ended" || status.Status == "failed" {
			fmt.Printf("\n  ⚡ El flujo terminó (status: %s)\n", status.Status)
			break
		}

		stepName := formatCurrentStep(status.CurrentStep)
		fmt.Printf("  ─── Workflow espera en: %s ───\n", stepName)

		fmt.Print("  Tú > ")
		if !promptReader.Scan() {
			fmt.Println()
			break
		}
		input := strings.TrimSpace(promptReader.Text())

		if input == "" || strings.ToLower(input) == "done" {
			break
		}

		messages = append(messages, input)
		fmt.Println()

		if err := client.ResumeExecution(ctx, workflowID, execResp.ExecutionID, input); err != nil {
			return fmt.Errorf("error enviando mensaje: %v", err)
		}
	}

	if len(messages) == 0 {
		return fmt.Errorf("no se grabaron mensajes, no se generó spec")
	}

	// Get events and build the spec
	fmt.Println("\n  Obteniendo eventos y generando spec...")

	evts, err := client.GetEvents(ctx, workflowID, execResp.ExecutionID)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("error obteniendo eventos: %v", err)
	}

	// Reverse to chronological order
	for i, j := 0, len(evts)-1; i < j; i, j = i+1, j-1 {
		evts[i], evts[j] = evts[j], evts[i]
	}

	var actualPath []string
	actualDecisions := map[string]string{}
	terminalStatus := statusForEvents(evts)

	for _, ev := range evts {
		if ev.EventType == "step_entered" || ev.EventType == "execution_started" {
			if step, ok := ev.Step["identifier"]; ok {
				if stepID, ok := step.(string); ok {
					actualPath = append(actualPath, stepID)
				}
			}
		}
		if ev.EventType == "decision_evaluated" && ev.EdgeLabel != "" {
			if step, ok := ev.Step["identifier"]; ok {
				if stepID, ok := step.(string); ok {
					actualDecisions[stepID] = ev.EdgeLabel
				}
			}
		}
	}

	// Deduplicate path (keep order, remove consecutive duplicates)
	var deduped []string
	for _, id := range actualPath {
		if len(deduped) == 0 || deduped[len(deduped)-1] != id {
			deduped = append(deduped, id)
		}
	}

	// Build spec messages
	specMessages := make([]spec.Message, len(messages))
	for i, m := range messages {
		specMessages[i] = spec.Message{User: m}
	}

	// Build spec
	s := &spec.Spec{
		Workflow: workflowID,
		Given: spec.Given{
			PhoneNumber: phone,
		},
		When: spec.When{
			Messages: specMessages,
		},
		Then: spec.Then{
			Path:           deduped,
			Decisions:      actualDecisions,
			TerminalStatus: terminalStatus,
		},
	}

	// Prompt for spec name
	defaultName := fmt.Sprintf("%s-learned", workflowName)
	s.Name = prompt("Nombre del spec", defaultName)
	if s.Name == "" {
		s.Name = defaultName
	}

	// Prompt for filename
	defaultFile := strings.ToLower(strings.ReplaceAll(s.Name, " ", "-")) + ".yaml"
	fileName := prompt("Archivo a guardar", defaultFile)
	if fileName == "" {
		fileName = defaultFile
	}

	specsDir := filepath.Join(root, cfg.SpecsDir)
	os.MkdirAll(specsDir, 0755)
	filePath := filepath.Join(specsDir, fileName)

	if err := spec.Save(filePath, s); err != nil {
		return fmt.Errorf("error guardando spec: %v", err)
	}

	fmt.Printf("\n  ✓ Spec guardado: %s\n", filePath)
	fmt.Printf("    - %d mensajes grabados\n", len(specMessages))
	fmt.Printf("    - %d nodos en la ruta\n", len(deduped))
	fmt.Printf("    - %d decisiones capturadas\n", len(actualDecisions))
	fmt.Println()
	fmt.Println("  Para ejecutarlo:")
	fmt.Printf("    kfs run --spec %s\n", filePath)

	return nil
}

func formatCurrentStep(step interface{}) string {
	if step == nil {
		return "?"
	}
	m, ok := step.(map[string]interface{})
	if !ok {
		return fmt.Sprintf("%v", step)
	}
	if id, ok := m["identifier"].(string); ok {
		return id
	}
	return fmt.Sprintf("%v", step)
}

func statusForEvents(evts []kapso.Event) string {
	for i := len(evts) - 1; i >= 0; i-- {
		if evts[i].EventType == "execution_ended" {
			return "ended"
		}
		if evts[i].EventType == "execution_failed" {
			return "failed"
		}
	}
	return "waiting"
}

func init() {
	learnCmd.Flags().StringVar(&learnWorkflow, "workflow", "", "ID del flujo a grabar")
	RootCmd.AddCommand(learnCmd)
}
