package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

func TestGitAnalysisEndpointReturnsLocalHistoryMetrics(t *testing.T) {
	t.Parallel()

	repository := createAPIGitFixture(t)
	payload, err := json.Marshal(map[string]any{
		"repositoryPath": repository,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/git-analysis", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	NewHandler("test-version").ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	var report domain.GitAnalysisReport
	if err := json.NewDecoder(response.Body).Decode(&report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.CommitCount != 1 || len(report.Files) != 1 {
		t.Fatalf("report = %+v, want one commit and one file", report)
	}
	if report.Files[0].Path != "sample.go" || report.Files[0].GitChurn != 2 || report.Files[0].ChangeFrequency != 1 {
		t.Errorf("file metrics = %+v, want sample.go churn=2 frequency=1", report.Files[0])
	}
}

func TestGitAnalysisEndpointRejectsInvalidHistoryRange(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"repositoryPath":"/tmp/repository",
		"history":{"from":"2024-02-01T00:00:00Z","to":"2024-01-01T00:00:00Z"}
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/git-analysis", bytes.NewReader(payload))
	response := httptest.NewRecorder()
	NewHandler("test-version").ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	assertErrorCode(t, response, "VALIDATION_ERROR")
}

func createAPIGitFixture(t *testing.T) string {
	t.Helper()

	repository := t.TempDir()
	runAPIGit(t, repository, "init")
	runAPIGit(t, repository, "config", "user.name", "Fixture User")
	runAPIGit(t, repository, "config", "user.email", "fixture@example.test")
	if err := os.WriteFile(filepath.Join(repository, "sample.go"), []byte("package sample\nfunc value() int { return 1 }\n"), 0o600); err != nil {
		t.Fatalf("write fixture source: %v", err)
	}
	runAPIGit(t, repository, "add", "sample.go")
	runAPIGit(t, repository, "commit", "-m", "initial")
	return repository
}

func runAPIGit(t *testing.T, repository string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repository}, arguments...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
}
