package application_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/analysisguard"
	"github.com/Erfangh-80/code-analysis/backend/internal/analyzer/static"
	"github.com/Erfangh-80/code-analysis/backend/internal/application"
	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
	"github.com/Erfangh-80/code-analysis/backend/internal/githistory"
	"github.com/Erfangh-80/code-analysis/backend/internal/infrastructure/sqlite"
	"github.com/Erfangh-80/code-analysis/backend/internal/scoring"
	"github.com/Erfangh-80/code-analysis/backend/internal/storage/failedfiles"
)

func TestOrchestratorMarksTimedOutAnalysisAsControlledFailure(t *testing.T) {
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "analyses.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()
	limits := analysisguard.DefaultLimits()
	limits.AnalysisTimeout = time.Second
	orchestrator, err := application.NewOrchestratorWithLimits(
		store,
		blockingScanner{},
		application.NewStaticAnalysisService(static.NewAnalyzer(), failedfiles.NewRegistry()),
		githistory.NewAnalyzer(),
		scoring.NewEngine(),
		1,
		limits,
	)
	if err != nil {
		t.Fatalf("new bounded orchestrator: %v", err)
	}

	run, err := orchestrator.Start(context.Background(), domain.AnalysisRequest{ProjectPath: "/fake/project"})
	if err != nil {
		t.Fatalf("start analysis: %v", err)
	}
	terminal := waitForTerminalRun(t, orchestrator, run.ID)
	if terminal.Status != domain.AnalysisStatusFailed {
		t.Fatalf("status = %q, want failed; diagnostics=%+v", terminal.Status, terminal.Diagnostics)
	}
	if !hasRunDiagnostic(terminal, "ANALYSIS_TIMEOUT") {
		t.Errorf("diagnostics = %+v, want ANALYSIS_TIMEOUT", terminal.Diagnostics)
	}
}

func hasRunDiagnostic(run domain.AnalysisRun, code string) bool {
	for _, diagnostic := range run.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
