package focus

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrNoActiveSession = errors.New("no active session")
	ErrTooShort        = errors.New("duration below minimal threshold")
)

type Store struct{ db *sql.DB }

type Current struct {
	SessionID      int64     `json:"session_id"`
	Status         string    `json:"status"` // started|paused
	Mode           string    `json:"mode"`   // stopwatch|countdown
	PlannedMinutes *int      `json:"planned_minutes,omitempty"`
	TaskName       *string   `json:"task_name,omitempty"`
	StartedAt      time.Time `json:"started_at"`
	ElapsedSec     int64     `json:"elapsed_sec"`
}

func (s *Store) Current(ctx context.Context, visitorID string) (Current, error) {
	var c Current
	err := s.db.QueryRowContext(ctx, `
	  SELECT id, status, mode, planned_minutes, task_name, start_at
	  FROM focus_sessions
	  WHERE visitor_id=$1 AND status IN ('started','paused')
	  ORDER BY start_at DESC LIMIT 1
	`, visitorID).Scan(&c.SessionID, &c.Status, &c.Mode, &c.PlannedMinutes, &c.TaskName, &c.StartedAt)
	if err == sql.ErrNoRows {
		return c, ErrNoActiveSession
	}
	if err != nil {
		return c, err
	}

	var elapsed float64
	if err := s.db.QueryRowContext(ctx, `
	  SELECT COALESCE(SUM(EXTRACT(EPOCH FROM (COALESCE(seg_end_at, now()) - seg_start_at))),0)
	  FROM focus_segments WHERE session_id=$1
	`, c.SessionID).Scan(&elapsed); err != nil {
		return c, err
	}
	c.ElapsedSec = int64(elapsed)
	return c, nil
}

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// 最近一条可变更（started/paused）会话
func (s *Store) lastMutableSession(ctx context.Context, visitorID string) (int64, string, error) {
	var id int64
	var status string
	err := s.db.QueryRowContext(ctx, `
	  SELECT id, status FROM focus_sessions
	  WHERE visitor_id=$1 AND status IN ('started','paused')
	  ORDER BY start_at DESC LIMIT 1
	`, visitorID).Scan(&id, &status)
	if err == sql.ErrNoRows {
		return 0, "", ErrNoActiveSession
	}
	return id, status, err
}

// 计算累计秒数
func (s *Store) totalSeconds(ctx context.Context, sessionID int64) (int64, error) {
	var sec float64
	err := s.db.QueryRowContext(ctx, `
	  SELECT COALESCE(SUM(EXTRACT(EPOCH FROM (COALESCE(seg_end_at, now()) - seg_start_at))),0)
	  FROM focus_segments WHERE session_id=$1
	`, sessionID).Scan(&sec)
	return int64(sec), err
}

func (s *Store) Start(ctx context.Context, visitorID, mode string, planned *int, taskName *string) (int64, time.Time, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, time.Time{}, err
	}
	defer tx.Rollback()

	var sid int64
	var startedAt time.Time
	err = tx.QueryRowContext(ctx, `
	  INSERT INTO focus_sessions(visitor_id, mode, planned_minutes, task_name, status)
	  VALUES ($1,$2,$3,$4,'started') RETURNING id, start_at
	`, visitorID, mode, planned, taskName).Scan(&sid, &startedAt)
	if err != nil {
		return 0, time.Time{}, err
	}

	if _, err = tx.ExecContext(ctx, `
	  INSERT INTO focus_segments(session_id, seg_start_at) VALUES ($1, now())
	`, sid); err != nil {
		return 0, time.Time{}, err
	}

	if err := tx.Commit(); err != nil {
		return 0, time.Time{}, err
	}
	return sid, startedAt, nil
}

func (s *Store) Pause(ctx context.Context, visitorID string) (int64, error) {
	sid, status, err := s.lastMutableSession(ctx, visitorID)
	if err != nil {
		return 0, err
	}
	if status != "started" {
		return 0, fmt.Errorf("not started")
	}

	if _, err = s.db.ExecContext(ctx, `
	  UPDATE focus_segments SET seg_end_at=now()
	  WHERE session_id=$1 AND seg_end_at IS NULL
	`, sid); err != nil {
		return 0, err
	}

	total, err := s.totalSeconds(ctx, sid)
	if err != nil {
		return 0, err
	}

	_, err = s.db.ExecContext(ctx, `
	  UPDATE focus_sessions SET status='paused', duration_sec=$2 WHERE id=$1
	`, sid, total)
	return total, err
}

