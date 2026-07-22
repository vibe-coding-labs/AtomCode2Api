package openai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vibe-coding-labs/AtomCode2API/pkg/atmc"
)

func mockDaemon() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			json.NewEncoder(w).Encode(map[string]any{"status": "ok", "version": "v4.25.0"})
		case "/auth/status":
			json.NewEncoder(w).Encode(map[string]any{"logged_in": true, "user": map[string]any{"name": "T"}})
		case "/models":
			json.NewEncoder(w).Encode([]map[string]any{
				{"id": "deepseek-chat", "model": "deepseek-chat"},
			})
		case "/providers":
			json.NewEncoder(w).Encode(map[string]any{
				"providers": []map[string]any{
					{"name": "deepseek", "model": "deepseek-v4-flash"},
				},
			})
		case "/chat":
			flusher := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(200)
			for _, e := range []string{
				`data: {"type":"text","content":"Hello"}`,
				`data: {"type":"tokens","prompt":1,"completion":2,"total":3}`,
				`data: {"type":"done"}`,
			} {
				fmt.Fprintf(w, "%s\n\n", e)
				flusher.Flush()
			}
		}
	}))
}

func TestHandleHealth(t *testing.T) {
	d := mockDaemon()
	defer d.Close()
	srv := NewServer(atmc.NewClient(d.URL), nil)
	r := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	srv.handleHealth(w, r)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleModels(t *testing.T) {
	d := mockDaemon()
	defer d.Close()
	srv := NewServer(atmc.NewClient(d.URL), nil)
	r := httptest.NewRequest("GET", "/v1/models", nil)
	w := httptest.NewRecorder()
	srv.handleModels(w, r)

	var resp ModelsListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 model, got %d", len(resp.Data))
	}
}

func TestHandleChatNonStream(t *testing.T) {
	d := mockDaemon()
	defer d.Close()
	srv := NewServer(atmc.NewClient(d.URL), nil)

	body := `{"model":"deepseek-v4-flash","messages":[{"role":"user","content":"hi"}],"stream":false}`
	r := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleChat(w, r)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp ChatCompletionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Choices) != 1 {
		t.Errorf("expected 1 choice, got %d", len(resp.Choices))
	}
}

func TestHandleChatError(t *testing.T) {
	d := mockDaemon()
	defer d.Close()
	srv := NewServer(atmc.NewClient(d.URL), nil)

	body := `{"messages":[]}`
	r := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleChat(w, r)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleChatOptions(t *testing.T) {
	d := mockDaemon()
	defer d.Close()
	srv := NewServer(atmc.NewClient(d.URL), nil)
	r := httptest.NewRequest("OPTIONS", "/v1/chat/completions", nil)
	w := httptest.NewRecorder()
	srv.handleChat(w, r)
	if w.Code != 200 {
		t.Errorf("expected 200 for OPTIONS, got %d", w.Code)
	}
}
