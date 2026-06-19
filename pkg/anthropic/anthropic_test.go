package anthropic

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// Local version of atmc.NewAnthropicState for test independence.
func testNewAnthropicState() *testAnthropicState {
	return &testAnthropicState{MessageID: fmt.Sprintf("msg_%x", time.Now().UnixNano())}
}

type testAnthropicState struct {
	MessageID    string
	ContentIndex int
	HasSentStart bool
}

func (s *testAnthropicState) translate(ev *testSSEEvent, model string) []string {
	switch ev.Type {
	case "text":
		if !s.HasSentStart {
			s.HasSentStart = true
			return []string{
				`{"type":"message_start","message":{"id":"` + s.MessageID + `","type":"message","role":"assistant","content":[],"model":"` + model + `","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":0,"output_tokens":0}},"model":"` + model + `"}`,
				fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"text","text":""}}`, s.ContentIndex),
				`{"type":"content_block_delta","index":` + fmt.Sprintf("%d", s.ContentIndex) + `,"delta":{"type":"text_delta","text":"` + ev.Content + `"}}`,
			}
		}
		return []string{
			`{"type":"content_block_delta","index":` + fmt.Sprintf("%d", s.ContentIndex) + `,"delta":{"type":"text_delta","text":"` + ev.Content + `"}}`,
		}
	case "reasoning":
		if !s.HasSentStart {
			s.HasSentStart = true
			return []string{
				`{"type":"message_start","message":{"id":"` + s.MessageID + `","type":"message","role":"assistant","content":[],"model":"` + model + `","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":0,"output_tokens":0}},"model":"` + model + `"}`,
				fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"thinking","thinking":""}}`, s.ContentIndex),
				`{"type":"content_block_delta","index":` + fmt.Sprintf("%d", s.ContentIndex) + `,"delta":{"type":"thinking_delta","thinking":"` + ev.Content + `"}}`,
			}
		}
		return []string{
			`{"type":"content_block_delta","index":` + fmt.Sprintf("%d", s.ContentIndex) + `,"delta":{"type":"thinking_delta","thinking":"` + ev.Content + `"}}`,
		}
	case "tool_start":
		if !s.HasSentStart {
			s.HasSentStart = true
			s.ContentIndex = -1
			return []string{
				`{"type":"message_start","message":{"id":"` + s.MessageID + `","type":"message","role":"assistant","content":[],"model":"` + model + `","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":0,"output_tokens":0}},"model":"` + model + `"}`,
				`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"` + ev.ID + `","name":"` + ev.Name + `","input":{}}}`,
				`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"` + ev.Arguments + `"}}`,
			}
		}
		s.ContentIndex++
		return []string{
			`{"type":"content_block_start","index":` + fmt.Sprintf("%d", s.ContentIndex) + `,"content_block":{"type":"tool_use","id":"` + ev.ID + `","name":"` + ev.Name + `","input":{}}}`,
			`{"type":"content_block_delta","index":` + fmt.Sprintf("%d", s.ContentIndex) + `,"delta":{"type":"input_json_delta","partial_json":"` + ev.Arguments + `"}}`,
		}
	case "tool_output", "tool_result":
		return nil
	case "tokens":
		return []string{
			fmt.Sprintf(`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":%d,"input_tokens":%d},"message":"%s"}`,
				ev.Completion, ev.Prompt, s.MessageID),
		}
	case "done", "stopped":
		return []string{
			`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{}}`,
			`{"type":"message_stop"}`,
		}
	case "error":
		return []string{
			`{"type":"error","error":{"type":"api_error","message":"` + ev.Message + `"}}`,
		}
	default:
		return nil
	}
}

type testSSEEvent struct {
	Type       string `json:"type"`
	Content    string `json:"content,omitempty"`
	ID         string `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	Arguments  string `json:"arguments,omitempty"`
	Prompt     int    `json:"prompt,omitempty"`
	Completion int    `json:"completion,omitempty"`
	Message    string `json:"message,omitempty"`
}

func TestFormatAnthropicMessages(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: []ContentBlock{{Type: "text", Text: "you are helpful"}}},
		{Role: "user", Content: []ContentBlock{{Type: "text", Text: "hello"}}},
		{Role: "assistant", Content: []ContentBlock{{Type: "text", Text: "hi there"}}},
	}
	formatted, system := FormatAnthropicMessages(messages)
	if system != "you are helpful" {
		t.Errorf("expected system prompt, got %q", system)
	}
	if formatted == "" {
		t.Errorf("expected formatted messages, got empty")
	}
}

func TestTranslateToAnthropicResponse(t *testing.T) {
	state := testNewAnthropicState()
	ev := &testSSEEvent{Type: "text", Content: "hello world"}
	lines := state.translate(ev, "test-model")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(lines))
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &parsed); err != nil {
		t.Fatalf("line 0 not valid json: %s", lines[0])
	}
	if parsed["type"] != "message_start" {
		t.Errorf("expected message_start, got %s", parsed["type"])
	}
}

func TestTranslateToAnthropicResponseThinking(t *testing.T) {
	state := testNewAnthropicState()
	ev := &testSSEEvent{Type: "reasoning", Content: "thinking step 1"}
	lines := state.translate(ev, "m")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[1], "thinking") {
		t.Errorf("expected thinking content_block, got: %s", lines[1])
	}
}

func TestTranslateToAnthropicTokens(t *testing.T) {
	state := testNewAnthropicState()
	state.HasSentStart = true
	ev := &testSSEEvent{Type: "tokens", Prompt: 10, Completion: 20}
	lines := state.translate(ev, "m")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "message_delta") {
		t.Errorf("expected message_delta, got %s", lines[0])
	}
}

func TestTranslateToAnthropicToolStart(t *testing.T) {
	state := testNewAnthropicState()
	ev := &testSSEEvent{Type: "tool_start", ID: "tool1", Name: "read_file", Arguments: `{"path":"a.txt"}`}
	lines := state.translate(ev, "m")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[1], "tool_use") {
		t.Errorf("expected tool_use block, got: %s", lines[1])
	}
}

func TestTranslateToAnthropicError(t *testing.T) {
	state := testNewAnthropicState()
	ev := &testSSEEvent{Type: "error", Message: "oops"}
	lines := state.translate(ev, "m")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "api_error") {
		t.Errorf("expected api_error, got: %s", lines[0])
	}
}

func TestTranslateToAnthropicDone(t *testing.T) {
	state := testNewAnthropicState()
	ev := &testSSEEvent{Type: "done"}
	lines := state.translate(ev, "m")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (message_delta + message_stop), got %d", len(lines))
	}
	if !strings.Contains(lines[1], "message_stop") {
		t.Errorf("expected message_stop, got: %s", lines[1])
	}
}
