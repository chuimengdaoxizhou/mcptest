package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	Log "github.com/charmbracelet/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"golang.org/x/crypto/bcrypt"
	"log"
	"mcpclient/config"
	"mcpclient/llm"
	"mcpclient/llm/history"
	"mcpclient/llm/ollama"
	"mcpclient/models"
	"strings"
	"time"
)

const (
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
	maxRetries     = 5 // 最多重试次数
)

var (
	messageWindow int
)

// runPrompt 函数：负责发送用户输入的提示并处理 AI 模型的响应，
// 并通过 Channel 将响应结果输出到外部。
// 参数：
// - provider：llm.Provider，负责与 AI 模型进行交互的提供程序
// - mcpClients：map[string]*client.SSEMCPClient，MCP 客户端，用于工具调用
// - tools：[]llm.Tool，支持的工具列表
// - prompt：string，用户输入的提示内容
// - messages：*[]history.HistoryMessage，消息历史记录
// - responseChan：chan<- string，输出到外部的 Channel
func RunPromptmcp(
	provider llm.Provider, // llm 提供程序，处理 AI 模型请求
	mcpClients map[string]*client.SSEMCPClient, // MCP 客户端，执行工具调用
	tools []llm.Tool, // 支持的工具列表
	prompt string, // 用户输入的提示
	messages *[]history.HistoryMessage, // 消息历史记录
	responseChan chan<- string,
) error {
	// 如果 prompt 为空（即工具调用的返回结果），显示用户输入的提示
	if prompt != "" {
		*messages = append(
			*messages,
			history.HistoryMessage{
				Role: "user",
				Content: []history.ContentBlock{{
					Type: "text",
					Text: prompt,
				}},
			},
		)
	}

	var message llm.Message
	var err error
	backoff := initialBackoff // 初始重试间隔
	retries := 0              // 重试次数

	// 将 HistoryMessage 转换为 llm.Message
	llmMessages := make([]llm.Message, len(*messages))
	for i := range *messages {
		llmMessages[i] = &(*messages)[i]
	}

	fmt.Println("进入重试机制")
	// 重试机制：如果请求失败且是“过载”错误，则重试
	for {
		fmt.Printf("重试+++")
		message, err = provider.CreateMessage(
			context.Background(),
			prompt,
			llmMessages,
			tools,
		)
		if err != nil {
			// 检查是否为过载错误
			fmt.Println(err)
			if strings.Contains(err.Error(), "overloaded_error") {
				// 如果重试次数已达最大值，返回错误
				if retries >= maxRetries {
					return fmt.Errorf(
						"claude 当前过载，请稍等几分钟后再试",
					)
				}

				Log.Warn("Ollama 过载，正在退避...",
					"attempt", retries+1,
					"backoff", backoff.String())

				// 退避策略：增加重试间隔
				time.Sleep(backoff)
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				retries++
				continue
			}
			// 如果不是过载错误，直接返回该错误
			return err
		}
		// 如果没有出错，表示请求成功
		break
	}

	var messageContent []history.ContentBlock
	// 处理 AI 返回的文本内容
	if message.GetContent() != "" {
		// 将 AI 输出的内容写入 Channel，供外部程序使用
		//responseChan <- message.GetContent() + "\n"
		fmt.Println("处理 AI 输出的内容：" + message.GetContent() + "\n")
		writeToChannel(responseChan, message.GetContent()+"\n")
		// 将 AI 的文本内容存储在历史记录中
		messageContent = append(messageContent, history.ContentBlock{
			Type: "text",
			Text: message.GetContent(),
		})
	}
	toolResults := []history.ContentBlock{}
	messageContent = []history.ContentBlock{}
	// 处理工具调用
	for _, toolCall := range message.GetToolCalls() {
		fmt.Printf("处理 %s工具调用", toolCall.GetName())
		Log.Info("🔧 使用工具", "name", toolCall.GetName())

		input, _ := json.Marshal(toolCall.GetArguments()) // 序列化工具参数
		messageContent = append(messageContent, history.ContentBlock{
			Type:  "tool_use",
			ID:    toolCall.GetID(),
			Name:  toolCall.GetName(),
			Input: input,
		})

		// 如果有使用统计，记录日志
		inputTokens, outputTokens := message.GetUsage()
		if inputTokens > 0 || outputTokens > 0 {
			Log.Info("使用统计",
				"input_tokens", inputTokens,
				"output_tokens", outputTokens,
				"total_tokens", inputTokens+outputTokens)
		}

		// 分割工具名称
		parts := strings.Split(toolCall.GetName(), "__")
		if len(parts) != 2 {
			fmt.Printf(
				"错误: 无效的工具名称格式: %s\n",
				toolCall.GetName(),
			)
			continue
		}

		serverName, toolName := parts[0], parts[1]
		mcpClient, ok := mcpClients[serverName]
		if !ok {
			fmt.Printf("错误: 找不到服务器: %s\n", serverName)
			continue
		}

		var toolArgs map[string]interface{}
		if err := json.Unmarshal(input, &toolArgs); err != nil {
			fmt.Printf("解析工具参数时出错: %v\n", err)
			continue
		}

		var toolResultPtr *mcp.CallToolResult
		// 调用工具
		action := func() {
			req := mcp.CallToolRequest{}
			req.Params.Name = toolName
			req.Params.Arguments = toolArgs
			toolResultPtr, err = mcpClient.CallTool(
				context.Background(),
				req,
			)
		}
		action()

		// 如果调用失败，显示错误信息并记录工具调用结果
		if err != nil {
			errMsg := fmt.Sprintf(
				"调用工具 %s 时出错: %v",
				toolName,
				err,
			)
			fmt.Printf("调用工具出错：%s\n", errMsg)

			// 添加错误信息作为工具调用结果
			toolResults = append(toolResults, history.ContentBlock{
				Type:      "tool_result",
				ToolUseID: toolCall.GetID(),
				Content: []history.ContentBlock{{
					Type: "text",
					Text: errMsg,
				}},
			})
			continue
		}

		toolResult := *toolResultPtr

		if toolResult.Content != nil {

			fmt.Println("原始工具结果内容", "content", toolResult.Content)

			// 创建工具结果块
			resultBlock := history.ContentBlock{
				Type:      "tool_result",
				ToolUseID: toolCall.GetID(),
				Content:   toolResult.Content,
			}

			// 提取文本内容
			var resultText string
			// 处理数组内容
			for _, item := range toolResult.Content {
				// 处理文本内容
				if textContent, ok := item.(mcp.TextContent); ok {
					//resultText += fmt.Sprintf("%v ", textContent.Text)
					//responseChan <- resultText + "\n"
					writeToChannel(responseChan, textContent.Text)
				}
				// 处理其他数据类型
			}

			resultBlock.Text = strings.TrimSpace(resultText)
			Log.Debug("创建工具结果块",
				"block", resultBlock,
				"tool_id", toolCall.GetID())

			toolResults = append(toolResults, resultBlock)
		}
	}

	fmt.Println("将 AI 响应消息添加到历史记录,如果没有工具调用，推出")
	// 将 AI 响应消息添加到历史记录
	*messages = append(*messages, history.HistoryMessage{
		Role:    message.GetRole(),
		Content: messageContent,
	})

	// 如果有工具结果，继续执行
	if len(toolResults) > 0 {
		fmt.Println("继续调用工具")
		*messages = append(*messages, history.HistoryMessage{
			Role:    "user",
			Content: toolResults,
		})
		// 对工具结果再次调用 AI 获取响应
		return RunPromptmcp(provider, mcpClients, tools, "", messages, responseChan)
	}

	fmt.Println() // 输出换行
	return nil
}

