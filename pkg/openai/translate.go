package openai

import (
	"encoding/json"
	"time"

	"github.com/vibe-coding-labs/AtomCodeProxy/pkg/atmc"
)

// TranslateToOpenAIResponse converts daemon SSE events into a non-streaming ChatCompletionResponse.
func TranslateToOpenAIResponse(events []atmc.SSEEvent, model string) *ChatCompletionResponse {
	resp := &ChatCompletionResponse{
		ID:      "chatcmpl-atomcode",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []ChatChoice{{
			Index:        0,
			Message:      &ChatMessage{Role: "assistant", Content: ""},
			FinishReason: strPtr("stop"),
		}},
	}

	var toolCalls []ToolCall
	for _, ev := range events {
		switch ev.Type {
		case "text":
			resp.Choices[0].Message.Content += ev.Content
		case "tool_start":
			tc := ToolCall{
				ID:   ev.ID,
				Type: "function",
				Function: ToolCallFunction{
					Name:      ev.Name,
					Arguments: ev.Arguments,
				},
			}
			toolCalls = append(toolCalls, tc)
		case "tokens":
			resp.Usage = &ChatUsage{
				PromptTokens:     ev.Prompt,
				CompletionTokens: ev.Completion,
				TotalTokens:      ev.Total,
			}
		case "error":
			resp.Choices[0].FinishReason = strPtr("error")
		}
	}

	if len(toolCalls) > 0 {
		resp.Choices[0].Message.ToolCalls = toolCalls
		resp.Choices[0].FinishReason = strPtr("tool_calls")
	}

	return resp
}

// NewErrorResponse creates an OpenAI-style error response JSON.
func NewErrorResponse(code int, message string) []byte {
	data, _ := json.Marshal(ErrorResponse{
		Error: ErrorDetail{Message: message, Code: code},
	})
	return data
}

// TranslateModels converts daemon model info to OpenAI /v1/models format.
func TranslateModels(atmcModels []atmc.ModelInfo) *ModelsListResponse {
	data := make([]ModelResponse, 0, len(atmcModels))
	for _, m := range atmcModels {
		data = append(data, ModelResponse{
			ID:      m.ID,
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "atomcode",
		})
	}
	return &ModelsListResponse{
		Object: "list",
		Data:   data,
	}
}

func strPtr(s string) *string {
	return &s
}
