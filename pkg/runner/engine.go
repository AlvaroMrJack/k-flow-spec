package runner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/AlvaroMrJack/k-flow-spec/internal/config"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/kapso"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/spec"
)

type Result struct {
	SpecName        string
	WorkflowID      string
	Passed          bool
	Errors          []RunError
	Duration        time.Duration
	ActualPath      []string
	ActualDecisions map[string]string
	Snapshot        *spec.Snapshot
}

type RunError struct {
	Type     string
	Expected interface{}
	Actual   interface{}
	Message  string
}

type Engine struct {
	client      *kapso.Client
	cfg         *config.KfsConfig
	Interactive bool // If true, prints debug info during execution
	Progress    func(format string, args ...interface{})
}

func NewEngine(client *kapso.Client, cfg *config.KfsConfig) *Engine {
	return &Engine{client: client, cfg: cfg}
}

func (e *Engine) logProgress(format string, args ...interface{}) {
	if e.Progress != nil {
		e.Progress(format, args...)
	} else {
		slog.Info(fmt.Sprintf(format, args...))
	}
}

func (e *Engine) Run(ctx context.Context, s *spec.Spec) *Result {
	start := time.Now()
	result := &Result{
		SpecName:   s.Name,
		WorkflowID: s.Workflow,
		Passed:     true,
	}

	timeout := s.Timeout
	if timeout == 0 {
		timeout = e.cfg.Defaults.Timeout
	}
	if timeout == 0 {
		timeout = 60
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	e.logProgress("▶ Iniciando ejecución de spec: %s", s.Name)

	phone := s.Given.PhoneNumber
	if phone == "" {
		phone = e.cfg.PhoneNumber
	}

	execResp, err := e.client.StartExecution(ctx, s.Workflow, phone, s.Given.Variables)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, RunError{
			Type:    "execution_error",
			Message: fmt.Sprintf("failed to start execution: %v", err),
		})
		result.Duration = time.Since(start)
		return result
	}
	e.logProgress("  ✓ Ejecución iniciada (id: %s)", execResp.ExecutionID)

	cleanupNeeded := true
	defer func() {
		if !cleanupNeeded {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := e.client.UpdateExecutionStatus(cleanupCtx, s.Workflow, execResp.ExecutionID, "ended"); err != nil {
			slog.Warn("cleanup failed", "execution_id", execResp.ExecutionID, "error", err)
		}
	}()

	actualPath := []string{}
	actualDecisions := map[string]string{}

	numMessages := len(s.When.Messages)

	for i, msg := range s.When.Messages {
		e.logProgress("  ⏳ Esperando que el workflow esté listo para recibir mensaje %d/%d...", i+1, numMessages)
		status, err := PollUntil(ctx, e.client, s.Workflow, execResp.ExecutionID,
			time.Duration(timeout)*time.Second, nil, "waiting", "ended", "failed")
		if err != nil {
			result.Passed = false
			result.Errors = append(result.Errors, RunError{
				Type:    "timeout",
				Message: fmt.Sprintf("timeout waiting for execution to be ready (message %d): %v", i+1, err),
			})
			result.Duration = time.Since(start)
			return result
		}

		if status.Status == "ended" || status.Status == "failed" {
			e.logProgress("  ⚠ La ejecución terminó antes de enviar el mensaje %d/%d (status: %s)", i+1, numMessages, status.Status)
			break
		}

		payload := msg.ToPayload()
		e.logProgress("  → Enviando mensaje %d/%d: %s", i+1, numMessages, msg.Display())
		if err := e.client.ResumeExecution(ctx, s.Workflow, execResp.ExecutionID, payload); err != nil {
			result.Passed = false
			result.Errors = append(result.Errors, RunError{
				Type:    "execution_error",
				Message: fmt.Sprintf("failed to resume execution (message %d): %v", i+1, err),
			})
			result.Duration = time.Since(start)
			return result
		}
	}

	e.logProgress("  ⏳ Esperando que la ejecución finalice...")
	finalStatus, err := PollUntil(ctx, e.client, s.Workflow, execResp.ExecutionID,
		time.Duration(timeout)*time.Second, nil, "ended", "failed", "waiting", "handoff")
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, RunError{
			Type:    "timeout",
			Message: fmt.Sprintf("timeout waiting for execution to finish: %v", err),
		})
		result.Duration = time.Since(start)
		return result
	}
	e.logProgress("  ✓ Ejecución finalizada (status: %s)", finalStatus.Status)
	cleanupNeeded = false

	evts, err := e.client.GetEvents(ctx, s.Workflow, execResp.ExecutionID)
	if err != nil {
		e.logProgress("  ⚠ Error obteniendo eventos: %v", err)
	} else {
		e.logProgress("  ✓ Eventos obtenidos: %d", len(evts))
		// API returns newest-first; reverse for chronological order
		for i, j := 0, len(evts)-1; i < j; i, j = i+1, j-1 {
			evts[i], evts[j] = evts[j], evts[i]
		}
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
	}

	snapshot := &spec.Snapshot{
		Workflow: s.Workflow,
		RunAt:    time.Now().UTC().Format(time.RFC3339),
		Events:   evts,
	}
	if finalStatus != nil {
		snapshot.ExecutionContext = finalStatus.ExecutionContext
	}
	result.Snapshot = snapshot
	result.ActualPath = actualPath
	result.ActualDecisions = actualDecisions

	e.logProgress("  ✓ Ruta real: %v", actualPath)
	e.logProgress("  ✓ Decisiones reales: %v", actualDecisions)

	errs := Validate(s, result)
	if len(errs) > 0 {
		result.Passed = false
		result.Errors = append(result.Errors, errs...)
	}

	result.Duration = time.Since(start)
	if result.Passed {
		e.logProgress("  ✓ ¡Spec superado! (%v)", result.Duration)
	} else {
		e.logProgress("  ✗ Spec fallido (%v) — %s", result.Duration, result.Errors[0].Message)
	}
	return result
}
