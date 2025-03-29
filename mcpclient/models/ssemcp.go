package models

// ServerConfig 定义 MCP 服务器的配置
type SSEServerConfig struct {
	Name         string `json:"name"`
	MCPServerURL string `json:"url"`
}
