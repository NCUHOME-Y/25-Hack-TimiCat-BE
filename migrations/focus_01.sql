-- 会话事件 sessions
CREATE TABLE IF NOT EXISTS focus_sessions (
  id              BIGSERIAL PRIMARY KEY,
  visitor_id      UUID        NOT NULL,
  mode            TEXT        NOT NULL CHECK (mode IN ('stopwatch','countdown')),
  planned_minutes INT,
  task_name       TEXT,
  task_labels     JSONB,
  status          TEXT        NOT NULL CHECK (status IN ('started','paused','finished','canceled')),
  start_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  end_at          TIMESTAMPTZ,
  duration_sec    INT         NOT NULL DEFAULT 0,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_fs_visitor_created ON focus_sessions(visitor_id, created_at);
CREATE INDEX IF NOT EXISTS idx_fs_status            ON focus_sessions(status);

-- 专注时间段 segments（暂停/继续按片段累加）
CREATE TABLE IF NOT EXISTS focus_segments (
  id            BIGSERIAL PRIMARY KEY,
  session_id    BIGINT NOT NULL REFERENCES focus_sessions(id) ON DELETE CASCADE,
  seg_start_at  TIMESTAMPTZ NOT NULL,
  seg_end_at    TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_fseg_session ON focus_segments(session_id);

-- 成长事件 growth events（结束会话时写入一条）
CREATE TABLE IF NOT EXISTS growth_events (
  id            BIGSERIAL PRIMARY KEY,
  visitor_id    UUID        NOT NULL,
  session_id    BIGINT      NOT NULL REFERENCES focus_sessions(id) ON DELETE CASCADE,
  minutes       INT         NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  handled       BOOLEAN     NOT NULL DEFAULT FALSE
);
