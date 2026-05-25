package flow

import (
	"fmt"
	"time"
)

type ScreenAction string

const (
	ActionNext    ScreenAction = "next"
	ActionSelect  ScreenAction = "select"
	ActionSubmit  ScreenAction = "submit"
	ActionConfirm ScreenAction = "confirm"
)

type FlowSpec struct {
	Name        string                 `json:"name"`
	FlowID      string                 `json:"flow_id"`
	InitialData map[string]interface{} `json:"initial_data"`
	Screens     []FlowScreenStep       `json:"screens"`
	Then        FlowThen               `json:"then"`
}

type FlowScreenStep struct {
	Screen string                 `json:"screen"`
	Action string                 `json:"action"`
	Fields map[string]interface{} `json:"fields,omitempty"`
}

type FlowThen struct {
	TerminalScreen  string                 `json:"terminal_screen"`
	SubmittedData   map[string]interface{} `json:"submitted_data,omitempty"`
	WebhookReceived bool                   `json:"webhook_received,omitempty"`
}

type FlowResult struct {
	Name           string                 `json:"name"`
	Passed         bool                   `json:"passed"`
	CurrentScreen  string                 `json:"current_screen"`
	TerminalScreen string                 `json:"terminal_screen"`
	Errors         []string               `json:"errors,omitempty"`
	ScreensVisited []string               `json:"screens_visited"`
	SubmittedData  map[string]interface{} `json:"submitted_data,omitempty"`
	DurationMs     int64                  `json:"duration_ms"`
}

type Simulator struct {
	baseURL string
	apiKey  string
	mock    bool
}

func NewSimulator(baseURL, apiKey string, mock bool) *Simulator {
	return &Simulator{baseURL: baseURL, apiKey: apiKey, mock: mock}
}

func (s *Simulator) Run(spec *FlowSpec) *FlowResult {
	start := time.Now()
	result := &FlowResult{
		Name:           spec.Name,
		Passed:         true,
		ScreensVisited: make([]string, 0),
		SubmittedData:  make(map[string]interface{}),
	}

	submittedData := make(map[string]interface{})
	for k, v := range spec.InitialData {
		submittedData[k] = v
	}

	for i, step := range spec.Screens {
		result.ScreensVisited = append(result.ScreensVisited, step.Screen)
		result.CurrentScreen = step.Screen

		// Simulate field entry
		for k, v := range step.Fields {
			submittedData[k] = v
		}

		// Validate screen transitions
		switch step.Action {
		case "next", "select", "submit", "confirm":
			// Valid actions
		default:
			result.Errors = append(result.Errors,
				fmt.Sprintf("screen %d (%s): acción inválida %q", i+1, step.Screen, step.Action))
			result.Passed = false
		}
	}

	result.SubmittedData = submittedData

	// Validate terminal screen
	if spec.Then.TerminalScreen != "" {
		result.TerminalScreen = spec.Then.TerminalScreen
		if result.CurrentScreen != spec.Then.TerminalScreen {
			// Check if last visited screen matches expected terminal
			if len(result.ScreensVisited) > 0 {
				last := result.ScreensVisited[len(result.ScreensVisited)-1]
				if last != spec.Then.TerminalScreen {
					result.Errors = append(result.Errors,
						fmt.Sprintf("terminal screen esperada: %s, actual: %s", spec.Then.TerminalScreen, last))
					result.Passed = false
				}
			}
		}
	}

	// Validate submitted data
	if spec.Then.SubmittedData != nil {
		for k, expectedV := range spec.Then.SubmittedData {
			actualV, ok := submittedData[k]
			if !ok {
				result.Errors = append(result.Errors,
					fmt.Sprintf("campo %q no encontrado en datos enviados", k))
				result.Passed = false
				continue
			}
			if fmt.Sprintf("%v", expectedV) != fmt.Sprintf("%v", actualV) {
				result.Errors = append(result.Errors,
					fmt.Sprintf("campo %q: esperado %v, actual %v", k, expectedV, actualV))
				result.Passed = false
			}
		}
	}

	result.DurationMs = time.Since(start).Milliseconds()
	return result
}
