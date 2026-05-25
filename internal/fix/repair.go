package fix

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/AlvaroMrJack/k-flow-spec/pkg/kapso"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/spec"
)

type Repairer struct {
	analyzer *Analyzer
}

func NewRepairer(specsDir string) *Repairer {
	return &Repairer{
		analyzer: NewAnalyzer(specsDir),
	}
}

func (r *Repairer) Repair(client *kapso.Client, workflowID string, apply bool, interactive bool) ([]Issue, error) {
	issues, err := r.analyzer.Analyze(client, workflowID)
	if err != nil {
		return nil, err
	}

	var repaired []Issue
	for _, issue := range issues {
		if !issue.AutoFix {
			slog.Info("cannot auto-fix", "type", issue.Type, "file", issue.SpecFile, "message", issue.Message)
			continue
		}

		if interactive {
			fmt.Printf("Reparar %s en %s? [y/N] ", issue.Message, issue.SpecFile)
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				continue
			}
		}

		if apply {
			if err := r.applyFix(issue); err != nil {
				slog.Error("fix failed", "file", issue.SpecFile, "error", err)
				continue
			}
			repaired = append(repaired, issue)
			slog.Info("fixed", "type", issue.Type, "file", issue.SpecFile)
		} else {
			slog.Info("would fix", "type", issue.Type, "file", issue.SpecFile, "message", issue.Message)
		}
	}

	return repaired, nil
}

func (r *Repairer) applyFix(issue Issue) error {
	if issue.Fix == nil {
		return fmt.Errorf("no fix available for issue: %s", issue.Message)
	}

	s, err := spec.Load(issue.Fix.File)
	if err != nil {
		return err
	}

	switch issue.Type {
	case IssueBranchRemoved:
		for nodeID := range s.Then.Decisions {
			if issue.Fix.OldValue != "" && nodeID == issue.Fix.OldValue {
				delete(s.Then.Decisions, nodeID)
			}
		}
	case IssueNodeRenamed:
		for i, nodeID := range s.Then.Path {
			if nodeID == issue.Fix.OldValue {
				s.Then.Path[i] = issue.Fix.NewValue
			}
		}
		if _, ok := s.Then.Decisions[issue.Fix.OldValue]; ok {
			s.Then.Decisions[issue.Fix.NewValue] = s.Then.Decisions[issue.Fix.OldValue]
			delete(s.Then.Decisions, issue.Fix.OldValue)
		}
	}

	return spec.Save(issue.Fix.File, s)
}

// UpdateSnapshots regenerates snapshots by running specs and saving fresh baselines.
// In production this would use the runner engine. For now it walks the snapshots dir.
func UpdateSnapshots(snapshotsDir string) error {
	entries, err := os.ReadDir(snapshotsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		ext := filepath.Ext(entry.Name())
		if ext != ".snap.yml" && ext != ".snap.yaml" {
			continue
		}
		snappath := filepath.Join(snapshotsDir, entry.Name())
		slog.Info("snapshot ready for update", "path", snappath)
	}

	return nil
}
