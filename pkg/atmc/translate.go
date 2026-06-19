package atmc

import (
	"crypto/md5"
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
}

func NewAnthropicState() *AnthropicState {
	return &AnthropicState{MessageID: fmt.Sprintf("msg_%x", time.Now().UnixNano())}
}

// TranslateToAnthropicSSE converts a daemon SSEEvent to Anthropic SSE data lines.
func TranslateToAnthropicSSE(ev *SSEEvent, model string, state *AnthropicState) []string {
	switch ev.Type {
	case "text":
		if !state.HasSentStart {
			state.HasSentStart = true
			return []string{
				fmt.Sprintf(`{"type":"message_start","message":{"id":%s,"type":"message","role":"assistant","content":[],"model":%s,"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":0,"output_tokens":0}},"model":%s}`,
					jsonString(state.MessageID), jsonString(model), jsonString(model)),
				fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"text","text":""}}`, state.ContentIndex),
				fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"text_delta","text":%s}}`, state.ContentIndex, jsonString(ev.Content)),
			}
		}
		return []string{
			fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"text_delta","text":%s}}`, state.ContentIndex, jsonString(ev.Content)),
		}
	case "reasoning":
		if !state.HasSentStart {
			state.HasSentStart = true
			return []string{
				fmt.Sprintf(`{"type":"message_start","message":{"id":%s,"type":"message","role":"assistant","content":[],"model":%s,"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":0,"output_tokens":0}},"model":%s}`,
					jsonString(state.MessageID), jsonString(model), jsonString(model)),
				fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"thinking","thinking":""}}`, state.ContentIndex),
				fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"thinking_delta","thinking":%s}}`, state.ContentIndex, jsonString(ev.Content)),
			}
		}
		return []string{
			fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"thinking_delta","thinking":%s}}`, state.ContentIndex, jsonString(ev.Content)),
		}
	case "tool_start":
		if !state.HasSentStart {
			state.HasSentStart = true
			state.ContentIndex = -1
			return []string{
				fmt.Sprintf(`{"type":"message_start","message":{"id":%s,"type":"message","role":"assistant","content":[],"model":%s,"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":0,"output_tokens":0}},"model":%s}`,
					jsonString(state.MessageID), jsonString(model), jsonString(model)),
				fmt.Sprintf(`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":%s,"name":%s,"input":{}}}`, jsonString(ev.ID), jsonString(ev.Name)),
				fmt.Sprintf(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":%s}}`, jsonString(ev.Arguments)),
			}
		}
		state.ContentIndex++
		return []string{
			fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"tool_use","id":%s,"name":%s,"input":{}}}`, state.ContentIndex, jsonString(ev.ID), jsonString(ev.Name)),
			fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"input_json_delta","partial_json":%s}}`, state.ContentIndex, jsonString(ev.Arguments)),
		}
	case "tool_output", "tool_result":
		return nil
	case "tokens":
		return []string{
			fmt.Sprintf(`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":%d,"input_tokens":%d},"message":%s}`,
				ev.Completion, ev.Prompt, jsonString(state.MessageID)),
		}
	case "done", "stopped":
		return []string{
			`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{}}`,
			`{"type":"message_stop"}`,
		}
	case "error":
		return []string{
			fmt.Sprintf(`{"type":"error","error":{"type":"api_error","message":%s}}`, jsonString(ev.Message)),
		}
	default:
		return nil
	}
}

func jsonString(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}
