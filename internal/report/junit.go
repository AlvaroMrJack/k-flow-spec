package report

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/AlvaroMrJack/k-flow-spec/pkg/runner"
)

type JUnitReporter struct{}

func (r *JUnitReporter) Format() Format { return FormatJUnit }

type junitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Time      string          `xml:"time,attr"`
	TestCases []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Content string `xml:",chardata"`
}

func (r *JUnitReporter) Write(w io.Writer, results []*runner.Result) error {
	suite := junitTestSuite{
		Name:      "k-flow-spec",
		Tests:     len(results),
		TestCases: make([]junitTestCase, 0, len(results)),
	}

	var totalMs int64
	for _, res := range results {
		totalMs += res.Duration.Milliseconds()

		tc := junitTestCase{
			Name:      res.SpecName,
			ClassName: res.WorkflowID,
			Time:      fmt.Sprintf("%.3f", res.Duration.Seconds()),
		}

		if !res.Passed && len(res.Errors) > 0 {
			suite.Failures++
			msg := res.Errors[0].Message
			tc.Failure = &junitFailure{
				Message: msg,
				Content: formatErrorsForJUnit(res.Errors),
			}
		}

		suite.TestCases = append(suite.TestCases, tc)
	}

	suite.Time = fmt.Sprintf("%.3f", float64(totalMs)/1000.0)

	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(suite); err != nil {
		return err
	}
	fmt.Fprintln(w)
	return nil
}

func formatErrorsForJUnit(errs []runner.RunError) string {
	var s string
	for _, e := range errs {
		s += fmt.Sprintf("%s: %s\n", e.Type, e.Message)
	}
	return s
}
