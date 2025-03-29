package models

import "github.com/golang-jwt/jwt/v5"

type Claims struct {
	UserID               int64  `json:"userid"`
	UserName             string `json:"username"`
	jwt.RegisteredClaims        // 包含标准的 JWT 声明
}
