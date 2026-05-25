package broadcast

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Recipient struct {
	Phone     string            `json:"phone"`
	Variables map[string]string `json:"variables,omitempty"`
}

type BroadcastSpec struct {
	Name          string      `json:"name"`
	TemplateID    string      `json:"template_id"`
	PhoneNumberID string      `json:"phone_number_id"`
	Recipients    []Recipient `json:"recipients"`
}

type BroadcastResult struct {
	BroadcastID  string   `json:"broadcast_id"`
	Status       string   `json:"status"`
	SentCount    int      `json:"sent_count"`
	FailedCount  int      `json:"failed_count"`
	TotalCount   int      `json:"total_count"`
	DeliveryRate float64  `json:"delivery_rate"`
	Passed       bool     `json:"passed"`
	Errors       []string `json:"errors,omitempty"`
}

type Tester struct {
	baseURL string
	apiKey  string
	client  *http.Client
	dryRun  bool
}

func NewTester(baseURL, apiKey string, dryRun bool) *Tester {
	return &Tester{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 120 * time.Second},
		dryRun:  dryRun,
	}
}

func (t *Tester) Run(ctx context.Context, spec *BroadcastSpec) *BroadcastResult {
	result := &BroadcastResult{
		Passed: true,
	}

	if t.dryRun {
		result.Status = "dry_run"
		result.TotalCount = len(spec.Recipients)
		result.Passed = len(spec.Recipients) > 0

		if len(spec.Recipients) == 0 {
			result.Errors = append(result.Errors, "broadcast debe tener al menos 1 recipient")
			result.Passed = false
		}

		for _, r := range spec.Recipients {
			if r.Phone == "" {
				result.Errors = append(result.Errors, "recipient sin phone_number")
				result.Passed = false
			}
		}

		return result
	}

	// Step 1: Create broadcast
	broadcastID, err := t.createBroadcast(ctx, spec)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("create: %v", err))
		return result
	}
	result.BroadcastID = broadcastID

	// Step 2: Add recipients (batches of 1000)
	for _, r := range spec.Recipients {
		if err := t.addRecipient(ctx, broadcastID, r); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("recipient %s: %v", r.Phone, err))
		}
	}

	// Step 3: Send
	if err := t.sendBroadcast(ctx, broadcastID); err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("send: %v", err))
		return result
	}

	// Step 4: Poll for completion
	status, err := t.pollUntilComplete(ctx, broadcastID, 5*time.Minute)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("poll: %v", err))
		return result
	}

	result.Status = status.Status
	result.SentCount = status.SentCount
	result.FailedCount = status.FailedCount
	result.TotalCount = status.TotalCount
	result.DeliveryRate = status.DeliveryRate

	if result.Status != "completed" {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("broadcast terminó en estado %s", result.Status))
	}

	return result
}

func (t *Tester) do(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, t.baseURL+path, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", t.apiKey)

	return t.client.Do(req)
}

func (t *Tester) createBroadcast(ctx context.Context, spec *BroadcastSpec) (string, error) {
	body := map[string]interface{}{
		"whatsapp_broadcast": map[string]interface{}{
			"name":                spec.Name,
			"phone_number_id":     spec.PhoneNumberID,
			"whatsapp_template_id": spec.TemplateID,
		},
	}

	resp, err := t.do(ctx, "POST", "/whatsapp/broadcasts", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var out struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}

	return out.Data.ID, nil
}

func (t *Tester) addRecipient(ctx context.Context, broadcastID string, r Recipient) error {
	components := make([]map[string]interface{}, 0)
	if len(r.Variables) > 0 {
		params := make([]map[string]interface{}, 0)
		for name, value := range r.Variables {
			params = append(params, map[string]interface{}{
				"type":           "text",
				"parameter_name": name,
				"text":           value,
			})
		}
		components = append(components, map[string]interface{}{
			"type":       "body",
			"parameters": params,
		})
	}

	body := map[string]interface{}{
		"whatsapp_broadcast": map[string]interface{}{
			"recipients": []map[string]interface{}{
				{
					"phone_number": r.Phone,
					"components":   components,
				},
			},
		},
	}

	resp, err := t.do(ctx, "POST", fmt.Sprintf("/whatsapp/broadcasts/%s/recipients", broadcastID), body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (t *Tester) sendBroadcast(ctx context.Context, broadcastID string) error {
	resp, err := t.do(ctx, "POST", fmt.Sprintf("/whatsapp/broadcasts/%s/send", broadcastID), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (t *Tester) pollUntilComplete(ctx context.Context, broadcastID string, timeout time.Duration) (*BroadcastResult, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		resp, err := t.do(ctx, "GET", fmt.Sprintf("/whatsapp/broadcasts/%s", broadcastID), nil)
		if err != nil {
			return nil, err
		}

		var out struct {
			Data struct {
				Status          string  `json:"status"`
				SentCount       int     `json:"sent_count"`
				FailedCount     int     `json:"failed_count"`
				TotalRecipients int     `json:"total_recipients"`
				CompletedAt     *string `json:"completed_at"`
			} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		if out.Data.CompletedAt != nil || out.Data.Status == "completed" || out.Data.Status == "failed" {
			rate := 0.0
			if out.Data.TotalRecipients > 0 {
				rate = float64(out.Data.SentCount) / float64(out.Data.TotalRecipients)
			}
			return &BroadcastResult{
				Status:      out.Data.Status,
				SentCount:   out.Data.SentCount,
				FailedCount: out.Data.FailedCount,
				TotalCount:  out.Data.TotalRecipients,
				DeliveryRate: rate,
			}, nil
		}

		time.Sleep(5 * time.Second)
	}

	return nil, fmt.Errorf("timeout esperando broadcast %s", broadcastID)
}
