package anthropic

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// toMessagesRequest 将 ADK 请求转换为 Anthropic Messages API 请求
func toMessagesRequest(req *model.LLMRequest, modelName string, maxTokens int) (MessagesRequest, error) {
	apiReq := MessagesRequest{
		Model:     modelName,
		MaxTokens: maxTokens,
	}

	// 提取系统指令
	if req.Config != nil && req.Config.SystemInstruction != nil {
		apiReq.System = extractSystemText(req.Config.SystemInstruction)
	}

	// 转换消息
	messages, err := toAnthropicMessages(req.Contents)
	if err != nil {
		return MessagesRequest{}, err
	}
	apiReq.Messages = messages

	// 转换工具
	if req.Config != nil && len(req.Config.Tools) > 0 {
		apiReq.Tools = convertTools(req.Config.Tools)
	}

	// 应用配置参数
	if req.Config != nil {
		if req.Config.Temperature != nil {
			t := float64(*req.Config.Temperature)
			apiReq.Temperature = &t
		}
		if req.Config.MaxOutputTokens > 0 {
			apiReq.MaxTokens = int(req.Config.MaxOutputTokens)
		}
		if req.Config.TopP != nil {
			p := float64(*req.Config.TopP)
			apiReq.TopP = &p
		}
		if len(req.Config.StopSequences) > 0 {
			apiReq.StopSequences = req.Config.StopSequences
		}
	}

	return apiReq, nil
}

// toAnthropicMessages 将 ADK Contents 转换为 Anthropic 消息列表
// 关键：Anthropic 要求严格交替的 user/assistant 消息
func toAnthropicMessages(contents []*genai.Content) ([]Message, error) {
	var raw []Message

	for _, content := range contents {
		if content == nil {
			continue
		}
		role := convertRole(content.Role)
		blocks, err := toContentBlocks(content)
		if err != nil {
			return nil, err
		}
		if len(blocks) == 0 {
			continue
		}
		raw = append(raw, Message{Role: role, Content: blocks})
	}

	// 合并连续相同角色的消息（Anthropic 要求交替）
	return mergeConsecutiveMessages(raw), nil
}

// toContentBlocks 将 genai.Content 的 Parts 转换为 Anthropic ContentBlock 列表
func toContentBlocks(content *genai.Content) ([]ContentBlock, error) {
	var blocks []ContentBlock

	for _, part := range content.Parts {
		if part == nil {
			continue
		}

		// thinking 内容
		if part.Thought && part.Text != "" {
			blocks = append(blocks, ContentBlock{
				Type:     "thinking",
				Thinking: part.Text,
			})
			continue
		}

		// 普通文本
		if part.Text != "" {
			blocks = append(blocks, ContentBlock{
				Type: "text",
				Text: part.Text,
			})
			continue
		}

		// 函数调用 -> tool_use
		if part.FunctionCall != nil {
			blocks = append(blocks, ContentBlock{
				Type:  "tool_use",
				ID:    part.FunctionCall.ID,
				Name:  part.FunctionCall.Name,
				Input: part.FunctionCall.Args,
			})
			continue
		}

		// 函数响应 -> tool_result
		if part.FunctionResponse != nil {
			resultJSON, err := json.Marshal(part.FunctionResponse.Response)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal function response: %w", err)
			}
			blocks = append(blocks, ContentBlock{
				Type:      "tool_result",
				ToolUseID: part.FunctionResponse.ID,
				Content:   string(resultJSON),
			})
			continue
		}
	}

	return blocks, nil
}

// mergeConsecutiveMessages 合并连续相同角色的消息
func mergeConsecutiveMessages(messages []Message) []Message {
	if len(messages) <= 1 {
		return messages
	}

	var merged []Message
	for _, msg := range messages {
		if len(merged) > 0 && merged[len(merged)-1].Role == msg.Role {
			// 合并内容块
			last := &merged[len(merged)-1]
			prevBlocks := toBlockSlice(last.Content)
			curBlocks := toBlockSlice(msg.Content)
			last.Content = append(prevBlocks, curBlocks...)
		} else {
			merged = append(merged, msg)
		}
	}
	return merged
}

