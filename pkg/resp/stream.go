package resp

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// item is one output item accumulated during generation.
type item struct {
	kind     string // "message" | "reasoning" | "function_call"
	id       string
	role     string
	name     string
	callID   string
	text     strings.Builder
	summary  strings.Builder
	args     strings.Builder
}

// usage records token accounting from the engine.
type usage struct {
	input, output, total int
	reasoning            bool
}

// Stream is a Responses-API event emitter. Feed it normalized EngineEvents;
// it produces the canonical SSE line sequence (response.created → … →
// response.completed / response.failed).
type Stream struct {
	respID  string
	model   string
	created int64
	seq     int
	items   []*item
	cur     *item
	usage   *usage
	msgItem *item
	msgIdx  int
	started bool
	failed  bool
	idSeq   int
}

// NewStream starts a response stream for the given model. A generated
// response id is used when respID is empty.
func NewStream(respID, model string) *Stream {
	if respID == "" {
		respID = fmt.Sprintf("resp_%x", time.Now().UnixNano())
	}
	return &Stream{respID: respID, model: model, created: time.Now().Unix()}
}

// Event translates one engine event into SSE lines.
func (s *Stream) Event(ev EngineEvent) []string {
	if s.failed {
		return nil
	}
	var out []string
	if !s.started {
		out = append(out, s.sse("response.created", map[string]any{
			"response": s.snapshot("in_progress", nil),
		})...)
		out = append(out, s.sse("response.in_progress", map[string]any{
			"response": s.snapshot("in_progress", nil),
		})...)
		s.started = true
	}

	switch ev.Type {
	case "text":
		if ev.Content == "" {
			return out
		}
		// Upstream may interleave reasoning and text; the Responses shape is
		// reasoning* → message. Keep one message open; reopen after a tool
		// call closed it.
		if s.msgItem == nil || s.cur != s.msgItem {
			out = append(out, s.closeItem()...)
			out = append(out, s.openItem(&item{kind: "message", id: s.newID("msg"), role: "assistant"})...)
			s.msgItem = s.cur
			s.msgIdx = s.outIdx()
		}
		s.msgItem.text.WriteString(ev.Content)
		s.cur = s.msgItem
		out = append(out, s.sse("response.output_text.delta", map[string]any{
			"item_id": s.msgItem.id, "output_index": s.msgIdx, "content_index": 0, "delta": ev.Content,
		})...)
	case "reasoning":
		if ev.Content == "" {
			return out
		}
		if s.msgItem != nil {
			// Reasoning after the answer started: fold into the message text.
			s.msgItem.text.WriteString(ev.Content)
			out = append(out, s.sse("response.output_text.delta", map[string]any{
				"item_id": s.msgItem.id, "output_index": s.msgIdx, "content_index": 0, "delta": ev.Content,
			})...)
			break
		}
		if s.cur == nil || s.cur.kind != "reasoning" {
			out = append(out, s.closeItem()...)
			out = append(out, s.openItem(&item{kind: "reasoning", id: s.newID("rs")})...)
		}
		s.cur.summary.WriteString(ev.Content)
		out = append(out, s.sse("response.reasoning_summary_text.delta", map[string]any{
			"item_id": s.cur.id, "output_index": s.outIdx(), "summary_index": 0, "delta": ev.Content,
		})...)
	case "tool_start":
		out = append(out, s.closeItem()...)
		s.msgItem = nil
		callID := ev.ID
		if callID == "" {
			callID = s.newID("call")
		}
		out = append(out, s.openItem(&item{kind: "function_call", id: s.newID("fc"), name: ev.Name, callID: callID})...)
		s.cur.args.WriteString(ev.Arguments)
		out = append(out, s.sse("response.function_call_arguments.delta", map[string]any{
			"item_id": s.cur.id, "output_index": s.outIdx(), "call_id": s.cur.callID, "name": s.cur.name, "delta": ev.Arguments,
		})...)
	case "tokens":
		reasoned := false
		for _, it := range s.items {
			if it.kind == "reasoning" && it.summary.Len() > 0 {
				reasoned = true
			}
		}
		s.usage = &usage{input: ev.Prompt, output: ev.Completion, total: ev.Total, reasoning: reasoned}
	}
	return out
}

// Finish closes the stream with response.completed.
func (s *Stream) Finish() []string {
	if s.failed {
		return nil
	}
	out := s.closeItem()
	out = append(out, s.sse("response.completed", map[string]any{
		"response": s.snapshot("completed", nil),
	})...)
	s.failed = true
	return out
}

// Error terminates the stream with response.failed. When the stream has not
// started yet, the opening created/in_progress events are emitted first so the
// envelope is complete even on immediate failure.
func (s *Stream) Error(msg string) []string {
	var out []string
	if !s.started {
		out = append(out, s.sse("response.created", map[string]any{
			"response": s.snapshot("in_progress", nil),
		})...)
		out = append(out, s.sse("response.in_progress", map[string]any{
			"response": s.snapshot("in_progress", nil),
		})...)
		s.started = true
	}
	s.failed = true
	return append(out, s.sse("response.failed", map[string]any{
		"response": s.snapshot("failed", map[string]any{"type": "server_error", "message": msg}),
	})...)
}

