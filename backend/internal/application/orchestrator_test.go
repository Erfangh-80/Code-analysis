package application_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/analysisguard"
	"github.com/Erfangh-80/code-analysis/backend/internal/analyzer/static"
	"github.com/Erfangh-80/code-analysis/backend/internal/application"
	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
	"github.com/Erfangh-80/code-analysis/backend/internal/githistory"
	"github.com/Erfangh-80/code-analysis/backend/internal/infrastructure/sqlite"
	"github.com/Erfangh-80/code-analysis/backend/internal/repositoryscan"
	"github.com/Erfangh-80/code-analysis/backend/internal/scoring"
	"github.com/Erfangh-80/code-analysis/backend/internal/storage/failedfiles"
)

func TestOrchestratorCompletesNonGitProjectWithStaticOnlyRisk(t *testing.T) {
	project := t.TempDir()
	writeOrchestratorFile(t, filepath.Join(project, ".gitignore"), "ignored.py\n")
	writeOrchestratorFile(t, filepath.Join(project, "sample.go"), "package sample\nfunc Value() int { return 1 }\n")
	writeOrchestratorFile(t, filepath.Join(project, "ignored.py"), "print('ignore')\n")

	orchestrator, closeStore := newTestOrchestrator(t)
	defer closeStore()
	run, err := orchestrator.Start(context.Background(), domain.AnalysisRequest{ProjectPath: project})
	if err != nil {
		t.Fatalf("start non-Git analysis: %v", err)
	}
	completed := waitForTerminalRun(t, orchestrator, run.ID)
	if completed.Status != domain.AnalysisStatusCompleted {
		t.Fatalf("status = %q, want completed; diagnostics=%+v", completed.Status, completed.Diagnostics)
	}
	if stageStatus(completed, domain.AnalysisStageStatic) != domain.StageStatusCompleted || stageStatus(completed, domain.AnalysisStageGit) != domain.StageStatusUnavailable || stageStatus(completed, domain.AnalysisStageRisk) != domain.StageStatusCompleted {
		t.Errorf("stage statuses = %+v", completed.Stages)
	}

	report, err := orchestrator.Report(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get completed report: %v", err)
	}
	if report.Artifacts.Static == nil || len(report.Artifacts.Static.Files) != 1 || report.Artifacts.Static.Files[0].Path != "sample.go" {
		t.Errorf("static artifact = %+v", report.Artifacts.Static)
	}
	if report.Artifacts.Git != nil {
		t.Errorf("Git artifact = %+v, want nil for non-Git project", report.Artifacts.Git)
	}
	if report.Artifacts.Risk == nil || len(report.Artifacts.Risk.UnavailableMetrics) != 2 {
		t.Errorf("risk artifact = %+v, want static-only scoring", report.Artifacts.Risk)
	}
}

func TestOrchestratorCompletesProjectWithNoSupportedSourceFiles(t *testing.T) {
	project := t.TempDir()
	writeOrchestratorFile(t, filepath.Join(project, "README.md"), "No supported source files.\n")

	orchestrator, closeStore := newTestOrchestrator(t)
	defer closeStore()
	run, err := orchestrator.Start(context.Background(), domain.AnalysisRequest{ProjectPath: project})
	if err != nil {
		t.Fatalf("start empty analysis: %v", err)
	}
	completed := waitForTerminalRun(t, orchestrator, run.ID)
	if completed.Status != domain.AnalysisStatusCompleted {
		t.Fatalf("status = %q, want completed; diagnostics=%+v", completed.Status, completed.Diagnostics)
	}
	if stageStatus(completed, domain.AnalysisStageStatic) != domain.StageStatusCompleted || stageStatus(completed, domain.AnalysisStageRisk) != domain.StageStatusUnavailable {
		t.Errorf("stage statuses = %+v", completed.Stages)
	}
}

