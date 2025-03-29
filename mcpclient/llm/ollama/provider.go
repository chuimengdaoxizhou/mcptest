package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	api "github.com/ollama/ollama/api"
	"mcpclient/llm"
	"mcpclient/llm/history"
)

// boolPtr 将布尔值转换为指针
func boolPtr(b bool) *bool {
	return &b
}

// Provider 实现了 Ollama 提供者接口
type Provider struct {
	client *api.Client // 与 Ollama API 的客户端连接
	model  string      // 使用的模型名称
}

// NewProvider 创建一个新的 Ollama 提供者实例
func NewProvider(model string) (*Provider, error) {
	// 从环境变量中创建 Ollama 客户端
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, err
	}
	return &Provider{
		client: client, // 初始化客户端
		model:  model,  // 设置模型
	}, nil
}
func (p *Provider) CreateMessagestream(
	ctx context.Context,
	prompt string,
	messages []llm.Message,
	tools []llm.Tool,
	contentChan chan<- string, // 新增参数：用于流式传输内容的通道
) (llm.Message, error) {
	// 日志记录信息
	log.Debug("creating message",
		"prompt", prompt,
		"num_messages", len(messages),
		"num_tools", len(tools))

	// 将传入的消息转换为 Ollama 格式
	ollamaMessages := make([]api.Message, 0, len(messages)+1)
	for _, msg := range messages {
		if msg.IsToolResponse() {
			var content string
			if historyMsg, ok := msg.(*history.HistoryMessage); ok {
				for _, block := range historyMsg.Content {
					if block.Type == "tool_result" {
						content = block.Text
						break
					}
				}
			}
			if content == "" {
				content = msg.GetContent()
			}
			if content == "" {
				continue
			}
			ollamaMessages = append(ollamaMessages, api.Message{
				Role:    "tool",
				Content: content,
			})
			continue
		}
		if msg.GetContent() == "" && len(msg.GetToolCalls()) == 0 {
			continue
		}
		ollamaMsg := api.Message{
			Role:    msg.GetRole(),
			Content: msg.GetContent(),
		}
		if msg.GetRole() == "assistant" {
			for _, call := range msg.GetToolCalls() {
				if call.GetName() != "" {
					args := call.GetArguments()
					ollamaMsg.ToolCalls = append(ollamaMsg.ToolCalls, api.ToolCall{
						Function: api.ToolCallFunction{
							Name:      call.GetName(),
							Arguments: args,
						},
					})
				}
			}
		}
		ollamaMessages = append(ollamaMessages, ollamaMsg)
	}

	if prompt != "" {
		ollamaMessages = append(ollamaMessages, api.Message{
			Role:    "user",
			Content: prompt,
		})
	}

	// 将工具转换为 Ollama 格式
	ollamaTools := make([]api.Tool, len(tools))
	for i, tool := range tools {
		ollamaTools[i] = api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters: struct {
					Type       string   `json:"type"`
					Required   []string `json:"required"`
					Properties map[string]struct {
						Type        string   `json:"type"`
						Description string   `json:"description"`
						Enum        []string `json:"enum,omitempty"`
					} `json:"properties"`
				}{
					Type:       tool.InputSchema.Type,
					Required:   tool.InputSchema.Required,
					Properties: convertProperties(tool.InputSchema.Properties),
				},
			},
		}
	}

	// 日志记录请求发送信息
	log.Debug("sending messages to Ollama",
		"messages", ollamaMessages,
		"num_tools", len(tools))

	// 流式处理响应
	var sb strings.Builder
	var role string
	var response api.Message

	err := p.client.Chat(ctx, &api.ChatRequest{
		Model:    p.model,
		Messages: ollamaMessages,
		Tools:    ollamaTools,
		Stream:   boolPtr(true), // 启用流式传输
	}, func(r api.ChatResponse) error {
		if r.Message.Content != "" {
			if role == "" { // 仅从第一个分片获取角色
				role = r.Message.Role
			}
			// 将分片内容写入通道
			contentChan <- r.Message.Content
			sb.WriteString(r.Message.Content) // 累积内容以返回完整消息
		}
		if r.Done {
			response = api.Message{
				Role:    role,
				Content: sb.String(),
			}
			close(contentChan) // 流结束时关闭通道
		}
		return nil
	})

	if err != nil {
		close(contentChan) // 出现错误时关闭通道
		return nil, err
	}

	return &OllamaMessage{Message: response}, nil
}

