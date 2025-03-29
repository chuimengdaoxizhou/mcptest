package models

import (
	"mcpclient/llm/history"
)

type UserHistoryMessage struct {
	UserID         string                   `json:"userid"`
	CreateTime     int64                    `json:"createtime"`
	HistoryMessage []history.HistoryMessage `json:"historymessage"`
}

type ManageHistoryMessage struct {
	// 使用UserID+CreateTime作为key
	Data map[string]*UserHistoryMessage
}
