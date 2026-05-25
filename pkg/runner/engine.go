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
}

func NewEngine(client *kapso.Client, cfg *config.KfsConfig) *Engine {
	return &Engine{client: client, cfg: cfg}
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

	slog.Debug("starting execution", "spec", s.Name, "workflow", s.Workflow)

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
	slog.Debug("execution started", "execution_id", execResp.ExecutionID)

	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := e.client.UpdateExecutionStatus(cleanupCtx, s.Workflow, execResp.ExecutionID, "ended"); err != nil {
			slog.Warn("cleanup failed", "execution_id", execResp.ExecutionID, "error", err)
		}
	}()

	actualPath := []string{}
	actualDecisions := map[string]string{}

	for i, msg := range s.When.Messages {
		status, err := PollUntil(ctx, e.client, s.Workflow, execResp.ExecutionID,
			time.Duration(timeout)*time.Second, "waiting", "ended", "failed")
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
			break
		}

		if err := e.client.ResumeExecution(ctx, s.Workflow, execResp.ExecutionID, msg.User); err != nil {
			result.Passed = false
			result.Errors = append(result.Errors, RunError{
				Type:    "execution_error",
				Message: fmt.Sprintf("failed to resume execution (message %d): %v", i+1, err),
			})
			result.Duration = time.Since(start)
			return result
		}
		slog.Debug("sent message", "msg", msg.User)
	}

	finalStatus, err := PollUntil(ctx, e.client, s.Workflow, execResp.ExecutionID,
		time.Duration(timeout)*time.Second, "ended", "failed")
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, RunError{
			Type:    "timeout",
			Message: fmt.Sprintf("timeout waiting for execution to finish: %v", err),
		})
		result.Duration = time.Since(start)
		return result
	}

	evts, err := e.client.GetEvents(ctx, s.Workflow, execResp.ExecutionID)
	if err == nil {
		for _, ev := range evts {
			if ev.EventType == "step_entered" || ev.EventType == "execution_started" {
				if stepID, ok := ev.Step["identifier"]; ok {
					actualPath = append(actualPath, stepID)
				}
			}
			if ev.EventType == "decision_evaluated" && ev.EdgeLabel != "" {
				if stepID, ok := ev.Step["identifier"]; ok {
					actualDecisions[stepID] = ev.EdgeLabel
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

	errs := Validate(s, result)
	if len(errs) > 0 {
		result.Passed = false
		result.Errors = append(result.Errors, errs...)
	}

	result.Duration = time.Since(start)
	return result
}
