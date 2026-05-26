package mock

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/AlvaroMrJack/k-flow-spec/pkg/kapso"
)

type executionState struct {
	ID         string
	WorkflowID string
	Status     string
	Phone      string
	Variables  map[string]interface{}
	Messages   []string
	Events     []kapso.Event
	StartTime  time.Time
}

type Server struct {
	addr       string
	mu         sync.Mutex
	workflows  []kapso.Workflow
	definitions map[string]*kapso.Definition
	variables   map[string][]kapso.Variable
	triggers    map[string][]kapso.Trigger
	executions  map[string]*executionState
	nextID     int
}

func NewServer(addr string) *Server {
	s := &Server{
		addr:        addr,
		workflows:   make([]kapso.Workflow, 0),
		definitions: make(map[string]*kapso.Definition),
		variables:   make(map[string][]kapso.Variable),
		triggers:    make(map[string][]kapso.Trigger),
		executions:  make(map[string]*executionState),
		nextID:      1,
	}

	s.loadDefaultFixture()

	return s
}

func (s *Server) loadDefaultFixture() {
	s.workflows = append(s.workflows, kapso.Workflow{
		ID:        "support-router",
		Name:      "Support Router",
		Status:    "active",
		CreatedAt: time.Now(),
	})

	s.definitions["support-router"] = &kapso.Definition{
		Nodes: []kapso.Node{
			{ID: "start", Data: map[string]interface{}{
				"node_type": "start",
				"config": map[string]interface{}{
					"message": "Bienvenido al soporte de CorteYa",
				},
			}},
			{ID: "intro", Data: map[string]interface{}{
				"node_type": "send_text",
				"config": map[string]interface{}{
					"message": "Hola {{customer_name}}, ¿cómo podemos ayudarte?",
				},
			}},
			{ID: "wait_reply", Data: map[string]interface{}{
				"node_type": "wait_for_response",
				"config": map[string]interface{}{
					"save_response_to": "user_input",
				},
			}},
			{ID: "classify", Data: map[string]interface{}{
				"node_type": "decide",
				"config": map[string]interface{}{
					"decision_type": "ai",
					"conditions": []interface{}{
						map[string]interface{}{"label": "billing", "description": "Facturación"},
						map[string]interface{}{"label": "technical", "description": "Soporte técnico"},
						map[string]interface{}{"label": "other", "description": "Otros"},
					},
				},
			}},
			{ID: "handoff", Data: map[string]interface{}{
				"node_type": "handoff",
				"config": map[string]interface{}{
					"message": "Te transferimos con un agente. Gracias por tu paciencia.",
				},
			}},
			{ID: "end", Data: map[string]interface{}{
				"node_type": "end",
			}},
		},
		Edges: []kapso.Edge{
			{Source: "start", Target: "intro", Label: "next"},
			{Source: "intro", Target: "wait_reply", Label: "next"},
			{Source: "wait_reply", Target: "classify", Label: "next"},
			{Source: "classify", Target: "handoff", Label: "billing"},
			{Source: "classify", Target: "handoff", Label: "technical"},
			{Source: "classify", Target: "handoff", Label: "other"},
			{Source: "handoff", Target: "end", Label: "next"},
		},
	}

	s.variables["support-router"] = []kapso.Variable{
		{Name: "customer_name", SampleValue: "Juan"},
		{Name: "service", SampleValue: "corte clásico"},
	}

	s.triggers["support-router"] = []kapso.Trigger{
		{ID: "trg_1", Type: "inbound"},
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/platform/v1/", s.handleAPI)

	slog.Info("Mock server starting", "addr", s.addr)
	return http.ListenAndServe(s.addr, mux)
}

func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/platform/v1")
	path = strings.TrimSuffix(path, "/")
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")

	slog.Debug("mock request", "method", r.Method, "path", r.URL.Path, "parts", parts)

	w.Header().Set("Content-Type", "application/json")

	switch {
	case len(parts) == 1 && parts[0] == "workflows" && r.Method == "GET":
		s.handleListWorkflows(w, r)
	case len(parts) == 2 && parts[0] == "workflows" && r.Method == "GET":
		s.handleGetWorkflow(w, r, parts[1])
	case len(parts) == 3 && parts[0] == "workflows" && parts[2] == "definition" && r.Method == "GET":
		s.handleGetDefinition(w, r, parts[1])
	case len(parts) == 3 && parts[0] == "workflows" && parts[2] == "variables" && r.Method == "GET":
		s.handleGetVariables(w, r, parts[1])
	case len(parts) == 3 && parts[0] == "workflows" && parts[2] == "triggers" && r.Method == "GET":
		s.handleGetTriggers(w, r, parts[1])
	case len(parts) == 3 && parts[0] == "workflows" && parts[2] == "executions" && r.Method == "POST":
		s.handleStartExecution(w, r, parts[1])
	case len(parts) == 2 && parts[0] == "workflow_executions" && r.Method == "GET":
		s.handleGetExecution(w, r, parts[1])
	case len(parts) == 2 && parts[0] == "workflow_executions" && r.Method == "PATCH":
		s.handleUpdateStatus(w, r, parts[1])
	case len(parts) == 3 && parts[0] == "workflow_executions" && parts[2] == "resume" && r.Method == "POST":
		s.handleResumeExecution(w, r, parts[1])
	case len(parts) == 3 && parts[0] == "workflow_executions" && parts[2] == "events" && r.Method == "GET":
		s.handleGetEvents(w, r, parts[1])
	default:
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}
}

