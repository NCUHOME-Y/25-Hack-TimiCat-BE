package model

import (
	"time"
)

type User struct {
	ID             uint64     `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
	UserName       string     `gorm:"unique;type:varchar(64)" json:"user_name,omitempty"`
	StartTime      *time.Time `gorm:"type:datetime;null" json:"start_time"`
	EndTime        *time.Time `json:"end_time,omitempty"`
	DurationSecond int64      `json:"duration_second,omitempty"`
	TotalDuration  int64      `json:"total_duration,omitempty"`
	IsCompleted    *bool      `json:"is_completed,omitempty" gorm:"type:boolean"`     //状态检验，用于计时
	Achievement    string     `json:"achievement,omitempty" gorm:"type:varchar(255)"` // 成就记录，保存用户获得了哪些成就
	IsGuest        bool       `json:"is_guest" gorm:"default:false"`                  //检验是否为游客，后续开发正式用户或许能用到
	CreatedAt      time.Time  `gorm:"autoCreateTime"`
}