func TestOrchestratorCancelsAnInProgressAnalysis(t *testing.T) {
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "analyses.db"))
	if err != nil {
		t.Fatalf("open test SQLite store: %v", err)
	}
	defer store.Close()
	orchestrator := application.NewOrchestrator(
		store,
		blockingScanner{},
		application.NewStaticAnalysisService(static.NewAnalyzer(), failedfiles.NewRegistry()),
		githistory.NewAnalyzer(),
		scoring.NewEngine(),
		1,
	)
	run, err := orchestrator.Start(context.Background(), domain.AnalysisRequest{ProjectPath: "/fake/project"})
	if err != nil {
		t.Fatalf("start blocking analysis: %v", err)
	}
	waitForStatus(t, orchestrator, run.ID, domain.AnalysisStatusRunning)
	if _, err := orchestrator.Cancel(context.Background(), run.ID); err != nil {
		t.Fatalf("cancel analysis: %v", err)
	}
	cancelled := waitForTerminalRun(t, orchestrator, run.ID)
	if cancelled.Status != domain.AnalysisStatusCancelled {
		t.Errorf("status = %q, want cancelled", cancelled.Status)
	}
	if stageStatus(cancelled, domain.AnalysisStageDiscovery) != domain.StageStatusCancelled {
		t.Errorf("stage statuses = %+v", cancelled.Stages)
	}
}

func TestOrchestratorRetainsStaticOnlyReportWhenGitStageIsUnavailable(t *testing.T) {
	project := t.TempDir()
	writeOrchestratorFile(t, filepath.Join(project, "sample.go"), "package sample\nfunc Value() int { return 1 }\n")
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "analyses.db"))
	if err != nil {
		t.Fatalf("open test SQLite store: %v", err)
	}
	defer store.Close()
	scope := domain.ProjectScope{ProjectPath: project, ScopeKey: "mock-git-project", GitAvailable: true, GitRepositoryRoot: project}
	orchestrator := application.NewOrchestrator(
		store,
		fixedScanner{scope: scope, files: []string{"sample.go"}},
		application.NewStaticAnalysisService(static.NewAnalyzer(), failedfiles.NewRegistry()),
		unavailableGitAnalyzer{},
		scoring.NewEngine(),
		1,
	)
	run, err := orchestrator.Start(context.Background(), domain.AnalysisRequest{ProjectPath: project})
	if err != nil {
		t.Fatalf("start Git-unavailable analysis: %v", err)
	}
	completed := waitForTerminalRun(t, orchestrator, run.ID)
	if completed.Status != domain.AnalysisStatusCompleted || stageStatus(completed, domain.AnalysisStageGit) != domain.StageStatusUnavailable || stageStatus(completed, domain.AnalysisStageRisk) != domain.StageStatusCompleted {
		t.Fatalf("run = %+v", completed)
	}
	report, err := orchestrator.Report(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get report: %v", err)
	}
	if report.Artifacts.Risk == nil || len(report.Artifacts.Risk.UnavailableMetrics) != 2 {
		t.Errorf("static-only risk report = %+v", report.Artifacts.Risk)
	}
}

func TestOrchestratorScopesStaticAndGitAnalysisToSelectedGitSubdirectory(t *testing.T) {
	repository := t.TempDir()
	runOrchestratorGit(t, repository, "init")
	runOrchestratorGit(t, repository, "config", "user.name", "Orchestrator Fixture")
	runOrchestratorGit(t, repository, "config", "user.email", "orchestrator@example.test")
	writeOrchestratorFile(t, filepath.Join(repository, "backend", "worker.go"), "package backend\nfunc Work() {}\n")
	writeOrchestratorFile(t, filepath.Join(repository, "frontend", "app.js"), "function app() {}\n")
	runOrchestratorGit(t, repository, "add", ".")
	runOrchestratorGit(t, repository, "commit", "-m", "initial")

	orchestrator, closeStore := newTestOrchestrator(t)
	defer closeStore()
	run, err := orchestrator.Start(context.Background(), domain.AnalysisRequest{ProjectPath: filepath.Join(repository, "backend")})
	if err != nil {
		t.Fatalf("start scoped analysis: %v", err)
	}
	completed := waitForTerminalRun(t, orchestrator, run.ID)
	if completed.Status != domain.AnalysisStatusCompleted {
		t.Fatalf("status = %q, want completed; diagnostics=%+v", completed.Status, completed.Diagnostics)
	}
	report, err := orchestrator.Report(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get scoped report: %v", err)
	}
	if report.Artifacts.Static == nil || len(report.Artifacts.Static.Files) != 1 || report.Artifacts.Static.Files[0].Path != "worker.go" {
		t.Errorf("static scoped report = %+v", report.Artifacts.Static)
	}
	if report.Artifacts.Git == nil || len(report.Artifacts.Git.Files) != 1 || report.Artifacts.Git.Files[0].Path != "worker.go" {
		t.Errorf("Git scoped report = %+v", report.Artifacts.Git)
	}
}

