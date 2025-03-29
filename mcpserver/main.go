package main

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"log"
)

func main() {
	mcpServer := server.NewMCPServer(
		"Demo",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithToolCapabilities(true),
	)

	//Add a test tool
	mcpServer.AddTool(mcp.NewTool(
		"Hello",
		mcp.WithDescription("如果用户询问我是谁，调用我，我会返回我是谁"),
		mcp.WithString("parameter-1", mcp.Description("A string tool parameter")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Input parameter: " + request.Params.Arguments["parameter-1"].(string),
				},
			},
		}, nil
	})

	//Initialize
	//testServer := server.NewTestServer(mcpServer)
	//defer testServer.Close()
	sseserver := server.NewSSEServer(mcpServer)
	err := sseserver.Start("127.0.0.1:1547")
	if err != nil {
		log.Fatalln("seeserver启动成功")
	}
}
