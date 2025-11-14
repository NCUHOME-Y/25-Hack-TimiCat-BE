package handlers

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/models"
)

type Focus struct {
	DB *gorm.DB
}

func NewFocus(db *gorm.DB) *Focus { return &Focus{DB: db} }

// visitorID 从 cookie 中取游客 ID（tcid）
// 返回 ID 字符串和是否成功（cookie 存在且不为空）
func (f *Focus) visitorID(c *gin.Context) (string, bool) {
	vid, err := c.Cookie("tcid")
	return vid, err == nil && vid != ""
}

// POST /api/v1/sessions/start
type startReq struct {
	Mode           string  `json:"mode"` // stopwatch|countdown
	PlannedMinutes *int    `json:"planned_minutes"`
	TaskName       *string `json:"task_name"`
}

func (f *Focus) Start(c *gin.Context) {
	var req startReq
	_ = c.ShouldBindJSON(&req)
	if req.Mode != "stopwatch" && req.Mode != "countdown" {
		req.Mode = "stopwatch"
	}
	vid, ok := f.visitorID(c)
	if !ok {
		c.JSON(401, gin.H{"message": "无访客"})
		return
	}
	sess := models.Session{
		VisitorID:      vid,
		Mode:           req.Mode,
		PlannedMinutes: req.PlannedMinutes,
		TaskName:       req.TaskName,
		Status:         "started",
	}
	if err := f.DB.Create(&sess).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}
	seg := models.Segment{SessionID: sess.ID}
	_ = f.DB.Create(&seg).Error
	c.JSON(200, gin.H{
		"session_id": sess.ID,
		"status":     "started",
		"started_at": sess.StartAt.UTC(),
	})
}

// Pause 暂停当前计时会话
// 逻辑：结束当前片段的计时，保存累计秒数，改状态为 paused
func (f *Focus) Pause(c *gin.Context) {
	vid, ok := f.visitorID(c)
	if !ok {
		c.JSON(401, gin.H{"message": "no visitor"})
		return
	}
	sess, ok := f.findMutable(vid)
	if !ok || sess.Status != "started" {
		c.JSON(400, gin.H{"message": "not started"})
		return
	}
	now := time.Now()

	// 结束最后一个未结束的片段（记录片段的结束时间）
	f.DB.Model(&models.Segment{}).
		Where("session_id=? AND end_at IS NULL", sess.ID).
		Update("end_at", &now)

	// 计算本次会话已用的总秒数并更新会话状态
	total := f.totalSeconds(sess.ID)
	f.DB.Model(&models.Session{}).Where("id=?", sess.ID).
		Updates(map[string]any{
			"status":       "paused",
			"duration_sec": total,
		})
	c.JSON(200, gin.H{
		"status":    "paused",
		"total_sec": total,
	})
}

// Resume 恢复暂停的计时会话
// 逻辑：新建一个片段（开始新的计时），改状态为 started
func (f *Focus) Resume(c *gin.Context) {
	vid, ok := f.visitorID(c)
	if !ok {
		c.JSON(401, gin.H{"message": "无访客"})
		return
	}
	sess, ok := f.findMutable(vid)
	if !ok || sess.Status != "paused" {
		c.JSON(400, gin.H{"message": "无访客"})
		return
	}

	// 新开一个片段（记录新的开始时间）
	_ = f.DB.Create(&models.Segment{SessionID: sess.ID}).Error
	f.DB.Model(&models.Session{}).Where("id=?", sess.ID).
		Update("status", "started")
	c.JSON(200, gin.H{"status": "started"})
}

