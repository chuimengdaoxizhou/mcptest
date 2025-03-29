package ollama

import (
	"fmt"
	"strings"
	"time"

	api "github.com/ollama/ollama/api"
	"mcpclient/llm"
)

// OllamaMessage 将 Ollama 的消息格式适配为我们自己的 Message 接口
type OllamaMessage struct {
	Message    api.Message // 储存 Ollama 消息
	ToolCallID string      // 单独存储工具调用 ID，因为 Ollama API 并没有这个字段
}

// 获取消息的角色
func (m *OllamaMessage) GetRole() string {
	return m.Message.Role
}

// 获取消息内容（去掉两端的空白字符）
func (m *OllamaMessage) GetContent() string {
	// 对于工具响应和常规消息，直接返回内容字符串
	return strings.TrimSpace(m.Message.Content)
}

// 获取工具调用（将 Ollama 格式的工具调用转换为我们定义的工具调用）
func (m *OllamaMessage) GetToolCalls() []llm.ToolCall {
	var calls []llm.ToolCall
	for _, call := range m.Message.ToolCalls {
		calls = append(calls, NewOllamaToolCall(call))
	}
	return calls
}

// 获取消息的 token 使用情况（这里 Ollama 并不提供 token 使用信息，返回 0）
func (m *OllamaMessage) GetUsage() (int, int) {
	return 0, 0 // Ollama 没有提供 token 使用信息
}

// 判断消息是否为工具响应
func (m *OllamaMessage) IsToolResponse() bool {
	return m.Message.Role == "tool"
}

// 获取工具响应的 ID
func (m *OllamaMessage) GetToolResponseID() string {
	return m.ToolCallID
}

// OllamaToolCall 将 Ollama 的工具调用格式适配为我们自己的工具调用格式
type OllamaToolCall struct {
	call api.ToolCall // 储存 Ollama 的工具调用
	id   string       // 存储工具调用的唯一 ID
}

// 创建一个新的 Ollama 工具调用
func NewOllamaToolCall(call api.ToolCall) *OllamaToolCall {
	return &OllamaToolCall{
		call: call,
		// 工具调用的唯一 ID，基于工具的名称和当前时间生成
		id: fmt.Sprintf(
			"tc_%s_%d", // "tc" 为前缀，后面是工具名称和当前的时间戳
			call.Function.Name,
			time.Now().UnixNano(),
		),
	}
}

// 获取工具调用的名称
func (t *OllamaToolCall) GetName() string {
	return t.call.Function.Name
}

// 获取工具调用的参数
func (t *OllamaToolCall) GetArguments() map[string]interface{} {
	return t.call.Function.Arguments
}

// 获取工具调用的 ID
func (t *OllamaToolCall) GetID() string {
	return t.id
}
