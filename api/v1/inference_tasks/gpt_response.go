package inference_tasks

type MessageRole string

const (
	SystemRole    MessageRole = "system"
	UserRole      MessageRole = "user"
	AssistantRole MessageRole = "assistant"
)

type Message struct {
	Role    MessageRole `json:"role"`
	Content string      `json:"content"`
}

type Usage struct {
	PromptTokens     uint `json:"prompt_tokens" validate:"required"`
	CompletionTokens uint `json:"completion_tokens" validate:"required"`
	TotalTokens      uint `json:"total_tokens" validate:"required"`
}

type FinishReason string

const (
	ReasonStop   FinishReason = "stop"
	ReasonLength FinishReason = "length"
)

type ResponseChoice struct {
	Index        uint         `json:"index" validate:"required"`
	Message      Message      `json:"message" validate:"required"`
	FinishReason FinishReason `json:"finish_reason" validate:"required"`
}

type GPTTaskResponse struct {
	Model   string           `json:"model" validate:"required"`
	Choices []ResponseChoice `json:"choices" validate:"required"`
	Usage   Usage            `json:"usage" validate:"required"`
}
