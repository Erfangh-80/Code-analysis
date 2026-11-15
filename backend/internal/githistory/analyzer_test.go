package githistory_test

import (
	"context"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
	"github.com/Erfangh-80/code-analysis/backend/internal/githistory"
)

func TestAnalyzerCalculatesFileHistoryMetricsAndPreservesRenames(t *testing.T) {
	t.Parallel()

	repository := createHistoryFixture(t)
	report, err := githistory.NewAnalyzer().Analyze(context.Background(), domain.GitAnalysisRequest{
		RepositoryPath: repository,
	})
	if err != nil {
		t.Fatalf("analyze repository history: %v", err)
	}

	if report.CommitCount != 4 {
		t.Fatalf("commit count = %d, want 4", report.CommitCount)
	}
	if len(report.Files) != 1 {
		t.Fatalf("file count = %d, want 1", len(report.Files))
	}

	file := report.Files[0]
	if file.Path != "src/current.go" {
		t.Errorf("path = %q, want src/current.go", file.Path)
	}
	if !file.Deleted {
		t.Error("deleted = false, want true after final delete")
	}
	if len(file.PreviousPaths) != 1 || file.PreviousPaths[0] != "src/legacy.go" {
		t.Errorf("previous paths = %v, want [src/legacy.go]", file.PreviousPaths)
	}
	if file.GitChurn != 6 {
		t.Errorf("git churn = %d, want 6", file.GitChurn)
	}
	if file.ChangeFrequency != 4 {
		t.Errorf("change frequency = %d, want 4", file.ChangeFrequency)
	}
	if file.AuthorCount != 2 {
		t.Errorf("author count = %d, want 2", file.AuthorCount)
	}
	if got := file.FirstChangedAt.UTC().Format(time.DateOnly); got != "2024-01-10" {
		t.Errorf("first change = %s, want 2024-01-10", got)
	}
	if got := file.LastChangedAt.UTC().Format(time.DateOnly); got != "2024-04-20" {
		t.Errorf("last change = %s, want 2024-04-20", got)
	}
	if math.Abs(file.ChangeConcentration-0.25) > 0.0001 {
		t.Errorf("change concentration = %.4f, want 0.25", file.ChangeConcentration)
	}
	if math.Abs(file.ChangeDispersion-0.75) > 0.0001 {
		t.Errorf("change dispersion = %.4f, want 0.75", file.ChangeDispersion)
	}
}

func TestAnalyzerSupportsOptionalHistoryRange(t *testing.T) {
	t.Parallel()

	repository := createHistoryFixture(t)
	from := mustTime(t, "2024-03-01T00:00:00Z")
	to := mustTime(t, "2024-12-31T23:59:59Z")
	report, err := githistory.NewAnalyzer().Analyze(context.Background(), domain.GitAnalysisRequest{
		RepositoryPath: repository,
		History: domain.GitHistoryRange{
			From: &from,
			To:   &to,
		},
	})
	if err != nil {
		t.Fatalf("analyze time-limited history: %v", err)
	}
	if report.CommitCount != 2 {
		t.Fatalf("commit count = %d, want 2", report.CommitCount)
	}
	if len(report.Files) != 1 {
		t.Fatalf("file count = %d, want 1", len(report.Files))
	}
	file := report.Files[0]
	if file.ChangeFrequency != 2 || file.GitChurn != 2 {
		t.Errorf("time-limited metrics = %+v, want frequency=2 churn=2", file)
	}
}

func TestAnalyzerRejectsInvalidRepositoryAndHistoryRange(t *testing.T) {
	t.Parallel()

	_, err := githistory.NewAnalyzer().Analyze(context.Background(), domain.GitAnalysisRequest{
		RepositoryPath: t.TempDir(),
	})
	if !errors.Is(err, githistory.ErrInvalidRepository) {
		t.Fatalf("invalid repository error = %v, want ErrInvalidRepository", err)
	}

	from := mustTime(t, "2024-02-01T00:00:00Z")
	to := mustTime(t, "2024-01-01T00:00:00Z")
	_, err = githistory.NewAnalyzer().Analyze(context.Background(), domain.GitAnalysisRequest{
		RepositoryPath: t.TempDir(),
		History:        domain.GitHistoryRange{From: &from, To: &to},
	})
	if !errors.Is(err, githistory.ErrInvalidHistoryRange) {
		t.Fatalf("invalid range error = %v, want ErrInvalidHistoryRange", err)
	}
}

func TestAnalyzerReportsRepositoryWithoutHistory(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	runGit(t, repository, nil, "init")

	report, err := githistory.NewAnalyzer().Analyze(context.Background(), domain.GitAnalysisRequest{
		RepositoryPath: repository,
	})
	if err != nil {
		t.Fatalf("analyze empty repository: %v", err)
	}
	if report.CommitCount != 0 || len(report.Files) != 0 {
		t.Errorf("empty repository result = %+v, want no commits or files", report)
	}
	if len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != domain.DiagnosticCodeInsufficientHistory {
		t.Errorf("diagnostics = %+v, want insufficient-history diagnostic", report.Diagnostics)
	}
}

func createHistoryFixture(t *testing.T) string {
	t.Helper()

	repository := t.TempDir()
	runGit(t, repository, nil, "init")
	runGit(t, repository, nil, "config", "user.name", "Fixture User")
	runGit(t, repository, nil, "config", "user.email", "fixture@example.test")

	legacyPath := filepath.Join(repository, "src", "legacy.go")
	writeHistoryFile(t, legacyPath, "package sample\nfunc value() int { return 1 }\n")
	commitAll(t, repository, "initial", "Alice", "alice@example.test", "2024-01-10T12:00:00Z")

	writeHistoryFile(t, legacyPath, "package sample\nfunc value() int { return 2 }\n")
	commitAll(t, repository, "modify", "Bob", "bob@example.test", "2024-02-15T12:00:00Z")

	runGit(t, repository, nil, "mv", "src/legacy.go", "src/current.go")
	commitAll(t, repository, "rename", "Alice", "alice@example.test", "2024-03-18T12:00:00Z")

	if err := os.Remove(filepath.Join(repository, "src", "current.go")); err != nil {
		t.Fatalf("delete renamed fixture: %v", err)
	}
	commitAll(t, repository, "delete", "Bob", "bob@example.test", "2024-04-20T12:00:00Z")
	return repository
}

func commitAll(t *testing.T, repository, message, authorName, authorEmail, timestamp string) {
	t.Helper()
	runGit(t, repository, nil, "add", "-A")
	environment := map[string]string{
		"GIT_AUTHOR_NAME":     authorName,
		"GIT_AUTHOR_EMAIL":    authorEmail,
		"GIT_AUTHOR_DATE":     timestamp,
		"GIT_COMMITTER_NAME":  authorName,
		"GIT_COMMITTER_EMAIL": authorEmail,
		"GIT_COMMITTER_DATE":  timestamp,
	}
	runGit(t, repository, environment, "commit", "-m", message)
}

func writeHistoryFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
}

func runGit(t *testing.T, repository string, environment map[string]string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repository}, arguments...)...)
	command.Env = os.Environ()
	for key, value := range environment {
		command.Env = append(command.Env, key+"="+value)
	}
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}
