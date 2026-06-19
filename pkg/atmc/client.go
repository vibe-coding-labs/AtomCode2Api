package atmc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is the HTTP client for the AtomCode daemon REST API.
type Client struct {
	BaseURL    string
	httpClient *http.Client
}

// NewClient creates a new daemon client.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// SetTimeout overrides the default HTTP client timeout.
func (c *Client) SetTimeout(d time.Duration) {
	c.httpClient.Timeout = d
}

// ─── Sync helpers ──────────────────────────────────────────────────────────────

func (c *Client) syncRequest(method, path string, body any) (int, []byte) {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	u, _ := url.JoinPath(c.BaseURL, path)
	req, err := http.NewRequest(method, u, bytes.NewReader(reqBody))
	if err != nil {
		return 500, []byte(fmt.Sprintf(`{"error":"build request: %s"}`, err))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 503, []byte(fmt.Sprintf(`{"error":"daemon unreachable: %s"}`, err))
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}

func (c *Client) syncGet(path string) (int, []byte) {
	return c.syncRequest("GET", path, nil)
}

func (c *Client) syncPost(path string, body any) (int, []byte) {
	return c.syncRequest("POST", path, body)
}

// ─── Auth ──────────────────────────────────────────────────────────────────────

func (c *Client) Health() (*HealthResponse, error) {
	code, data := c.syncGet("/health")
	if code != 200 {
		return nil, fmt.Errorf("health check failed (HTTP %d): %s", code, string(data))
	}
	var h HealthResponse
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("parse health response: %w", err)
	}
	return &h, nil
}

func (c *Client) AuthStatus() (*AuthStatusResponse, error) {
	code, data := c.syncGet("/auth/status")
	if code != 200 {
		return nil, fmt.Errorf("auth status failed (HTTP %d): %s", code, string(data))
	}
	var a AuthStatusResponse
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("parse auth status: %w", err)
	}
	return &a, nil
}

func (c *Client) LoginStart() (*LoginStartResponse, error) {
	_, data := c.syncPost("/auth/login/start", LoginStartRequest{OpenBrowser: false})
	var l LoginStartResponse
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, fmt.Errorf("parse login start: %w", err)
	}
	return &l, nil
}

func (c *Client) LoginPoll(loginID string) (*LoginPollResponse, error) {
	_, data := c.syncPost(fmt.Sprintf("/auth/login/%s/poll", loginID), map[string]any{})
	var p LoginPollResponse
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse login poll: %w", err)
	}
	return &p, nil
}

// ─── CodingPlan ───────────────────────────────────────────────────────────────

func (c *Client) CodingPlanSetup() (*CodingPlanSetupResponse, error) {
	_, data := c.syncPost("/codingplan/setup", map[string]any{})
	var s CodingPlanSetupResponse
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse codingplan setup: %w", err)
	}
	return &s, nil
}

// ─── Models / Providers ──────────────────────────────────────────────────────

func (c *Client) ListModels() ([]ModelInfo, error) {
	_, data := c.syncGet("/models")
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse models: %w", err)
	}
	models := make([]ModelInfo, 0, len(raw))
	for _, r := range raw {
		var m ModelInfo
		if err := json.Unmarshal(r, &m); err != nil {
			continue
		}
		if m.ID == "" {
			m.ID = m.Model
		}
		if m.ID == "" {
			continue
		}
		models = append(models, m)
	}
	return models, nil
}

func (c *Client) ListProviders() ([]ProviderConfig, error) {
	_, data := c.syncGet("/providers")
	// Providers can return as {"providers": [...]} or as a flat array
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err == nil {
		if provJSON, ok := rawMap["providers"]; ok {
			var providers []ProviderConfig
			if err := json.Unmarshal(provJSON, &providers); err == nil {
				return providers, nil
			}
		}
	}
	// Try flat array
	var providers []ProviderConfig
	if err := json.Unmarshal(data, &providers); err != nil {
		return nil, fmt.Errorf("parse providers: %w", err)
	}
	return providers, nil
}

// ─── Chat (SSE stream) ───────────────────────────────────────────────────────

// ChatStream sends a chat request and returns a channel of SSE events.
// The caller must read from the channel until it closes.
func (c *Client) ChatStream(req *ChatRequest) (<-chan SSEEvent, error) {
	body, _ := json.Marshal(req)
	u, _ := url.JoinPath(c.BaseURL, "/chat")

	httpReq, err := http.NewRequest("POST", u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build chat request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("chat request failed: %w", err)
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("chat returned HTTP %d", resp.StatusCode)
	}

	ch := make(chan SSEEvent, 64)
	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 256*1024), 256*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			dataStr := line[6:]
			if dataStr == "[DONE]" {
				ch <- SSEEvent{Type: "done"}
				return
			}
			var ev SSEEvent
			if err := json.Unmarshal([]byte(dataStr), &ev); err != nil {
				log.Printf("atmc: failed to parse SSE event: %v (line: %s)", err, line)
				continue
			}
			ch <- ev
		}
		if err := scanner.Err(); err != nil {
			log.Printf("atmc: SSE scanner error: %v", err)
		}
	}()

	return ch, nil
}
