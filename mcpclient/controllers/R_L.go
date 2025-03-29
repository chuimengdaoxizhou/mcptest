package controllers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"log"
	"mcpclient/config"
	"mcpclient/models"
	"mcpclient/utils"
	"net/http"
)

func RegisterUser(ctx *gin.Context) {
	// 初始化一个用户对象
	var user models.User

	// 将客户端传入的用户信息注册到user中
	if err := ctx.ShouldBindBodyWithJSON(&user); err != nil {
		// 如果注册失败
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Fatalln("获取用户输入信息失败")
		return
	}

	// 初始化数据库
	db := config.InitDB()
	err := db.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatalln("在mysql中创建User表失败，controllers.RegisterUser")
		return
	}

	// 初始化一个临时用户信息变量，存储数据库查找的信息
	var u models.User
	result := db.Where("UserName = ?", user.UserName).First(&u)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) || u.DeletedAt.Valid == false { // 如果数据库中没有找到
		// 加密用户密码
		if user.Password, err = utils.GetHashPassword(user.Password); err != nil {
			// 加密密码失败
			ctx.JSON(http.StatusFailedDependency, gin.H{"error": "服务器出错"})
			log.Fatalf("%s注册账户时，加密密码失败", user.UserName)
		}
		// 注册用户到数据库中
		res := db.Create(user)
		if res.Error != nil {
			ctx.JSON(http.StatusFailedDependency, gin.H{"errors": "服务器出错"})
			log.Fatalln("用户数据注册到数据库时失败，controllers.RegisterUser")
		} else {
			// 注册成功
			ctx.JSON(http.StatusOK, gin.H{})
			log.Fatalf("%s注册成功", user.UserName)
		}
	} else {
		// 用户已经被注册，而且没有被软删除
		ctx.JSON(http.StatusFailedDependency, gin.H{"error": "用户已存在"})
		log.Fatalf("%s用户重复注册", user.UserName)
	}
	return

}

func Loginuser(ctx *gin.Context) {
	// 获取用户输入
	var input struct {
		Username string `json:"username"`
		Userid   string `json:"userid"`
		Password string `json:"password"`
	}
	if err := ctx.ShouldBindBodyWithJSON(&input); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	//查找用户信息
	var user models.User
	db := config.InitDB()
	if err := db.Where("username = ?", input.Username).First(&user).Error; err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "wrong credentials"})
		return
	}
	// 验证账户密码
	if !utils.CheckPassword(input.Password, user.Password) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "wrong credentials"})
		return
	}
	// 添加jwtToken
	token, err := utils.GenerateToken(user.UserID, user.UserName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 返回token
	ctx.JSON(http.StatusOK, gin.H{"token": token})
}
