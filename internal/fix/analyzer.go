package fix

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlvaroMrJack/k-flow-spec/pkg/kapso"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/spec"
)

type Analyzer struct {
	specsDir string
}

func NewAnalyzer(specsDir string) *Analyzer {
	return &Analyzer{specsDir: specsDir}
}

// Analyze all specs in the directory and return detected issues.
func (a *Analyzer) Analyze(client *kapso.Client, workflowID string) ([]Issue, error) {
	var issues []Issue

	entries, err := os.ReadDir(a.specsDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read specs directory %s: %w", a.specsDir, err)
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".yaml" && filepath.Ext(entry.Name()) != ".yml" {
			continue
		}

		specPath := filepath.Join(a.specsDir, entry.Name())
		s, err := spec.Load(specPath)
		if err != nil {
			issues = append(issues, Issue{
				Type:     IssueInvalidYAML,
				Severity: SeverityError,
				SpecFile: specPath,
				Message:  fmt.Sprintf("failed to parse YAML: %v", err),
				AutoFix:  false,
			})
			continue
		}

		// Check for placeholder messages
		for i, msg := range s.When.Messages {
			if msg.User == "__EDIT_ME__" {
				issues = append(issues, Issue{
					Type:     IssuePlaceholderMessage,
					Severity: SeverityWarning,
					SpecFile: specPath,
					Message:  fmt.Sprintf("message #%d is still __EDIT_ME__", i+1),
					AutoFix:  false,
				})
			}
		}

		// If we have a Kapso client, check against live definitions
		if client != nil {
			def, err := client.GetDefinition(context.Background(), s.Workflow)
			if err != nil {
				continue // skip if can't fetch definition
			}

			a.analyzePath(s, def, specPath, &issues)
			a.analyzeDecisions(s, def, specPath, &issues)
		}
	}

	return issues, nil
}

func (a *Analyzer) analyzePath(s *spec.Spec, def *kapso.Definition, specPath string, issues *[]Issue) {
	// Build set of valid node IDs
	validNodes := make(map[string]bool)
	for _, node := range def.Nodes {
		validNodes[node.ID] = true
	}

	// Check each node in path exists in definition
	for _, nodeID := range s.Then.Path {
		if !validNodes[nodeID] {
			*issues = append(*issues, Issue{
				Type:     IssueNodeRenamed,
				Severity: SeverityError,
				SpecFile: specPath,
				Message:  fmt.Sprintf("node %q not found in current workflow definition", nodeID),
				AutoFix:  false,
			})
		}
	}
}

func (a *Analyzer) analyzeDecisions(s *spec.Spec, def *kapso.Definition, specPath string, issues *[]Issue) {
	// Find decide nodes and their valid labels
	validLabels := make(map[string]map[string]bool)
	for _, node := range def.Nodes {
		if nodeType, ok := node.Data["node_type"].(string); ok && nodeType == "decide" {
			labels := make(map[string]bool)
			if config, ok := node.Data["config"].(map[string]interface{}); ok {
				if conditions, ok := config["conditions"].([]interface{}); ok {
					for _, c := range conditions {
						if cond, ok := c.(map[string]interface{}); ok {
							if label, ok := cond["label"].(string); ok {
								labels[label] = true
							}
						}
					}
				}
			}
			validLabels[node.ID] = labels
		}
	}

	for nodeID, expectedLabel := range s.Then.Decisions {
		labels, ok := validLabels[nodeID]
		if !ok {
			*issues = append(*issues, Issue{
				Type:     IssueNodeRenamed,
				Severity: SeverityError,
				SpecFile: specPath,
				Message:  fmt.Sprintf("decide node %q not found in current workflow", nodeID),
				AutoFix:  false,
			})
			continue
		}
		if !labels[expectedLabel] {
			*issues = append(*issues, Issue{
				Type:     IssueBranchRemoved,
				Severity: SeverityError,
				SpecFile: specPath,
				Message:  fmt.Sprintf("decision label %q no longer exists on node %q", expectedLabel, nodeID),
				AutoFix:  true,
			})
		}
	}
}