// Finish 完成计时会话
// 收口所有片段，计算总秒数，若小于 1 分钟视为无效，否则创建成长事件
func (f *Focus) Finish(c *gin.Context) {
	vid, ok := f.visitorID(c)
	if !ok {
		c.JSON(401, gin.H{"message": "no visitor"})
		return
	}
	sess, ok := f.findMutable(vid)
	if !ok {
		c.JSON(400, gin.H{"message": "no active session"})
		return
	}
	now := time.Now()

	// 收口当前片段（把未结束的 seg 结束掉）
	f.DB.Model(&models.Segment{}).
		Where("session_id=? AND end_at IS NULL", sess.ID).
		Update("end_at", &now)

	// 统计本次秒数
	total := f.totalSeconds(sess.ID)

	// 少于 1 分钟视为太短（短短的也很可爱呢:)）
	minLimit := int64(60)
	if total < minLimit {
		c.String(400, "It ended too quickly")
		return
	}
	// 会话结束，并标记结束时间与总秒数
	f.DB.Model(&models.Session{}).Where("id=?", sess.ID).
		Updates(map[string]any{
			"status":       "finished",
			"end_at":       &now,
			"duration_sec": total,
		})

	// 分钟数向上取整（61s -> 2min），且至少 1 分钟
	// 用于统一计算成长值：比如 61 秒和 120 秒都算 2 分钟
	minutes := int((total + 59) / 60)
	if minutes < 1 {
		minutes = 1
	}
	// 创建成长事件记录，供前端和宠物系统使用
	_ = f.DB.Create(&models.GrowthEvent{
		VisitorID: vid,
		SessionID: sess.ID,
		Minutes:   minutes,
	}).Error

	c.JSON(200, gin.H{
		"status":       "finished",
		"session_id":   sess.ID,
		"duration_sec": total,
		"minutes":      minutes,
	})
}

// Cancel POST /api/v1/sessions/cancel
func (f *Focus) Cancel(c *gin.Context) {
	vid, ok := f.visitorID(c)
	if !ok {
		c.JSON(401, gin.H{"message": "无访客"})
		return
	}
	sess, ok := f.findMutable(vid)
	if !ok {
		c.JSON(400, gin.H{"message": "no active session"})
		return
	}
	now := time.Now()
	f.DB.Model(&models.Session{}).Where("id=?", sess.ID).
		Updates(map[string]any{"status": "canceled", "end_at": &now})

		// 把未结束的片段也收口
	f.DB.Model(&models.Segment{}).Where("session_id=? AND end_at IS NULL", sess.ID).
		Update("end_at", &now)
	c.JSON(200, gin.H{"status": "canceled"})
}

// Current GET /api/v1/sessions/current
func (f *Focus) Current(c *gin.Context) {
	vid, ok := f.visitorID(c)
	if !ok {
		c.JSON(401, gin.H{"message": "no visitor"})
		return
	}
	sess, ok := f.findMutable(vid)
	if !ok {
		c.JSON(200, nil)
		return
	}
	elapsed := f.elapsedNow(sess.ID)
	c.JSON(200, gin.H{
		"session_id":  sess.ID,
		"status":      sess.Status,
		"mode":        sess.Mode,
		"started_at":  sess.StartAt.UTC(),
		"elapsed_sec": elapsed,
	})
}

// Summary 获取统计数据：今日时长/次数、近 7 天每天分钟、总分钟
// 逻辑：分别查询三个时间段内已完成的会话，累计计算分钟数
func (f *Focus) Summary(c *gin.Context) {
	vid, ok := f.visitorID(c)
	if !ok {
		c.JSON(401, gin.H{"message": "no visitor"})
		return
	}
	// 今日完成的会话（从今天 00:00:00 起）
	startOfDay := time.Now().Truncate(24 * time.Hour)
	var today []models.Session
	f.DB.Where("visitor_id=? AND status='finished' AND end_at >= ?", vid, startOfDay).
		Find(&today)
	todayMin := 0
	for _, s := range today {
		todayMin += int(s.DurationSec / 60)
	}

	// 近 7 天（含今天）的数据，用 Go 填充为 0（没有数据的日期也显示为 0）
	last7 := make([]map[string]any, 0, 7)
	dayMap := map[string]int{}
	// 找出近 7 天已完成的会话，累计每天的分钟数
	weekAgo := time.Now().AddDate(0, 0, -6).Truncate(24 * time.Hour)
	var all []models.Session
	f.DB.Where("visitor_id=? AND status='finished' AND end_at >= ?", vid, weekAgo).Find(&all)
	for _, s := range all {
		// 将 end_at 时间戳转换为日期字符串，作为 dayMap 的 key
		d := s.EndAt.Truncate(24 * time.Hour).Format("2006-01-02")
		dayMap[d] += int(s.DurationSec / 60)
	}

	// 构造返回的 7 天数组，倒序遍历（从前 6 天到今天）
	for i := 6; i >= 0; i-- {
		d := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		last7 = append(last7, map[string]any{
			"date":    d,
			"minutes": dayMap[d],
		})
	}

	// 总分钟（全历史）
	var total []models.Session
	f.DB.Where("visitor_id=? AND status='finished'", vid).Find(&total)
	totalMin := 0
	for _, s := range total {
		totalMin += int(s.DurationSec / 60)
	}

	c.JSON(200, gin.H{
		"today_minutes": todayMin,
		"today_count":   len(today),
		"last7d":        last7,
		"total_minutes": totalMin,
	})
}

