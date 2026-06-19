package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vibe-coding-labs/AtomCodeProxy/pkg/atmc"
	"github.com/vibe-coding-labs/AtomCodeProxy/pkg/openai"
)

// mockDaemon creates a test server that mimics the AtomCode daemon.
func mockDaemon() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			json.NewEncoder(w).Encode(map[string]any{
				"status": "ok", "version": "v4.25.0", "service": "atomcode-daemon",
			})

		case "/auth/status":
			json.NewEncoder(w).Encode(map[string]any{
				"logged_in": true,
				"user":      map[string]any{"name": "TestUser", "username": "test"},
				"token":     map[string]any{"expires_in": 604800, "has_refresh_token": true},
			})

		case "/models":
			json.NewEncoder(w).Encode([]map[string]any{
				{"id": "deepseek-chat", "model": "deepseek-chat", "provider": "deepseek"},
				{"id": "deepseek-v4-flash", "model": "deepseek-v4-flash", "provider": "deepseek"},
				{"id": "gpt-4", "model": "gpt-4", "provider": "openai"},
			})

		case "/providers":
			json.NewEncoder(w).Encode(map[string]any{
				"providers": []map[string]any{
					{"name": "deepseek", "model": "deepseek-v4-flash", "provider_type": "openai", "is_default": true},
					{"name": "openai", "model": "gpt-4", "provider_type": "openai", "is_default": false},
				},
			})

		case "/chat":
			flusher := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(200)

			var reqBody map[string]any
			json.NewDecoder(r.Body).Decode(&reqBody)
			_ = reqBody

			events := []string{
				`data: {"type":"text","content":"Hello! I'm a mock daemon. "}`,
				`data: {"type":"text","content":"How can I help you?"}`,
				`data: {"type":"tokens","prompt":15,"completion":12,"total":27}`,
				`data: {"type":"done","session_id":"mock-sess-123"}`,
			}
			for _, e := range events {
				fmt.Fprintf(w, "%s\n\n", e)
				flusher.Flush()
			}

		default:
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	}))
}

func TestIntegration_HealthCheck(t *testing.T) {
	daemon := mockDaemon()
	defer daemon.Close()

	client := atmc.NewClient(daemon.URL)
	h, err := client.Health()
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	if h.Status != "ok" {
		t.Errorf("expected ok, got %s", h.Status)
	}
	if h.Version != "v4.25.0" {
		t.Errorf("expected v4.25.0, got %s", h.Version)
	}
}

func TestIntegration_AuthStatus(t *testing.T) {
	daemon := mockDaemon()
	defer daemon.Close()

	client := atmc.NewClient(daemon.URL)
	auth, err := client.AuthStatus()
	if err != nil {
		t.Fatalf("auth status failed: %v", err)
	}
	if !auth.LoggedIn {
		t.Errorf("expected logged in")
	}
	if auth.User == nil || auth.User.Name != "TestUser" {
		t.Errorf("expected TestUser, got %v", auth.User)
	}
}

func TestIntegration_ListModels(t *testing.T) {
	daemon := mockDaemon()
	defer daemon.Close()

	client := atmc.NewClient(daemon.URL)
	models, err := client.ListModels()
	if err != nil {
		t.Fatalf("list models failed: %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("expected 3 models, got %d", len(models))
	}
	if models[1].ID != "deepseek-v4-flash" {
		t.Errorf("expected deepseek-v4-flash, got %s", models[1].ID)
	}
}

func TestIntegration_FindProvider(t *testing.T) {
	daemon := mockDaemon()
	defer daemon.Close()

	client := atmc.NewClient(daemon.URL)
	providers, err := client.ListProviders()
	if err != nil {
		t.Fatalf("list providers failed: %v", err)
	}
	provider := atmc.FindProviderForModel(providers, "deepseek-v4-flash")
	if provider != "deepseek" {
		t.Errorf("expected deepseek, got %s", provider)
	}
}

func TestIntegration_OpenAIChatNonStream(t *testing.T) {
	daemon := mockDaemon()
	defer daemon.Close()

	client := atmc.NewClient(daemon.URL)
	srv := openai.NewServer(client, nil)

	reqBody := `{
		"model": "deepseek-v4-flash",
		"messages": [{"role": "user", "content": "hello"}],
		"stream": false
	}`

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	srvServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
	}))
	defer srvServer.Close()

	resp, err := http.Post(srvServer.URL+"/v1/chat/completions", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("chat request failed: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if result["object"] != "chat.completion" {
		t.Errorf("expected chat.completion, got %s", result["object"])
	}
	choices, ok := result["choices"].([]any)
	if !ok || len(choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(choices))
	}
	choice := choices[0].(map[string]any)
	msg := choice["message"].(map[string]any)
	content := msg["content"].(string)
	if !strings.Contains(content, "Hello!") {
		t.Errorf("expected Hello in response, got %s", content)
	}
	if result["usage"] == nil {
		t.Errorf("expected usage in response")
	}
}

func TestIntegration_OpenAIChatStream(t *testing.T) {
	daemon := mockDaemon()
	defer daemon.Close()

	client := atmc.NewClient(daemon.URL)
	srv := openai.NewServer(client, nil)

	reqBody := `{
		"model": "deepseek-v4-flash",
		"messages": [{"role": "user", "content": "hello"}],
		"stream": true
	}`

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	srvServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
	}))
	defer srvServer.Close()

	resp, err := http.Post(srvServer.URL+"/v1/chat/completions", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("stream request failed: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %s", ct)
	}

	scanner := bufio.NewScanner(resp.Body)
	chunks := 0
	gotDone := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "data: ") {
			data := line[6:]
			if data == "[DONE]" {
				gotDone = true
				break
			}
			var chunk map[string]any
			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				if chunk["object"] == "chat.completion.chunk" {
					chunks++
				}
			}
		}
	}
	if chunks == 0 {
		t.Errorf("expected at least 1 chunk, got %d", chunks)
	}
	if !gotDone {
		t.Errorf("expected [DONE] signal")
	}
}

func TestIntegration_ModelListEndpoint(t *testing.T) {
	daemon := mockDaemon()
	defer daemon.Close()

	client := atmc.NewClient(daemon.URL)
	srv := openai.NewServer(client, nil)

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	srvServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
	}))
	defer srvServer.Close()

	resp, err := http.Get(srvServer.URL + "/v1/models")
	if err != nil {
		t.Fatalf("models request failed: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if result["object"] != "list" {
		t.Errorf("expected list, got %s", result["object"])
	}
	data, ok := result["data"].([]any)
	if !ok || len(data) != 3 {
		t.Errorf("expected 3 models, got %d", len(data))
	}
}