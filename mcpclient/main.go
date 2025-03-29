package main

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"log"
	"mcpclient/router"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	//设置 Gin 运行模式（可选: gin.ReleaseMode, gin.DebugMode, gin.TestMode）
	gin.SetMode(gin.ReleaseMode)

	// 初始化路由
	r := router.SetupRouter()

	// 服务器配置
	server := &http.Server{
		Addr:    "localhost:8080", // 监听端口
		Handler: r,                // 使用 Gin 路由
		//ReadTimeout:    10 * time.Second, // 读超时
		//WriteTimeout:   30 * time.Second, // 写超时
		MaxHeaderBytes: 1 << 20, // 1MB 请求头限制
	}

	// 使用 goroutine 启动服务器
	go func() {
		log.Println("服务器启动，监听端口 8080...")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("服务器启动失败:", err)
		}
	}()
	// 监听系统信号 (如 Ctrl+C 或 kill 命令)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt) // 监听 SIGINT（Ctrl+C）信号

	// 阻塞，直到接收到终止信号
	<-quit
	log.Println("收到终止信号，正在关闭服务器...")

	// 创建超时上下文，确保服务器在 5 秒内关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 优雅地关闭服务器
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("服务器关闭失败:", err)
	}

	log.Println("服务器已成功关闭")
}

//client, err := client.NewSSEMCPClient("http://127.0.0.1:1547" + "/sse")
//if err != nil {
//	log.Fatalf("Failed to create client: %v", err)
//}
//defer client.Close()
//
//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//defer cancel()
//
//// Start the client
//if err := client.Start(ctx); err != nil {
//	log.Fatalf("Failed to start client: %v", err)
//}
//
//// Initialize
//initRequest := mcp.InitializeRequest{}
//initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
//initRequest.Params.ClientInfo = mcp.Implementation{
//	Name:    "test-client",
//	Version: "1.0.0",
//}
//
//result, err := client.Initialize(ctx, initRequest)
//if err != nil {
//	log.Fatalf("Failed to initialize: %v", err)
//}
//
//if result.ServerInfo.Name != "Demo" {
//	log.Println(
//		"Expected server name 'test-server', got '%s'",
//		result.ServerInfo.Name,
//	)
//}
//
//// Test Ping
//if err := client.Ping(ctx); err != nil {
//	log.Fatalln("Ping failed: %v", err)
//}
//
//// Test ListTools
//toolsRequest := mcp.ListToolsRequest{}
//tools, err := client.ListTools(ctx, toolsRequest)
//fmt.Println(tools)
//if err != nil {
//	log.Println("ListTools failed: %v", err)
//}
