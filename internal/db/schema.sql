-- todoloo database schema

-- Raw queue table: append-only log of every submitted input
CREATE TABLE IF NOT EXISTS raw_queue (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    raw_text        TEXT        NOT NULL,
    submitted_at    DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status          TEXT        NOT NULL DEFAULT 'pending',
    attempts        INTEGER     NOT NULL DEFAULT 0,
    last_attempted  DATETIME,
    error_msg       TEXT
);

-- Todos table: structured, queryable records
CREATE TABLE IF NOT EXISTS todos (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    raw_id          INTEGER     REFERENCES raw_queue(id),
    title           TEXT        NOT NULL,
    description     TEXT,
    category        TEXT,
    due_date        TEXT,
    due_time        TEXT,
    urgency         INTEGER     DEFAULT 3,
    tags            TEXT,
    location        TEXT,
    recurrence      TEXT,
    status          TEXT        NOT NULL DEFAULT 'open',
    source_text     TEXT,
    created_at      DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at    DATETIME,
    completed_at    DATETIME
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_todos_due_date ON todos(due_date);
CREATE INDEX IF NOT EXISTS idx_todos_category ON todos(category);
CREATE INDEX IF NOT EXISTS idx_todos_urgency ON todos(urgency);
CREATE INDEX IF NOT EXISTS idx_todos_status ON todos(status);
CREATE INDEX IF NOT EXISTS idx_raw_queue_status ON raw_queue(status);
CREATE INDEX IF NOT EXISTS idx_raw_queue_last_attempted ON raw_queue(last_attempted);
