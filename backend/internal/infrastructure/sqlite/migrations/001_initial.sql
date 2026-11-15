CREATE TABLE IF NOT EXISTS analysis_runs (
    id TEXT PRIMARY KEY,
    scope_key TEXT NOT NULL,
    project_path TEXT NOT NULL,
    git_repository_root TEXT NOT NULL,
    git_relative_scope TEXT NOT NULL,
    status TEXT NOT NULL,
    current_stage TEXT,
    history_json TEXT NOT NULL,
    profile_json TEXT,
    stages_json TEXT NOT NULL,
    progress_json TEXT NOT NULL,
    diagnostics_json TEXT NOT NULL,
    summary_json TEXT,
    created_at TEXT NOT NULL,
    started_at TEXT,
    completed_at TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS active_analysis_scope
ON analysis_runs(scope_key)
WHERE status IN ('queued', 'running');

CREATE TABLE IF NOT EXISTS analysis_reports (
    analysis_id TEXT PRIMARY KEY REFERENCES analysis_runs(id) ON DELETE CASCADE,
    result_json TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS analysis_runs_status_created
ON analysis_runs(status, created_at DESC);
