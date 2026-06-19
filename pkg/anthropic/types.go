package anthropic

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
	Role    string            `json:"role"`
	Content []ContentBlock    `json:"content,omitempty"`
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