func (s *Server) json(w http.ResponseWriter, status int, v interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (s *Server) handleListWorkflows(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.json(w, http.StatusOK, map[string]interface{}{"data": s.workflows})
}

func (s *Server) handleGetWorkflow(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, wf := range s.workflows {
		if wf.ID == id {
			s.json(w, http.StatusOK, wf)
			return
		}
	}
	http.Error(w, `{"error":"workflow not found"}`, http.StatusNotFound)
}

func (s *Server) handleGetDefinition(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	def, ok := s.definitions[id]
	if !ok {
		http.Error(w, `{"error":"workflow not found"}`, http.StatusNotFound)
		return
	}
	// Return full workflow with definition wrapped in data
	var workflow *kapso.Workflow
	for i := range s.workflows {
		if s.workflows[i].ID == id {
			workflow = &s.workflows[i]
			break
		}
	}
	if workflow == nil {
		http.Error(w, `{"error":"workflow not found"}`, http.StatusNotFound)
		return
	}
	s.json(w, http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"id":         workflow.ID,
			"name":       workflow.Name,
			"status":     workflow.Status,
			"slug":       workflow.ID,
			"created_at": workflow.CreatedAt,
			"updated_at": workflow.CreatedAt,
			"definition": def,
		},
	})
}

func (s *Server) handleGetVariables(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	vars, ok := s.variables[id]
	if !ok {
		http.Error(w, `{"error":"workflow not found"}`, http.StatusNotFound)
		return
	}

	type varResponse struct {
		Fixed      []kapso.Variable `json:"fixed"`
		Discovered []kapso.Variable `json:"discovered"`
	}

	s.json(w, http.StatusOK, varResponse{Fixed: vars, Discovered: nil})
}

func (s *Server) handleGetTriggers(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	triggers, ok := s.triggers[id]
	if !ok {
		http.Error(w, `{"error":"workflow not found"}`, http.StatusNotFound)
		return
	}
	s.json(w, http.StatusOK, map[string]interface{}{"data": triggers})
}

