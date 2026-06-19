package anthropic

import (
	"testing"
)

func TestTruncateMessages(t *testing.T) {
	messages := []map[string]any{
		{"role": "system", "content": "you are helpful"},
		{"role": "user", "content": "msg1"},
		{"role": "assistant", "content": "response1"},
		{"role": "user", "content": "msg2"},
		{"role": "assistant", "content": "response2"},
	}
	truncated := TruncateMessages(messages, 100)
	if len(truncated) != 4 {
		t.Errorf("expected 4 messages, got %d", len(truncated))
	}
	// System should be preserved
	if truncated[0]["role"] != "system" {
		t.Errorf("expected system message first")
	}
}

func TestEstimateTokens(t *testing.T) {
	messages := []map[string]any{
		{"role": "user", "content": "hello world this is a test message with some more content for token counting"},
	}
	est := EstimateTokens(messages)
	if est <= 0 {
		t.Errorf("expected positive token estimate, got %d", est)
	}
}

func TestEnsureMaxTokensSmall(t *testing.T) {
	messages := []map[string]any{
		{"role": "user", "content": "hi"},
	}
	result := EnsureMaxTokens(messages, 1000)
	if len(result) != len(messages) {
		t.Errorf("expected unchanged messages")
	}
}