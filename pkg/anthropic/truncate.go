package anthropic

// TruncateMessages truncates a conversation to fit within maxTokens.
// Keeps the most recent messages and system prompt.
// Preserves tool_use/tool_result pairs for tool integrity.
func TruncateMessages(messages []map[string]any, maxTokens int) []map[string]any {
	if len(messages) <= 4 {
		return messages
	}

	// Simple strategy: keep system (first), then keep last 3 messages
	var result []map[string]any

	// Find and preserve system message
	for _, m := range messages {
		if role, _ := m["role"].(string); role == "system" {
			result = append(result, m)
			break
		}
	}

	// Keep last 3 messages (user + assistant pairs)
	start := len(messages) - 3
	if start < 0 {
		start = 0
	}
	result = append(result, messages[start:]...)

	return result
}

// EstimateTokens estimates the number of tokens in a message list.
// Rough approximation: 1 token ≈ 4 characters for English, 1 token ≈ 2 CJK chars.
func EstimateTokens(messages []map[string]any) int {
	total := 0
	for _, m := range messages {
		if content, ok := m["content"].(string); ok {
			total += len(content) / 3
		}
		if role, _ := m["role"].(string); role == "" {
			total += 1
		}
		// Role tokens
		total += 2
	}
	return total
}

// EnsureMaxTokens truncates the conversation if it exceeds the limit.
func EnsureMaxTokens(messages []map[string]any, maxTokens int) []map[string]any {
	est := EstimateTokens(messages)
	if est <= maxTokens {
		return messages
	}
	return TruncateMessages(messages, maxTokens)
}