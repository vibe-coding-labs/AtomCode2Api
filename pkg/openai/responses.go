package openai

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/vibe-coding-labs/AtomCode2API/pkg/atmc"
	"github.com/vibe-coding-labs/AtomCode2API/pkg/resp"
)

// handleResponses implements the OpenAI Responses API (/v1/responses), the
// protocol Codex uses natively. Engine events from the daemon are translated
// into the canonical Responses SSE event sequence.
func (s *Server) handleResponses(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req resp.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, fmt.Sprintf("invalid request: %v", err))
		return
	}

	daemonMsg := req.InputToText()
	if daemonMsg == "" {
		writeError(w, 400, "no input to process")
		return
	}
	messages := req.Messages()
	system := req.Instructions

	provider := ""
	if req.Model != "" {
		if providers, err := s.Client.ListProviders(); err == nil {
			provider = atmc.FindProviderForModel(providers, req.Model)
		}
	}

	convKey := atmc.ConversationKey(messages, system)
	daemonSessionID := s.sessions.Get(convKey)

	log.Printf("responses: model=%s stream=%t provider=%s segs=%d sid=%s",
		req.Model, req.Stream, strOr(provider, "(auto)"), len(messages), strOr(daemonSessionID, "(new)"))

	daemonReq := &atmc.ChatRequest{
		Message:   daemonMsg,
		Stream:    true,
		Provider:  provider,
		System:    system,
		SessionID: daemonSessionID,
	}

	if !req.Stream {
		s.handleResponsesNonStream(w, &req, daemonReq, convKey, daemonSessionID)
		return
	}
	s.handleResponsesStream(w, r, &req, daemonReq, convKey, daemonSessionID)
}

func (s *Server) handleResponsesStream(w http.ResponseWriter, r *http.Request, req *resp.Request,
	daemonReq *atmc.ChatRequest, convKey, sessionID string) {

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

	ch, err := s.Client.ChatStream(daemonReq)
	stream := resp.NewStream("", req.Model)
	if err != nil {
		writeRespLines(w, flusher, stream.Error(err.Error()))
		return
	}

	var lastSessionID string
	for ev := range ch {
		if ev.Type == "done" || ev.Type == "stopped" {
			if ev.SessionID != "" {
				lastSessionID = ev.SessionID
			}
			continue
		}
		writeRespLines(w, flusher, stream.Event(respEvent(ev)))
		if ev.Type == "error" {
			writeRespLines(w, flusher, stream.Error(ev.Message))
			lastSessionID = ev.SessionID
			break
		}
	}
	writeRespLines(w, flusher, stream.Finish())

	if lastSessionID != "" && lastSessionID != sessionID {
		s.sessions.Set(convKey, lastSessionID)
	}
}

func (s *Server) handleResponsesNonStream(w http.ResponseWriter, req *resp.Request,
	daemonReq *atmc.ChatRequest, convKey, sessionID string) {

	ch, err := s.Client.ChatStream(daemonReq)
	if err != nil {
		writeError(w, 502, fmt.Sprintf("daemon chat error: %v", err))
		return
	}

	var events []resp.EngineEvent
	var lastSessionID string
	for ev := range ch {
		if ev.Type == "done" || ev.Type == "stopped" {
			if ev.SessionID != "" {
				lastSessionID = ev.SessionID
			}
			break
		}
		events = append(events, respEvent(ev))
	}
	if lastSessionID != "" && lastSessionID != sessionID {
		s.sessions.Set(convKey, lastSessionID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(200)
	w.Write(resp.NonStreaming("", req.Model, events))
}

func respEvent(ev atmc.SSEEvent) resp.EngineEvent {
	return resp.EngineEvent{
		Type:       ev.Type,
		Content:    ev.Content,
		ID:         ev.ID,
		Name:       ev.Name,
		Arguments:  ev.Arguments,
		Prompt:     ev.Prompt,
		Completion: ev.Completion,
		Total:      ev.Total,
		Message:    ev.Message,
		SessionID:  ev.SessionID,
	}
}

func writeRespLines(w http.ResponseWriter, flusher http.Flusher, lines []string) {
	for _, line := range lines {
		if line == "" {
			fmt.Fprint(w, "\n")
			continue
		}
		fmt.Fprint(w, line, "\n")
	}
	flusher.Flush()
}
