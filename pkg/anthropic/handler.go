package anthropic

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vibe-coding-labs/AtomCode2API/pkg/atmc"
	"github.com/vibe-coding-labs/AtomCode2API/pkg/store"
)

// Handler implements the Anthropic Messages API.
type Handler struct {
	Client   *atmc.Client
	store    *store.Store
	sessions *sessionTracker
}

// NewHandler creates a new Anthropic Messages API handler.
func NewHandler(client *atmc.Client, s *store.Store) *Handler {
	return &Handler{
		Client:   client,
		store:    s,
		sessions: newSessionTracker(30 * time.Minute),
	}
}

// RequestIDKey is the context key for request ID.
type RequestIDKey struct{}

// WithRequestID stores the request ID in the request.
func WithRequestID(r *http.Request, id uint64) *http.Request {
	r.Header.Set("X-Request-ID", strconv.FormatUint(id, 10))
	return r
}

// RegisterRoutes registers Anthropic Messages API endpoints.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/messages", h.handleMessages)
}

func (h *Handler) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-api-key")
		w.WriteHeader(200)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, 405, "method not allowed")
		return
	}

	var req MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Extract system prompt from system field or first message
	systemPrompt := ""
	if len(req.System) > 0 {
		var blocks []ContentBlock
		if err := json.Unmarshal(req.System, &blocks); err == nil {
			for _, b := range blocks {
				if b.Type == "text" {
					systemPrompt += b.Text
				}
			}
		} else {
			// Try as plain string
			var s string
			if json.Unmarshal(req.System, &s) == nil {
				systemPrompt = s
			}
		}
	}

	// Format messages for daemon
	formatted, sp := FormatAnthropicMessages(req.Messages)
	if systemPrompt == "" {
		systemPrompt = sp
	}

	if formatted == "" {
		writeError(w, 400, "no messages to process")
		return
	}

	// Resolve provider from model
	provider := ""
	if req.Model != "" {
		providers, err := h.Client.ListProviders()
		if err == nil {
			provider = atmc.FindProviderForModel(providers, req.Model)
		}
	}

	// Build conversation key & session tracking
	msgs := messagesToMap(req.Messages)
	convKey := atmc.ConversationKey(msgs, systemPrompt)
	daemonSessionID := h.sessions.get(convKey)

	log.Printf("anthropic messages: model=%s stream=%t provider=%s messages=%d system=%t sid=%s",
		req.Model, req.Stream, strOr(provider, "(auto)"), len(req.Messages),
		systemPrompt != "", strOr(daemonSessionID, "(new)"))

	if req.Stream {
		h.handleStreamChat(w, r, &req, formatted, provider, systemPrompt, daemonSessionID, convKey)
	} else {
		h.handleNonStreamChat(w, r, &req, formatted, provider, systemPrompt, daemonSessionID, convKey)
	}
}

func (h *Handler) handleNonStreamChat(w http.ResponseWriter, r *http.Request, req *MessageRequest,
	daemonMsg, provider, system, sessionID, convKey string) {

	daemonReq := &atmc.ChatRequest{
		Message:   daemonMsg,
		Stream:    true,
		Provider:  provider,
		System:    system,
		SessionID: sessionID,
	}

	var events []atmc.SSEEvent
	var lastSessionID string
	ch, err := h.Client.ChatStream(daemonReq)
	if err != nil {
		writeError(w, 502, fmt.Sprintf("daemon error: %v", err))
		return
	}
	for ev := range ch {
		if ev.Type == "done" {
			lastSessionID = ev.SessionID
			break
		}
		events = append(events, ev)
	}

	if lastSessionID != "" && lastSessionID != sessionID {
		h.sessions.set(convKey, lastSessionID)
	}

	resp := translateToAnthropicResponse(events, req.Model)
	writeJSON(w, 200, resp)
}

func (h *Handler) handleStreamChat(w http.ResponseWriter, r *http.Request, req *MessageRequest,
	daemonMsg, provider, system, sessionID, convKey string) {

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, 500, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(200)

	daemonReq := &atmc.ChatRequest{
		Message:   daemonMsg,
		Stream:    true,
		Provider:  provider,
		System:    system,
		SessionID: sessionID,
	}

	ch, err := h.Client.ChatStream(daemonReq)
	if err != nil {
		fmt.Fprintf(w, "data: {\"type\":\"error\",\"error\":{\"type\":\"api_error\",\"message\":\"%s\"}}\n\n", err.Error())
		flusher.Flush()
		return
	}

	state := atmc.NewAnthropicState()
	var lastSessionID string
	hasSentStop := false

	for ev := range ch {
		if ev.Type == "done" {
			lastSessionID = ev.SessionID
			if !hasSentStop {
				lines := atmc.TranslateToAnthropicSSE(&ev, req.Model, state)
				for _, line := range lines {
					fmt.Fprintf(w, "data: %s\n\n", line)
				}
				hasSentStop = true
			}
			flusher.Flush()
			break
		}

		lines := atmc.TranslateToAnthropicSSE(&ev, req.Model, state)
		for _, line := range lines {
			if strings.Contains(line, `"type":"message_stop"`) {
				hasSentStop = true
			}
			fmt.Fprintf(w, "data: %s\n\n", line)
			flusher.Flush()
		}
	}

	if !hasSentStop {
		fmt.Fprint(w, "data: {\"type\":\"message_stop\"}\n\n")
		flusher.Flush()
	}

	if lastSessionID != "" && lastSessionID != sessionID {
		h.sessions.set(convKey, lastSessionID)
	}
}

