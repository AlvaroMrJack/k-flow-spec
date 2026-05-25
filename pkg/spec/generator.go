package spec

import (
	"fmt"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/kapso"
)

// Generate creates a stub spec based on a workflow definition
func Generate(w *kapso.Workflow, def *kapso.Definition) *Spec {
	spec := &Spec{
		Name:     fmt.Sprintf("%s - Auto Generated", w.Name),
		Workflow: w.ID,
		Given: Given{
			Variables: map[string]interface{}{
				"__FILL_ME__": "value",
			},
		},
		When: When{
			Messages: []Message{},
		},
		Then: Then{
			Path:           []string{"start"},
			TerminalStatus: "ended",
			Decisions:      make(map[string]string),
			Snapshot:       true,
		},
	}

	// Basic discovery of wait_for_response nodes
	for _, node := range def.Nodes {
		if nodeType, ok := node.Data["node_type"].(string); ok {
			if nodeType == "wait_for_response" {
				spec.When.Messages = append(spec.When.Messages, Message{User: "__EDIT_ME__"})
			} else if nodeType == "decide" {
				spec.Then.Decisions[node.ID] = "__CHOOSE_LABEL__"
			}
			spec.Then.Path = append(spec.Then.Path, node.ID)
		}
	}

	return spec
}
