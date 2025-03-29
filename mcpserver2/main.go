package main

import (
	"github.com/mark3labs/mcp-go/server"
	"log"
)

func main() {
	mcpServer := server.NewMCPServer(
		"Demo2",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithToolCapabilities(true),
	)

	//Add a test tool
	//mcpServer.AddTool(mcp.NewTool(
	//	"assagent",
	//	mcp.WithDescription("assagent"),
	//	mcp.WithString("parameter-1", mcp.Description("A string tool parameter")),
	//), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	//	return &mcp.CallToolResult{
	//		Content: []mcp.Content{
	//			mcp.TextContent{
	//				Type: "text",
	//				Text: "Input parameter: " + request.Params.Arguments["parameter-1"].(string),
	//			},
	//		},
	//	}, nil
	//})

	//Initialize
	//testServer := server.NewTestServer(mcpServer)
	//defer testServer.Close()
	sseserver := server.NewSSEServer(mcpServer)
	err := sseserver.Start("127.0.0.1:1548")
	if err != nil {
		log.Fatalln("seeserver启动成功")
	}
}
