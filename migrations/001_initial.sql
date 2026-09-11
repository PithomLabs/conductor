-- Conductor Schema: Core Tables
-- Version: 001
-- Description: Initial schema for project, task, and activity tables

CREATE TABLE IF NOT EXISTS conductor_project (
    id          TEXT PRIMARY KEY,           -- UUID
    name        TEXT NOT NULL,
    description TEXT,
    status      TEXT NOT NULL DEFAULT 'active'
                CHECK (status IN ('active', 'completed', 'archived')),
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS conductor_task (
    id              TEXT PRIMARY KEY,       -- UUID
    project_id      TEXT NOT NULL REFERENCES conductor_project(id),
    title           TEXT NOT NULL,
    description     TEXT,                   -- Markdown specification
    status          TEXT NOT NULL DEFAULT 'proposed'
                    CHECK (status IN ('proposed', 'active', 'review', 'accepted', 'blocked', 'cancelled')),
    priority        TEXT NOT NULL DEFAULT 'medium'
                    CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    current_agent   TEXT,                   -- Agent identifier (nullable)
    governance_ref  TEXT,                   -- Opaque JSON governance reference (nullable)
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_task_project ON conductor_task(project_id);
CREATE INDEX IF NOT EXISTS idx_task_status ON conductor_task(status);
CREATE INDEX IF NOT EXISTS idx_task_agent ON conductor_task(current_agent);

CREATE TABLE IF NOT EXISTS conductor_activity (
    id          TEXT PRIMARY KEY,           -- UUID
    task_id     TEXT NOT NULL REFERENCES conductor_task(id),
    actor_type  TEXT NOT NULL CHECK (actor_type IN ('human', 'agent', 'system')),
    actor_id    TEXT NOT NULL,              -- Authenticated actor identifier
    action      TEXT NOT NULL,              -- Event type (see lifecycle)
    details     TEXT,                       -- JSON metadata (non-authoritative)
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_activity_task ON conductor_activity(task_id);
CREATE INDEX IF NOT EXISTS idx_activity_created ON conductor_activity(task_id, created_at DESC);
