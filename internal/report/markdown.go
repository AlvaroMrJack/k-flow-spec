package report

import (
	"fmt"
	"io"
	"time"

	"github.com/AlvaroMrJack/k-flow-spec/pkg/runner"
)

type MarkdownReporter struct{}

func (r *MarkdownReporter) Format() Format { return FormatMarkdown }

func (r *MarkdownReporter) Write(w io.Writer, results []*runner.Result) error {
	var summary Summary

	fmt.Fprintf(w, "# k-flow-spec Report\n\n")
	fmt.Fprintf(w, "Generated: %s\n\n", time.Now().UTC().Format(time.RFC3339))

	fmt.Fprintf(w, "## Summary\n\n")
	fmt.Fprintf(w, "| Status | Count |\n")
	fmt.Fprintf(w, "|--------|-------|\n")

	for _, res := range results {
		if res.Passed {
			summary.Passed++
		} else {
			summary.Failed++
		}
		summary.DurationMs += res.Duration.Milliseconds()
	}

	fmt.Fprintf(w, "| ✅ Passed | %d |\n", summary.Passed)
	fmt.Fprintf(w, "| ❌ Failed | %d |\n", summary.Failed)
	fmt.Fprintf(w, "| ⏱ Duration | %dms |\n\n", summary.DurationMs)

	fmt.Fprintf(w, "## Results\n\n")

	for _, res := range results {
		status := "✅"
		if !res.Passed {
			status = "❌"
		}
		fmt.Fprintf(w, "### %s %s (%s)\n\n", status, res.SpecName, res.Duration)
		fmt.Fprintf(w, "- **Workflow:** `%s`\n", res.WorkflowID)
		fmt.Fprintf(w, "- **Status:** %s\n", map[bool]string{true: "Passed", false: "Failed"}[res.Passed])

		if !res.Passed && len(res.Errors) > 0 {
			fmt.Fprintf(w, "\n#### Errors\n\n")
			for _, e := range res.Errors {
				fmt.Fprintf(w, "- **%s:** %s\n", e.Type, e.Message)
				if e.Expected != nil && e.Actual != nil {
					fmt.Fprintf(w, "  - Expected: `%v`\n", e.Expected)
					fmt.Fprintf(w, "  - Actual: `%v`\n", e.Actual)
				}
			}
		}
		fmt.Fprintln(w)
	}

	return nil
}
