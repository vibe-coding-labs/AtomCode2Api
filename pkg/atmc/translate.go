package atmc

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ConversationKey generates a deterministic hash for a conversation prefix.
func ConversationKey(messages []map[string]any, system string) string {
	payload := system
	for i, m := range messages {
		if i == len(messages)-1 {
			break
		}
		role, _ := m["role"].(string)
		content := contentString(m["content"])
		payload += fmt.Sprintf("|%s:%s", role, content)
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(payload)))[:16]
}

// FormatMessages formats the messages list into a single daemon message string.
func FormatMessages(messages []map[string]any, systemPrompt string) string {
	var parts []string
	for _, m := range messages {
		role, _ := m["role"].(string)
		content := contentString(m["content"])
		if content == "" {
			continue
		}

		label := "User"
		switch role {
		case "user", "tool":
			label = "User"
		case "assistant":
			label = "Assistant"
		case "system":
			continue
		default:
			label = strings.Title(role)
		}
		parts = append(parts, fmt.Sprintf("%s: %s", label, content))
	}
	return strings.Join(parts, "\n\n")
}

// contentString extracts a string from message content.
func contentString(content any) string {
	if content == nil {
		return ""
	}
	switch v := content.(type) {
	case string:
		return v
	case []any:
		var texts []string
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				if m["type"] == "text" {
					if t, ok := m["text"].(string); ok {
						texts = append(texts, t)
					}
				}
			}
		}
		return strings.Join(texts, "\n")
	default:
		return fmt.Sprintf("%v", content)
	}
}

// FindProviderForModel searches the provider list for a matching model name.
func FindProviderForModel(providers []ProviderConfig, model string) string {
	modelLower := strings.ToLower(model)
	for _, p := range providers {
		if strings.ToLower(p.Model) == modelLower {
			return p.Name
		}
	}
	return ""
}

// ─── OpenAI SSE Translation ──────────────────────────────────────────────────

// TranslateToOpenAIChunk converts a daemon SSEEvent to an OpenAI SSE data line.
// Returns the SSE data string, or empty string to skip, or "__DONE__" to signal completion.
func TranslateToOpenAIChunk(ev *SSEEvent, model string, toolIdx *int) string {
	switch ev.Type {
	case "text":
		return fmt.Sprintf(`{"choices":[{"delta":{"content":%s},"index":0}]}`, jsonString(ev.Content))
	case "reasoning":
		return fmt.Sprintf(`{"choices":[{"delta":{"reasoning_content":%s},"index":0}]}`, jsonString(ev.Content))
	case "tool_start":
		*toolIdx++
		id := ev.ID
		if id == "" {
			id = fmt.Sprintf("call_%d", *toolIdx)
		}
		return fmt.Sprintf(`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":%s,"type":"function","function":{"name":%s,"arguments":%s}}]},"index":0}]}`,
			jsonString(id), jsonString(ev.Name), jsonString(ev.Arguments))
	case "tool_output":
		return ""
	case "tool_result":
		return ""
	case "tokens":
		return fmt.Sprintf(`{"choices":[],"usage":{"prompt_tokens":%d,"completion_tokens":%d,"total_tokens":%d}`,
			ev.Prompt, ev.Completion, ev.Total)
	case "done", "stopped":
		return "__DONE__"
	case "error":
		return fmt.Sprintf(`{"choices":[{"delta":{},"finish_reason":"error","index":0}]}`)
	default:
		return ""
	}
}

// BuildOpenAIFullChunk wraps a delta into a full OpenAI SSE chunk JSON string.
func BuildOpenAIFullChunk(deltaJSON string, model string) string {
	if deltaJSON == "__DONE__" {
		return ""
	}
	return fmt.Sprintf(`{"id":"chatcmpl-atomcode","object":"chat.completion.chunk","created":%d,"model":%s,%s}`,
		time.Now().Unix(), jsonString(model), deltaJSON[1:])
}

// ─── Anthropic SSE Translation ───────────────────────────────────────────────

// AnthropicState tracks Anthropic SSE translation progress.
type AnthropicState struct {
	MessageID    string
	ContentIndex int
	HasSentStart bool
	CurrentBlock string // "text", "thinking", "tool_use"
}

func NewAnthropicState() *AnthropicState {
	return &AnthropicState{
		MessageID: fmt.Sprintf("msg_%x", time.Now().UnixNano()),
	}
}

