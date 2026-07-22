package openai

import (
	"testing"

	"github.com/vibe-coding-labs/AtomCode2API/pkg/atmc"
)

func TestTranslateToOpenAIResponse(t *testing.T) {
	events := []atmc.SSEEvent{
		{Type: "text", Content: "Hello"},
		{Type: "text", Content: " world"},
		{Type: "tokens", Prompt: 10, Completion: 20, Total: 30},
	}
	resp := TranslateToOpenAIResponse(events, "test-model")
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Model != "test-model" {
		t.Errorf("expected model test-model, got %s", resp.Model)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "Hello world" {
		t.Errorf("content should concatenate: got %q", resp.Choices[0].Message.Content)
	}
	if resp.Usage == nil {
		t.Fatal("expected usage")
	}
	if resp.Usage.TotalTokens != 30 {
		t.Errorf("expected 30 total tokens, got %d", resp.Usage.TotalTokens)
	}
	if *resp.Choices[0].FinishReason != "stop" {
		t.Errorf("expected stop finish reason")
	}
}

func TestTranslateToOpenAIResponseWithToolCalls(t *testing.T) {
	events := []atmc.SSEEvent{
		{Type: "text", Content: "Let me check"},
		{Type: "tool_start", ID: "call_1", Name: "read_file", Arguments: `{"path":"a.txt"}`},
		{Type: "tokens", Prompt: 5, Completion: 10, Total: 15},
	}
	resp := TranslateToOpenAIResponse(events, "test-model")
	if resp.Choices[0].FinishReason == nil || *resp.Choices[0].FinishReason != "tool_calls" {
		t.Errorf("expected tool_calls finish reason")
	}
	if len(resp.Choices[0].Message.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.Choices[0].Message.ToolCalls))
	}
	if resp.Choices[0].Message.ToolCalls[0].ID != "call_1" {
		t.Errorf("tool call ID mismatch")
	}
}

func TestTranslateToOpenAIResponseWithError(t *testing.T) {
	events := []atmc.SSEEvent{
		{Type: "text", Content: "partial"},
		{Type: "error", Message: "something went wrong"},
	}
	resp := TranslateToOpenAIResponse(events, "m")
	if resp.Choices[0].FinishReason == nil || *resp.Choices[0].FinishReason != "error" {
		t.Errorf("expected error finish reason")
	}
}

func TestTranslateModels(t *testing.T) {
	atmcModels := []atmc.ModelInfo{
		{ID: "deepseek-chat", Model: "deepseek-chat"},
		{ID: "gpt-4", Model: "gpt-4"},
	}
	resp := TranslateModels(atmcModels)
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 models, got %d", len(resp.Data))
	}
	if resp.Data[0].ID != "deepseek-chat" {
		t.Errorf("first model should be deepseek-chat")
	}
	if resp.Data[0].OwnedBy != "atomcode" {
		t.Errorf("owned_by should be atomcode")
	}
}

func TestNewErrorResponse(t *testing.T) {
	data := NewErrorResponse(400, "bad request")
	if len(data) == 0 {
		t.Fatal("expected non-empty error response")
	}
}

func TestStrPtr(t *testing.T) {
	p := strPtr("hello")
	if *p != "hello" {
		t.Errorf("expected hello, got %s", *p)
	}
}
