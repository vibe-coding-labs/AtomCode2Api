package openai

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/vibe-coding-labs/AtomCode2API/pkg/atmc"
	"github.com/vibe-coding-labs/AtomCode2API/pkg/store"
)

// Server implements the OpenAI-compatible HTTP API.
type Server struct {
	Client  *atmc.Client
	store   *store.Store
	sessions *SessionTracker
}

// NewServer creates a new OpenAI-compatible proxy server.
func NewServer(client *atmc.Client, s *store.Store) *Server {
	return &Server{
		Client:   client,
		store:    s,
		sessions: NewSessionTracker(30 * time.Minute),
	}
}

// RegisterRoutes registers all OpenAI-compatible endpoints on the mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/chat/completions", s.handleChat)
	mux.HandleFunc("/v1/models", s.handleModels)
	mux.HandleFunc("/v1/web-search", s.handleWebSearch)
	mux.HandleFunc("/v1/rerank", s.handleRerank)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/v1/health", s.handleHealth)
}

// ─── Helper ──────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	data := NewErrorResponse(code, msg)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	w.Write(data)
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(200)
		return false
	}
	if r.Method != method {
		writeError(w, 405, "method not allowed")
		return false
	}
	return true
}

// ─── Endpoints ───────────────────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(200)
		return
	}

	h, err := s.Client.Health()
	daemonOK := err == nil && h.Status == "ok"

	auth, authErr := s.Client.AuthStatus()
	loggedIn := authErr == nil && auth.LoggedIn

	status := "ok"
	if !daemonOK {
		status = "degraded"
	}

	daemonVersion := "?"
	if h != nil && h.Version != "" {
		daemonVersion = h.Version
	}
	resp := map[string]any{
		"status":   status,
		"service":  "atomcode-2api",
		"daemon": map[string]any{
			"connected": daemonOK,
			"version":   daemonVersion,
		},
		"logged_in": loggedIn,
		"uptime":    time.Now().Unix(),
	}
	if daemonOK && loggedIn && auth.User != nil {
		resp["user"] = auth.User
	}
	writeJSON(w, 200, resp)
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	models, err := s.Client.ListModels()
	if err != nil {
		log.Printf("openai: list models error: %v", err)
		writeError(w, 502, fmt.Sprintf("daemon error: %v", err))
		return
	}

	writeJSON(w, 200, TranslateModels(models))
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Parse messages
	var messages []map[string]any
	if err := json.Unmarshal(req.Messages, &messages); err != nil {
		writeError(w, 400, "invalid messages format")
		return
	}

	// Extract system prompt + filter
	systemPrompt := ""
	var remaining []map[string]any
	for _, m := range messages {
		role, _ := m["role"].(string)
		if role == "system" {
			if content, ok := m["content"].(string); ok {
				systemPrompt = content
			}
		} else {
			remaining = append(remaining, m)
		}
	}

	if len(remaining) == 0 {
		writeError(w, 400, "no messages to process")
		return
	}

	// Resolve provider from model name
	provider := ""
	if req.Model != "" {
		providers, err := s.Client.ListProviders()
		if err == nil {
			provider = atmc.FindProviderForModel(providers, req.Model)
		}
	}

	// Build daemon message
	daemonMessage := atmc.FormatMessages(remaining, systemPrompt)

	// Session tracking
	convKey := atmc.ConversationKey(remaining, systemPrompt)
	daemonSessionID := s.sessions.Get(convKey)

	log.Printf("openai chat: model=%s stream=%t provider=%s messages=%d system=%t sid=%s",
		req.Model, req.Stream, strOr(provider, "(auto)"), len(remaining),
		systemPrompt != "", strOr(daemonSessionID, "(new)"))

	if req.Stream {
		s.handleStreamChat(w, r, &req, daemonMessage, provider, systemPrompt, daemonSessionID, convKey)
	} else {
		s.handleNonStreamChat(w, r, &req, daemonMessage, provider, systemPrompt, daemonSessionID, convKey)
	}
}