// TranslateToAnthropicSSE converts a daemon SSEEvent to Anthropic SSE data lines.
func TranslateToAnthropicSSE(ev *SSEEvent, model string, state *AnthropicState) []string {
	switch ev.Type {
	case "text":
		return translateAnthropicText(ev, model, state)
	case "reasoning":
		return translateAnthropicReasoning(ev, model, state)
	case "tool_start":
		return translateAnthropicToolStart(ev, model, state)
	case "tool_output", "tool_result":
		return nil
	case "tokens":
		return []string{
			fmt.Sprintf(`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":%d,"input_tokens":%d},"message":%s}`,
				ev.Completion, ev.Prompt, jsonString(state.MessageID)),
		}
	case "done", "stopped":
		var lines []string
		if state.CurrentBlock != "" {
			lines = append(lines, fmt.Sprintf(`{"type":"content_block_stop","index":%d}`, state.ContentIndex))
			state.CurrentBlock = ""
		}
		lines = append(lines,
			`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{}}`,
			`{"type":"message_stop"}`)
		return lines
	case "error":
		if state.CurrentBlock != "" {
			state.CurrentBlock = ""
		}
		return []string{
			fmt.Sprintf(`{"type":"error","error":{"type":"api_error","message":%s}}`, jsonString(ev.Message)),
		}
	default:
		return nil
	}
}

func translateAnthropicText(ev *SSEEvent, model string, state *AnthropicState) []string {
	// If we are in a non-text block, close it first
	if state.CurrentBlock != "" && state.CurrentBlock != "text" {
		closeBlock := []string{
			fmt.Sprintf(`{"type":"content_block_stop","index":%d}`, state.ContentIndex),
		}
		state.ContentIndex++
		state.CurrentBlock = "text"
		openBlock := []string{
			fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"text","text":""}}`, state.ContentIndex),
			fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"text_delta","text":%s}}`, state.ContentIndex, jsonString(ev.Content)),
		}
		return append(closeBlock, openBlock...)
	}

	if !state.HasSentStart {
		state.HasSentStart = true
		state.CurrentBlock = "text"
		return []string{
			fmt.Sprintf(`{"type":"message_start","message":{"id":%s,"type":"message","role":"assistant","content":[],"model":%s,"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":0,"output_tokens":0}},"model":%s}`,
				jsonString(state.MessageID), jsonString(model), jsonString(model)),
			fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"text","text":""}}`, state.ContentIndex),
			fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"text_delta","text":%s}}`, state.ContentIndex, jsonString(ev.Content)),
		}
	}
	state.CurrentBlock = "text"
	return []string{
		fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"text_delta","text":%s}}`, state.ContentIndex, jsonString(ev.Content)),
	}
}

func translateAnthropicReasoning(ev *SSEEvent, model string, state *AnthropicState) []string {
	// If we are in a non-thinking block, close it first
	if state.CurrentBlock != "" && state.CurrentBlock != "thinking" {
		closeBlock := []string{
			fmt.Sprintf(`{"type":"content_block_stop","index":%d}`, state.ContentIndex),
		}
		state.ContentIndex++
		state.CurrentBlock = "thinking"
		openBlock := []string{
			fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"thinking","thinking":""}}`, state.ContentIndex),
			fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"thinking_delta","thinking":%s}}`, state.ContentIndex, jsonString(ev.Content)),
		}
		return append(closeBlock, openBlock...)
	}

	if !state.HasSentStart {
		state.HasSentStart = true
		state.CurrentBlock = "thinking"
		return []string{
			fmt.Sprintf(`{"type":"message_start","message":{"id":%s,"type":"message","role":"assistant","content":[],"model":%s,"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":0,"output_tokens":0}},"model":%s}`,
				jsonString(state.MessageID), jsonString(model), jsonString(model)),
			fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"thinking","thinking":""}}`, state.ContentIndex),
			fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"thinking_delta","thinking":%s}}`, state.ContentIndex, jsonString(ev.Content)),
		}
	}
	state.CurrentBlock = "thinking"
	return []string{
		fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"thinking_delta","thinking":%s}}`, state.ContentIndex, jsonString(ev.Content)),
	}
}

func translateAnthropicToolStart(ev *SSEEvent, model string, state *AnthropicState) []string {
	var lines []string

	// Close current block if any
	if state.CurrentBlock != "" && state.CurrentBlock != "tool_use" {
		lines = append(lines, fmt.Sprintf(`{"type":"content_block_stop","index":%d}`, state.ContentIndex))
		state.ContentIndex++
		state.HasSentStart = true
	} else if !state.HasSentStart {
		state.HasSentStart = true
		state.ContentIndex = 0
		lines = append(lines,
			fmt.Sprintf(`{"type":"message_start","message":{"id":%s,"type":"message","role":"assistant","content":[],"model":%s,"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":0,"output_tokens":0}},"model":%s}`,
				jsonString(state.MessageID), jsonString(model), jsonString(model)))
	} else if state.CurrentBlock == "tool_use" {
		// Already in a tool_use, close previous
		lines = append(lines, fmt.Sprintf(`{"type":"content_block_stop","index":%d}`, state.ContentIndex))
		state.ContentIndex++
	}

	state.CurrentBlock = "tool_use"
	lines = append(lines,
		fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"tool_use","id":%s,"name":%s,"input":{}}}`, state.ContentIndex, jsonString(ev.ID), jsonString(ev.Name)),
		fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"input_json_delta","partial_json":%s}}`, state.ContentIndex, jsonString(ev.Arguments)),
	)
	return lines
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}