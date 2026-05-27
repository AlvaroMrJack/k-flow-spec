package spec

import (
	"github.com/AlvaroMrJack/k-flow-spec/pkg/kapso"
)

type Spec struct {
	Name     string `yaml:"name"`
	Workflow string `yaml:"workflow"`
	Kind     string `yaml:"kind,omitempty"`
	Timeout  int    `yaml:"timeout,omitempty"`
	Given    Given  `yaml:"given"`
	When     When   `yaml:"when"`
	Then     Then   `yaml:"then"`
}

type Given struct {
	Variables   map[string]interface{} `yaml:"variables,omitempty"`
	PhoneNumber string                 `yaml:"phone_number,omitempty"`
}

type ButtonMessage struct {
	ID    string `yaml:"id"`
	Title string `yaml:"title"`
}

type Message struct {
	User   string         `yaml:"user,omitempty"`
	Button *ButtonMessage `yaml:"button,omitempty"`
}

func (m Message) ToPayload() kapso.MessagePayload {
	if m.Button != nil {
		return kapso.ButtonMessage(m.Button.ID, m.Button.Title)
	}
	return kapso.TextMessage(m.User)
}

func (m Message) Display() string {
	if m.Button != nil {
		return "[button: " + m.Button.Title + "]"
	}
	return m.User
}

type When struct {
	Messages []Message `yaml:"messages,omitempty"`
}

type Then struct {
	Path           []string               `yaml:"path"`
	TerminalStatus string                 `yaml:"terminal_status"`
	Decisions      map[string]string      `yaml:"decisions,omitempty"`
	VariablesSet   map[string]interface{} `yaml:"variables_set,omitempty"`
	Snapshot       bool                   `yaml:"snapshot,omitempty"`
	EventsInclude  []kapso.Event          `yaml:"events_include,omitempty"`
}

type Snapshot struct {
	Workflow         string                 `yaml:"workflow"`
	RunAt            string                 `yaml:"run_at"`
	ExecutionContext kapso.ExecutionContext `yaml:"execution_context"`
	Events           []kapso.Event          `yaml:"events"`
}

// BroadcastSpec defines a broadcast campaign test.
type BroadcastSpec struct {
	Name  string         `yaml:"name"`
	Kind  string         `yaml:"kind"` // "broadcast"
	Given BroadcastGiven `yaml:"given"`
	When  BroadcastWhen  `yaml:"when"`
	Then  BroadcastThen  `yaml:"then"`
}

type BroadcastGiven struct {
	TemplateID    string               `yaml:"template_id"`
	PhoneNumberID string               `yaml:"phone_number_id"`
	Recipients    []BroadcastRecipient `yaml:"recipients"`
}

type BroadcastRecipient struct {
	Phone     string            `yaml:"phone"`
	Variables map[string]string `yaml:"variables,omitempty"`
}

type BroadcastWhen struct {
	Send struct{} `yaml:"send"`
}

type BroadcastThen struct {
	Status     string                       `yaml:"status"`
	Metrics    BroadcastMetrics             `yaml:"metrics"`
	Recipients []BroadcastRecipientStatus   `yaml:"recipients,omitempty"`
}

type BroadcastMetrics struct {
	SentCount    int     `yaml:"sent_count"`
	FailedCount  int     `yaml:"failed_count"`
	DeliveryRate float64 `yaml:"delivery_rate"`
}

type BroadcastRecipientStatus struct {
	Phone  string `yaml:"phone"`
	Status string `yaml:"status"`
}

// FlowSpec defines a WhatsApp Flow navigation test.
type FlowSpec struct {
	Name  string    `yaml:"name"`
	Kind  string    `yaml:"kind"` // "flow"
	Given FlowGiven `yaml:"given"`
	When  FlowWhen  `yaml:"when"`
	Then  FlowThen  `yaml:"then"`
}

type FlowGiven struct {
	FlowID      string                 `yaml:"flow_id"`
	Screen      string                 `yaml:"screen"`
	InitialData map[string]interface{} `yaml:"initial_data"`
}

type FlowWhen struct {
	Screens []FlowScreen `yaml:"screens"`
}

type FlowScreen struct {
	Screen string                 `yaml:"screen"`
	Action string                 `yaml:"action"`
	Fields map[string]interface{} `yaml:"fields,omitempty"`
}

type FlowThen struct {
	TerminalScreen  string                 `yaml:"terminal_screen"`
	SubmittedData   map[string]interface{} `yaml:"submitted_data,omitempty"`
	WebhookReceived bool                   `yaml:"webhook_received,omitempty"`
}
