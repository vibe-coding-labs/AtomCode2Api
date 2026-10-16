package resp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRequestInputToText(t *testing.T) {
	var r Request
	if err := json.Unmarshal([]byte(`{"model":"m","input":"hello"}`), &r); err != nil {
		t.Fatalf("string input: %v", err)
	}
	if got := r.InputToText(); got != "hello" {
		t.Fatalf("string input = %q, want %q", got, "hello")
	}

	raw := `{"model":"m","instructions":"sys","input":[
	  {"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]},
	  {"type":"function_call","call_id":"c1","name":"shell","arguments":"{\"a\":1}"},
	  {"type":"function_call_output","call_id":"c1","output":"done"},
	  {"type":"reasoning","id":"rs1","summary":[{"type":"summary_text","text":"deep thought"}]}
	]}`
	var r2 Request
	if err := json.Unmarshal([]byte(raw), &r2); err != nil {
		t.Fatalf("item input: %v", err)
	}
	txt := r2.InputToText()
	for _, want := range []string{"hi", "shell", `{"a":1}`, "done", "deep thought"} {
		if !strings.Contains(txt, want) {
			t.Errorf("InputToText missing %q; got:\n%s", want, txt)
		}
	}
	if r2.Instructions != "sys" {
		t.Errorf("instructions = %q", r2.Instructions)
	}
}

func TestStreamEventSequence(t *testing.T) {
	evs := []EngineEvent{
		{Type: "reasoning", Content: "thinking"},
		{Type: "text", Content: "hello"},
		{Type: "tool_start", Name: "shell", Arguments: `{"cmd":"ls"}`},
		{Type: "tokens", Prompt: 10, Completion: 5, Total: 15},
	}
	var types []string
	s := NewStream("", "gpt-5")
	for _, ev := range evs {
		for _, line := range s.Event(ev) {
			if strings.HasPrefix(line, "event: ") {
				types = append(types, strings.TrimPrefix(line, "event: "))
			}
		}
	}
	for _, line := range s.Finish() {
		if strings.HasPrefix(line, "event: ") {
			types = append(types, strings.TrimPrefix(line, "event: "))
		}
	}

	wantOrder := []string{
		"response.created",
		"response.in_progress",
		"response.output_item.added",
		"response.reasoning_summary_text.delta",
		"response.output_item.added",
		"response.output_text.delta",
		"response.output_item.added",
		"response.function_call_arguments.delta",
		"response.completed",
	}
	if len(types) < len(wantOrder) {
		t.Fatalf("too few events: %v", types)
	}
	idx := 0
	for _, want := range wantOrder {
		found := false
		for idx < len(types) {
			if types[idx] == want {
				found = true
				idx++
				break
			}
			idx++
		}
		if !found {
			t.Errorf("missing/out-of-order event %q in %v", want, types)
		}
	}
}

// TestInterleavedReasoningText locks the real-daemon behavior: upstream emits
// reasoning and text alternately; the stream must collapse them into ONE
// reasoning item + ONE message item (reasoning after first text is folded
// into the message text instead of opening extra items).
func TestInterleavedReasoningText(t *testing.T) {
	s := NewStream("resp_test", "m")
	var lines []string
	for _, ev := range []EngineEvent{
		{Type: "reasoning", Content: "think1 "},
		{Type: "text", Content: "hello "},
		{Type: "reasoning", Content: "think2 "},
		{Type: "text", Content: "world"},
	} {
		lines = append(lines, s.Event(ev)...)
	}
	lines = append(lines, s.Finish()...)

	added := 0
	var completed string
	for _, l := range lines {
		if l == "event: response.output_item.added" {
			added++
		}
		if strings.HasPrefix(l, "data: ") && strings.Contains(l, `"type":"response.completed"`) {
			completed = strings.TrimPrefix(l, "data: ")
		}
	}
	if added != 2 {
		t.Fatalf("want 2 output items (reasoning+message), got %d added events", added)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(completed), &m); err != nil {
		t.Fatal(err)
	}
	out := m["response"].(map[string]any)["output"].([]any)
	types := make([]string, 0, len(out))
	for _, it := range out {
		types = append(types, it.(map[string]any)["type"].(string))
	}
	if strings.Join(types, ",") != "reasoning,message" {
		t.Errorf("output item types = %v, want [reasoning message]", types)
	}
	msg := out[1].(map[string]any)
	txt := msg["content"].([]any)[0].(map[string]any)["text"].(string)
	if txt != "hello think2 world" {
		t.Errorf("message text = %q, want %q (folded reasoning)", txt, "hello think2 world")
	}
	rs := out[0].(map[string]any)
	rst := rs["summary"].([]any)[0].(map[string]any)["text"].(string)
	if rst != "think1 " {
		t.Errorf("reasoning summary = %q, want %q", rst, "think1 ")
	}
}

