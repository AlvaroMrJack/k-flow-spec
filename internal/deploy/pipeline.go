package deploy

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Pipeline struct {
	WorkDir     string
	Environment string
	DryRun      bool
	Full        bool
	Workflows   []string
	KapsoCLI    string
}

type DeployResult struct {
	Passed   bool          `json:"passed"`
	Steps    []StepResult  `json:"steps"`
	Duration time.Duration `json:"duration"`
}

type StepResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

func NewPipeline(workDir, environment string, dryRun, full bool, workflows []string) *Pipeline {
	kapsoPath, _ := exec.LookPath("kapso")

	return &Pipeline{
		WorkDir:     workDir,
		Environment: environment,
		DryRun:      dryRun,
		Full:        full,
		Workflows:   workflows,
		KapsoCLI:    kapsoPath,
	}
}

func (p *Pipeline) Run(ctx context.Context) *DeployResult {
	start := time.Now()
	result := &DeployResult{
		Passed: true,
		Steps:  make([]StepResult, 0),
	}

	// Step 1: Build
	step1 := p.runStep("build", func() (string, error) {
		if p.KapsoCLI == "" {
			return "", fmt.Errorf("kapso CLI no encontrado en PATH")
		}
		if p.DryRun {
			return "[dry-run] kapso build ejecutado", nil
		}
		return p.execCommand(ctx, p.KapsoCLI, "build")
	})
	result.Steps = append(result.Steps, step1)
	if !step1.Passed {
		result.Passed = false
		result.Duration = time.Since(start)
		return result
	}

	// Step 2: Push
	step2 := p.runStep("push", func() (string, error) {
		if p.DryRun {
			return "[dry-run] kapso push ejecutado", nil
		}
		args := []string{"push", "workflow"}
		if p.Environment != "" {
			args = append(args, "--environment", p.Environment)
		}
		if len(p.Workflows) > 0 {
			args = append(args, p.Workflows...)
		}
		return p.execCommand(ctx, p.KapsoCLI, args...)
	})
	result.Steps = append(result.Steps, step2)
	if !step2.Passed {
		result.Passed = false
		result.Duration = time.Since(start)
		return result
	}

	// Step 3: Generate (regenerate specs)
	step3 := p.runStep("generate", func() (string, error) {
		if p.DryRun {
			return "[dry-run] kfs generate ejecutado", nil
		}
		return p.execCommand(ctx, "kfs", "generate")
	})
	result.Steps = append(result.Steps, step3)
	if !step3.Passed {
		slog.Warn("generate step failed, continuing", "error", step3.Error)
	}

	// Step 4: Run tests
	step4 := p.runStep("test", func() (string, error) {
		if p.DryRun {
			return "[dry-run] kfs run --ci ejecutado", nil
		}
		return p.execCommand(ctx, "kfs", "run", "--ci")
	})
	result.Steps = append(result.Steps, step4)
	if !step4.Passed {
		result.Passed = false
	}

	// Step 5 (optional): Full deploy with broadcast + webhook
	if p.Full {
		step5 := p.runStep("broadcast-test", func() (string, error) {
			if p.DryRun {
				return "[dry-run] kfs run-broadcast ejecutado", nil
			}
			return p.execCommand(ctx, "kfs", "run-broadcast", "--dry-run")
		})
		result.Steps = append(result.Steps, step5)
	}

	result.Duration = time.Since(start)
	return result
}

func (p *Pipeline) runStep(name string, fn func() (string, error)) StepResult {
	slog.Info("deploy step", "step", name)
	output, err := fn()
	if err != nil {
		return StepResult{
			Name:   name,
			Passed: false,
			Error:  err.Error(),
			Output: output,
		}
	}
	return StepResult{
		Name:   name,
		Passed: true,
		Output: output,
	}
}

func (p *Pipeline) execCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = p.WorkDir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func FindWorkflows(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var workflows []string
	for _, e := range entries {
		if !e.IsDir() {
			ext := filepath.Ext(e.Name())
			if ext == ".js" || ext == ".ts" || ext == ".json" {
				workflows = append(workflows, e.Name())
			}
		}
	}

	return workflows, nil
}
