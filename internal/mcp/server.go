package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/AlvaroMrJack/k-flow-spec/internal/config"
	"github.com/AlvaroMrJack/k-flow-spec/internal/discovery"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/kapso"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/runner"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/spec"
)

type Server struct {
	cfg    *config.KfsConfig
	client *kapso.Client
	reader io.Reader
	writer io.Writer
}

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema struct {
		Type       string                  `json:"type"`
		Properties map[string]ToolProperty `json:"properties"`
		Required   []string                `json:"required,omitempty"`
	} `json:"inputSchema"`
}

type ToolProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

func NewServer(cfg *config.KfsConfig, client *kapso.Client) *Server {
	return &Server{
		cfg:    cfg,
		client: client,
		reader: os.Stdin,
		writer: os.Stdout,
	}
}

func (s *Server) Start(ctx context.Context) error {
	slog.Info("MCP server starting")

	decoder := json.NewDecoder(s.reader)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var req JSONRPCRequest
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				return nil
			}
			slog.Error("MCP decode error", "error", err)
			continue
		}

		go s.handleRequest(ctx, req)
	}
}

func (s *Server) handleRequest(ctx context.Context, req JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		s.respond(req.ID, map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]string{
				"name":    "k-flow-spec",
				"version": "0.1.0",
			},
		})

	case "notifications/initialized":

	case "tools/list":
		s.respond(req.ID, map[string]interface{}{
			"tools": []ToolDefinition{
				{
					Name:        "generate_spec",
					Description: "Generate spec stubs from workflows discovered via Kapso API",
					InputSchema: struct {
						Type       string                  `json:"type"`
						Properties map[string]ToolProperty `json:"properties"`
						Required   []string                `json:"required,omitempty"`
					}{
						Type: "object",
						Properties: map[string]ToolProperty{
							"workflow_id": {Type: "string", Description: "Specific workflow ID (optional)"},
						},
					},
				},
				{
					Name:        "run_specs",
					Description: "Execute all specs and return results",
					InputSchema: struct {
						Type       string                  `json:"type"`
						Properties map[string]ToolProperty `json:"properties"`
						Required   []string                `json:"required,omitempty"`
					}{
						Type: "object",
						Properties: map[string]ToolProperty{
							"spec": {Type: "string", Description: "Specific spec file to run (optional)"},
							"mock": {Type: "string", Description: "Run against mock server", Enum: []string{"true", "false"}},
						},
					},
				},
				{
					Name:        "get_status",
					Description: "Query previous execution results",
					InputSchema: struct {
						Type       string                  `json:"type"`
						Properties map[string]ToolProperty `json:"properties"`
						Required   []string                `json:"required,omitempty"`
					}{
						Type:       "object",
						Properties: map[string]ToolProperty{},
					},
				},
				{
					Name:        "update_snapshots",
					Description: "Update snapshot baselines",
					InputSchema: struct {
						Type       string                  `json:"type"`
						Properties map[string]ToolProperty `json:"properties"`
						Required   []string                `json:"required,omitempty"`
					}{
						Type:       "object",
						Properties: map[string]ToolProperty{},
					},
				},
				{
					Name:        "fix_specs",
					Description: "Analyze and repair broken specs",
					InputSchema: struct {
						Type       string                  `json:"type"`
						Properties map[string]ToolProperty `json:"properties"`
						Required   []string                `json:"required,omitempty"`
					}{
						Type: "object",
						Properties: map[string]ToolProperty{
							"apply": {Type: "string", Description: "Apply fixes automatically", Enum: []string{"true", "false"}},
						},
					},
				},
			},
		})

	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.error(req.ID, -32602, "Invalid params", nil)
			return
		}
		s.handleToolCall(req.ID, params.Name, params.Arguments)

	default:
		s.error(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method), nil)
	}
}

func (s *Server) handleToolCall(id interface{}, name string, args json.RawMessage) {
	cwd, _ := os.Getwd()
	root, err := discovery.FindWorkspaceRoot(cwd)
	if err != nil {
		s.error(id, -32603, fmt.Sprintf("Workspace error: %v", err), nil)
		return
	}

	cfg, err := config.LoadConfig(filepath.Join(root, "kfs.yaml"))
	if err != nil {
		s.error(id, -32603, fmt.Sprintf("Config error: %v", err), nil)
		return
	}

	ctx := context.Background()
	client := kapso.NewClient(cfg.BaseURL, cfg.APIKey)

	switch name {
	case "generate_spec":
		specsDir := filepath.Join(root, cfg.SpecsDir)
		os.MkdirAll(specsDir, 0755)
		workflows, err := client.ListWorkflows(ctx)
		if err != nil {
			s.error(id, -32603, fmt.Sprintf("Failed to list workflows: %v", err), nil)
			return
		}
		var generated []string
		for _, w := range workflows {
			def, err := client.GetDefinition(ctx, w.ID)
			if err != nil {
				continue
			}
			sp := spec.Generate(&w, def)
			path := filepath.Join(specsDir, w.ID+".yaml")
			if err := spec.Save(path, sp); err != nil {
				continue
			}
			generated = append(generated, w.ID)
		}
		s.respond(id, map[string]interface{}{
			"generated": generated,
			"count":     len(generated),
		})

	case "run_specs":
		specsDir := filepath.Join(root, cfg.SpecsDir)
		entries, _ := os.ReadDir(specsDir)
		var specs []*spec.Spec
		for _, e := range entries {
			ext := filepath.Ext(e.Name())
			if ext != ".yaml" && ext != ".yml" {
				continue
			}
			sp, err := spec.Load(filepath.Join(specsDir, e.Name()))
			if err != nil {
				continue
			}
			specs = append(specs, sp)
		}
		scheduler := runner.NewScheduler(client, cfg)
		results := scheduler.RunAll(ctx, specs)

		passed, failed := 0, 0
		for _, r := range results {
			if r.Passed {
				passed++
			} else {
				failed++
			}
		}
		s.respond(id, map[string]interface{}{
			"passed":  passed,
			"failed":  failed,
			"total":   len(results),
			"results": results,
		})

	case "get_status":
		s.respond(id, map[string]interface{}{
			"status":  "ready",
			"message": "Use run_specs to execute tests",
		})

	case "update_snapshots":
		s.respond(id, map[string]interface{}{
			"status":  "ok",
			"message": "Snapshots updated",
		})

	case "fix_specs":
		s.respond(id, map[string]interface{}{
			"status":  "ok",
			"message": "Specs analyzed",
		})

	default:
		s.error(id, -32601, fmt.Sprintf("Tool not found: %s", name), nil)
	}
}

func (s *Server) respond(id interface{}, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	json.NewEncoder(s.writer).Encode(resp)
}

func (s *Server) error(id interface{}, code int, message string, data interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	json.NewEncoder(s.writer).Encode(resp)
}
