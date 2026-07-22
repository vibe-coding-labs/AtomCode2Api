package anthropic

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FormatAnthropicMessages formats Anthropic messages for the daemon.
// Returns the formatted message string and system prompt.
func FormatAnthropicMessages(messages []Message) (formatted string, systemPrompt string) {
	var parts []string
	for _, msg := range messages {
		if msg.Role == "system" {
			for _, block := range msg.Content {
				if block.Type == "text" {
					if systemPrompt != "" {
						systemPrompt += "\n"
					}
					systemPrompt += block.Text
				}
			}
			continue
		}

		content := extractTextContent(msg.Content)
		if content == "" {
			continue
		}

		label := "User"
		switch msg.Role {
		case "user":
			label = "User"
		case "assistant":
			label = "Assistant"
		case "tool_result":
			label = "User"
		}
		parts = append(parts, fmt.Sprintf("%s: %s", label, content))
	}
	return strings.Join(parts, "\n\n"), systemPrompt
}

// extractTextContent extracts text from content blocks.
func extractTextContent(blocks []ContentBlock) string {
	var texts []string
	for _, block := range blocks {
		switch block.Type {
		case "text":
			texts = append(texts, block.Text)
		case "thinking":
			texts = append(texts, block.Thinking)
		case "tool_use":
			texts = append(texts, fmt.Sprintf("[Using tool: %s]", block.Name))
		case "tool_result":
			if block.Content != nil {
				var nested []ContentBlock
				if err := json.Unmarshal(block.Content, &nested); err == nil {
					texts = append(texts, extractTextContent(nested))
				} else {
					var s string
					if json.Unmarshal(block.Content, &s) == nil {
						texts = append(texts, s)
					} else {
						texts = append(texts, string(block.Content))
					}
				}
				continue
			}
		}
	}
	return strings.Join(texts, "\n")
}

// ParseToolResultContent extracts text from tool_result content blocks.
func ParseToolResultContent(content []ContentBlock) string {
	return extractTextContent(content)
}
