package focus

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/google/uuid"
)

func getVisitorID(w http.ResponseWriter, r *http.Request) string {
	if c, err := r.Cookie("tcid"); err == nil && c.Value != "" {
		return c.Value
	}
	vid := uuid.NewString()
	http.SetCookie(w, &http.Cookie{
		Name: "tcid", Value: vid, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	return vid
}

type StartReq struct {
	Mode           string   `json:"mode"` // stopwatch|countdown
	PlannedMinutes *int     `json:"planned_minutes,omitempty"`
	TaskName       *string  `json:"task_name,omitempty"`
	TaskLabels     []string `json:"task_labels,omitempty"`
}

func StartHandler(db *sql.DB) http.HandlerFunc {
	s := NewStore(db)
	return func(w http.ResponseWriter, r *http.Request) {
		var req StartReq
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Mode != "stopwatch" && req.Mode != "countdown" {
			req.Mode = "stopwatch"
		}
		vid := getVisitorID(w, r)
		id, startedAt, err := s.Start(r.Context(), vid, req.Mode, req.PlannedMinutes, req.TaskName)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session_id": id, "status": "started", "started_at": startedAt.UTC(),
		})
	}
}

func PauseHandler(db *sql.DB) http.HandlerFunc {
	s := NewStore(db)
	return func(w http.ResponseWriter, r *http.Request) {
		vid := getVisitorID(w, r)
		total, err := s.Pause(r.Context(), vid)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "paused", "total_sec": total})
	}
}

func ResumeHandler(db *sql.DB) http.HandlerFunc {
	s := NewStore(db)
	return func(w http.ResponseWriter, r *http.Request) {
		vid := getVisitorID(w, r)
		if err := s.Resume(r.Context(), vid); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "started"})
	}
}

func FinishHandler(db *sql.DB) http.HandlerFunc {
	s := NewStore(db)
	return func(w http.ResponseWriter, r *http.Request) {
		vid := getVisitorID(w, r)
		minSec := int64(60)
		if v := os.Getenv("MIN_SESSION_SEC"); v != "" {
			if i, err := strconv.Atoi(v); err == nil {
				minSec = int64(i)
			}
		}
		sid, total, err := s.Finish(r.Context(), vid, minSec)
		if err != nil {
			if err == ErrTooShort {
				http.Error(w, err.Error(), 400)
				return
			}
			http.Error(w, err.Error(), 400)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "finished", "session_id": sid,
			"duration_sec": total, "minutes": total / 60,
		})
	}
}

func CancelHandler(db *sql.DB) http.HandlerFunc {
	s := NewStore(db)
	return func(w http.ResponseWriter, r *http.Request) {
		vid := getVisitorID(w, r)
		if err := s.Cancel(r.Context(), vid); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "canceled"})
	}
}

func SummaryHandler(db *sql.DB) http.HandlerFunc {
	s := NewStore(db)
	return func(w http.ResponseWriter, r *http.Request) {
		vid := getVisitorID(w, r)
		days := 7
		if q := r.URL.Query().Get("range"); q == "30d" {
			days = 30
		}
		sum, err := s.Summary(r.Context(), vid, days)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		_ = json.NewEncoder(w).Encode(sum)
	}
}

// 当前会话
func CurrentHandler(db *sql.DB) http.HandlerFunc {
	s := NewStore(db)
	return func(w http.ResponseWriter, r *http.Request) {
		vid := getVisitorID(w, r)
		cur, err := s.Current(r.Context(), vid)
		if err != nil {
			if err == ErrNoActiveSession {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(nil)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(cur)
	}
}

// 拉取成长事件
func GrowthPullHandler(db *sql.DB) http.HandlerFunc {
	s := NewStore(db)
	return func(w http.ResponseWriter, r *http.Request) {
		vid := getVisitorID(w, r)
		limit := 50
		if v := r.URL.Query().Get("limit"); v != "" {
			if i, err := strconv.Atoi(v); err == nil && i > 0 && i <= 200 {
				limit = i
			}
		}
		evs, err := s.PullGrowth(r.Context(), vid, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(evs) // []GrowthEvent
	}
}

// 确认成长事件（游标式）
type ackReq struct {
	LastID int64 `json:"last_id"`
}

func GrowthAckHandler(db *sql.DB) http.HandlerFunc {
	s := NewStore(db)
	return func(w http.ResponseWriter, r *http.Request) {
		vid := getVisitorID(w, r)
		var req ackReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.LastID <= 0 {
			http.Error(w, "invalid last_id", http.StatusBadRequest)
			return
		}
		if err := s.AckGrowthUpTo(r.Context(), vid, req.LastID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}
}
