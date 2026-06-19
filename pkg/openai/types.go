package openai

import "encoding/json"

// ChatRequest is the OpenAI-compatible chat completion request.
type ChatRequest struct {
	Model            string          `json:"model"`
	Messages         json.RawMessage `json:"messages"`
	Stream           bool            `json:"stream,omitempty"`
	MaxTokens        int             `json:"max_tokens,omitempty"`
	Temperature      *float64        `json:"temperature,omitempty"`
	TopP             *float64        `json:"top_p,omitempty"`
	FrequencyPenalty *float64        `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64        `json:"presence_penalty,omitempty"`
	Stop             json.RawMessage `json:"stop,omitempty"`
	User             string          `json:"user,omitempty"`
	Tools            json.RawMessage `json:"tools,omitempty"`
	ToolChoice       json.RawMessage `json:"tool_choice,omitempty"`
}

// ChatUsage represents token usage in a chat completion.
type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatChoice represents a single choice in a chat completion response.
type ChatChoice struct {
	Index        int              `json:"index"`
	Message      *ChatMessage     `json:"message,omitempty"`
	Delta        *ChatDelta       `json:"delta,omitempty"`
	FinishReason *string          `json:"finish_reason,omitempty"`
}

// ChatMessage represents a message in a chat completion response.
type ChatMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ChatDelta represents a delta in streaming chat completion.
type ChatDelta struct {
	Role             string     `json:"role,omitempty"`
	Content          string     `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a function call in a chat response.
type ToolCall struct {
	Index    *int           `json:"index,omitempty"`
	ID       string         `json:"id,omitempty"`
	Type     string         `json:"type,omitempty"`
	Function ToolCallFunction `json:"function,omitempty"`
}

// ToolCallFunction represents the function details in a tool call.
type ToolCallFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// ChatCompletionResponse is the non-streaming chat completion response.
type ChatCompletionResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   *ChatUsage   `json:"usage,omitempty"`
}

// ModelResponse is the OpenAI /v1/models response entry.
type ModelResponse struct {
	ID       string `json:"id"`
	Object   string `json:"object"`
	Created  int64  `json:"created"`
	OwnedBy  string `json:"owned_by"`
}

// ModelsListResponse is the full /v1/models response.
type ModelsListResponse struct {
	Object string          `json:"object"`
	Data   []ModelResponse `json:"data"`
}

// ErrorResponse is a standard OpenAI-style error.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details.
type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
	Code    int    `json:"code,omitempty"`
}
