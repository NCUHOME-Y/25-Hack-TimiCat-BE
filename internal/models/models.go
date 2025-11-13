package models

import (
	"time"

	"gorm.io/gorm"
)

// 一次专注会话
type Session struct {
	ID             uint    `json:"id" gorm:"primaryKey"`
	VisitorID      string  `json:"visitor_id" gorm:"type:uuid"`
	Mode           string  `json:"mode"` // stopwatch、countdown
	PlannedMinutes *int    `json:"planned_minutes"`
	TaskName       *string `json:"task_name"`

	Status      string         `json:"status"` // started、paused、finished、canceled
	StartAt     time.Time      `json:"start_at" gorm:"autoCreateTime"`
	EndAt       *time.Time     `json:"end_at"`
	DurationSec int64          `json:"duration_sec"` // 结束时写入
	Segments    []Segment      `json:"segments"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// 一个专注片段（开始->结束或未结束）
type Segment struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	SessionID uint       `json:"session_id" gorm:"index"`
	StartAt   time.Time  `json:"seg_start_at" gorm:"autoCreateTime"`
	EndAt     *time.Time `json:"seg_end_at"`
}

// 成长事件：当一次会话结束（>=60s）就写一条 minutes，用于前端/宠物系统消费
type GrowthEvent struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	VisitorID string    `json:"visitor_id" gorm:"type:uuid;index"`
	SessionID uint      `json:"session_id"`
	Minutes   int       `json:"minutes"`
	Handled   bool      `json:"handled" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}