func TestStreamDataIsValidResponseEnvelope(t *testing.T) {
	s := NewStream("", "gpt-5")
	var lines []string
	lines = append(lines, s.Event(EngineEvent{Type: "text", Content: "hi"})...)
	lines = append(lines, s.Event(EngineEvent{Type: "tokens", Prompt: 1, Completion: 2, Total: 3})...)
	lines = append(lines, s.Finish()...)

	var objs int
	for _, line := range lines {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(payload), &m); err != nil {
			t.Fatalf("bad json payload %q: %v", payload, err)
		}
		if ev, ok := m["type"].(string); !ok || ev == "" {
			t.Errorf("event without type: %s", payload)
		}
		if _, ok := m["sequence_number"]; ok {
			objs++
		}
	}
	if objs == 0 {
		t.Error("no sequence_number found on any event payload")
	}
}

func TestCompletedCarriesUsageShape(t *testing.T) {
	s := NewStream("", "gpt-5")
	s.Event(EngineEvent{Type: "text", Content: "x"})
	s.Event(EngineEvent{Type: "tokens", Prompt: 100, Completion: 50, Total: 150})
	lines := s.Finish()

	var data string
	for _, l := range lines {
		if strings.HasPrefix(l, "data: ") {
			data = strings.TrimPrefix(l, "data: ")
		}
	}
	if data == "" {
		t.Fatal("no data payload in finish")
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		t.Fatal(err)
	}
	if m["type"] != "response.completed" {
		t.Errorf("type = %v", m["type"])
	}
	respObj, _ := m["response"].(map[string]any)
	usage, _ := respObj["usage"].(map[string]any)
	for _, k := range []string{"input_tokens", "output_tokens", "total_tokens", "input_tokens_details", "output_tokens_details"} {
		if _, ok := usage[k]; !ok {
			t.Errorf("usage missing %s in %v", k, usage)
		}
	}
	if usage["total_tokens"].(float64) != 150 {
		t.Errorf("total_tokens = %v, want 150", usage["total_tokens"])
	}
	// output[] must contain the assistant message item.
	out, _ := respObj["output"].([]any)
	var foundMsg bool
	for _, it := range out {
		im, _ := it.(map[string]any)
		if im["type"] == "message" {
			foundMsg = true
		}
	}
	if !foundMsg {
		t.Errorf("completed output missing message item: %v", out)
	}
}

func TestErrorEvent(t *testing.T) {
	s := NewStream("", "gpt-5")
	lines := append(s.Event(EngineEvent{Type: "text", Content: "partial"}), s.Error("boom")...)
	var sawFailed, sawErrTerm bool
	for _, l := range lines {
		if l == "event: response.failed" {
			sawFailed = true
		}
		if strings.Contains(l, `"boom"`) {
			sawErrTerm = true
		}
	}
	if !sawFailed {
		t.Error("missing response.failed")
	}
	if !sawErrTerm {
		t.Error("missing error message")
	}
}

func TestNonStreamingResponse(t *testing.T) {
	evs := []EngineEvent{
		{Type: "reasoning", Content: "r"},
		{Type: "text", Content: "answer"},
		{Type: "tool_start", Name: "shell", Arguments: `{"cmd":"ls"}`},
		{Type: "tokens", Prompt: 3, Completion: 4, Total: 7},
	}
	out := string(NonStreaming("", "gpt-5", evs))
	if _, err := parseObject([]byte(out)); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	for _, want := range []string{"\"object\":\"response\"", "\"status\":\"completed\"", "shell", "\"input_tokens\":3"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}

// parseObject is a helper so tests can validate the non-streaming payload shape.
func parseObject(b []byte) (map[string]any, error) {
	var m map[string]any
	err := json.Unmarshal(b, &m)
	return m, err
}
