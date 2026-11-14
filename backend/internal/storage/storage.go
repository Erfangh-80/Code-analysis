// Package storage persists analysis metadata and results in a local SQLite file.
package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Erfangh-80/Code-analysis/backend/internal/domain"
	_ "github.com/mattn/go-sqlite3"
)

type Store struct{ db *sql.DB }

type FileQuery struct {
	Limit, Offset            int
	Search, Language, Module string
	MinComplexity, MinChurn  *float64
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY);
CREATE TABLE IF NOT EXISTS analyses(
 id TEXT PRIMARY KEY, status TEXT NOT NULL, repository_path TEXT NOT NULL, ref TEXT NOT NULL,
 commit_sha TEXT NOT NULL DEFAULT '', languages_json TEXT NOT NULL, history_since TEXT NOT NULL,
 history_until TEXT NOT NULL, weights_json TEXT NOT NULL, formula_version TEXT NOT NULL,
 history_complete INTEGER NOT NULL, current_stage TEXT NOT NULL DEFAULT '', files_processed INTEGER NOT NULL DEFAULT 0,
 files_discovered INTEGER NOT NULL DEFAULT 0, warnings_json TEXT NOT NULL DEFAULT '[]', failure_code TEXT,
 failure_message TEXT, created_at TEXT NOT NULL, started_at TEXT, finished_at TEXT
);
CREATE TABLE IF NOT EXISTS files(
 id TEXT PRIMARY KEY, analysis_id TEXT NOT NULL REFERENCES analyses(id) ON DELETE CASCADE, path TEXT NOT NULL,
 module TEXT NOT NULL, language TEXT NOT NULL, static_json TEXT NOT NULL, git_json TEXT NOT NULL,
 normalized_json TEXT NOT NULL, weight_json TEXT NOT NULL, contribution_json TEXT NOT NULL,
 hotspot_score REAL, score_reason TEXT, UNIQUE(analysis_id,path)
);
CREATE INDEX IF NOT EXISTS idx_files_analysis_score ON files(analysis_id, hotspot_score DESC);
CREATE TABLE IF NOT EXISTS modules(
 id TEXT PRIMARY KEY, analysis_id TEXT NOT NULL REFERENCES analyses(id) ON DELETE CASCADE,
 name TEXT NOT NULL, metrics_json TEXT NOT NULL, UNIQUE(analysis_id,name)
);
CREATE TABLE IF NOT EXISTS evaluations(
 id TEXT PRIMARY KEY, analysis_id TEXT NOT NULL REFERENCES analyses(id) ON DELETE CASCADE, result_json TEXT NOT NULL
);
INSERT OR IGNORE INTO schema_migrations(version) VALUES(1);`)
	return err
}

func (s *Store) CreateAnalysis(a domain.Analysis) error {
	languages, weights, warnings := marshal(a.Languages), marshal(a.Weights), marshal(a.Warnings)
	_, err := s.db.Exec(`INSERT INTO analyses(id,status,repository_path,ref,commit_sha,languages_json,history_since,history_until,weights_json,formula_version,history_complete,current_stage,files_processed,files_discovered,warnings_json,failure_code,failure_message,created_at,started_at,finished_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, a.ID, a.Status, a.RepositoryPath, a.Ref, a.CommitSHA, languages, timeText(a.History.Since), timeText(a.History.Until), weights, a.FormulaVersion, boolInt(a.HistoryComplete), a.CurrentStage, a.FilesProcessed, a.FilesDiscovered, warnings, a.FailureCode, a.FailureMessage, timeText(a.CreatedAt), nullableTime(a.StartedAt), nullableTime(a.FinishedAt))
	return err
}
func (s *Store) UpdateAnalysis(a domain.Analysis) error {
	languages, weights, warnings := marshal(a.Languages), marshal(a.Weights), marshal(a.Warnings)
	result, err := s.db.Exec(`UPDATE analyses SET status=?, repository_path=?, ref=?, commit_sha=?, languages_json=?, history_since=?, history_until=?, weights_json=?, formula_version=?, history_complete=?, current_stage=?, files_processed=?, files_discovered=?, warnings_json=?, failure_code=?, failure_message=?, started_at=?, finished_at=? WHERE id=?`, a.Status, a.RepositoryPath, a.Ref, a.CommitSHA, languages, timeText(a.History.Since), timeText(a.History.Until), weights, a.FormulaVersion, boolInt(a.HistoryComplete), a.CurrentStage, a.FilesProcessed, a.FilesDiscovered, warnings, a.FailureCode, a.FailureMessage, nullableTime(a.StartedAt), nullableTime(a.FinishedAt), a.ID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (s *Store) GetAnalysis(id string) (domain.Analysis, error) {
	row := s.db.QueryRow(analysisSelect+" WHERE id=?", id)
	return scanAnalysis(row)
}
func (s *Store) ListAnalyses() ([]domain.Analysis, error) {
	rows, err := s.db.Query(analysisSelect + " ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Analysis{}
	for rows.Next() {
		item, err := scanAnalysis(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (s *Store) RecoverRunning() error {
	_, err := s.db.Exec(`UPDATE analyses SET status='failed', current_stage='interrupted', failure_code='SERVER_RESTARTED', failure_message='سرور هنگام اجرای تحلیل متوقف شد؛ این Job دوباره اجرا نمی‌شود.', finished_at=? WHERE status='running'`, timeText(time.Now().UTC()))
	return err
}

const analysisSelect = `SELECT id,status,repository_path,ref,commit_sha,languages_json,history_since,history_until,weights_json,formula_version,history_complete,current_stage,files_processed,files_discovered,warnings_json,failure_code,failure_message,created_at,started_at,finished_at FROM analyses`

type scanner interface{ Scan(...any) error }

func scanAnalysis(row scanner) (domain.Analysis, error) {
	var a domain.Analysis
	var languages, weights, warnings string
	var since, until, created string
	var started, finished sql.NullString
	var complete int
	err := row.Scan(&a.ID, &a.Status, &a.RepositoryPath, &a.Ref, &a.CommitSHA, &languages, &since, &until, &weights, &a.FormulaVersion, &complete, &a.CurrentStage, &a.FilesProcessed, &a.FilesDiscovered, &warnings, &a.FailureCode, &a.FailureMessage, &created, &started, &finished)
	if err != nil {
		return a, err
	}
	if err := json.Unmarshal([]byte(languages), &a.Languages); err != nil {
		return a, err
	}
	if err := json.Unmarshal([]byte(weights), &a.Weights); err != nil {
		return a, err
	}
	if err := json.Unmarshal([]byte(warnings), &a.Warnings); err != nil {
		return a, err
	}
	if a.Warnings == nil {
		a.Warnings = []string{}
	}
	var parseErr error
	a.History.Since, parseErr = parseTime(since)
	if parseErr != nil {
		return a, parseErr
	}
	a.History.Until, parseErr = parseTime(until)
	if parseErr != nil {
		return a, parseErr
	}
	a.CreatedAt, parseErr = parseTime(created)
	if parseErr != nil {
		return a, parseErr
	}
	a.HistoryComplete = complete == 1
	a.StartedAt = parseNullable(started)
	a.FinishedAt = parseNullable(finished)
	return a, nil
}

func (s *Store) SaveFiles(analysisID string, files []domain.FileResult) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM files WHERE analysis_id=?", analysisID); err != nil {
		return err
	}
	statement, err := tx.Prepare(`INSERT INTO files(id,analysis_id,path,module,language,static_json,git_json,normalized_json,weight_json,contribution_json,hotspot_score,score_reason) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	for _, f := range files {
		_, err = statement.Exec(f.ID, analysisID, f.Path, f.Module, f.Static.Language, marshal(f.Static), marshal(f.Git), marshal(f.Normalized), marshal(f.EffectiveWeight), marshal(f.Contributions), f.HotspotScore, f.ScoreReason)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *Store) ListFiles(analysisID string, query FileQuery) ([]domain.FileResult, int, error) {
	if query.Limit <= 0 || query.Limit > 10000 {
		query.Limit = 50
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	where, args := fileFilter(analysisID, query)
	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM files"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Query(`SELECT id,analysis_id,path,module,language,static_json,git_json,normalized_json,weight_json,contribution_json,hotspot_score,score_reason FROM files`+where+` ORDER BY hotspot_score IS NULL,hotspot_score DESC,path ASC LIMIT ? OFFSET ?`, append(args, query.Limit, query.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []domain.FileResult{}
	for rows.Next() {
		f, err := scanFile(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, f)
	}
	return items, total, rows.Err()
}
func (s *Store) GetFile(analysisID, fileID string) (domain.FileResult, error) {
	return scanFile(s.db.QueryRow(`SELECT id,analysis_id,path,module,language,static_json,git_json,normalized_json,weight_json,contribution_json,hotspot_score,score_reason FROM files WHERE analysis_id=? AND id=?`, analysisID, fileID))
}
func (s *Store) AllFiles(analysisID string) ([]domain.FileResult, error) {
	// Higher-level aggregations use a bounded, intentionally explicit cap; the
	// analysis has its own source and time limits before this point.
	files, _, err := s.ListFiles(analysisID, FileQuery{Limit: 10000})
	return files, err
}
func fileFilter(analysisID string, q FileQuery) (string, []any) {
	parts, args := []string{" WHERE analysis_id=?"}, []any{analysisID}
	if q.Search != "" {
		parts = append(parts, " AND path LIKE ?")
		args = append(args, "%"+q.Search+"%")
	}
	if q.Language != "" {
		parts = append(parts, " AND language=?")
		args = append(args, q.Language)
	}
	if q.Module != "" {
		parts = append(parts, " AND module=?")
		args = append(args, q.Module)
	}
	return strings.Join(parts, ""), args
}
func scanFile(row scanner) (domain.FileResult, error) {
	var f domain.FileResult
	var staticM, gitM, normal, weight, contribution string
	var score sql.NullFloat64
	err := row.Scan(&f.ID, &f.AnalysisID, &f.Path, &f.Module, &f.Static.Language, &staticM, &gitM, &normal, &weight, &contribution, &score, &f.ScoreReason)
	if err != nil {
		return f, err
	}
	if err = json.Unmarshal([]byte(staticM), &f.Static); err != nil {
		return f, err
	}
	if err = json.Unmarshal([]byte(gitM), &f.Git); err != nil {
		return f, err
	}
	if err = json.Unmarshal([]byte(normal), &f.Normalized); err != nil {
		return f, err
	}
	if err = json.Unmarshal([]byte(weight), &f.EffectiveWeight); err != nil {
		return f, err
	}
	if err = json.Unmarshal([]byte(contribution), &f.Contributions); err != nil {
		return f, err
	}
	if score.Valid {
		value := score.Float64
		f.HotspotScore = &value
	}
	return f, nil
}

func (s *Store) SaveModules(analysisID string, modules []domain.ModuleResult) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec("DELETE FROM modules WHERE analysis_id=?", analysisID); err != nil {
		return err
	}
	for _, m := range modules {
		if _, err = tx.Exec("INSERT INTO modules(id,analysis_id,name,metrics_json) VALUES(?,?,?,?)", m.ID, analysisID, m.Name, marshal(m)); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *Store) ListModules(analysisID string) ([]domain.ModuleResult, error) {
	rows, err := s.db.Query("SELECT metrics_json FROM modules WHERE analysis_id=? ORDER BY name", analysisID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ModuleResult{}
	for rows.Next() {
		var raw string
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		var m domain.ModuleResult
		if err = json.Unmarshal([]byte(raw), &m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (s *Store) SaveEvaluation(e domain.Evaluation) error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO evaluations(id,analysis_id,result_json) VALUES(?,?,?)", e.ID, e.AnalysisID, marshal(e))
	return err
}
func (s *Store) GetEvaluation(analysisID, id string) (domain.Evaluation, error) {
	var raw string
	err := s.db.QueryRow("SELECT result_json FROM evaluations WHERE analysis_id=? AND id=?", analysisID, id).Scan(&raw)
	if err != nil {
		return domain.Evaluation{}, err
	}
	var e domain.Evaluation
	err = json.Unmarshal([]byte(raw), &e)
	return e, err
}

func marshal(value any) string {
	bytes, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}
func timeText(value time.Time) string {
	if value.IsZero() {
		return time.Time{}.UTC().Format(time.RFC3339Nano)
	}
	return value.UTC().Format(time.RFC3339Nano)
}
func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return timeText(*value)
}
func parseTime(value string) (time.Time, error) { return time.Parse(time.RFC3339Nano, value) }
func parseNullable(value sql.NullString) *time.Time {
	if !value.Valid {
		return nil
	}
	parsed, err := parseTime(value.String)
	if err != nil {
		return nil
	}
	return &parsed
}
func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

var _ = errors.New
var _ = finite
var _ = fmt.Sprintf
