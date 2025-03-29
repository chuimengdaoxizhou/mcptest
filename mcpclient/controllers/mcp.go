package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"mcpclient/llm/history"
	"mcpclient/models"
	"mcpclient/utils"
	"net/http"
	"sync"
)

var AllUserHistoryMessage = models.ManageHistoryMessage{
	Data: make(map[string]*models.UserHistoryMessage),
}

func HandleUserPrompt2(ctx *gin.Context) {
	fmt.Println("开始执行")
	// 获取模型提供者
	provider, _, _, err := utils.Getproviderclientstools()
	if err != nil {
		log.Println("获取初始化失败")
		ctx.String(http.StatusInternalServerError, "初始化失败")
		return
	}

	var requestData models.Question
	if err := ctx.ShouldBindJSON(&requestData); err != nil {
		// 如果绑定失败，返回错误
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
		})
		return
	}

	// 获取解析后的参数
	UserID := requestData.Userid
	if UserID == "" {
		ctx.String(http.StatusBadRequest, "用户 ID 不能为空")
		return
	}
	prompt := requestData.Prompt
	if prompt == "" {
		ctx.String(http.StatusBadRequest, "提示语不能为空")
		return
	}
	createTime := requestData.Createtime
	if createTime == 0 {
		ctx.String(http.StatusBadRequest, "创建时间不能为0")
		return
	}
	fmt.Println("createTime:" + string(createTime))

	// 构建 key
	key := utils.GenerateCustomId(createTime, UserID)

	// 查找历史消息
	historyMsg, ok := AllUserHistoryMessage.Data[key]
	if !ok {
		historyMsg = &models.UserHistoryMessage{
			UserID:         UserID,
			CreateTime:     createTime,
			HistoryMessage: []history.HistoryMessage{},
		}
		AllUserHistoryMessage.Data[key] = historyMsg
	}

	// 创建 responseChan
	responseChan := make(chan string, 10)

	// 设置流式响应头
	ctx.Writer.Header().Set("Content-Type", "text/event-stream")
	ctx.Writer.Header().Set("Cache-Control", "no-cache")
	ctx.Writer.Header().Set("Connection", "keep-alive")
	ctx.Writer.Flush()

	// 监听客户端断开
	notify := ctx.Writer.CloseNotify()

	// 启动 goroutine 监听 responseChan
	go func() {
		for {
			select {
			case response, ok := <-responseChan:
				fmt.Println(response)
				if !ok {
					return // channel 关闭后退出
				}
				_, err := fmt.Fprintf(ctx.Writer, "%s\n", response)
				if err != nil {
					log.Println("写入响应失败:", err)
					return
				}
				ctx.Writer.Flush() // 立即刷新缓冲区，避免客户端等待
			case <-notify:
				log.Println("客户端已断开连接")
				return
			}
		}
	}()

	// 调用 RunPrompt
	err = utils.RunPrompt(provider, prompt, &historyMsg.HistoryMessage, responseChan)
	if err != nil {
		log.Println("RunPrompt 出错:", err)
		ctx.String(http.StatusInternalServerError, "运行错误")
		return
	}
	fmt.Println("RunPrompt 调用完成")
	n := len(historyMsg.HistoryMessage)
	var allhistory string
	for i := 0; i < n; i++ {
		allhistory += historyMsg.HistoryMessage[i].Content[0].Text + "\n"
	}
	fmt.Println("总历史记录 :", allhistory)
	// 关闭 channel，通知 goroutine 退出
	close(responseChan)
}

func HandleUserPrompt(AllUserHistoryMessage *models.ManageHistoryMessage) {
	fmt.Println("进入")
	provider, _, _, err := utils.Getproviderclientstools()
	if err != nil {
		log.Println("获取初始化失败")
	}
	fmt.Println("初始化完成")

	UserID := "3220921037"
	prompt := "你是谁"
	createTime := int64(1742894838)
	key := utils.GenerateCustomId(createTime, UserID)
	fmt.Println("key:", key)
	historyMsg, ok := AllUserHistoryMessage.Data[key]
	if !ok {
		historyMsg = &models.UserHistoryMessage{
			UserID:         UserID,
			CreateTime:     createTime,
			HistoryMessage: []history.HistoryMessage{},
		}
		AllUserHistoryMessage.Data[key] = historyMsg
	}

	responseChan := make(chan string, 10)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		for response := range responseChan {
			fmt.Println(response)
		}
	}()

	// 调用 RunPrompt
	err = utils.RunPrompt(provider, prompt, &historyMsg.HistoryMessage, responseChan)
	if err != nil {
		fmt.Println("出错:", err)
	} else {
		fmt.Println("RunPrompt调用完成")
	}

	fmt.Println("+++++++++")
	fmt.Println("RunPrompt 调用完成")

	n := len(historyMsg.HistoryMessage)
	fmt.Println("n:", n)
	var allhistory string
	for i := 0; i < n; i++ {
		role := historyMsg.HistoryMessage[i].Role
		fmt.Printf("role%s:%s\n", i, role)
		allhistory = historyMsg.HistoryMessage[i].Content[0].Text + "\n"
		fmt.Printf("allhistory%s:%s\n", i, allhistory)
	}
	wg.Wait() // 等待 goroutine 结束
	fmt.Println("channel已关闭")
	fmt.Println("执行完成")
}