func RunPrompt(
	provider llm.Provider, // llm 提供程序，处理 AI 模型请求
	prompt string, // 用户输入的提示
	messages *[]history.HistoryMessage, // 消息历史记录
	responseChan chan<- string, // 输出到外部的 Channel
) error {
	// 如果 prompt 为空（即工具调用的返回结果），显示用户输入的提示
	if prompt != "" {
		*messages = append(
			*messages,
			history.HistoryMessage{
				Role: "user",
				Content: []history.ContentBlock{{
					Type: "text",
					Text: prompt,
				}},
			},
		)
	}

	var message llm.Message
	var err error
	backoff := initialBackoff // 初始重试间隔
	retries := 0              // 重试次数
	tools := make([]llm.Tool, 0)
	// 将 HistoryMessage 转换为 llm.Message
	llmMessages := make([]llm.Message, len(*messages))
	for i := range *messages {
		llmMessages[i] = &(*messages)[i]
	}

	fmt.Println("进入重试机制")
	// 重试机制：如果请求失败且是“过载”错误，则重试
	for {
		fmt.Printf("重试+++")
		message, err = provider.CreateMessagestream(
			context.Background(),
			prompt,
			llmMessages,
			tools,
			responseChan,
		)
		fmt.Println("调用完成", err)
		if err != nil {
			// 检查是否为过载错误
			fmt.Println(err)
			if strings.Contains(err.Error(), "overloaded_error") {
				// 如果重试次数已达最大值，返回错误
				if retries >= maxRetries {
					return fmt.Errorf(
						"claude 当前过载，请稍等几分钟后再试",
					)
				}

				Log.Warn("Ollama 过载，正在退避...",
					"attempt", retries+1,
					"backoff", backoff.String())

				// 退避策略：增加重试间隔
				time.Sleep(backoff)
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				retries++
				continue
			}
			// 如果不是过载错误，直接返回该错误
			return err
		}
		// 如果没有出错，表示请求成功
		break
	}

	var messageContent []history.ContentBlock

	text := message.GetContent()

	if text != "" {
		messageContent = append(messageContent, history.ContentBlock{
			Type: "text",
			Text: text,
		})
	} else {
		fmt.Println("AI输出内容为空")
		return nil
	}
	// 将 AI 响应消息添加到历史记录
	*messages = append(*messages, history.HistoryMessage{
		Role:    message.GetRole(),
		Content: messageContent,
	})

	return nil
}

