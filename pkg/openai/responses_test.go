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

// mockDaemonFull responds to /chat with a realistic event sequence: reasoning,
// text, a tool call, usage tokens, done with a session id.
func mockDaemonFull(t *testing.T, wantMsg string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/providers":
			json.NewEncoder(w).Encode(map[string]any{"providers": []map[string]any{}})
		case "/chat":
			var got atmc.ChatRequest
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Errorf("daemon decode: %v", err)
			}
			if wantMsg != "" && !strings.Contains(got.Message, wantMsg) {
				t.Errorf("daemon got message %q, want substring %q", got.Message, wantMsg)
			}
			flusher := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(200)
			for _, e := range []string{
				`data: {"type":"reasoning","content":"planning"}`,
				`data: {"type":"text","content":"Hi there"}`,
				`data: {"type":"tool_start","id":"call_9","name":"shell","arguments":"{\"cmd\":\"ls\"}"}`,
				`data: {"type":"tokens","prompt":11,"completion":22,"total":33}`,
				`data: {"type":"done","session_id":"sess-42"}`,
			} {
				fmt.Fprintf(w, "%s\n\n", e)
				flusher.Flush()
			}
		}
	}))
}

func collectSSE(t *testing.T, body string) (events []string, datas []map[string]any) {
	t.Helper()
	lines := strings.Split(body, "\n")
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "event: ") {
			events = append(events, strings.TrimPrefix(lines[i], "event: "))
			if i+1 < len(lines) && strings.HasPrefix(lines[i+1], "data: ") {
				var m map[string]any
				if err := json.Unmarshal([]byte(strings.TrimPrefix(lines[i+1], "data: ")), &m); err != nil {
					t.Fatalf("bad data json: %v", err)
				}
				datas = append(datas, m)
			}
		}
	}
	return
}

func serveResponses(srv *Server, r *http.Request) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func TestResponsesStream(t *testing.T) {
	d := mockDaemonFull(t, "hi")
	defer d.Close()
	srv := NewServer(atmc.NewClient(d.URL), nil)

	reqBody := `{"model":"deepseek-chat","stream":true,"instructions":"be nice","input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]}]}`
	r := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(reqBody))
	r.Header.Set("Content-Type", "application/json")
	w := serveResponses(srv, r)

	if w.Code != 200 {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	events, datas := collectSSE(t, w.Body.String())

	// First/last event are the envelope markers.
	if len(events) < 2 || events[0] != "response.created" || events[len(events)-1] != "response.completed" {
		t.Fatalf("bad envelope: %v", events)
	}
	// Data payloads align 1:1 with events and every one carries the right type.
	for i, d := range datas {
		if d["type"] != events[i] {
			t.Errorf("event %d: sse=%q payload.type=%q", i, events[i], d["type"])
		}
	}

	// completed carries usage and full output items.
	last := datas[len(datas)-1]
	respObj, _ := last["response"].(map[string]any)
	if respObj["status"] != "completed" {
		t.Errorf("status = %v", respObj["status"])
	}
	usage, _ := respObj["usage"].(map[string]any)
	if usage["total_tokens"].(float64) != 33 {
		t.Errorf("usage.total_tokens = %v want 33", usage["total_tokens"])
	}
	out, _ := respObj["output"].([]any)
	var kinds []string
	for _, it := range out {
		im, _ := it.(map[string]any)
		kinds = append(kinds, im["type"].(string))
	}
	want := map[string]bool{"reasoning": false, "message": false, "function_call": false}
	for _, k := range kinds {
		want[k] = true
	}
	for k, seen := range want {
		if !seen {
			t.Errorf("completed.output missing item type %q; got %v", k, kinds)
		}
	}

	// Session persisted: second identical request reuses sess-42.
	if sid := srv.sessions.Get(atmc.ConversationKey(
		[]map[string]any{{"role": "user", "content": "hi"}}, "be nice")); sid != "sess-42" {
		t.Errorf("session not persisted, got %q", sid)
	}
}

func TestResponsesNonStream(t *testing.T) {
	d := mockDaemonFull(t, "hi")
	defer d.Close()
	srv := NewServer(atmc.NewClient(d.URL), nil)

	reqBody := `{"model":"deepseek-chat","input":"hi"}`
	r := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(reqBody))
	w := serveResponses(srv, r)

	if w.Code != 200 {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["object"] != "response" || m["status"] != "completed" {
		t.Errorf("bad shape: %v", m)
	}
	out, _ := m["output"].([]any)
	if len(out) != 3 {
		t.Errorf("want 3 output items, got %d: %v", len(out), out)
	}
	u, _ := m["usage"].(map[string]any)
	if u["input_tokens"].(float64) != 11 || u["output_tokens"].(float64) != 22 {
		t.Errorf("usage = %v", u)
	}
}

func TestResponsesErrorFromDaemon(t *testing.T) {
	d := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/providers":
			json.NewEncoder(w).Encode(map[string]any{"providers": []map[string]any{}})
		case "/chat":
			flusher := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(200)
			fmt.Fprint(w, "data: {\"type\":\"text\",\"content\":\"partial\"}\n\n")
			flusher.Flush()
			fmt.Fprint(w, "data: {\"type\":\"error\",\"message\":\"daemon exploded\"}\n\n")
			flusher.Flush()
		}
	}))
	defer d.Close()
	srv := NewServer(atmc.NewClient(d.URL), nil)

	r := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(`{"model":"m","stream":true,"input":"x"}`))
	w := serveResponses(srv, r)

	events, _ := collectSSE(t, w.Body.String())
	if len(events) == 0 || events[len(events)-1] != "response.failed" {
		t.Fatalf("expected terminal response.failed, got %v", events)
	}
	for _, e := range events {
		if e == "response.completed" {
			t.Error("completed must not follow failed")
		}
	}
}

func TestResponsesRejectsBadInput(t *testing.T) {
	d := mockDaemonFull(t, "")
	defer d.Close()
	srv := NewServer(atmc.NewClient(d.URL), nil)
	r := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(`{"model":"m","input":""}`))
	w := serveResponses(srv, r)
	if w.Code != 400 {
		t.Errorf("want 400 for empty input, got %d", w.Code)
	}
}
