package anthropic

import (
	"encoding/json"
	"fmt"
)

// MessageRequest is an Anthropic Messages API request.
type MessageRequest struct {
	Model       string          `json:"model"`
	Messages    []Message       `json:"messages"`
	System      jsonField       `json:"system,omitempty"`
	MaxTokens   int             `json:"max_tokens"`
	Stream      bool            `json:"stream,omitempty"`
	Metadata    map[string]any  `json:"metadata,omitempty"`
	StopSequences []string      `json:"stop_sequences,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	TopK        *int            `json:"top_k,omitempty"`
	Tools       jsonField       `json:"tools,omitempty"`
	Thinking    jsonField       `json:"thinking,omitempty"`
}

type jsonField []byte

func (j *jsonField) UnmarshalJSON(data []byte) error {
	*j = data
	return nil
}

func (j jsonField) MarshalJSON() ([]byte, error) {
	return j, nil
}

// Message represents a message in the Anthropic Messages API.
type Message struct {
	Role    string       `json:"role"`
	Content ContentField `json:"content,omitempty"`
}

// ContentField handles content that can be a string or []ContentBlock.
type ContentField []ContentBlock

func (c *ContentField) UnmarshalJSON(data []byte) error {
	// Try as array of ContentBlock first
	var blocks []ContentBlock
	if err := json.Unmarshal(data, &blocks); err == nil {
		*c = blocks
		return nil
	}
	// Try as plain string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*c = []ContentBlock{{Type: "text", Text: s}}
		return nil
	}
	return fmt.Errorf("content must be a string or array of content blocks")
}

func (c ContentField) MarshalJSON() ([]byte, error) {
	if len(c) == 1 && c[0].Type == "text" {
		return json.Marshal(c[0].Text)
	}
	return json.Marshal([]ContentBlock(c))
}

// ContentBlock represents a content block in an Anthropic message.
type ContentBlock struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Input  any    `json:"input,omitempty"`
	Source *ImageSource `json:"source,omitempty"`
	Thinking string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`
	Content  jsonField `json:"content,omitempty"` // for tool_result
}

// ImageSource represents an image source in a content block.
type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}