func (s *Stream) openItem(it *item) []string {
	s.items = append(s.items, it)
	s.cur = it
	out := s.sse("response.output_item.added", map[string]any{
		"output_index": len(s.items) - 1, "item": itemShell(it),
	})
	switch it.kind {
	case "message":
		out = append(out, s.sse("response.content_part.added", map[string]any{
			"item_id": it.id, "output_index": s.outIdx(), "content_index": 0,
			"part": map[string]any{"type": "output_text", "text": "", "annotations": []any{}},
		})...)
	case "reasoning":
		out = append(out, s.sse("response.reasoning_summary_part.added", map[string]any{
			"item_id": it.id, "output_index": s.outIdx(), "summary_index": 0,
			"part": map[string]any{"type": "summary_text", "text": ""},
		})...)
	}
	return out
}

func (s *Stream) closeItem() []string {
	it := s.cur
	if it == nil {
		return nil
	}
	s.cur = nil
	idx := s.indexOf(it)
	switch it.kind {
	case "message":
		out := s.sse("response.output_text.done", map[string]any{
			"item_id": it.id, "output_index": idx, "content_index": 0, "text": it.text.String(),
		})
		out = append(out, s.sse("response.content_part.done", map[string]any{
			"item_id": it.id, "output_index": idx, "content_index": 0,
			"part": map[string]any{"type": "output_text", "text": it.text.String(), "annotations": []any{}},
		})...)
		return append(out, s.sse("response.output_item.done", map[string]any{
			"output_index": idx, "item": itemDone(it),
		})...)
	case "reasoning":
		out := s.sse("response.reasoning_summary_text.done", map[string]any{
			"item_id": it.id, "output_index": idx, "summary_index": 0, "text": it.summary.String(),
		})
		out = append(out, s.sse("response.reasoning_summary_part.done", map[string]any{
			"item_id": it.id, "output_index": idx, "summary_index": 0,
			"part": map[string]any{"type": "summary_text", "text": it.summary.String()},
		})...)
		return append(out, s.sse("response.output_item.done", map[string]any{
			"output_index": idx, "item": itemDone(it),
		})...)
	case "function_call":
		out := s.sse("response.function_call_arguments.done", map[string]any{
			"item_id": it.id, "output_index": idx, "call_id": it.callID, "name": it.name, "arguments": it.args.String(),
		})
		return append(out, s.sse("response.output_item.done", map[string]any{
			"output_index": idx, "item": itemDone(it),
		})...)
	}
	return nil
}

func (s *Stream) indexOf(it *item) int {
	for i, x := range s.items {
		if x == it {
			return i
		}
	}
	return len(s.items) - 1
}

func (s *Stream) outIdx() int { return len(s.items) - 1 }

func (s *Stream) newID(prefix string) string {
	s.idSeq++
	return fmt.Sprintf("%s_%d", prefix, s.idSeq)
}

func (s *Stream) sse(event string, fields map[string]any) []string {
	payload := map[string]any{"type": event}
	for k, v := range fields {
		payload[k] = v
	}
	if event != "response.created" && event != "response.in_progress" {
		payload["response_id"] = s.respID
	}
	s.seq++
	payload["sequence_number"] = s.seq
	b, _ := json.Marshal(payload)
	return []string{"event: " + event, "data: " + string(b), ""}
}

// snapshot renders the response object embedded in created/completed/failed.
func (s *Stream) snapshot(status string, err map[string]any) map[string]any {
	output := make([]map[string]any, 0, len(s.items))
	for _, it := range s.items {
		output = append(output, itemDone(it))
	}
	resp := map[string]any{
		"id": s.respID, "object": "response", "created_at": s.created, "status": status,
		"model": s.model, "output": output,
	}
	if err != nil {
		resp["error"] = err
	}
	resp["usage"] = usageMap(s.usage)
	return resp
}

func itemShell(it *item) map[string]any {
	switch it.kind {
	case "reasoning":
		return map[string]any{"type": "reasoning", "id": it.id, "summary": []any{}}
	case "function_call":
		return map[string]any{"type": "function_call", "id": it.id, "call_id": it.callID, "name": it.name, "arguments": ""}
	default:
		return map[string]any{"type": "message", "id": it.id, "role": "assistant", "status": "in_progress", "content": []any{}}
	}
}

func itemDone(it *item) map[string]any {
	switch it.kind {
	case "reasoning":
		return map[string]any{"type": "reasoning", "id": it.id,
			"summary": []any{map[string]any{"type": "summary_text", "text": it.summary.String()}}}
	case "function_call":
		return map[string]any{"type": "function_call", "id": it.id, "call_id": it.callID,
			"name": it.name, "arguments": it.args.String()}
	default:
		return map[string]any{"type": "message", "id": it.id, "role": "assistant", "status": "completed",
			"content": []any{map[string]any{"type": "output_text", "text": it.text.String(), "annotations": []any{}}}}
	}
}

func usageMap(u *usage) map[string]any {
	rt := 0
	if u != nil && u.reasoning {
		rt = u.output
	}
	in, out, total := 0, 0, 0
	if u != nil {
		in, out, total = u.input, u.output, u.total
	}
	if total == 0 {
		total = in + out
	}
	return map[string]any{
		"input_tokens":  in, "output_tokens": out, "total_tokens": total,
		"input_tokens_details":  map[string]any{"cached_tokens": 0},
		"output_tokens_details": map[string]any{"reasoning_tokens": rt},
	}
}

// NonStreaming runs the engine events through the same state machine and
// returns the response body for stream=false requests.
func NonStreaming(respID, model string, events []EngineEvent) []byte {
	s := NewStream(respID, model)
	for _, ev := range events {
		s.Event(ev)
	}
	s.closeItem()
	s.failed = true
	snap := s.snapshot("completed", nil)
	b, _ := json.Marshal(snap)
	return b
}