func (s *Store) Resume(ctx context.Context, visitorID string) error {
	sid, status, err := s.lastMutableSession(ctx, visitorID)
	if err != nil {
		return err
	}
	if status != "paused" {
		return fmt.Errorf("not paused")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO focus_segments(session_id, seg_start_at) VALUES ($1, now())`,
		sid,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE focus_sessions SET status='started' WHERE id=$1`,
		sid,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) Finish(ctx context.Context, visitorID string, minSeconds int64) (int64, int64, error) {
	sid, _, err := s.lastMutableSession(ctx, visitorID)
	if err != nil {
		return 0, 0, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	// 统计当前片段
	if _, err = tx.ExecContext(ctx, `
	  UPDATE focus_segments SET seg_end_at=now()
	  WHERE session_id=$1 AND seg_end_at IS NULL
	`, sid); err != nil {
		return 0, 0, err
	}

	// 计算总时长
	var totalSec int64
	if err := tx.QueryRowContext(ctx, `
  SELECT COALESCE(SUM(EXTRACT(EPOCH FROM (seg_end_at - seg_start_at)))::bigint, 0)
  FROM focus_segments WHERE session_id=$1
`, sid).Scan(&totalSec); err != nil {
		return 0, 0, err
	}

	// 幂等： finished 的直接返回
	var prev string
	_ = tx.QueryRowContext(ctx, `SELECT status FROM focus_sessions WHERE id=$1`, sid).Scan(&prev)
	if prev == "finished" {
		var duration int64
		_ = tx.QueryRowContext(ctx, `SELECT duration_sec FROM focus_sessions WHERE id=$1`, sid).Scan(&duration)
		return sid, duration, nil
	}

	if totalSec < minSeconds {
		return 0, 0, ErrTooShort
	}

	if _, err := tx.ExecContext(ctx, `
	  UPDATE focus_sessions SET status='finished', end_at=now(), duration_sec=$2 WHERE id=$1
	`, sid, totalSec); err != nil {
		return 0, 0, err
	}

	minutes := int64(totalSec / 60)
	if minutes < 1 {
		minutes = 1
	}
	if _, err := tx.ExecContext(ctx, `
	  INSERT INTO growth_events(visitor_id, session_id, minutes) VALUES ($1,$2,$3)
	`, visitorID, sid, minutes); err != nil {
		return 0, 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return sid, totalSec, nil
}

func (s *Store) Cancel(ctx context.Context, visitorID string) error {
	sid, _, err := s.lastMutableSession(ctx, visitorID)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`UPDATE focus_sessions SET status='canceled', end_at=now() WHERE id=$1`,
		sid,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE focus_segments SET seg_end_at=COALESCE(seg_end_at, now()) WHERE session_id=$1`,
		sid,
	); err != nil {
		return err
	}

	return tx.Commit()
}

type DayItem struct {
	Date    string `json:"date"`
	Minutes int    `json:"minutes"`
}

type Summary struct {
	TodayMinutes int       `json:"today_minutes"`
	TodayCount   int       `json:"today_count"`
	Last7d       []DayItem `json:"last7d"`        // 最近 7 天，含 0 填充
	TotalMinutes int       `json:"total_minutes"` // 全部历史累计分钟
}

func (s *Store) Summary(ctx context.Context, visitorID string, _ int) (Summary, error) {
	var res Summary
	res.Last7d = make([]DayItem, 0, 7)

	// 1) 最近 7 天（按天聚合），只统计已完成会话的片段
	rows, err := s.db.QueryContext(ctx, `
	  SELECT date_trunc('day', seg_end_at)::date AS d,
	         SUM(EXTRACT(EPOCH FROM (seg_end_at - seg_start_at))/60)::int AS m
	  FROM focus_segments seg
	  JOIN focus_sessions fs ON fs.id=seg.session_id
	  WHERE fs.visitor_id=$1 AND fs.status='finished'
	    AND seg_end_at >= now() - 7 * interval '1 day'
	  GROUP BY 1 ORDER BY 1
	`, visitorID)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	agg := map[string]int{}
	for rows.Next() {
		var d string
		var m int
		if err := rows.Scan(&d, &m); err != nil {
			return res, err
		}
		agg[d] = m
	}

	// 用 0 填满 近 7 天（含今天）
	for i := 6; i >= 0; i-- {
		d := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		res.Last7d = append(res.Last7d, DayItem{Date: d, Minutes: agg[d]})
	}

	// 2) 今日分钟 & 今日次数
	_ = s.db.QueryRowContext(ctx, `
	  SELECT
	    COALESCE(SUM(EXTRACT(EPOCH FROM (seg_end_at - seg_start_at))/60)::int, 0) AS m,
	    COALESCE(COUNT(DISTINCT fs.id), 0) AS c
	  FROM focus_segments seg
	  JOIN focus_sessions fs ON fs.id=seg.session_id
	  WHERE fs.visitor_id=$1 AND fs.status='finished'
	    AND seg_end_at >= date_trunc('day', now())
	`, visitorID).Scan(&res.TodayMinutes, &res.TodayCount)

	// 3) 累计总分钟（历史全部 finished）
	_ = s.db.QueryRowContext(ctx, `
	  SELECT COALESCE(SUM(EXTRACT(EPOCH FROM (seg_end_at - seg_start_at))/60)::int, 0)
	  FROM focus_segments seg
	  JOIN focus_sessions fs ON fs.id=seg.session_id
	  WHERE fs.visitor_id=$1 AND fs.status='finished'
	`, visitorID).Scan(&res.TotalMinutes)

	return res, nil
}

// 仅趋势（近 N 天）
func (s *Store) Trend(ctx context.Context, visitorID string, days int) ([]DayItem, error) {
	items := make([]DayItem, 0)
	rows, err := s.db.QueryContext(ctx, `
	  SELECT date_trunc('day', seg_end_at)::date AS d,
	         SUM(EXTRACT(EPOCH FROM (seg_end_at - seg_start_at))/60)::int AS m
	  FROM focus_segments seg
	  JOIN focus_sessions fs ON fs.id=seg.session_id
	  WHERE fs.visitor_id=$1 AND fs.status='finished'
	    AND seg_end_at >= now() - $2 * interval '1 day'
	  GROUP BY 1 ORDER BY 1
	`, visitorID, days)
	if err != nil {
		return items, err
	}
	defer rows.Close()
	for rows.Next() {
		var d string
		var m int
		if err := rows.Scan(&d, &m); err != nil {
			return items, err
		}
		items = append(items, DayItem{Date: d, Minutes: m})
	}
	return items, nil
}

// period: day|week|month 的总分钟与会话次数
type Overview struct {
	Period       string `json:"period"` // day|week|month
	TotalMinutes int    `json:"total_minutes"`
	SessionCount int    `json:"session_count"`
}

func (s *Store) Overview(ctx context.Context, visitorID, period string) (Overview, error) {
	res := Overview{Period: period}
	var trunc string
	switch period {
	case "day":
		trunc = "day"
	case "week":
		trunc = "week"
	case "month":
		trunc = "month"
	default:
		trunc = "day"
		res.Period = "day"
	}
	err := s.db.QueryRowContext(ctx, `
	  WITH x AS (
	    SELECT fs.id
	    FROM focus_sessions fs
	    WHERE fs.visitor_id=$1 AND fs.status='finished'
	      AND fs.end_at >= date_trunc($2, now())
	  )
	  SELECT
	    COALESCE((
	      SELECT SUM(EXTRACT(EPOCH FROM (seg_end_at - seg_start_at))/60)::int
	      FROM focus_segments seg
	      JOIN focus_sessions fs2 ON fs2.id=seg.session_id
	      WHERE fs2.visitor_id=$1 AND fs2.status='finished'
	        AND seg_end_at >= date_trunc($2, now())
	    ), 0) AS minutes,
	    (SELECT COUNT(*) FROM x) AS cnt
	`, visitorID, trunc).Scan(&res.TotalMinutes, &res.SessionCount)
	return res, err
}

// 成长事件确认
type GrowthEvent struct {
	ID        int64     `json:"id"`
	SessionID int64     `json:"session_id"`
	Minutes   int       `json:"minutes"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) PullGrowth(ctx context.Context, visitorID string, limit int) ([]GrowthEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
	  SELECT id, session_id, minutes, created_at
	  FROM growth_events
	  WHERE visitor_id=$1 AND handled=false
	  ORDER BY id ASC
	  LIMIT $2
	`, visitorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []GrowthEvent
	for rows.Next() {
		var ge GrowthEvent
		if err := rows.Scan(&ge.ID, &ge.SessionID, &ge.Minutes, &ge.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, ge)
	}
	return out, nil
}

// 将 <= lastID 的事件标记为已处理（当前访客）
func (s *Store) AckGrowthUpTo(ctx context.Context, visitorID string, lastID int64) error {
	_, err := s.db.ExecContext(ctx, `
	  UPDATE growth_events SET handled=true
	  WHERE visitor_id=$1 AND id <= $2
	`, visitorID, lastID)
	return err
}
