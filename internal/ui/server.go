package ui

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/AlvaroMrJack/k-flow-spec/internal/config"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/runner"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/spec"
)

type Server struct {
	port     int
	cfg      *config.KfsConfig
	results  []*runner.Result
	mockMode bool
	mockAddr string
}

func NewServer(port int, cfg *config.KfsConfig) *Server {
	return &Server{
		port: port,
		cfg:  cfg,
	}
}

func (s *Server) SetResults(results []*runner.Result) {
	s.results = results
}

func (s *Server) SetMockMode(addr string) {
	s.mockMode = true
	s.mockAddr = addr
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/results", s.handleResults)
	mux.HandleFunc("/api/results/", s.handleResultByIndex)
	mux.HandleFunc("/api/specs", s.handleSpecs)
	mux.HandleFunc("/api/config", s.handleConfig)

	staticFS, err := fs.Sub(Assets, ".")
	if err != nil {
		return fmt.Errorf("failed to create sub filesystem: %w", err)
	}
	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("/", fileServer)

	addr := fmt.Sprintf(":%d", s.port)
	slog.Info("UI dashboard starting", "addr", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleResults(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.results == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
		return
	}

	type resultResponse struct {
		Name       string            `json:"name"`
		WorkflowID string            `json:"workflow_id"`
		Passed     bool              `json:"passed"`
		DurationMs int64             `json:"duration_ms"`
		Errors     []runner.RunError `json:"errors,omitempty"`
		Timestamp  string            `json:"timestamp"`
	}

	results := make([]resultResponse, 0, len(s.results))
	for _, res := range s.results {
		results = append(results, resultResponse{
			Name:       res.SpecName,
			WorkflowID: res.WorkflowID,
			Passed:     res.Passed,
			DurationMs: res.Duration.Milliseconds(),
			Errors:     res.Errors,
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": results,
	})
}

func (s *Server) handleResultByIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "not implemented"})
}

func (s *Server) handleSpecs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.cfg == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
		return
	}

	specsDir := s.cfg.SpecsDir
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
		return
	}

	type specInfo struct {
		Name     string `json:"name"`
		File     string `json:"file"`
		Workflow string `json:"workflow"`
	}

	var specs []specInfo
	for _, entry := range entries {
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		s, err := spec.Load(filepath.Join(specsDir, entry.Name()))
		if err != nil {
			continue
		}
		specs = append(specs, specInfo{
			Name:     s.Name,
			File:     entry.Name(),
			Workflow: s.Workflow,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"data": specs})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.cfg == nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "no config"})
		return
	}
	json.NewEncoder(w).Encode(s.cfg)
}
