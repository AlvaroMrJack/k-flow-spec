package runner

import (
	"fmt"

	"github.com/AlvaroMrJack/k-flow-spec/pkg/kapso"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/spec"
)

func Validate(s *spec.Spec, result *Result) []RunError {
	var errs []RunError

	if len(s.Then.Path) > 0 {
		errs = append(errs, validatePath(s.Then.Path, result.ActualPath)...)
	}

	if len(s.Then.Decisions) > 0 {
		errs = append(errs, validateDecisions(s.Then.Decisions, result.ActualDecisions)...)
	}

	if s.Then.TerminalStatus != "" && result.Snapshot != nil {
		found := false
		for _, ev := range result.Snapshot.Events {
			if ev.EventType == "execution_ended" && s.Then.TerminalStatus == "ended" {
				found = true
				break
			}
			if ev.EventType == "execution_failed" && s.Then.TerminalStatus == "failed" {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, RunError{
				Type:     "execution_error",
				Expected: s.Then.TerminalStatus,
				Message:  fmt.Sprintf("expected terminal status %q not reached", s.Then.TerminalStatus),
			})
		}
	}

	if len(s.Then.VariablesSet) > 0 && result.Snapshot != nil {
		for k, expectedV := range s.Then.VariablesSet {
			actualV, ok := result.Snapshot.ExecutionContext.Vars[k]
			if !ok {
				errs = append(errs, RunError{
					Type:     "variable_mismatch",
					Expected: expectedV,
					Actual:   nil,
					Message:  fmt.Sprintf("variable %q not found in execution context", k),
				})
				continue
			}
			if fmt.Sprintf("%v", expectedV) != fmt.Sprintf("%v", actualV) {
				errs = append(errs, RunError{
					Type:     "variable_mismatch",
					Expected: expectedV,
					Actual:   actualV,
					Message:  fmt.Sprintf("variable %q mismatch", k),
				})
			}
		}
	}

	if len(s.Then.EventsInclude) > 0 {
		errs = append(errs, validateEventsInclude(s.Then.EventsInclude, result.Snapshot.Events)...)
	}

	return errs
}

func validatePath(expected, actual []string) []RunError {
	if len(actual) == 0 {
		return []RunError{{
			Type:     "path_mismatch",
			Expected: expected,
			Actual:   actual,
			Message:  "no path recorded",
		}}
	}

	i := 0
	for _, exp := range expected {
		found := false
		for i < len(actual) {
			if actual[i] == exp {
				found = true
				i++
				break
			}
			i++
		}
		if !found {
			return []RunError{{
				Type:     "path_mismatch",
				Expected: expected,
				Actual:   actual,
				Message:  fmt.Sprintf("expected node %q not found in path", exp),
			}}
		}
	}
	return nil
}

func validateDecisions(expected, actual map[string]string) []RunError {
	var errs []RunError
	for nodeID, expectedLabel := range expected {
		actualLabel, ok := actual[nodeID]
		if !ok {
			errs = append(errs, RunError{
				Type:     "decision_mismatch",
				Expected: expectedLabel,
				Actual:   nil,
				Message:  fmt.Sprintf("no decision recorded for node %q", nodeID),
			})
			continue
		}
		if actualLabel != expectedLabel {
			errs = append(errs, RunError{
				Type:     "decision_mismatch",
				Expected: expectedLabel,
				Actual:   actualLabel,
				Message:  fmt.Sprintf("node %q: expected decision %q, got %q", nodeID, expectedLabel, actualLabel),
			})
		}
	}
	return errs
}

func validateEventsInclude(expected []kapso.Event, actual []kapso.Event) []RunError {
	var errs []RunError
	for _, exp := range expected {
		found := false
		for _, act := range actual {
			if act.EventType == exp.EventType {
				if exp.EdgeLabel == "" || act.EdgeLabel == exp.EdgeLabel {
					found = true
					break
				}
			}
		}
		if !found {
			errs = append(errs, RunError{
				Type:     "event_mismatch",
				Expected: exp,
				Message:  fmt.Sprintf("expected event %q not found", exp.EventType),
			})
		}
	}
	return errs
}
