// Package sqlite provides local durable RunStore infrastructure.
package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/Erfangh-80/code-analysis/backend/internal/application"
	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

//go:embed migrations/001_initial.sql
var initialMigration string

// Store is a local SQLite implementation of application.RunStore.
type Store struct {
	database *sql.DB
}

// Open opens or creates a local SQLite database and applies embedded migrations.
func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("SQLite database path is required")
	}
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, fmt.Errorf("create SQLite data directory: %w", err)
		}
	}
	dsn := path + "?_foreign_keys=on&_busy_timeout=5000"
	if path == ":memory:" {
		dsn = fmt.Sprintf("file:code-risk-analyzer-%d?mode=memory&cache=shared&_foreign_keys=on&_busy_timeout=5000", time.Now().UTC().UnixNano())
	}
	database, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	database.SetMaxOpenConns(1)
	store := &Store{database: database}
	if err := store.migrate(context.Background()); err != nil {
		_ = database.Close()
		return nil, err
	}
	if _, err := database.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("enable SQLite WAL mode: %w", err)
	}
	if err := store.RecoverInterruptedRuns(context.Background()); err != nil {
		_ = database.Close()
		return nil, err
	}
	return store, nil
}

// OpenMemory creates an isolated ephemeral SQLite store suitable for handler tests.
func OpenMemory() (*Store, error) {
	return Open(":memory:")
}

// Close closes the local SQLite database.
func (store *Store) Close() error {
	if store == nil || store.database == nil {
		return nil
	}
	return store.database.Close()
}

func (store *Store) migrate(ctx context.Context) error {
	if _, err := store.database.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}
	var count int
	if err := store.database.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version = 1`).Scan(&count); err != nil {
		return fmt.Errorf("inspect migration state: %w", err)
	}
	if count != 0 {
		return nil
	}
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration transaction: %w", err)
	}
	defer transaction.Rollback()
	if _, err := transaction.ExecContext(ctx, initialMigration); err != nil {
		return fmt.Errorf("apply initial migration: %w", err)
	}
	if _, err := transaction.ExecContext(ctx, `INSERT INTO schema_migrations(version, applied_at) VALUES(1, ?)`, timestamp(time.Now().UTC())); err != nil {
		return fmt.Errorf("record initial migration: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit initial migration: %w", err)
	}
	return nil
}

// CreateQueued inserts an immutable queued-run identity. SQLite enforces the
// single active-run constraint for each canonical scope key.
func (store *Store) CreateQueued(ctx context.Context, run domain.AnalysisRun) error {
	values, err := runValues(run)
	if err != nil {
		return err
	}
	_, err = store.database.ExecContext(ctx, `
		INSERT INTO analysis_runs (
			id, scope_key, project_path, git_repository_root, git_relative_scope,
			status, current_stage, history_json, profile_json, stages_json,
			progress_json, diagnostics_json, summary_json, created_at, started_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, values...)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: analysis_runs.scope_key") {
			return application.ErrAnalysisAlreadyRunning
		}
		return fmt.Errorf("create queued analysis: %w", err)
	}
	return nil
}

// Get returns one local persisted run without exposing its internal paths in JSON.
func (store *Store) Get(ctx context.Context, id domain.AnalysisID) (domain.AnalysisRun, error) {
	row := store.database.QueryRowContext(ctx, selectRun+` WHERE id = ?`, string(id))
	run, err := scanRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AnalysisRun{}, application.ErrAnalysisNotFound
	}
	if err != nil {
		return domain.AnalysisRun{}, err
	}
	return run, nil
}

