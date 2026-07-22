package openai

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/vibe-coding-labs/AtomCode2API/pkg/atmc"
)

func (s *Server) handleWebSearch(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var body struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	if body.Query == "" {
		writeError(w, 400, "query is required")
		return
	}
	daemonReq := &atmc.ChatRequest{
		Message:  "User: " + body.Query,
		Provider: "deepseek",
	}
	ch, err := s.Client.ChatStream(daemonReq)
	if err != nil {
		slog.Error("web search upstream", "error", err)
		writeError(w, 502, err.Error())
		return
	}
	var result string
	for ev := range ch {
		if ev.Type == "text" {
			result += ev.Content
		}
	}
	writeJSON(w, 200, map[string]any{"search_result": result})
}

func (s *Server) handleRerank(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var body struct {
		Query     string   `json:"query"`
		Documents []string `json:"documents"`
		TopN      int      `json:"top_n"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	if body.Query == "" || len(body.Documents) == 0 {
		writeError(w, 400, "query and documents are required")
		return
	}
	daemonReq := &atmc.ChatRequest{
		Message:  "User: Rerank query: " + body.Query + "\nDocuments: " + strings.Join(body.Documents, ", "),
		Provider: "deepseek",
	}
	ch, err := s.Client.ChatStream(daemonReq)
	if err != nil {
		slog.Error("rerank upstream", "error", err)
		writeError(w, 502, err.Error())
		return
	}
	var result string
	for ev := range ch {
		if ev.Type == "text" {
			result += ev.Content
		}
	}
	writeJSON(w, 200, map[string]any{"result": result})
}
