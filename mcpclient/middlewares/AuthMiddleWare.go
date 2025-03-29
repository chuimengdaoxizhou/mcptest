package middlewares

import (
	"github.com/gin-gonic/gin"
	"mcpclient/utils"
	"net/http"
)

func AuthMiddleWare() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := ctx.GetHeader("Authorization")
		if token == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization"})
			ctx.Abort()
			return
		}
		username, err := utils.ParseJWT(token)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			ctx.Abort()
			return
		}

		ctx.Set("username", username)
		ctx.Next()
	}
}