// CreateMessage 创建并返回一条消息
func (p *Provider) CreateMessage(
	ctx context.Context,
	prompt string,
	messages []llm.Message,
	tools []llm.Tool,
) (llm.Message, error) {
	// 日志记录信息
	log.Debug("creating message",
		"prompt", prompt,
		"num_messages", len(messages),
		"num_tools", len(tools))

	// 将传入的消息转换为 Ollama 格式
	ollamaMessages := make([]api.Message, 0, len(messages)+1)

	// 添加现有的消息
	for _, msg := range messages {
		// 如果是工具响应消息
		if msg.IsToolResponse() {
			var content string

			// 如果消息是历史记录格式
			if historyMsg, ok := msg.(*history.HistoryMessage); ok {
				for _, block := range historyMsg.Content {
					// 如果是工具结果类型的内容块
					if block.Type == "tool_result" {
						content = block.Text
						break
					}
				}
			}

			// 如果没有找到内容，再尝试提取标准内容
			if content == "" {
				content = msg.GetContent()
			}

			// 如果没有内容，跳过该消息
			if content == "" {
				continue
			}

			// 将工具响应消息添加到 Ollama 消息中
			ollamaMsg := api.Message{
				Role:    "tool", // 角色设置为 "tool"
				Content: content,
			}
			ollamaMessages = append(ollamaMessages, ollamaMsg)
			continue
		}

		// 跳过完全没有内容且没有工具调用的消息
		if msg.GetContent() == "" && len(msg.GetToolCalls()) == 0 {
			continue
		}

		// 创建新的 Ollama 消息
		ollamaMsg := api.Message{
			Role:    msg.GetRole(),
			Content: msg.GetContent(),
		}

		// 如果是助手角色的消息，添加工具调用信息
		if msg.GetRole() == "assistant" {
			for _, call := range msg.GetToolCalls() {
				// 只有工具调用名称非空时才添加
				if call.GetName() != "" {
					args := call.GetArguments()
					ollamaMsg.ToolCalls = append(
						ollamaMsg.ToolCalls,
						api.ToolCall{
							Function: api.ToolCallFunction{
								Name:      call.GetName(),
								Arguments: args,
							},
						},
					)
				}
			}
		}

		// 将构建好的 Ollama 消息添加到消息列表中
		ollamaMessages = append(ollamaMessages, ollamaMsg)
	}

	// 如果 prompt 不为空，添加 prompt 消息
	if prompt != "" {
		ollamaMessages = append(ollamaMessages, api.Message{
			Role:    "user", // 角色设置为 "user"
			Content: prompt,
		})
	}

	// 将工具转换为 Ollama 格式
	ollamaTools := make([]api.Tool, len(tools))
	for i, tool := range tools {
		ollamaTools[i] = api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters: struct {
					Type       string   `json:"type"`
					Required   []string `json:"required"`
					Properties map[string]struct {
						Type        string   `json:"type"`
						Description string   `json:"description"`
						Enum        []string `json:"enum,omitempty"`
					} `json:"properties"`
				}{
					Type:       tool.InputSchema.Type,
					Required:   tool.InputSchema.Required,
					Properties: convertProperties(tool.InputSchema.Properties),
				},
			},
		}
	}

	// 日志记录请求发送信息
	log.Debug("sending messages to Ollama",
		"messages", ollamaMessages,
		"num_tools", len(tools))

	// 向 Ollama API 发送请求并获取响应
	var response api.Message
	err := p.client.Chat(ctx, &api.ChatRequest{
		Model:    p.model,
		Messages: ollamaMessages,
		Tools:    ollamaTools,
		Stream:   boolPtr(false),
	}, func(r api.ChatResponse) error {
		// 获取消息响应
		if r.Done {
			response = r.Message
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 返回 Ollama 消息
	return &OllamaMessage{Message: response}, nil
}

// SupportsTools 检查模型是否支持工具调用功能
func (p *Provider) SupportsTools() bool {
	// 向 Ollama 请求模型信息，检查是否支持工具功能
	resp, err := p.client.Show(context.Background(), &api.ShowRequest{
		Model: p.model,
	})
	if err != nil {
		return false
	}
	// 如果模型文件中包含 "<tools>"，则认为支持工具调用
	return strings.Contains(resp.Modelfile, "<tools>")
}

// Name 返回提供者的名称
func (p *Provider) Name() string {
	return "ollama"
}

// CreateToolResponse 创建并返回工具响应消息
func (p *Provider) CreateToolResponse(
	toolCallID string,
	content interface{},
) (llm.Message, error) {
	// 日志记录工具响应信息
	log.Debug("creating tool response",
		"tool_call_id", toolCallID,
		"content_type", fmt.Sprintf("%T", content),
		"content", content)

	// 将工具响应的内容转换为字符串
	contentStr := ""
	switch v := content.(type) {
	case string:
		contentStr = v
		log.Debug("using string content directly")
	default:
		// 如果内容不是字符串，则将其转换为 JSON 字符串
		bytes, err := json.Marshal(v)
		if err != nil {
			log.Error("failed to marshal tool response",
				"error", err,
				"content", content)
			return nil, fmt.Errorf("error marshaling tool response: %w", err)
		}
		contentStr = string(bytes)
		log.Debug("marshaled content to JSON string",
			"result", contentStr)
	}

	// 创建工具响应消息
	msg := &OllamaMessage{
		Message: api.Message{
			Role:    "tool", // 设置角色为 "tool"
			Content: contentStr,
			// 工具响应不需要设置 ToolCalls
		},
		ToolCallID: toolCallID, // 设置工具调用 ID
	}

	// 日志记录创建的工具响应消息
	log.Debug("created tool response message",
		"role", msg.GetRole(),
		"content", msg.GetContent(),
		"tool_call_id", msg.GetToolResponseID(),
		"raw_content", contentStr)

	// 返回消息
	return msg, nil
}

// convertProperties 将工具的属性转换为 Ollama 格式
func convertProperties(props map[string]interface{}) map[string]struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
} {
	result := make(map[string]struct {
		Type        string   `json:"type"`
		Description string   `json:"description"`
		Enum        []string `json:"enum,omitempty"`
	})

	// 遍历属性并将其转换为 Ollama 格式
	for name, prop := range props {
		if propMap, ok := prop.(map[string]interface{}); ok {
			prop := struct {
				Type        string   `json:"type"`
				Description string   `json:"description"`
				Enum        []string `json:"enum,omitempty"`
			}{
				Type:        getString(propMap, "type"),
				Description: getString(propMap, "description"),
			}

			// 如果属性有枚举值，提取枚举值
			if enumRaw, ok := propMap["enum"].([]interface{}); ok {
				for _, e := range enumRaw {
					if str, ok := e.(string); ok {
						prop.Enum = append(prop.Enum, str)
					}
				}
			}

			// 将转换后的属性添加到结果中
			result[name] = prop
		}
	}

	return result
}

// getString 从 map 中安全地获取字符串值
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
