package report

import (
	"fmt"
	"io"
	"time"

	"github.com/AlvaroMrJack/k-flow-spec/pkg/runner"
)

type TAPReporter struct{}

func (r *TAPReporter) Format() Format { return FormatTAP }

func (r *TAPReporter) Write(w io.Writer, results []*runner.Result) error {
	fmt.Fprintf(w, "TAP version 14\n")
	fmt.Fprintf(w, "1..%d\n", len(results))

	for i, res := range results {
		lineNum := i + 1
		if res.Passed {
			fmt.Fprintf(w, "ok %d - %s (%s)\n", lineNum, res.SpecName, res.Duration)
		} else {
			fmt.Fprintf(w, "not ok %d - %s (%s)\n", lineNum, res.SpecName, res.Duration)
			if len(res.Errors) > 0 {
				fmt.Fprintf(w, "  ---\n")
				for _, e := range res.Errors {
					fmt.Fprintf(w, "    %s: %s\n", e.Type, e.Message)
				}
				fmt.Fprintf(w, "  ...\n")
			}
		}
	}

	fmt.Fprintf(w, "# %s\n", time.Now().UTC().Format(time.RFC3339))
	return nil
}