const chunkSize = 100

func writeToChannel(responseChan chan<- string, resultText string) {
	for len(resultText) > 0 {
		fmt.Println("resultText", resultText)
		if len(resultText) > chunkSize {
			responseChan <- resultText[:chunkSize]
			resultText = resultText[chunkSize:]
			fmt.Println("----------")
		} else {
			responseChan <- resultText
			fmt.Println("=========")
			break
		}
	}
}

func GetSSEMCPClientsandTools(
	config []models.SSEServerConfig,
) (map[string]*client.SSEMCPClient, []llm.Tool, error) {

	clients := make(map[string]*client.SSEMCPClient)
	var allTools []llm.Tool

	// 遍历所有 MCP 服务配置
	for _, server := range config {
		// 创建 MCP 客户端
		client, err := client.NewSSEMCPClient(server.MCPServerURL + "/sse")
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Second)
		defer cancel()

		// Start the client
		if err := client.Start(ctx); err != nil {
			log.Fatalf("Failed to start client: %v", err)
		}

		// Initialize
		initRequest := mcp.InitializeRequest{}
		initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
		initRequest.Params.ClientInfo = mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		}

		result, err := client.Initialize(ctx, initRequest)
		if err != nil {
			log.Fatalf("Failed to initialize: %v", err)
		}

		fmt.Printf("获取服务名称：" + result.ServerInfo.Name + " ")

		// Test Ping
		if err := client.Ping(ctx); err != nil {
			log.Fatalln("Ping failed: %v", err)
		}
		// 获取Tools
		ctx2, cancel2 := context.WithTimeout(context.Background(), 1000*time.Second)

		//mcpClient.Start(ctx)
		toolsResult, err := client.ListTools(ctx2, mcp.ListToolsRequest{})
		cancel2() // 取消请求

		if err != nil {
			fmt.Println(
				"获取工具列表时出错",
				"服务器", server.Name,
				"错误", err,
			)
			continue // 获取工具失败则跳过
		}

		// 将 MCP 工具转换为符合标准的工具列表
		serverTools := McpToolsToAnthropicTools(server.Name, toolsResult.Tools)
		allTools = append(allTools, serverTools...) // 合并工具
		Log.Info(
			"工具加载成功",
			"服务器", server.Name,
			"工具数量", len(toolsResult.Tools),
		)
		// 保存客户端到映射中
		clients[server.Name] = client
	}

	return clients, allTools, nil
}

// 将 MCP 工具列表转换为 Anthropic 工具格式
func McpToolsToAnthropicTools(
	serverName string, // 服务器名称
	mcpTools []mcp.Tool, // MCP 工具列表
) []llm.Tool {
	// 创建与 MCP 工具数量相同的 llm.Tool 列表
	anthropicTools := make([]llm.Tool, len(mcpTools))

	// 遍历 MCP 工具，进行格式转换
	for i, tool := range mcpTools {
		// 添加命名空间（格式为：服务器名称__工具名称）
		namespacedName := fmt.Sprintf("%s__%s", serverName, tool.Name)

		// 转换为 Anthropic 格式
		anthropicTools[i] = llm.Tool{
			Name:        namespacedName,
			Description: tool.Description,
			InputSchema: llm.Schema{
				Type:       tool.InputSchema.Type,
				Properties: tool.InputSchema.Properties,
				Required:   tool.InputSchema.Required,
			},
		}
	}

	return anthropicTools
}

// 新增的函数：创建模型提供商
func CreateProvider(modelString string) (llm.Provider, error) {
	parts := strings.SplitN(modelString, ":", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf(
			"模型格式无效，预期为提供商:模型，实际为 %s",
			modelString,
		)
	}

	provider := parts[0]
	model := parts[1]

	// 根据提供商选择相应的模型
	switch provider {

	case "ollama":
		return ollama.NewProvider(model)

	default:
		return nil, fmt.Errorf("不支持的提供商：%s", provider)
	}
}

