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

	_, err = s.db.ExecContext(ctx, `
	  INSERT INTO focus_segments(session_id, seg_start_at) VALUES ($1, now());
	  UPDATE focus_sessions SET status='started' WHERE id=$1;
	`, sid)
	return err
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
	  SELECT COALESCE(SUM(EXTRACT(EPOCH FROM (seg_end_at - seg_start_at))),0)
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
	_, err = s.db.ExecContext(ctx, `
	  UPDATE focus_sessions SET status='canceled', end_at=now() WHERE id=$1;
	  UPDATE focus_segments SET seg_end_at=COALESCE(seg_end_at, now()) WHERE session_id=$1;
	`, sid)
	return err
}

type DayItem struct {
	Date    string `json:"date"`
	Minutes int    `json:"minutes"`
}
type Summary struct {
	TodayMinutes int       `json:"today_minutes"`
	TodayCount   int       `json:"today_count"`
	StreakDays   int       `json:"streak_days"`
	Trend        []DayItem `json:"trend"`
	Inactive48h  bool      `json:"inactive_48h"`
}

func (s *Store) Summary(ctx context.Context, visitorID string, days int) (Summary, error) {
	var res Summary

	// 近 n 天趋势
	rows, err := s.db.QueryContext(ctx, `
	  SELECT date_trunc('day', seg_end_at)::date AS d,
	         SUM(EXTRACT(EPOCH FROM (seg_end_at - seg_start_at))/60)::int AS m
	  FROM focus_segments seg
	  JOIN focus_sessions fs ON fs.id=seg.session_id
	  WHERE fs.visitor_id=$1 AND fs.status='finished'
	    AND seg_end_at >= now() - ($2 || ' days')::interval
	  GROUP BY 1 ORDER BY 1
	`, visitorID, days)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	for rows.Next() {
		var d string
		var m int
		if err := rows.Scan(&d, &m); err != nil {
			return res, err
		}
		res.Trend = append(res.Trend, DayItem{Date: d, Minutes: m})
	}

	// 今日分钟与次数
	_ = s.db.QueryRowContext(ctx, `
	  SELECT COALESCE(SUM(EXTRACT(EPOCH FROM (seg_end_at - seg_start_at))/60)::int,0) AS m,
	         COUNT(DISTINCT fs.id) AS c
	  FROM focus_segments seg
	  JOIN focus_sessions fs ON fs.id=seg.session_id
	  WHERE fs.visitor_id=$1 AND fs.status='finished'
	    AND seg_end_at >= date_trunc('day', now())
	`, visitorID).Scan(&res.TodayMinutes, &res.TodayCount)

	// 48h 不活跃
	_ = s.db.QueryRowContext(ctx, `
	  SELECT COALESCE((now() - MAX(fs.end_at)) > interval '48 hours', true)
	  FROM focus_sessions fs
	  WHERE fs.visitor_id=$1 AND fs.status='finished'
	`, visitorID).Scan(&res.Inactive48h)

	// 简化 streak：从今天起向前累加连续有数据的天数
	m := map[string]int{}
	for _, it := range res.Trend {
		m[it.Date] = it.Minutes
	}
	for i := 0; i < days; i++ {
		d := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		if m[d] > 0 {
			res.StreakDays++
		} else {
			break
		}
	}
	return res, nil
}