// List returns persisted lifecycle snapshots in newest-first order.
func (store *Store) List(ctx context.Context, query domain.AnalysisListQuery) (domain.AnalysisPage, error) {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 50
	}
	if query.PageSize > 200 {
		query.PageSize = 200
	}
	where := ""
	arguments := make([]any, 0, 3)
	if query.Status != nil {
		where = " WHERE status = ?"
		arguments = append(arguments, string(*query.Status))
	}
	var total int
	if err := store.database.QueryRowContext(ctx, "SELECT COUNT(*) FROM analysis_runs"+where, arguments...).Scan(&total); err != nil {
		return domain.AnalysisPage{}, fmt.Errorf("count analyses: %w", err)
	}
	arguments = append(arguments, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := store.database.QueryContext(ctx, selectRun+where+` ORDER BY created_at DESC LIMIT ? OFFSET ?`, arguments...)
	if err != nil {
		return domain.AnalysisPage{}, fmt.Errorf("list analyses: %w", err)
	}
	defer rows.Close()
	items := make([]domain.AnalysisRun, 0)
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return domain.AnalysisPage{}, err
		}
		items = append(items, run)
	}
	if err := rows.Err(); err != nil {
		return domain.AnalysisPage{}, fmt.Errorf("iterate analyses: %w", err)
	}
	return domain.AnalysisPage{Items: items, Page: query.Page, PageSize: query.PageSize, TotalItems: total}, nil
}

// SaveRun persists an atomic lifecycle/progress snapshot.
func (store *Store) SaveRun(ctx context.Context, run domain.AnalysisRun) error {
	values, err := runValues(run)
	if err != nil {
		return err
	}
	result, err := store.database.ExecContext(ctx, `
		UPDATE analysis_runs SET
			scope_key = ?, project_path = ?, git_repository_root = ?, git_relative_scope = ?,
			status = ?, current_stage = ?, history_json = ?, profile_json = ?, stages_json = ?,
			progress_json = ?, diagnostics_json = ?, summary_json = ?, created_at = ?, started_at = ?, completed_at = ?
		WHERE id = ?
	`, append(values[1:], values[0])...)
	if err != nil {
		return fmt.Errorf("save analysis run: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect saved analysis run: %w", err)
	}
	if changed == 0 {
		return application.ErrAnalysisNotFound
	}
	return nil
}

