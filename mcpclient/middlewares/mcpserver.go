package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/client"
	"go.mongodb.org/mongo-driver/mongo"
	"mcpclient/llm"
)

// 将clients放在ctx中
func LoadMCPSSEconfig(provider llm.Provider,
	ssemcpclients map[string]*client.SSEMCPClient,
	allTools []llm.Tool,
	mongodb *mongo.Collection) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set("provider", provider)
		ctx.Set("clients", ssemcpclients)
		ctx.Set("allTools", allTools)
		ctx.Set("mongodConnection", mongodb)
		ctx.Next()
	}
}
