package atmc

import (
	"strings"
	"testing"
)

func TestConversationKey(t *testing.T) {
	msgs := []map[string]any{
		{"role": "user", "content": "hello"},
	}
	key1 := ConversationKey(msgs, "")
	key2 := ConversationKey(msgs, "")
	if key1 != key2 {
		t.Errorf("same input should produce same key: %s vs %s", key1, key2)
	}
	if len(key1) != 16 {
		t.Errorf("key should be 16 chars, got %d: %s", len(key1), key1)
	}
}

func TestFormatMessages(t *testing.T) {
	msgs := []map[string]any{
		{"role": "system", "content": "you are helpful"},
		{"role": "user", "content": "hello"},
		{"role": "assistant", "content": "hi there"},
	}
	got := FormatMessages(msgs, "system prompt")
	if !strings.Contains(got, "User: hello") {
		t.Errorf("expected User: hello, got: %s", got)
	}
	if !strings.Contains(got, "Assistant: hi there") {
		t.Errorf("expected Assistant: hi there, got: %s", got)
	}
	if strings.Contains(got, "system") {
		t.Errorf("system message should be excluded: %s", got)
	}
}

func TestContentString(t *testing.T) {
	// String content
	if s := contentString("hello"); s != "hello" {
		t.Errorf("expected 'hello', got '%s'", s)
	}
	// Nil content
	if s := contentString(nil); s != "" {
		t.Errorf("expected '', got '%s'", s)
	}
	// Multi-part content
	multi := []any{
		map[string]any{"type": "text", "text": "hello"},
		map[string]any{"type": "text", "text": "world"},
	}
	if s := contentString(multi); s != "hello\nworld" {
		t.Errorf("expected 'hello\\nworld', got '%s'", s)
	}
	// Image (non-text) content
	withImage := []any{
		map[string]any{"type": "text", "text": "desc"},
		map[string]any{"type": "image_url", "image_url": map[string]any{"url": "data:image/png;base64,..."}},
	}
	if s := contentString(withImage); s != "desc" {
		t.Errorf("expected 'desc', got '%s'", s)
	}
}

func TestFindProviderForModel(t *testing.T) {
	providers := []ProviderConfig{
		{Name: "deepseek", Model: "deepseek-chat"},
		{Name: "openai", Model: "gpt-4"},
	}
	if p := FindProviderForModel(providers, "deepseek-chat"); p != "deepseek" {
		t.Errorf("expected 'deepseek', got '%s'", p)
	}
	if p := FindProviderForModel(providers, "UNKNOWN"); p != "" {
		t.Errorf("expected '', got '%s'", p)
	}
	if p := FindProviderForModel(providers, "DEEPSEEK-CHAT"); p != "deepseek" {
		t.Errorf("case insensitive match failed: got '%s'", p)
	}
}

func TestTranslateToOpenAIChunk(t *testing.T) {
	toolIdx := 0
	cases := []struct {
		name     string
		ev       SSEEvent
		expectFn func(string) bool
	}{
		{
			"text",
			SSEEvent{Type: "text", Content: "hello"},
			func(s string) bool { return strings.Contains(s, `"content":"hello"`) },
		},
		{
			"reasoning",
			SSEEvent{Type: "reasoning", Content: "thinking..."},
			func(s string) bool { return strings.Contains(s, `"reasoning_content"`) },
		},
		{
			"tool_start",
			SSEEvent{Type: "tool_start", ID: "call_1", Name: "read_file", Arguments: `{"path":"a.txt"}`},
			func(s string) bool { return strings.Contains(s, `"name":"read_file"`) },
		},
		{
			"tokens",
			SSEEvent{Type: "tokens", Prompt: 10, Completion: 20, Total: 30},
			func(s string) bool { return strings.Contains(s, `"prompt_tokens":10`) },
		},
		{
			"skip_tool_output",
			SSEEvent{Type: "tool_output"},
			func(s string) bool { return s == "" },
		},
		{
			"done",
			SSEEvent{Type: "done"},
			func(s string) bool { return s == "__DONE__" },
		},
		{
			"stopped",
			SSEEvent{Type: "stopped"},
			func(s string) bool { return s == "__DONE__" },
		},
		{
			"error",
			SSEEvent{Type: "error", Message: "oops"},
			func(s string) bool { return strings.Contains(s, `"finish_reason":"error"`) },
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := TranslateToOpenAIChunk(&c.ev, "test-model", &toolIdx)
			if !c.expectFn(got) {
				t.Errorf("unexpected result: %s", got)
			}
		})
	}
}

func TestTranslateToAnthropicSSE(t *testing.T) {
	state := NewAnthropicState()
	// First event should produce message_start + content_block_start + content_block_delta
	ev := SSEEvent{Type: "text", Content: "hello"}
	lines := TranslateToAnthropicSSE(&ev, "test-model", state)
	if len(lines) != 3 {
		t.Errorf("expected 3 lines for initial text, got %d", len(lines))
	}
	if !strings.Contains(lines[0], `"type":"message_start"`) {
		t.Errorf("first line should be message_start: %s", lines[0])
	}
	if !strings.Contains(lines[1], `"type":"content_block_start"`) {
		t.Errorf("second line should be content_block_start: %s", lines[1])
	}
	if !strings.Contains(lines[2], `"type":"content_block_delta"`) {
		t.Errorf("third line should be content_block_delta: %s", lines[2])
	}

	// Subsequent text event should only produce content_block_delta
	state2 := NewAnthropicState()
	state2.HasSentStart = true
	ev2 := SSEEvent{Type: "text", Content: "more"}
	lines2 := TranslateToAnthropicSSE(&ev2, "test-model", state2)
	if len(lines2) != 1 {
		t.Errorf("expected 1 line for subsequent text, got %d", len(lines2))
	}
	if !strings.Contains(lines2[0], `"delta":{"type":"text_delta"`) {
		t.Errorf("expected text_delta: %s", lines2[0])
	}
}

func TestBuildOpenAIFullChunk(t *testing.T) {
	delta := `{"choices":[{"delta":{"content":"hi"},"index":0}]}`
	full := BuildOpenAIFullChunk(delta, "test-model")
	if !strings.Contains(full, `"model":"test-model"`) {
		t.Errorf("expected model in chunk: %s", full)
	}
	if !strings.Contains(full, `"object":"chat.completion.chunk"`) {
		t.Errorf("expected chat.completion.chunk: %s", full)
	}
	if !strings.Contains(full, `"content":"hi"`) {
		t.Errorf("expected content in chunk: %s", full)
	}

	// DONE should return empty
	if s := BuildOpenAIFullChunk("__DONE__", "m"); s != "" {
		t.Errorf("expected empty for DONE, got %s", s)
	}
}

func TestJsonString(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"hello", `"hello"`},
		{`he"llo`, `"he\"llo"`},
		{"", `""`},
	}
	for _, c := range cases {
		got := jsonString(c.input)
		if got != c.expected {
			t.Errorf("jsonString(%q) = %s, want %s", c.input, got, c.expected)
		}
	}
}