func newTestOrchestrator(t *testing.T) (*application.Orchestrator, func()) {
	t.Helper()
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "analyses.db"))
	if err != nil {
		t.Fatalf("open test SQLite store: %v", err)
	}
	failures := failedfiles.NewRegistry()
	orchestrator := application.NewOrchestrator(
		store,
		repositoryscan.NewScanner(),
		application.NewStaticAnalysisService(static.NewAnalyzer(), failures),
		githistory.NewAnalyzer(),
		scoring.NewEngine(),
		2,
	)
	return orchestrator, func() { _ = store.Close() }
}

type fixedScanner struct {
	scope domain.ProjectScope
	files []string
}

func (scanner fixedScanner) Resolve(context.Context, string) (domain.ProjectScope, error) {
	return scanner.scope, nil
}

func (scanner fixedScanner) Scan(context.Context, domain.ProjectScope) (domain.ProjectScan, error) {
	return domain.ProjectScan{Scope: scanner.scope, Files: append([]string(nil), scanner.files...), Diagnostics: []domain.AnalysisDiagnostic{}}, nil
}

type blockingScanner struct{}

func (blockingScanner) Resolve(context.Context, string) (domain.ProjectScope, error) {
	return domain.ProjectScope{ProjectPath: "/fake/project", ScopeKey: "blocking-project"}, nil
}

func (blockingScanner) Scan(ctx context.Context, scope domain.ProjectScope) (domain.ProjectScan, error) {
	<-ctx.Done()
	return domain.ProjectScan{}, ctx.Err()
}

type unavailableGitAnalyzer struct{}

func (unavailableGitAnalyzer) Analyze(context.Context, domain.GitAnalysisRequest) (domain.GitAnalysisReport, error) {
	return domain.GitAnalysisReport{}, githistory.ErrHistoryUnavailable
}

func waitForStatus(t *testing.T, orchestrator *application.Orchestrator, id domain.AnalysisID, status domain.AnalysisRunStatus) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		run, err := orchestrator.Get(context.Background(), id)
		if err != nil {
			t.Fatalf("get analysis status: %v", err)
		}
		if run.Status == status {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("analysis %q did not reach status %q", id, status)
}

func waitForTerminalRun(t *testing.T, orchestrator *application.Orchestrator, id domain.AnalysisID) domain.AnalysisRun {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		run, err := orchestrator.Get(context.Background(), id)
		if err != nil {
			t.Fatalf("get analysis status: %v", err)
		}
		if run.Status == domain.AnalysisStatusCompleted || run.Status == domain.AnalysisStatusFailed || run.Status == domain.AnalysisStatusCancelled {
			return run
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("analysis %q did not reach a terminal state", id)
	return domain.AnalysisRun{}
}

func stageStatus(run domain.AnalysisRun, stage domain.AnalysisStage) domain.StageStatus {
	for _, snapshot := range run.Stages {
		if snapshot.Stage == stage {
			return snapshot.Status
		}
	}
	return ""
}

func writeOrchestratorFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
}

func runOrchestratorGit(t *testing.T, repository string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repository}, arguments...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
}