// SaveReport atomically persists the immutable completed result and changes its
// parent run from running to completed, so a completed lifecycle record always
// has a report after a restart.
func (store *Store) SaveReport(ctx context.Context, id domain.AnalysisID, report domain.AnalysisReport) error {
	if report.Analysis.ID != id || report.Analysis.Status != domain.AnalysisStatusCompleted {
		return fmt.Errorf("save analysis report: report must belong to a completed analysis")
	}
	payload, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal analysis report: %w", err)
	}
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin report persistence: %w", err)
	}
	defer transaction.Rollback()
	var status string
	if err := transaction.QueryRowContext(ctx, `SELECT status FROM analysis_runs WHERE id = ?`, string(id)).Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return application.ErrAnalysisNotFound
		}
		return fmt.Errorf("look up analysis report parent: %w", err)
	}
	if domain.AnalysisRunStatus(status) != domain.AnalysisStatusRunning {
		return fmt.Errorf("save analysis report: analysis is not running")
	}
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO analysis_reports(analysis_id, result_json, created_at)
		VALUES(?, ?, ?)
		ON CONFLICT(analysis_id) DO UPDATE SET result_json = excluded.result_json, created_at = excluded.created_at
	`, string(id), string(payload), timestamp(time.Now().UTC())); err != nil {
		return fmt.Errorf("save analysis report: %w", err)
	}
	values, err := runValues(report.Analysis)
	if err != nil {
		return err
	}
	result, err := transaction.ExecContext(ctx, `
		UPDATE analysis_runs SET
			scope_key = ?, project_path = ?, git_repository_root = ?, git_relative_scope = ?,
			status = ?, current_stage = ?, history_json = ?, profile_json = ?, stages_json = ?,
			progress_json = ?, diagnostics_json = ?, summary_json = ?, created_at = ?, started_at = ?, completed_at = ?
		WHERE id = ?
	`, append(values[1:], values[0])...)
	if err != nil {
		return fmt.Errorf("finalize analysis report: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect finalized analysis report: %w", err)
	}
	if changed == 0 {
		return application.ErrAnalysisNotFound
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit completed analysis report: %w", err)
	}
	return nil
}

// GetReport restores a completed immutable report after restart.
func (store *Store) GetReport(ctx context.Context, id domain.AnalysisID) (domain.AnalysisReport, error) {
	var payload string
	err := store.database.QueryRowContext(ctx, `SELECT result_json FROM analysis_reports WHERE analysis_id = ?`, string(id)).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AnalysisReport{}, application.ErrAnalysisNotFound
	}
	if err != nil {
		return domain.AnalysisReport{}, fmt.Errorf("get analysis report: %w", err)
	}
	var report domain.AnalysisReport
	if err := json.Unmarshal([]byte(payload), &report); err != nil {
		return domain.AnalysisReport{}, fmt.Errorf("decode persisted analysis report: %w", err)
	}
	return report, nil
}

// RecoverInterruptedRuns marks runs that cannot safely resume after a backend
// restart as failed while preserving their persisted stage snapshots.
func (store *Store) RecoverInterruptedRuns(ctx context.Context) error {
	rows, err := store.database.QueryContext(ctx, selectRun+` WHERE status IN ('queued', 'running')`)
	if err != nil {
		return fmt.Errorf("list interrupted analyses: %w", err)
	}
	interrupted := make([]domain.AnalysisRun, 0)
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			rows.Close()
			return err
		}
		interrupted = append(interrupted, run)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, run := range interrupted {
		finished := time.Now().UTC()
		run.Status = domain.AnalysisStatusFailed
		run.CurrentStage = nil
		run.CompletedAt = &finished
		for index := range run.Stages {
			switch run.Stages[index].Status {
			case domain.StageStatusRunning:
				run.Stages[index].Status = domain.StageStatusFailed
				run.Stages[index].CompletedAt = &finished
			case domain.StageStatusPending:
				run.Stages[index].Status = domain.StageStatusSkipped
				run.Stages[index].CompletedAt = &finished
			}
		}
		run.Diagnostics = append(run.Diagnostics, domain.AnalysisDiagnostic{
			Stage:      domain.AnalysisStageValidation,
			Code:       "BACKEND_RESTARTED",
			Severity:   "error",
			Message:    "The local backend restarted before this analysis could finish.",
			RecordedAt: finished,
		})
		if err := store.SaveRun(ctx, run); err != nil {
			return err
		}
	}
	return nil
}

const selectRun = `SELECT
	id, scope_key, project_path, git_repository_root, git_relative_scope,
	status, current_stage, history_json, profile_json, stages_json,
	progress_json, diagnostics_json, summary_json, created_at, started_at, completed_at
