package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/client"
	"log"
	"mcpclient/config"
	"mcpclient/controllers"
	"mcpclient/llm"
	"mcpclient/middlewares"
	"mcpclient/utils"
	"time"
)

func SetupRouter() *gin.Engine {
	// 初始化一个引擎
	r := gin.New()
	// 注册全局中间件
	// 全局 CORS 中间件
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // 允许所有域（可改为指定域名）
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour, // 预检请求缓存时间
	}))
	//r.Use(middlewares.AuthMiddleWare())
	auth := r.Group("/api/auth")
	{
		auth.POST("/login", controllers.Loginuser)
		auth.POST("/register", controllers.RegisterUser)
	}
	// 注册中间件
	provider, ssemcpclients, allTools := mcpseeconfig("/home/chenyun/program/Go/mcptest/mcpclient/config/ssemcpserver.json")
	mongodb, err := config.ConnectMongoDB()
	if err != nil {
		log.Fatalln("连接MongoDB失败")
	}
	// 注册路由
	chat := r.Group("/api/chat")
	chat.Use(middlewares.LoadMCPSSEconfig(provider, ssemcpclients, allTools, mongodb))
	{
		chat.POST("/send", controllers.HandleUserPrompt2)
	}
	return r
}

func mcpseeconfig(path string) (llm.Provider, map[string]*client.SSEMCPClient, []llm.Tool) {
	// 初始化服务
	var modelFlag string
	modelsource := "ollama:"
	con := config.GetConfig()
	_, modelname := con.Getollama()
	modelFlag = modelsource + modelname
	provider, err := utils.CreateProvider(modelFlag)
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
	ssemcpclients, allTools, err := utils.GetSSEMCPClientsandTools(ssemcpconfig)

	return provider, ssemcpclients, allTools
}
