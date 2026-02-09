package anthropic

// ===== Anthropic Messages API 请求类型 =====

// MessagesRequest POST /v1/messages 请求体
type MessagesRequest struct {
	Model         string           `json:"model"`
	MaxTokens     int              `json:"max_tokens"`
	System        string           `json:"system,omitempty"`
	Messages      []Message        `json:"messages"`
	Temperature   *float64         `json:"temperature,omitempty"`
	TopP          *float64         `json:"top_p,omitempty"`
	Stream        bool             `json:"stream,omitempty"`
	Tools         []ToolDefinition `json:"tools,omitempty"`
	StopSequences []string         `json:"stop_sequences,omitempty"`
}

// Message 消息
type Message struct {
	Role    string `json:"role"`    // "user" 或 "assistant"
	Content any    `json:"content"` // string 或 []ContentBlock
}

// ContentBlock 内容块
type ContentBlock struct {
	Type string `json:"type"` // "text", "tool_use", "tool_result", "thinking"
	// text 块
	Text string `json:"text,omitempty"`
	// tool_use 块
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input any    `json:"input,omitempty"`
	// tool_result 块
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   any    `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
	// thinking 块
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`
}

// ToolDefinition 工具定义
type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema"`
}

// ===== Anthropic Messages API 响应类型 =====

// MessagesResponse 非流式响应
type MessagesResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"` // "message"
	Role         string         `json:"role"` // "assistant"
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence,omitempty"`
	Usage        *Usage         `json:"usage"`
}

// Usage 用量信息
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ErrorResponse API 错误响应
type ErrorResponse struct {
	Type  string   `json:"type"` // "error"
	Error APIError `json:"error"`
}

// APIError 错误详情
type APIError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// ===== 流式 SSE 事件类型 =====

// MessageStartEvent message_start 事件
type MessageStartEvent struct {
	Type    string           `json:"type"`
	Message MessagesResponse `json:"message"`
}

// ContentBlockStartEvent content_block_start 事件
type ContentBlockStartEvent struct {
	Type         string       `json:"type"`
	Index        int          `json:"index"`
	ContentBlock ContentBlock `json:"content_block"`
}

// ContentBlockDeltaEvent content_block_delta 事件
type ContentBlockDeltaEvent struct {
	Type  string     `json:"type"`
	Index int        `json:"index"`
	Delta DeltaBlock `json:"delta"`
}

// DeltaBlock 增量内容
type DeltaBlock struct {
	Type        string `json:"type"` // "text_delta", "input_json_delta", "thinking_delta"
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
	Thinking    string `json:"thinking,omitempty"`
}

// ContentBlockStopEvent content_block_stop 事件
type ContentBlockStopEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

// MessageDeltaEvent message_delta 事件
type MessageDeltaEvent struct {
	Type  string       `json:"type"`
	Delta MessageDelta `json:"delta"`
	Usage *DeltaUsage  `json:"usage,omitempty"`
}

// MessageDelta 消息增量
type MessageDelta struct {
	StopReason   string  `json:"stop_reason,omitempty"`
	StopSequence *string `json:"stop_sequence,omitempty"`
}

// DeltaUsage 增量用量
type DeltaUsage struct {
	OutputTokens int `json:"output_tokens"`
}