// ─── Translation helpers ────────────────────────────────────────────────────

func translateToAnthropicResponse(events []atmc.SSEEvent, model string) map[string]any {
	resp := map[string]any{
		"id":         fmt.Sprintf("msg_%x", time.Now().UnixNano()),
		"type":       "message",
		"role":       "assistant",
		"model":      model,
		"content":    []any{},
		"stop_reason":    "end_turn",
		"stop_sequence":  nil,
		"usage": map[string]any{
			"input_tokens":  0,
			"output_tokens": 0,
		},
	}

	var contentBlocks []any
	textContent := ""
	hasToolUse := false

	for _, ev := range events {
		switch ev.Type {
		case "text":
			textContent += ev.Content
		case "reasoning":
			// Add thinking block
			contentBlocks = append(contentBlocks, map[string]any{
				"type":     "thinking",
				"thinking": ev.Content,
				"signature": "",
			})
		case "tool_start":
			hasToolUse = true
			var input any = ev.Arguments
			var parsed any
			if json.Unmarshal([]byte(ev.Arguments), &parsed) == nil {
				input = parsed
			}
			contentBlocks = append(contentBlocks, map[string]any{
				"type":  "tool_use",
				"id":    ev.ID,
				"name":  ev.Name,
				"input": input,
			})
		case "tokens":
			resp["usage"] = map[string]any{
				"input_tokens":  ev.Prompt,
				"output_tokens": ev.Completion,
			}
		case "error":
			resp["stop_reason"] = "error"
			resp["error"] = map[string]any{
				"type":    "api_error",
				"message": ev.Message,
			}
		}
	}

	if textContent != "" && !hasToolUse {
		// Single text block
		if len(contentBlocks) == 0 {
			contentBlocks = append(contentBlocks, map[string]any{
				"type": "text",
				"text": textContent,
			})
		} else {
			contentBlocks = append([]any{map[string]any{
				"type": "text",
				"text": textContent,
			}}, contentBlocks...)
		}
	}

	resp["content"] = contentBlocks
	return resp
}

// ─── Session Tracker ─────────────────────────────────────────────────────────

type sessionEntry struct {
	sessionID string
	updatedAt time.Time
}

type sessionTracker struct {
	sessions map[string]sessionEntry
	ttl      time.Duration
}

func newSessionTracker(ttl time.Duration) *sessionTracker {
	st := &sessionTracker{
		sessions: make(map[string]sessionEntry),
		ttl:      ttl,
	}
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			st.cleanup()
		}
	}()
	return st
}

func (st *sessionTracker) get(key string) string {
	entry, ok := st.sessions[key]
	if !ok {
		return ""
	}
	if time.Since(entry.updatedAt) > st.ttl {
		delete(st.sessions, key)
		return ""
	}
	return entry.sessionID
}

func (st *sessionTracker) set(key, sessionID string) {
	st.sessions[key] = sessionEntry{
		sessionID: sessionID,
		updatedAt: time.Now(),
	}
}

func (st *sessionTracker) cleanup() {
	now := time.Now()
	for k, v := range st.sessions {
		if now.Sub(v.updatedAt) > st.ttl {
			delete(st.sessions, k)
		}
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	data, _ := json.Marshal(map[string]any{
		"error": map[string]any{
			"type":    "api_error",
			"message": msg,
		},
	})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	w.Write(data)
}

func strOr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func messagesToMap(msgs []Message) []map[string]any {
	result := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		item := map[string]any{
			"role":    m.Role,
			"content": contentBlocksToText(m.Content),
		}
		result = append(result, item)
	}
	return result
}

func contentBlocksToText(blocks []ContentBlock) string {
	var texts []string
	for _, b := range blocks {
		switch b.Type {
		case "text":
			texts = append(texts, b.Text)
		case "thinking":
			texts = append(texts, b.Thinking)
		case "tool_use":
			texts = append(texts, fmt.Sprintf("[tool_use: %s]", b.Name))
		case "tool_result":
			if b.Content != nil {
				continue
			}
		}
	}
	return strings.Join(texts, "\n")
}