// 成长事件

// GrowthPull 拉取该游客未处理的成长事件
// 支持 limit 参数（默认 50，上限 200），按事件 ID 升序返回
func (f *Focus) GrowthPull(c *gin.Context) {
	vid, ok := f.visitorID(c)
	if !ok {
		c.JSON(401, gin.H{"message": "no visitor"})
		return
	}
	limit := 50
	// 读取 ?limit=N 参数，验证范围（1-200）
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	// 查询未处理的事件，按 ID 升序排列（保证顺序）
	var evs []models.GrowthEvent
	f.DB.Where("visitor_id=? AND handled=false", vid).
		Order("id ASC").Limit(limit).Find(&evs)
	c.JSON(200, evs)
}

// GrowthAck 确认已处理的成长事件（标记为已处理）
// 请求体：{"last_id":123}，将 ≤ last_id 的所有事件标记为 handled=true
type ackReq struct {
	LastID uint `json:"last_id"`
}

func (f *Focus) GrowthAck(c *gin.Context) {
	vid, ok := f.visitorID(c)
	if !ok {
		c.JSON(401, gin.H{"message": "no visitor"})
		return
	}
	var req ackReq
	if err := c.ShouldBindJSON(&req); err != nil || req.LastID == 0 {
		c.JSON(400, gin.H{"message": "bad last_id"})
		return
	}
	// 将 ≤ last_id 的所有事件标记为已处理，防止重复拉取
	f.DB.Model(&models.GrowthEvent{}).
		Where("visitor_id=? AND id <= ?", vid, req.LastID).
		Update("handled", true)
	c.JSON(200, gin.H{"ok": true})
}

// findMutable 查找该游客最近一条可变更的会话（状态为 started 或 paused）
// 这样可以确保同一时间只有一个活跃会话被修改
func (f *Focus) findMutable(visitorID string) (models.Session, bool) {
	var s models.Session
	err := f.DB.Where("visitor_id=? AND status IN ('started','paused')", visitorID).
		Order("start_at DESC").Take(&s).Error
	return s, err == nil
}

// totalSeconds 计算会话的总计时秒数
// 逻辑：遍历所有片段，对每个片段计算 end_at - start_at 的秒数，然后累加
// 如果片段还未结束（end_at 为 nil），则用当前时间作为 end_at 进行计算
func (f *Focus) totalSeconds(sessionID uint) int64 {
	var segs []models.Segment
	f.DB.Where("session_id=?", sessionID).Find(&segs)
	var sum int64
	now := time.Now()
	for _, sg := range segs {
		end := sg.EndAt
		// 如果片段未结束，用当前时间作为结束时间
		if end == nil {
			e := now
			end = &e
		}
		// 累加片段耗时（秒数）
		sum += int64(end.Sub(sg.StartAt).Seconds())
	}
	return sum
}

func (f *Focus) elapsedNow(sessionID uint) int64 {
	return f.totalSeconds(sessionID)
}

// Achievements 返回总专注分钟数与已解锁的成就列表（阈值用秒对比）
func (f *Focus) Achievements(c *gin.Context) {
	vid, ok := f.visitorID(c)
	if !ok {
		c.JSON(401, gin.H{"message": "no visitor"})
		return
	}
	// 汇总该游客所有已完成会话的总秒数
	var sessions []models.Session
	f.DB.Where("visitor_id=? AND status='finished'", vid).Find(&sessions)
	var totalSec int64
	for _, s := range sessions {
		totalSec += s.DurationSec
	}
	// 选择所有 threshold <= totalSec 的成就
	unlocked := make([]models.Achievement, 0, len(models.Achievements))
	for _, a := range models.Achievements {
		if totalSec >= a.Threshold {
			unlocked = append(unlocked, a)
		}
	}
	c.JSON(200, gin.H{
		"total_minutes": totalSec / 60, // 方便前端展示
		"Achievement":   unlocked,
	})
}
