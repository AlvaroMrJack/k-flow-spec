package kapso

import "time"

type Workflow struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Definition struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type Node struct {
	ID   string                 `json:"id"`
	Data map[string]interface{} `json:"data"` // node_type, config, etc.
}

type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label"`
}

type Variable struct {
	Name        string      `json:"name"`
	SampleValue interface{} `json:"sample_value"`
}

type ExecutionResponse struct {
	TrackingID  string `json:"tracking_id"`
	ExecutionID string `json:"id"`
}

type DataWrapper struct {
	Data interface{} `json:"data"`
}

type ExecutionContext struct {
	Vars     map[string]interface{} `json:"vars,omitempty" yaml:"vars,omitempty"`
	System   map[string]interface{} `json:"system,omitempty" yaml:"system,omitempty"`
	Context  map[string]interface{} `json:"context,omitempty" yaml:"context,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type ExecutionStatus struct {
	Status           string                 `json:"status"`
	CurrentStep      interface{}            `json:"current_step"`
	ExecutionContext ExecutionContext       `json:"execution_context"`
}

type Event struct {
	ID          interface{}            `json:"id,omitempty"`
	EventType   string                 `json:"event_type"`
	Direction   string                 `json:"direction,omitempty"`
	EdgeLabel   string                 `json:"edge_label,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	Step        map[string]interface{} `json:"step,omitempty"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
}

type Trigger struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type MessagePayload struct {
	Kind string      `json:"kind"`
	Data interface{} `json:"data"`
}

func TextMessage(text string) MessagePayload {
	return MessagePayload{Kind: "payload", Data: text}
}

func ButtonMessage(buttonID, buttonTitle string) MessagePayload {
	return MessagePayload{Kind: "button_reply", Data: map[string]string{"id": buttonID, "title": buttonTitle}}
}
