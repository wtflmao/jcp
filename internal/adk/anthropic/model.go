package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strings"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

var _ model.LLM = &AnthropicModel{}

const (
	DefaultBaseURL          = "https://api.anthropic.com"
	DefaultAnthropicVersion = "2023-06-01"
	DefaultMaxTokens        = 4096
)

// AnthropicModel 实现 model.LLM 接口，使用 Anthropic Messages API
type AnthropicModel struct {
	httpClient HTTPDoer
	baseURL    string
	apiKey     string
	modelName  string
	maxTokens  int
}

// HTTPDoer HTTP 客户端接口
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewAnthropicModel 创建 Anthropic 模型
func NewAnthropicModel(modelName, apiKey, baseURL string, maxTokens int, httpClient HTTPDoer) *AnthropicModel {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if maxTokens <= 0 {
		maxTokens = DefaultMaxTokens
	}
	return &AnthropicModel{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		modelName:  modelName,
		maxTokens:  maxTokens,
	}
}

// Name 返回模型名称
func (m *AnthropicModel) Name() string {
	return m.modelName
}

// GenerateContent 实现 model.LLM 接口
func (m *AnthropicModel) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	if stream {
		return m.generateStream(ctx, req)
	}
	return m.generate(ctx, req)
}

// messagesEndpoint 返回 Messages API 端点 URL
func (m *AnthropicModel) messagesEndpoint() string {
	return m.baseURL + "/v1/messages"
}

// doRequest 发送 HTTP 请求到 Messages API
func (m *AnthropicModel) doRequest(ctx context.Context, body []byte, stream bool) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.messagesEndpoint(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", m.apiKey)
	req.Header.Set("anthropic-version", DefaultAnthropicVersion)
	if stream {
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Connection", "keep-alive")
	}
	return m.httpClient.Do(req)
}

// generate 非流式生成
func (m *AnthropicModel) generate(ctx context.Context, req *model.LLMRequest) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		apiReq, err := toMessagesRequest(req, m.modelName, m.maxTokens)
		if err != nil {
			yield(nil, err)
			return
		}
		apiReq.Stream = false

		body, err := json.Marshal(apiReq)
		if err != nil {
			yield(nil, fmt.Errorf("序列化请求失败: %w", err))
			return
		}

		resp, err := m.doRequest(ctx, body, false)
		if err != nil {
			yield(nil, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			yield(nil, fmt.Errorf("Anthropic API 错误 (HTTP %d): %s", resp.StatusCode, string(respBody)))
			return
		}

		var apiResp MessagesResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			yield(nil, fmt.Errorf("解析响应失败: %w", err))
			return
		}

		llmResp, err := convertResponse(&apiResp)
		if err != nil {
			yield(nil, err)
			return
		}
		yield(llmResp, nil)
	}
}

// generateStream 流式生成
func (m *AnthropicModel) generateStream(ctx context.Context, req *model.LLMRequest) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		apiReq, err := toMessagesRequest(req, m.modelName, m.maxTokens)
		if err != nil {
			yield(nil, err)
			return
		}
		apiReq.Stream = true

		body, err := json.Marshal(apiReq)
		if err != nil {
			yield(nil, fmt.Errorf("序列化请求失败: %w", err))
			return
		}

		resp, err := m.doRequest(ctx, body, true)
		if err != nil {
			yield(nil, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			yield(nil, fmt.Errorf("Anthropic API 流式错误 (HTTP %d): %s", resp.StatusCode, string(respBody)))
			return
		}

		m.processStream(resp.Body, yield)
	}
}

// toolCallBuilder 用于聚合流式工具调用
type toolCallBuilder struct {
	id   string
	name string
	args string
}

