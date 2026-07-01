package ai

// ContentPart represents one part of an OpenAI-compatible multimodal message.
type ContentPart struct {
	Type     string       `json:"type"`
	Text     string       `json:"text,omitempty"`
	ImageURL *ImageURLRef `json:"image_url,omitempty"`
}

// ImageURLRef holds a data URL or remote URL for image inputs.
type ImageURLRef struct {
	URL string `json:"url"`
}

// ChatMessage is an OpenAI-compatible chat message.
type ChatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// ChatRequest is an OpenAI-compatible streaming chat completion request.
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

// SessionResetResult describes the response from the AI session-reset endpoint.
type SessionResetResult struct {
	Success    bool
	Message    string
	ParsedJSON bool
	StatusCode int
	Body       string
}

// ChatStream yields one AI token at a time.
type ChatStream interface {
	Next() (string, error)
	Close() error
}
