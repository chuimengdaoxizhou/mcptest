package models

type Question struct {
	Prompt     string `json:"prompt"`
	Createtime int64  `json:"createtime"`
	Userid     string `json:"userid"`
}