FROM analysis_runs`

type rowScanner interface {
	Scan(...any) error
}

func scanRun(scanner rowScanner) (domain.AnalysisRun, error) {
	var (
		id, scopeKey, projectPath, gitRoot, gitScope  string
		status, historyJSON, stagesJSON, progressJSON string
		diagnosticsJSON, createdAt                    string
		currentStage, profileJSON, summaryJSON        sql.NullString
		startedAt, completedAt                        sql.NullString
	)
	if err := scanner.Scan(
		&id, &scopeKey, &projectPath, &gitRoot, &gitScope,
		&status, &currentStage, &historyJSON, &profileJSON, &stagesJSON,
		&progressJSON, &diagnosticsJSON, &summaryJSON, &createdAt, &startedAt, &completedAt,
	); err != nil {
		return domain.AnalysisRun{}, err
	}
	created, err := parseTimestamp(createdAt)
	if err != nil {
		return domain.AnalysisRun{}, err
	}
	run := domain.AnalysisRun{
		ID:        domain.AnalysisID(id),
		Status:    domain.AnalysisRunStatus(status),
		CreatedAt: created,
		Scope: domain.ProjectScope{
			ProjectPath:       projectPath,
			ScopeKey:          scopeKey,
			GitRepositoryRoot: gitRoot,
			GitRelativeScope:  gitScope,
			GitAvailable:      gitRoot != "",
		},
	}
	if err := json.Unmarshal([]byte(historyJSON), &run.History); err != nil {
		return domain.AnalysisRun{}, fmt.Errorf("decode analysis history: %w", err)
	}
	if err := json.Unmarshal([]byte(stagesJSON), &run.Stages); err != nil {
		return domain.AnalysisRun{}, fmt.Errorf("decode analysis stages: %w", err)
	}
	if err := json.Unmarshal([]byte(progressJSON), &run.Progress); err != nil {
		return domain.AnalysisRun{}, fmt.Errorf("decode analysis progress: %w", err)
	}
	if err := json.Unmarshal([]byte(diagnosticsJSON), &run.Diagnostics); err != nil {
		return domain.AnalysisRun{}, fmt.Errorf("decode analysis diagnostics: %w", err)
	}
	if currentStage.Valid {
		stage := domain.AnalysisCurrentStage(currentStage.String)
		run.CurrentStage = &stage
	}
	if profileJSON.Valid && profileJSON.String != "" {
		var profile domain.RiskProfile
		if err := json.Unmarshal([]byte(profileJSON.String), &profile); err != nil {
			return domain.AnalysisRun{}, fmt.Errorf("decode analysis profile: %w", err)
		}
		run.Profile = &profile
	}
	if summaryJSON.Valid && summaryJSON.String != "" {
		var summary domain.AnalysisSummary
		if err := json.Unmarshal([]byte(summaryJSON.String), &summary); err != nil {
			return domain.AnalysisRun{}, fmt.Errorf("decode analysis summary: %w", err)
		}
		run.Summary = &summary
	}
	if startedAt.Valid && startedAt.String != "" {
		value, err := parseTimestamp(startedAt.String)
		if err != nil {
			return domain.AnalysisRun{}, err
		}
		run.StartedAt = &value
	}
	if completedAt.Valid && completedAt.String != "" {
		value, err := parseTimestamp(completedAt.String)
		if err != nil {
			return domain.AnalysisRun{}, err
		}
		run.CompletedAt = &value
	}
	return run, nil
}

func runValues(run domain.AnalysisRun) ([]any, error) {
	history, err := marshal(run.History)
	if err != nil {
		return nil, err
	}
	var profile any
	if run.Profile != nil {
		profile, err = marshal(run.Profile)
		if err != nil {
			return nil, err
		}
	}
	stages, err := marshal(run.Stages)
	if err != nil {
		return nil, err
	}
	progress, err := marshal(run.Progress)
	if err != nil {
		return nil, err
	}
	diagnostics, err := marshal(run.Diagnostics)
	if err != nil {
		return nil, err
	}
	var summary any
	if run.Summary != nil {
		summary, err = marshal(run.Summary)
		if err != nil {
			return nil, err
		}
	}
	var currentStage any
	if run.CurrentStage != nil {
		currentStage = string(*run.CurrentStage)
	}
	return []any{
		string(run.ID), run.Scope.ScopeKey, run.Scope.ProjectPath, run.Scope.GitRepositoryRoot, run.Scope.GitRelativeScope,
		string(run.Status), currentStage, history, profile, stages, progress, diagnostics, summary,
		timestamp(run.CreatedAt), nullableTimestamp(run.StartedAt), nullableTimestamp(run.CompletedAt),
	}, nil
}

func marshal(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func timestamp(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func nullableTimestamp(value *time.Time) any {
	if value == nil {
		return nil
	}
	return timestamp(*value)
}

func parseTimestamp(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse SQLite timestamp: %w", err)
	}
	return parsed.UTC(), nil
}
