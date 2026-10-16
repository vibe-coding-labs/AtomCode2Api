// Package resp implements the OpenAI Responses API (v1/responses) surface:
// request parsing for the `input` item array and translation of the AtomCode
// daemon's normalized engine events into the canonical Responses SSE event
// sequence that clients such as Codex expect.
package resp

import (
	"encoding/json"
	"strings"
)

// Request is an OpenAI Responses API request body.
type Request struct {
	Model        string          `json:"model"`
	Instructions string          `json:"instructions,omitempty"`
	Input        json.RawMessage `json:"input"`
	Stream       bool            `json:"stream,omitempty"`
}

// EngineEvent is a normalized upstream generation event, mirroring the fields
// the AtomCode daemon emits over its internal SSE channel.
type EngineEvent struct {
	Type       string // text | reasoning | tool_start | tokens | done | error
	Content    string
	ID         string
	Name       string
	Arguments  string
	Prompt     int
	Completion int
	Total      int
	Message    string
	SessionID  string
}

// segment is one conversation turn extracted from the Responses `input`.
type segment struct {
	role  string
	text  string
	isRaw bool // plain-string input: pass through without role labeling
}

// segments parses both the plain-string and item-array forms of `input`.
func (r *Request) segments() []segment {
	raw := strings.TrimSpace(string(r.Input))
	if raw == "" {
		return nil
	}
	if strings.HasPrefix(raw, `"`) {
		var s string
		if err := json.Unmarshal(r.Input, &s); err == nil {
			if s == "" {
				return nil
			}
			return []segment{{role: "user", text: s, isRaw: true}}
		}
	}

	var items []map[string]any
	if err := json.Unmarshal(r.Input, &items); err != nil {
		return nil
	}
	var segs []segment
	for _, it := range items {
		switch typeOf(it) {
		case "message":
			role, _ := it["role"].(string)
			if role == "system" || role == "developer" {
				role = "user"
			}
			if t := messageText(it["content"]); t != "" {
				segs = append(segs, segment{role: role, text: t})
			}
		case "function_call":
			name, _ := it["name"].(string)
			args, _ := it["arguments"].(string)
			segs = append(segs, segment{role: "assistant", text: "[tool_use: " + name + " " + args + "]"})
		case "function_call_output":
			segs = append(segs, segment{role: "user", text: "[tool_result: " + contentText(it["output"]) + "]"})
		case "reasoning":
			if t := summaryText(it["summary"]); t != "" {
				segs = append(segs, segment{role: "assistant", text: "[reasoning: " + t + "]"})
			}
		default:
			if t := messageText(it["content"]); t != "" {
				segs = append(segs, segment{role: "user", text: t})
			}
		}
	}
	return segs
}

// InputToText flattens `input` into a single prompt for the daemon. A plain
// string input is passed through as-is; multi-item inputs are labeled with the
// "Role: text" convention the chat bridge already uses.
func (r *Request) InputToText() string {
	var parts []string
	for _, s := range r.segments() {
		if s.isRaw {
			return s.text
		}
		label := "User"
		switch s.role {
		case "assistant":
			label = "Assistant"
		case "user", "tool":
			label = "User"
		default:
			label = strings.Title(s.role)
		}
		parts = append(parts, label+": "+s.text)
	}
	return strings.Join(parts, "\n\n")
}

// Messages returns pseudo chat messages so session-keying helpers can be
// reused across the chat and responses endpoints.
func (r *Request) Messages() []map[string]any {
	var msgs []map[string]any
	for _, s := range r.segments() {
		role := s.role
		if role != "assistant" {
			role = "user"
		}
		msgs = append(msgs, map[string]any{"role": role, "content": s.text})
	}
	return msgs
}

func typeOf(m map[string]any) string {
	t, _ := m["type"].(string)
	if t == "" {
		if _, ok := m["role"]; ok {
			return "message"
		}
	}
	return t
}

// messageText extracts text from string or content-part-array message content.
func messageText(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		var texts []string
		for _, p := range v {
			pm, _ := p.(map[string]any)
			if pm == nil {
				continue
			}
			if t, _ := pm["text"].(string); t != "" {
				texts = append(texts, t)
			}
		}
		return strings.Join(texts, "\n")
	}
	return ""
}

// contentText renders arbitrary content (string / part array / raw JSON) as text.
func contentText(content any) string {
	switch v := content.(type) {
	case nil:
		return ""
	case string:
		return v
	default:
		if t := messageText(v); t != "" {
			return t
		}
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

// summaryText extracts text from a reasoning summary part array.
func summaryText(raw any) string {
	arr, _ := raw.([]any)
	var texts []string
	for _, p := range arr {
		pm, _ := p.(map[string]any)
		if pm == nil {
			continue
		}
		if t, _ := pm["text"].(string); t != "" {
			texts = append(texts, t)
		}
	}
	return strings.Join(texts, "\n")
}
