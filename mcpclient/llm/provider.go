package llm

import (
	"context"
)

// ==========================
// 定义 Message 接口
// ==========================

// Message 表示在对话中传递的消息
// 这个接口定义了 LLM（大语言模型）中消息的基本结构和行为，
// 包括消息的角色、内容、工具调用等信息。
type Message interface {
	// GetRole 返回消息发送者的角色（例如："user"、"assistant"、"system"）
	// - "user"：表示用户发送的消息
	// - "assistant"：表示助手（LLM）生成的回复
	// - "system"：表示系统级别的提示（如上下文或规则）
	GetRole() string

	// GetContent 返回消息的文本内容
	// 例如：用户发送的文本或助手的回复内容
	GetContent() string

	// GetToolCalls 返回在消息中触发的工具调用（可能为空）
	// 工具调用是 LLM 在对话过程中调用外部工具（API、函数等）的操作
	GetToolCalls() []ToolCall

	// IsToolResponse 返回当前消息是否是某个工具的响应
	// - true：表示该消息是工具调用的结果
	// - false：表示该消息是正常的文本消息
	IsToolResponse() bool

	// GetToolResponseID 返回工具调用的 ID
	// 该 ID 标识了当前消息是对哪个工具调用的响应
	// 只有在 IsToolResponse() 返回 true 时才有意义
	GetToolResponseID() string

	// GetUsage 返回消息的 Token 使用情况（输入和输出的 Token 数）
	// 该方法用于统计 LLM 的消耗（如 OpenAI 的 token 计费）
	GetUsage() (input int, output int)
}

// ==========================
// 定义 ToolCall 接口
// ==========================

// ToolCall 表示一次工具调用
// 工具调用是 LLM 与外部函数或接口的交互方式，
// 例如：在对话中调用外部 API 进行数据查询或执行计算。
type ToolCall interface {
	// GetName 返回工具的名称
	// 例如："weather" 表示天气查询工具
	GetName() string

	// GetArguments 返回传递给工具的参数
	// 例如：{ "city": "Beijing", "unit": "Celsius" }
	GetArguments() map[string]interface{}

	// GetID 返回该工具调用的唯一标识符
	// 该 ID 用于在工具调用和响应之间建立关联
	GetID() string
}

// ==========================
// 定义 Tool 结构体
// ==========================

// Tool 表示工具的定义
// LLM 在生成响应时，可能会调用定义的工具，
// 这个结构体描述了工具的基本信息和输入参数。
type Tool struct {
	// Name 表示工具的名称（唯一标识）
	// 例如："translate" 表示翻译工具
	Name string `json:"name"`

	// Description 表示工具的描述（用于提示 LLM）
	// 例如："Translate text from one language to another"
	Description string `json:"description"`

	// InputSchema 表示工具的输入参数定义
	// 通过 Schema 定义了工具输入的结构、类型和必要参数
	InputSchema Schema `json:"input_schema"`
}

// ==========================
// 定义 Schema 结构体
// ==========================

// Schema 定义工具的输入参数结构
// 该结构用于描述工具所需的输入数据格式
type Schema struct {
	// Type 表示输入参数的数据类型
	// 可能的值：object、array、string、number、boolean 等
	Type string `json:"type"`

	// Properties 定义输入参数的结构和类型
	// 例如：
	// {
	//   "city": { "type": "string", "description": "City name" },
	//   "unit": { "type": "string", "description": "Temperature unit" }
	// }
	Properties map[string]interface{} `json:"properties"`

	// Required 定义必要的参数
	// 例如：["city", "unit"] 表示 city 和 unit 是必需的参数
	Required []string `json:"required"`
}

// ==========================
// 定义 Provider 接口
// ==========================

// Provider 定义了 LLM 服务提供者的接口
// LLM 的不同服务（如 OpenAI、Anthropic、Mistral）需要实现这个接口，
// 这样可以通过统一的方式调用不同的模型服务。
type Provider interface {
	// CreateMessage 发送一个消息到 LLM，并返回 LLM 的响应
	// - ctx：用于控制超时和取消操作的上下文对象
	// - prompt：为 LLM 提供的初始提示词
	// - messages：对话的上下文消息列表（用于保持对话状态）
	// - tools：LLM 可使用的工具列表（用于功能调用）
	// 返回值：
	// - Message：LLM 生成的响应消息
	// - error：如果发生错误，返回具体的错误信息
	CreateMessage(ctx context.Context, prompt string, messages []Message, tools []Tool) (Message, error)
	CreateMessagestream(ctx context.Context, prompt string, messages []Message, tools []Tool, contentChan chan<- string) (Message, error)
	// CreateToolResponse 创建一个工具调用的响应消息
	// - toolCallID：工具调用的唯一标识符
	// - content：工具调用的结果数据（格式可能是字符串或 JSON）
	// 返回值：
	// - Message：表示工具响应的消息对象
	// - error：如果发生错误，返回具体的错误信息
	CreateToolResponse(toolCallID string, content interface{}) (Message, error)

	// SupportsTools 返回当前 LLM 是否支持工具调用
	// 例如：OpenAI 的 GPT-4 支持工具调用，但部分模型不支持
	// - true：支持工具调用
	// - false：不支持工具调用
	SupportsTools() bool

	// Name 返回当前提供者的名称
	// 例如："OpenAI", "Anthropic", "Mistral"
	Name() string
}
