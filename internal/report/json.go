package report

import (
	"encoding/json"
	"io"
	"time"

	"github.com/AlvaroMrJack/k-flow-spec/pkg/runner"
)

type JSONReporter struct{}

func (r *JSONReporter) Format() Format { return FormatJSON }

func (r *JSONReporter) Write(w io.Writer, results []*runner.Result) error {
	var summary Summary
	specs := make([]map[string]interface{}, 0, len(results))

	for _, res := range results {
		if res.Passed {
			summary.Passed++
		} else {
			summary.Failed++
		}
		summary.DurationMs += res.Duration.Milliseconds()

		spec := map[string]interface{}{
			"name":       res.SpecName,
			"workflow":   res.WorkflowID,
			"passed":     res.Passed,
			"duration_ms": res.Duration.Milliseconds(),
		}

		if !res.Passed {
			errors := make([]map[string]interface{}, 0, len(res.Errors))
			for _, e := range res.Errors {
				errMap := map[string]interface{}{
					"type":    e.Type,
					"message": e.Message,
				}
				if e.Expected != nil {
					errMap["expected"] = e.Expected
				}
				if e.Actual != nil {
					errMap["actual"] = e.Actual
				}
				errors = append(errors, errMap)
			}
			spec["errors"] = errors
		}

		specs = append(specs, spec)
	}

	out := map[string]interface{}{
		"summary": map[string]interface{}{
			"passed":      summary.Passed,
			"failed":      summary.Failed,
			"skipped":     summary.Skipped,
			"duration_ms": summary.DurationMs,
		},
		"specs":     specs,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
