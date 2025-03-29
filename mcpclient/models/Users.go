package models

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	UserID       int64     `gorm:"primaryKey;autoIncrement;column:userid"`
	UserName     string    `gorm:"column:username;unique;not null"`
	Password     string    `gorm:"column:password;not null"`
	Email        string    `gorm:"column:email"`
	NickName     string    `gorm:"column:nickname"`
	RegisterTime time.Time `gorm:"column:registertime"`
	ChangeTime   time.Time `gorm:"column:changetime"`
	gorm.DeletedAt
}

// BeforeCreate 钩子函数，在创建记录前执行
func (u *User) BeforeCreate() (err error) {
	// 如果 NickName 为空，设置默认值
	if u.NickName == "" {
		u.NickName = "user"
	}

	// 如果 Email 为空，设置默认值
	if u.Email == "" {
		u.Email = "" // 设置一个默认的 email 地址
	}

	now := time.Now()
	// 如果 ResigterTime 和 ChangeTime 为空，则设置为当前时间
	if u.RegisterTime.IsZero() {
		u.RegisterTime = now
	}
	if u.ChangeTime.IsZero() {
		u.ChangeTime = now
	}
	return nil
}