func (s *Server) handleNonStreamChat(w http.ResponseWriter, r *http.Request, req *ChatRequest,
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
	ch, err := s.Client.ChatStream(daemonReq)
	if err != nil {
		writeError(w, 502, fmt.Sprintf("daemon chat error: %v", err))
		return
	}

	for ev := range ch {
		if ev.Type == "done" {
			lastSessionID = ev.SessionID
			break
		}
		events = append(events, ev)
	}

	// Persist session for multi-turn context
	if lastSessionID != "" && lastSessionID != sessionID {
		s.sessions.Set(convKey, lastSessionID)
	}

	resp := TranslateToOpenAIResponse(events, req.Model)
	writeJSON(w, 200, resp)
}

func (s *Server) handleStreamChat(w http.ResponseWriter, r *http.Request, req *ChatRequest,
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

	ch, err := s.Client.ChatStream(daemonReq)
	if err != nil {
		errJSON := fmt.Sprintf(`data: {"error":"%s"}\n\ndata: [DONE]\n\n`, err.Error())
		fmt.Fprint(w, errJSON)
		flusher.Flush()
		return
	}

	toolIdx := 0
	hasToolUse := false
	var lastSessionID string

	for ev := range ch {
		if ev.Type == "done" {
			lastSessionID = ev.SessionID
			// Send appropriate finish_reason chunk before [DONE]
			finishReason := "stop"
			if hasToolUse {
				finishReason = "tool_calls"
			}
			finishChunk := fmt.Sprintf(`{"id":"chatcmpl-atomcode","object":"chat.completion.chunk","created":%d,"model":"%s","choices":[{"delta":{},"finish_reason":"%s","index":0}]}`, time.Now().Unix(), req.Model, finishReason)
			fmt.Fprintf(w, "data: %s\n\n", finishChunk)
			flusher.Flush()
			fmt.Fprint(w, "data: [DONE]\n\n")
			flusher.Flush()
			break
		}

		if ev.Type == "tool_start" {
			hasToolUse = true
		}
		delta := atmc.TranslateToOpenAIChunk(&ev, req.Model, &toolIdx)
		if delta == "" {
			continue
		}
		if delta == "__DONE__" {
			fmt.Fprint(w, "data: [DONE]\n\n")
			flusher.Flush()
			break
		}

		full := atmc.BuildOpenAIFullChunk(delta, req.Model)
		if full != "" {
			fmt.Fprintf(w, "data: %s\n\n", full)
			flusher.Flush()
		}
	}

	// Persist session for multi-turn context
	if lastSessionID != "" && lastSessionID != sessionID {
		s.sessions.Set(convKey, lastSessionID)
	}
}

// ─── Session Tracker ─────────────────────────────────────────────────────────

type SessionTracker struct {
	sessions map[string]sessionEntry
	ttl      time.Duration
}

type sessionEntry struct {
	sessionID string
	updatedAt time.Time
}

func NewSessionTracker(ttl time.Duration) *SessionTracker {
	st := &SessionTracker{
		sessions: make(map[string]sessionEntry),
		ttl:      ttl,
	}
	// Start cleanup goroutine
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			st.cleanup()
		}
	}()
	return st
}

func (st *SessionTracker) Get(key string) string {
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

func (st *SessionTracker) Set(key, sessionID string) {
	st.sessions[key] = sessionEntry{
		sessionID: sessionID,
		updatedAt: time.Now(),
	}
}

func (st *SessionTracker) cleanup() {
	now := time.Now()
	for k, v := range st.sessions {
		if now.Sub(v.updatedAt) > st.ttl {
			delete(st.sessions, k)
		}
	}
}

// ─── Small helpers ───────────────────────────────────────────────────────────

func strOr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// mapValue is not used; inline instead.