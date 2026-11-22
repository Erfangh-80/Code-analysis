package application_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Erfangh-80/code-analysis/backend/internal/analyzer/static"
	"github.com/Erfangh-80/code-analysis/backend/internal/application"
	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
	"github.com/Erfangh-80/code-analysis/backend/internal/githistory"
	"github.com/Erfangh-80/code-analysis/backend/internal/infrastructure/sqlite"
	"github.com/Erfangh-80/code-analysis/backend/internal/scoring"
	"github.com/Erfangh-80/code-analysis/backend/internal/storage/failedfiles"
)

func TestOrchestratorKeepsStaticOnlyReportWhenGitResourceLimitIsReached(t *testing.T) {
	project := t.TempDir()
	writeOrchestratorFile(t, filepath.Join(project, "sample.go"), "package sample\nfunc Value() int { return 1 }\n")
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "analyses.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()
	scope := domain.ProjectScope{ProjectPath: project, ScopeKey: "resource-limited-git-project", GitAvailable: true, GitRepositoryRoot: project}
	orchestrator := application.NewOrchestrator(
		store,
		fixedScanner{scope: scope, files: []string{"sample.go"}},
		application.NewStaticAnalysisService(static.NewAnalyzer(), failedfiles.NewRegistry()),
		resourceLimitedGitAnalyzer{},
		scoring.NewEngine(),
		1,
	)

	run, err := orchestrator.Start(context.Background(), domain.AnalysisRequest{ProjectPath: project})
	if err != nil {
		t.Fatalf("start analysis: %v", err)
	}
	completed := waitForTerminalRun(t, orchestrator, run.ID)
	if completed.Status != domain.AnalysisStatusCompleted || stageStatus(completed, domain.AnalysisStageGit) != domain.StageStatusUnavailable || stageStatus(completed, domain.AnalysisStageRisk) != domain.StageStatusCompleted {
		t.Fatalf("run = %+v", completed)
	}
	if !hasRunDiagnostic(completed, "GIT_RESOURCE_LIMIT") {
		t.Errorf("diagnostics = %+v, want GIT_RESOURCE_LIMIT", completed.Diagnostics)
	}
	report, err := orchestrator.Report(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get report: %v", err)
	}
	if report.Artifacts.Static == nil || report.Artifacts.Risk == nil || report.Artifacts.Git != nil {
		t.Errorf("artifacts = %+v, want static-only report", report.Artifacts)
	}
}

type resourceLimitedGitAnalyzer struct{}

func (resourceLimitedGitAnalyzer) Analyze(context.Context, domain.GitAnalysisRequest) (domain.GitAnalysisReport, error) {
	return domain.GitAnalysisReport{}, githistory.ErrResourceLimit
}
