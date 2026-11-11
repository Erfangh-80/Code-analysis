package githistory

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func git(t *testing.T, dir string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return string(out)
}

func commit(t *testing.T, dir, name, email, date, message string) {
	t.Helper()
	env := []string{"GIT_AUTHOR_NAME=" + name, "GIT_AUTHOR_EMAIL=" + email, "GIT_COMMITTER_NAME=" + name, "GIT_COMMITTER_EMAIL=" + email, "GIT_AUTHOR_DATE=" + date, "GIT_COMMITTER_DATE=" + date}
	git(t, dir, env, "add", ".")
	git(t, dir, env, "commit", "-m", message)
}

func TestCollectComputesChurnFrequencyAuthorsAndDateBounds(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, nil, "init")
	if err := os.WriteFile(filepath.Join(dir, "a.py"), []byte("print(1)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	commit(t, dir, "Ada", "ada@example.test", "2025-01-02T00:00:00Z", "initial")
	if err := os.WriteFile(filepath.Join(dir, "a.py"), []byte("print(1)\nprint(2)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	commit(t, dir, "Ben", "ben@example.test", "2025-01-03T00:00:00Z", "fix parser")
	if err := os.WriteFile(filepath.Join(dir, "untouched.go"), []byte("package demo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	commit(t, dir, "Ben", "ben@example.test", "2025-01-04T00:00:00Z", "add Go file")

	result, err := Collect(dir, "HEAD", map[string]struct{}{"a.py": {}, "untouched.go": {}}, Options{Since: mustTime(t, "2025-01-01T00:00:00Z"), Until: mustTime(t, "2025-01-04T00:00:00Z"), MaxCommits: 50})
	if err != nil {
		t.Fatal(err)
	}
	a := result.Metrics["a.py"]
	if a.Churn != 2 || a.ChangeFrequency != 2 || a.AuthorCount != 2 {
		t.Fatalf("a metrics: %#v", a)
	}
	if a.AuthorConcentration == nil || *a.AuthorConcentration != .5 || a.AuthorDispersion == nil || *a.AuthorDispersion != .5 {
		t.Fatalf("concentration: %#v", a)
	}
	if a.FirstChange == nil || a.FirstChange.UTC().Format(time.RFC3339) != "2025-01-02T00:00:00Z" || a.LastChange == nil || a.LastChange.UTC().Format(time.RFC3339) != "2025-01-03T00:00:00Z" {
		t.Fatalf("dates: %#v", a)
	}
	untouched := result.Metrics["untouched.go"]
	if untouched.Churn != 0 || untouched.ChangeFrequency != 0 || untouched.AuthorConcentration != nil {
		t.Fatalf("untouched: %#v", untouched)
	}
}

func TestCollectMarksLimitAsIncompleteInsteadOfZero(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, nil, "init")
	if err := os.WriteFile(filepath.Join(dir, "x.js"), []byte("let x = 1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	commit(t, dir, "A", "a@test", "2025-01-01T00:00:00Z", "one")
	if err := os.WriteFile(filepath.Join(dir, "x.js"), []byte("let x = 2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	commit(t, dir, "A", "a@test", "2025-01-02T00:00:00Z", "two")
	result, err := Collect(dir, "HEAD", map[string]struct{}{"x.js": {}}, Options{Since: mustTime(t, "2024-01-01T00:00:00Z"), Until: mustTime(t, "2026-01-01T00:00:00Z"), MaxCommits: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Complete || len(result.Warnings) == 0 {
		t.Fatalf("limited result must disclose incompleteness: %#v", result)
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
