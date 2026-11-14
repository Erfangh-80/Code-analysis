package evaluation

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Erfangh-80/Code-analysis/backend/internal/domain"
)

func git(t *testing.T, repo string, env []string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repo}, args...)...)
	command.Env = append(os.Environ(), env...)
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return string(out)
}
func write(t *testing.T, repo, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
func commit(t *testing.T, repo, date, message string) {
	t.Helper()
	env := []string{"GIT_AUTHOR_NAME=A", "GIT_AUTHOR_EMAIL=a@test", "GIT_COMMITTER_NAME=A", "GIT_COMMITTER_EMAIL=a@test", "GIT_AUTHOR_DATE=" + date, "GIT_COMMITTER_DATE=" + date}
	git(t, repo, env, "add", ".")
	git(t, repo, env, "commit", "-m", message)
}
func TestRunUsesOnlyFutureFixSignalsAndComputesSpearman(t *testing.T) {
	repo := t.TempDir()
	git(t, repo, nil, "init")
	write(t, repo, "a.go", "a\n")
	write(t, repo, "b.py", "b\n")
	write(t, repo, "c.js", "c\n")
	commit(t, repo, "2025-01-01T00:00:00Z", "base")
	snapshot := stringTrim(git(t, repo, nil, "rev-parse", "HEAD"))
	write(t, repo, "a.go", "a\nfix\n")
	commit(t, repo, "2025-02-01T00:00:00Z", "fix top priority")
	write(t, repo, "a.go", "a\nfix\nagain\n")
	commit(t, repo, "2025-02-02T00:00:00Z", "resolve another bug")
	write(t, repo, "b.py", "b\nfix\n")
	commit(t, repo, "2025-02-03T00:00:00Z", "fix middle issue")
	files := []domain.FileResult{{Path: "a.go", HotspotScore: score(90)}, {Path: "b.py", HotspotScore: score(50)}, {Path: "c.js", HotspotScore: score(10)}}
	samples, fixes, rho, reason, err := Run(context.Background(), repo, snapshot, files, Options{Since: at(t, "2025-02-01T00:00:00Z"), Until: at(t, "2025-03-01T00:00:00Z")})
	if err != nil {
		t.Fatal(err)
	}
	if samples != 3 || fixes != 3 || reason != nil || rho == nil || *rho < .999 {
		t.Fatalf("samples=%d fixes=%d rho=%v reason=%v", samples, fixes, rho, reason)
	}
}
func TestRunReportsNullForNoFutureVariation(t *testing.T) {
	repo := t.TempDir()
	git(t, repo, nil, "init")
	write(t, repo, "a.go", "a\n")
	write(t, repo, "b.py", "b\n")
	write(t, repo, "c.js", "c\n")
	commit(t, repo, "2025-01-01T00:00:00Z", "base")
	snapshot := stringTrim(git(t, repo, nil, "rev-parse", "HEAD"))
	files := []domain.FileResult{{Path: "a.go", HotspotScore: score(1)}, {Path: "b.py", HotspotScore: score(2)}, {Path: "c.js", HotspotScore: score(3)}}
	_, _, rho, reason, err := Run(context.Background(), repo, snapshot, files, Options{Since: at(t, "2025-01-01T00:00:00Z"), Until: at(t, "2026-01-01T00:00:00Z")})
	if err != nil {
		t.Fatal(err)
	}
	if rho != nil || reason == nil {
		t.Fatalf("rho=%v reason=%v", rho, reason)
	}
}
func at(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
func score(value float64) *float64 { return &value }
func stringTrim(value string) string {
	for len(value) > 0 && (value[len(value)-1] == '\n' || value[len(value)-1] == '\r') {
		value = value[:len(value)-1]
	}
	return value
}
