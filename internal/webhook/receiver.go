package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"time"
)

type Event struct {
	EventType string                 `json:"eventType"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}

type Receiver struct {
	port       int
	server     *http.Server
	events     []Event
	eventCh    chan Event
	kapsoAPI   string
	apiKey     string
	tunnelURL  string
	webhookID  string
	captureDir string
}

type WebhookConfig struct {
	Port       int
	KapsoAPI   string
	APIKey     string
	UseTunnel  bool
	CaptureDir string
}

func NewReceiver(cfg WebhookConfig) *Receiver {
	return &Receiver{
		port:       cfg.Port,
		kapsoAPI:   cfg.KapsoAPI,
		apiKey:     cfg.APIKey,
		eventCh:    make(chan Event, 100),
		captureDir: cfg.CaptureDir,
	}
}

func (r *Receiver) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", r.handleWebhook)
	mux.HandleFunc("/health", r.handleHealth)

	addr := fmt.Sprintf(":%d", r.port)
	r.server = &http.Server{Addr: addr, Handler: mux}

	if r.captureDir != "" || true {
		slog.Info("Webhook receiver ready", "addr", addr)
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		r.server.Shutdown(shutdownCtx)
		r.unregisterWebhook()
	}()

	return r.server.ListenAndServe()
}

func (r *Receiver) handleWebhook(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "parse error", http.StatusBadRequest)
		return
	}

	evt := Event{
		EventType: guessEventType(payload),
		Timestamp: time.Now(),
		Payload:   payload,
	}

	r.events = append(r.events, evt)
	select {
	case r.eventCh <- evt:
	default:
	}

	slog.Debug("webhook received", "type", evt.EventType)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (r *Receiver) handleHealth(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func (r *Receiver) GetEvents() []Event {
	return r.events
}

func (r *Receiver) EventChannel() <-chan Event {
	return r.eventCh
}

func (r *Receiver) registerWebhook(url string) error {
	body := map[string]interface{}{
		"whatsapp_webhook": map[string]interface{}{
			"url":    url,
			"kind":   "kapso",
			"active": true,
			"events": []string{
				"workflow.execution.started",
				"workflow.execution.completed",
				"workflow.execution.failed",
				"workflow.execution.handoff",
			},
		},
	}

	data, _ := json.Marshal(body)
	resp, err := http.Post(
		r.kapsoAPI+"/whatsapp/webhooks",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return fmt.Errorf("failed to register webhook: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode webhook response: %w", err)
	}

	r.webhookID = result.Data.ID
	slog.Info("webhook registered", "id", r.webhookID, "url", url)
	return nil
}

func (r *Receiver) unregisterWebhook() {
	if r.webhookID == "" {
		return
	}

	req, _ := http.NewRequest("DELETE",
		r.kapsoAPI+"/whatsapp/webhooks/"+r.webhookID, nil)
	req.Header.Set("X-API-Key", r.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Warn("failed to unregister webhook", "error", err)
		return
	}
	resp.Body.Close()
	slog.Info("webhook unregistered", "id", r.webhookID)
}

func guessEventType(payload map[string]interface{}) string {
	if et, ok := payload["eventType"].(string); ok {
		return et
	}
	if status, ok := payload["status"].(string); ok {
		return "workflow.execution." + status
	}
	return "unknown"
}

func StartTunnel(port int) (string, error) {
	cmd := exec.Command("ngrok", "http", fmt.Sprintf("%d", port), "--log=stdout", "--log-level=info")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start ngrok: %w", err)
	}

	buf := make([]byte, 4096)
	n, _ := stdout.Read(buf)
	output := string(buf[:n])

	slog.Info("ngrok started", "output", output)

	time.Sleep(2 * time.Second)

	resp, err := http.Get("http://localhost:4040/api/tunnels")
	if err != nil {
		return "", fmt.Errorf("ngrok API not accessible: %w", err)
	}
	defer resp.Body.Close()

	var tunnels struct {
		Tunnels []struct {
			PublicURL string `json:"public_url"`
		} `json:"tunnels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tunnels); err != nil {
		return "", fmt.Errorf("failed to parse ngrok tunnels: %w", err)
	}

	if len(tunnels.Tunnels) > 0 {
		return tunnels.Tunnels[0].PublicURL, nil
	}

	return "", fmt.Errorf("no ngrok tunnels found")
}