func (s *Server) handleStartExecution(w http.ResponseWriter, r *http.Request, workflowID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.definitions[workflowID]; !ok {
		http.Error(w, `{"error":"workflow not found"}`, http.StatusNotFound)
		return
	}

	execID := fmt.Sprintf("exec_%d", s.nextID)
	trackingID := fmt.Sprintf("trk_%d", s.nextID)
	s.nextID++

	var req struct {
		WorkflowExecution struct {
			PhoneNumber string                 `json:"phone_number"`
			Variables   map[string]interface{} `json:"variables"`
		} `json:"workflow_execution"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	s.executions[execID] = &executionState{
		ID:         execID,
		WorkflowID: workflowID,
		Status:     "waiting",
		Phone:      req.WorkflowExecution.PhoneNumber,
		Variables:  req.WorkflowExecution.Variables,
		Messages:   make([]string, 0),
		StartTime:  time.Now(),
		Events: []kapso.Event{
			{
				EventType: "execution_started",
				CreatedAt: time.Now(),
				Payload:   map[string]interface{}{"workflow_id": workflowID},
			},
			{
				EventType: "step_entered",
				CreatedAt: time.Now(),
				Step:      map[string]interface{}{"identifier": "start"},
			},
			{
				EventType: "step_entered",
				CreatedAt: time.Now().Add(time.Millisecond),
				Step:      map[string]interface{}{"identifier": "intro"},
			},
			{
				EventType: "step_entered",
				CreatedAt: time.Now().Add(2 * time.Millisecond),
				Step:      map[string]interface{}{"identifier": "wait_reply"},
			},
		},
	}

	s.json(w, http.StatusAccepted, map[string]interface{}{
		"data": kapso.ExecutionResponse{
			TrackingID:  trackingID,
			ExecutionID: execID,
		},
	})
}

func (s *Server) handleGetExecution(w http.ResponseWriter, r *http.Request, execID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	exec, ok := s.executions[execID]
	if !ok {
		http.Error(w, `{"error":"execution not found"}`, http.StatusNotFound)
		return
	}

	s.json(w, http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"id":           execID,
			"status":       exec.Status,
			"current_step": fmt.Sprintf("step_%d", len(exec.Messages)+1),
			"execution_context": map[string]interface{}{
				"vars": exec.Variables,
				"system": map[string]interface{}{
					"flow_id":      execID,
					"started_at":   exec.StartTime.Format(time.RFC3339),
					"status":       exec.Status,
					"phone_number": exec.Phone,
				},
				"context": map[string]interface{}{
					"messages_count": len(exec.Messages),
				},
				"metadata": map[string]interface{}{},
			},
			"events": exec.Events,
		},
	})
}

func (s *Server) handleResumeExecution(w http.ResponseWriter, r *http.Request, execID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	exec, ok := s.executions[execID]
	if !ok {
		http.Error(w, `{"error":"execution not found"}`, http.StatusNotFound)
		return
	}

	var req struct {
		Message struct {
			Kind string `json:"kind"`
			Data string `json:"data"`
		} `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	exec.Messages = append(exec.Messages, req.Message.Data)

	exec.Events = append(exec.Events, kapso.Event{
		EventType: "user_input_received",
		CreatedAt: time.Now(),
		Step:      map[string]interface{}{"identifier": fmt.Sprintf("step_%d", len(exec.Messages))},
		Payload:   map[string]interface{}{"content": req.Message.Data},
	})

	// Simulate entering the classify node
	exec.Events = append(exec.Events, kapso.Event{
		EventType: "step_entered",
		CreatedAt: time.Now(),
		Step:      map[string]interface{}{"identifier": "classify"},
	})

	if strings.Contains(strings.ToLower(req.Message.Data), "factura") ||
		strings.Contains(strings.ToLower(req.Message.Data), "pago") ||
		strings.Contains(strings.ToLower(req.Message.Data), "cobro") {
		exec.Events = append(exec.Events, kapso.Event{
			EventType: "decision_evaluated",
			CreatedAt: time.Now(),
			Step:      map[string]interface{}{"identifier": "classify"},
			EdgeLabel: "billing",
			Payload:   map[string]interface{}{"reasoning": "User mentioned billing-related terms"},
		})
		if exec.Variables == nil {
			exec.Variables = make(map[string]interface{})
		}
		exec.Variables["classify_result"] = "billing"
	} else if strings.Contains(strings.ToLower(req.Message.Data), "técnic") ||
		strings.Contains(strings.ToLower(req.Message.Data), "error") ||
		strings.Contains(strings.ToLower(req.Message.Data), "problema") {
		exec.Events = append(exec.Events, kapso.Event{
			EventType: "decision_evaluated",
			CreatedAt: time.Now(),
			Step:      map[string]interface{}{"identifier": "classify"},
			EdgeLabel: "technical",
			Payload:   map[string]interface{}{"reasoning": "User mentioned technical issue terms"},
		})
		if exec.Variables == nil {
			exec.Variables = make(map[string]interface{})
		}
		exec.Variables["classify_result"] = "technical"
	} else {
		exec.Events = append(exec.Events, kapso.Event{
			EventType: "decision_evaluated",
			CreatedAt: time.Now(),
			Step:      map[string]interface{}{"identifier": "classify"},
			EdgeLabel: "other",
			Payload:   map[string]interface{}{"reasoning": "No specific keywords detected"},
		})
		if exec.Variables == nil {
			exec.Variables = make(map[string]interface{})
		}
		exec.Variables["classify_result"] = "other"
	}

	exec.Events = append(exec.Events, kapso.Event{
		EventType: "step_entered",
		CreatedAt: time.Now(),
		Step:      map[string]interface{}{"identifier": "handoff"},
	})

	exec.Events = append(exec.Events, kapso.Event{
		EventType: "execution_ended",
		CreatedAt: time.Now(),
	})
	exec.Status = "ended"

	s.json(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleUpdateStatus(w http.ResponseWriter, r *http.Request, execID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	exec, ok := s.executions[execID]
	if !ok {
		http.Error(w, `{"error":"execution not found"}`, http.StatusNotFound)
		return
	}

	var req struct {
		WorkflowExecution struct {
			Status string `json:"status"`
		} `json:"workflow_execution"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	exec.Status = req.WorkflowExecution.Status

	if req.WorkflowExecution.Status == "ended" {
		exec.Events = append(exec.Events, kapso.Event{
			EventType: "execution_ended",
			CreatedAt: time.Now(),
		})
	}

	s.json(w, http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"status": req.WorkflowExecution.Status,
		},
	})
}

func (s *Server) handleGetEvents(w http.ResponseWriter, r *http.Request, execID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	exec, ok := s.executions[execID]
	if !ok {
		http.Error(w, `{"error":"execution not found"}`, http.StatusNotFound)
		return
	}

	s.json(w, http.StatusOK, map[string]interface{}{"data": exec.Events})
}
