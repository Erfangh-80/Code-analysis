package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/application"
	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
	"github.com/Erfangh-80/code-analysis/backend/internal/infrastructure/sqlite"
)

func TestStorePersistsCompletedReportAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analyses.db")
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("open SQLite store: %v", err)
	}

	run := fixtureRun("analysis_completed", domain.AnalysisStatusQueued)
	if err := store.CreateQueued(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	run.Status = domain.AnalysisStatusRunning
	if err := store.SaveRun(context.Background(), run); err != nil {
		t.Fatalf("save running run: %v", err)
	}
	finished := time.Now().UTC()
	completed := run
	completed.Status = domain.AnalysisStatusCompleted
	completed.CompletedAt = &finished
	completed.Summary = &domain.AnalysisSummary{FilesDiscovered: 1, FilesAnalyzed: 1, GitStatus: domain.StageStatusUnavailable, RiskStatus: domain.StageStatusCompleted, ScoredFiles: 1}
	report := domain.AnalysisReport{
		Analysis: completed,
		Artifacts: domain.AnalysisArtifacts{
			Static: &domain.BatchAnalysis{Files: []domain.FileAnalysis{{Path: "sample.go", Language: domain.LanguageGo, Status: domain.AnalysisStatusAnalyzed}}},
			Risk:   &domain.RiskAnalysisReport{Profile: domain.RiskProfile{Version: "risk-v1"}},
		},
	}
	if err := store.SaveReport(context.Background(), run.ID, report); err != nil {
		t.Fatalf("save report: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	reopened, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("reopen SQLite store: %v", err)
	}
	defer reopened.Close()
	restored, err := reopened.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get persisted run: %v", err)
	}
	if restored.Status != domain.AnalysisStatusCompleted || restored.Summary == nil || restored.Summary.ScoredFiles != 1 {
		t.Errorf("restored run = %+v", restored)
	}
	restoredReport, err := reopened.GetReport(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get persisted report: %v", err)
	}
	if restoredReport.Artifacts.Static == nil || len(restoredReport.Artifacts.Static.Files) != 1 || restoredReport.Artifacts.Risk == nil {
		t.Errorf("restored report = %+v", restoredReport)
	}
}

func TestStoreRejectsSecondActiveRunForSameScope(t *testing.T) {
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "analyses.db"))
	if err != nil {
		t.Fatalf("open SQLite store: %v", err)
	}
	defer store.Close()

	first := fixtureRun("analysis_first", domain.AnalysisStatusQueued)
	second := fixtureRun("analysis_second", domain.AnalysisStatusQueued)
	if err := store.CreateQueued(context.Background(), first); err != nil {
		t.Fatalf("create first run: %v", err)
	}
	if err := store.CreateQueued(context.Background(), second); err != application.ErrAnalysisAlreadyRunning {
		t.Fatalf("create duplicate run error = %v, want ErrAnalysisAlreadyRunning", err)
	}
}

func TestStoreAllowsActiveRunsForDistinctScopes(t *testing.T) {
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "analyses.db"))
	if err != nil {
		t.Fatalf("open SQLite store: %v", err)
	}
	defer store.Close()

	first := fixtureRun("analysis_project_one", domain.AnalysisStatusQueued)
	second := fixtureRun("analysis_project_two", domain.AnalysisStatusQueued)
	second.Scope.ScopeKey = "another-scope-key"
	if err := store.CreateQueued(context.Background(), first); err != nil {
		t.Fatalf("create first distinct run: %v", err)
	}
	if err := store.CreateQueued(context.Background(), second); err != nil {
		t.Fatalf("create second distinct run: %v", err)
	}
}

func TestStoreRecoversInterruptedRunAfterReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analyses.db")
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("open SQLite store: %v", err)
	}
	run := fixtureRun("analysis_interrupted", domain.AnalysisStatusQueued)
	if err := store.CreateQueued(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	run.Status = domain.AnalysisStatusRunning
	if err := store.SaveRun(context.Background(), run); err != nil {
		t.Fatalf("save running run: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	reopened, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("reopen SQLite store: %v", err)
	}
	defer reopened.Close()
	recovered, err := reopened.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get recovered run: %v", err)
	}
	if recovered.Status != domain.AnalysisStatusFailed || len(recovered.Diagnostics) == 0 || recovered.Diagnostics[len(recovered.Diagnostics)-1].Code != "BACKEND_RESTARTED" {
		t.Errorf("recovered run = %+v", recovered)
	}
}

func fixtureRun(id domain.AnalysisID, status domain.AnalysisRunStatus) domain.AnalysisRun {
	now := time.Now().UTC()
	return domain.AnalysisRun{
		ID:        id,
		Status:    status,
		CreatedAt: now,
		History:   domain.GitHistoryRange{},
		Stages: []domain.StageSnapshot{
			{Stage: domain.AnalysisStageValidation, Status: domain.StageStatusPending},
		},
		Diagnostics: []domain.AnalysisDiagnostic{},
		Scope: domain.ProjectScope{
			ProjectPath: "/local/project",
			ScopeKey:    "scope-key",
		},
	}
}
