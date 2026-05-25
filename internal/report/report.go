package report

import (
	"io"

	"github.com/AlvaroMrJack/k-flow-spec/pkg/runner"
)

type Format string

const (
	FormatJSON     Format = "json"
	FormatMarkdown Format = "markdown"
	FormatJUnit    Format = "junit"
	FormatTAP      Format = "tap"
)

type Summary struct {
	Passed     int
	Failed     int
	Skipped    int
	DurationMs int64
}

type Reporter interface {
	Write(w io.Writer, results []*runner.Result) error
	Format() Format
}

func NewReporter(format Format) Reporter {
	switch format {
	case FormatJSON:
		return &JSONReporter{}
	case FormatMarkdown:
		return &MarkdownReporter{}
	case FormatJUnit:
		return &JUnitReporter{}
	case FormatTAP:
		return &TAPReporter{}
	default:
		return &JSONReporter{}
	}
}