// 修剪消息历史记录，确保不会超过指定的消息窗口大小
func PruneMessages(messages []history.HistoryMessage) []history.HistoryMessage {
	if len(messages) <= messageWindow {
		return messages
	}

	// 保留最近的消息
	messages = messages[len(messages)-messageWindow:]

	// 处理消息中的工具使用和工具结果
	toolUseIds := make(map[string]bool)
	toolResultIds := make(map[string]bool)

	// 第一次遍历：收集所有工具使用和结果的ID
	for _, msg := range messages {
		for _, block := range msg.Content {
			if block.Type == "tool_use" {
				toolUseIds[block.ID] = true
			} else if block.Type == "tool_result" {
				toolResultIds[block.ToolUseID] = true
			}
		}
	}

	// 第二次遍历：过滤掉孤立的工具调用/结果
	var prunedMessages []history.HistoryMessage
	for _, msg := range messages {
		var prunedBlocks []history.ContentBlock
		for _, block := range msg.Content {
			keep := true
			if block.Type == "tool_use" {
				keep = toolResultIds[block.ID]
			} else if block.Type == "tool_result" {
				keep = toolUseIds[block.ToolUseID]
			}
			if keep {
				prunedBlocks = append(prunedBlocks, block)
			}
		}
		// 仅包含有内容的消息，或者不是助手消息
		if (len(prunedBlocks) > 0 && msg.Role == "assistant") ||
			msg.Role != "assistant" {
			hasTextBlock := false
			for _, block := range msg.Content {
				if block.Type == "text" {
					hasTextBlock = true
					break
				}
			}
			if len(prunedBlocks) > 0 || hasTextBlock {
				msg.Content = prunedBlocks
				prunedMessages = append(prunedMessages, msg)
			}
		}
	}
	return prunedMessages
}
func Getproviderclientstools() (llm.Provider, map[string]*client.SSEMCPClient, []llm.Tool, error) {
	// 初始化服务
	var modelFlag string
	modelsource := "ollama:"
	con := config.GetConfig()
	_, modelname := con.Getollama()
	modelFlag = modelsource + modelname
	provider, err := CreateProvider(modelFlag)
	if err != nil {
		log.Fatalln("创建模型提供者时出错: %v", err) // 创建失败则返回错误
	}

	// 获取所有的mcpclients,allTools
	// 获取所有的mcpclients
	ssemcpconfigpath := "/home/chenyun/program/Go/mcptest/mcpclient/config/ssemcpserver.json"
	ssemcpconfig, err := config.LoadMCPConfig(ssemcpconfigpath)
	if err != nil {
		log.Fatalln("读取mcpconfig失败", err)
	}
	ssemcpclients, allTools, err := GetSSEMCPClientsandTools(ssemcpconfig)
	// 添加mcpclients,allTools
	return provider, ssemcpclients, allTools, nil
}

// 生成自定义的 _id
// userid 学号 3220921037
func GenerateCustomId(timestamp int64, userid string) string {
	// 拼接所有部分：时间戳（秒级） + 业务字符串 + 计数器
	id := fmt.Sprintf("%d-%s-%d", timestamp, userid)
	return id
}
func GenerateToken(userID int64, username string) (string, error) {
	claims := &models.Claims{
		UserID:   userID,
		UserName: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 12)), // 设置过期时间为 12 小时后
			IssuedAt:  jwt.NewNumericDate(time.Now()),                     // 设置签发时间
			NotBefore: jwt.NewNumericDate(time.Now()),                     // 设置生效时间
			Issuer:    "QASystem",                                         // 设置签发者
		},
	}

	// 使用指定的签名方式获取Token
	Token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 从配置文件中获取jwtSecret
	con := config.GetConfig()
	jwtSecret := con.GetsecretKey()

	// 使用密钥签名 Token 并获取完整编码后的字符串 token
	signedToken, err := Token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

func ParseJWT(tokenString string) (string, error) {
	con := config.GetConfig()
	secertKey := con.GetsecretKey()

	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("Unexpected Signing Method")
		}
		return []byte(secertKey), nil
	})
	if err != nil { // 错误
		fmt.Println("err", err)
		return "", err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		username, ok := claims["username"].(string)
		if !ok {
			return "", errors.New("Username claim is not a string")
		}
		return username, nil
	}
	return "", err
}
func GetHashPassword(password string) (string, error) {
	hashpassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(hashpassword), err
}

func CheckPassword(password string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
