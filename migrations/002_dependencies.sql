-- Conductor Schema: Dependencies
-- Version: 002
-- Description: Task dependency table with same-project constraint

CREATE TABLE IF NOT EXISTS conductor_dependency (
    id              TEXT PRIMARY KEY,       -- UUID
    task_id         TEXT NOT NULL REFERENCES conductor_task(id),
    blocked_by_id   TEXT NOT NULL REFERENCES conductor_task(id),
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(task_id, blocked_by_id),
    CHECK (task_id != blocked_by_id)
);

CREATE INDEX IF NOT EXISTS idx_dependency_task ON conductor_dependency(task_id);
CREATE INDEX IF NOT EXISTS idx_dependency_blocked_by ON conductor_dependency(blocked_by_id);