// toBlockSlice 将 Message.Content (any) 转换为 []ContentBlock
func toBlockSlice(content any) []ContentBlock {
	switch v := content.(type) {
	case []ContentBlock:
		return v
	case string:
		return []ContentBlock{{Type: "text", Text: v}}
	default:
		return nil
	}
}

// convertRole 转换 ADK 角色到 Anthropic 角色
func convertRole(role string) string {
	switch role {
	case "model":
		return "assistant"
	case "user":
		return "user"
	default:
		return "user"
	}
}

// extractSystemText 从 genai.Content 提取系统指令文本
func extractSystemText(content *genai.Content) string {
	if content == nil {
		return ""
	}
	var texts []string
	for _, part := range content.Parts {
		if part != nil && part.Text != "" {
			texts = append(texts, part.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// convertTools 转换 ADK 工具定义为 Anthropic 格式
func convertTools(genaiTools []*genai.Tool) []ToolDefinition {
	var tools []ToolDefinition
	for _, t := range genaiTools {
		if t == nil {
			continue
		}
		for _, fd := range t.FunctionDeclarations {
			schema := fd.ParametersJsonSchema
			if schema == nil {
				schema = fd.Parameters
			}
			tools = append(tools, ToolDefinition{
				Name:        fd.Name,
				Description: fd.Description,
				InputSchema: schema,
			})
		}
	}
	return tools
}

// convertResponse 将 Anthropic 响应转换为 ADK 响应
func convertResponse(resp *MessagesResponse) (*model.LLMResponse, error) {
	content := &genai.Content{
		Role:  genai.RoleModel,
		Parts: []*genai.Part{},
	}

	for _, block := range resp.Content {
		switch block.Type {
		case "thinking":
			content.Parts = append(content.Parts, &genai.Part{
				Text:    block.Thinking,
				Thought: true,
			})
		case "text":
			content.Parts = append(content.Parts, &genai.Part{
				Text: block.Text,
			})
		case "tool_use":
			args := parseInputArgs(block.Input)
			content.Parts = append(content.Parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					ID:   block.ID,
					Name: block.Name,
					Args: args,
				},
			})
		}
	}

	var usageMetadata *genai.GenerateContentResponseUsageMetadata
	if resp.Usage != nil {
		usageMetadata = &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     int32(resp.Usage.InputTokens),
			CandidatesTokenCount: int32(resp.Usage.OutputTokens),
			TotalTokenCount:      int32(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		}
	}

	return &model.LLMResponse{
		Content:       content,
		UsageMetadata: usageMetadata,
		FinishReason:  convertStopReason(resp.StopReason),
		TurnComplete:  true,
	}, nil
}

// convertStopReason 转换 Anthropic stop_reason 为 ADK FinishReason
func convertStopReason(reason string) genai.FinishReason {
	switch reason {
	case "end_turn":
		return genai.FinishReasonStop
	case "max_tokens":
		return genai.FinishReasonMaxTokens
	case "stop_sequence":
		return genai.FinishReasonStop
	case "tool_use":
		return genai.FinishReasonStop
	default:
		return genai.FinishReasonUnspecified
	}
}

// parseInputArgs 解析工具调用的 input 参数
func parseInputArgs(input any) map[string]any {
	if input == nil {
		return make(map[string]any)
	}
	switch v := input.(type) {
	case map[string]any:
		return v
	case string:
		var args map[string]any
		if err := json.Unmarshal([]byte(v), &args); err != nil {
			return make(map[string]any)
		}
		return args
	default:
		// 尝试通过 JSON 序列化/反序列化转换
		data, err := json.Marshal(v)
		if err != nil {
			return make(map[string]any)
		}
		var args map[string]any
		if err := json.Unmarshal(data, &args); err != nil {
			return make(map[string]any)
		}
		return args
	}
}