// processStream 处理 Anthropic Messages API 的 SSE 流
func (m *AnthropicModel) processStream(body io.Reader, yield func(*model.LLMResponse, error) bool) {
	scanner := bufio.NewScanner(body)
	// 增大 scanner 缓冲区以处理大型 SSE 事件
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// 聚合状态
	aggregatedContent := &genai.Content{Role: "model", Parts: []*genai.Part{}}
	var textContent string
	var thinkingContent string
	toolCallsMap := make(map[int]*toolCallBuilder)
	blockTypes := make(map[int]string)
	var usageMetadata *genai.GenerateContentResponseUsageMetadata
	var finishReason genai.FinishReason
	var currentEventType string

	for scanner.Scan() {
		line := scanner.Text()

		// SSE 格式解析
		if eventType, ok := strings.CutPrefix(line, "event: "); ok {
			currentEventType = strings.TrimSpace(eventType)
			continue
		}
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok || data == "" {
			continue
		}

		switch currentEventType {
		case "message_start":
			m.handleMessageStart(data, &usageMetadata)
		case "content_block_start":
			m.handleContentBlockStart(data, blockTypes, toolCallsMap)
		case "content_block_delta":
			m.handleContentBlockDelta(data, blockTypes, &textContent, &thinkingContent, toolCallsMap, yield)
		case "message_delta":
			m.handleMessageDelta(data, &finishReason, &usageMetadata)
		case "error":
			m.handleError(data, yield)
			return
		case "ping", "message_stop", "content_block_stop":
			// 忽略
		}
		currentEventType = ""
	}

	// 组装最终聚合响应
	if thinkingContent != "" {
		aggregatedContent.Parts = append(aggregatedContent.Parts, &genai.Part{
			Text:    thinkingContent,
			Thought: true,
		})
	}
	if textContent != "" {
		aggregatedContent.Parts = append(aggregatedContent.Parts, &genai.Part{
			Text: textContent,
		})
	}
	for _, builder := range toolCallsMap {
		aggregatedContent.Parts = append(aggregatedContent.Parts, &genai.Part{
			FunctionCall: &genai.FunctionCall{
				ID:   builder.id,
				Name: builder.name,
				Args: parseInputArgs(builder.args),
			},
		})
	}

	if finishReason == genai.FinishReasonUnspecified {
		finishReason = genai.FinishReasonStop
	}

	finalResp := &model.LLMResponse{
		Content:       aggregatedContent,
		UsageMetadata: usageMetadata,
		FinishReason:  finishReason,
		Partial:       false,
		TurnComplete:  true,
	}
	yield(finalResp, nil)
}

// handleMessageStart 处理 message_start 事件
func (m *AnthropicModel) handleMessageStart(data string, usageMetadata **genai.GenerateContentResponseUsageMetadata) {
	var event MessageStartEvent
	if json.Unmarshal([]byte(data), &event) != nil {
		return
	}
	if event.Message.Usage != nil {
		*usageMetadata = &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount: int32(event.Message.Usage.InputTokens),
		}
	}
}

// handleContentBlockStart 处理 content_block_start 事件
func (m *AnthropicModel) handleContentBlockStart(data string, blockTypes map[int]string, toolCallsMap map[int]*toolCallBuilder) {
	var event ContentBlockStartEvent
	if json.Unmarshal([]byte(data), &event) != nil {
		return
	}
	blockTypes[event.Index] = event.ContentBlock.Type
	if event.ContentBlock.Type == "tool_use" {
		toolCallsMap[event.Index] = &toolCallBuilder{
			id:   event.ContentBlock.ID,
			name: event.ContentBlock.Name,
		}
	}
}

// handleContentBlockDelta 处理 content_block_delta 事件
func (m *AnthropicModel) handleContentBlockDelta(
	data string,
	blockTypes map[int]string,
	textContent *string,
	thinkingContent *string,
	toolCallsMap map[int]*toolCallBuilder,
	yield func(*model.LLMResponse, error) bool,
) {
	var event ContentBlockDeltaEvent
	if json.Unmarshal([]byte(data), &event) != nil {
		return
	}

	switch event.Delta.Type {
	case "text_delta":
		*textContent += event.Delta.Text
		// 发送部分响应用于实时 UI 更新
		yield(&model.LLMResponse{
			Content: &genai.Content{
				Role:  "model",
				Parts: []*genai.Part{{Text: event.Delta.Text}},
			},
			Partial:      true,
			TurnComplete: false,
		}, nil)

	case "thinking_delta":
		*thinkingContent += event.Delta.Thinking

	case "input_json_delta":
		if builder, ok := toolCallsMap[event.Index]; ok {
			builder.args += event.Delta.PartialJSON
		}
	}
}

// handleMessageDelta 处理 message_delta 事件
func (m *AnthropicModel) handleMessageDelta(
	data string,
	finishReason *genai.FinishReason,
	usageMetadata **genai.GenerateContentResponseUsageMetadata,
) {
	var event MessageDeltaEvent
	if json.Unmarshal([]byte(data), &event) != nil {
		return
	}
	if event.Delta.StopReason != "" {
		*finishReason = convertStopReason(event.Delta.StopReason)
	}
	if event.Usage != nil && *usageMetadata != nil {
		(*usageMetadata).CandidatesTokenCount = int32(event.Usage.OutputTokens)
		(*usageMetadata).TotalTokenCount = (*usageMetadata).PromptTokenCount + int32(event.Usage.OutputTokens)
	}
}

// handleError 处理 error 事件
func (m *AnthropicModel) handleError(data string, yield func(*model.LLMResponse, error) bool) {
	var errResp ErrorResponse
	if json.Unmarshal([]byte(data), &errResp) != nil {
		yield(nil, fmt.Errorf("Anthropic API 流式错误: %s", data))
		return
	}
	yield(nil, fmt.Errorf("Anthropic API 错误 (%s): %s", errResp.Error.Type, errResp.Error.Message))
}
