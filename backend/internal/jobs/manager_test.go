package jobs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Erfangh-80/Code-analysis/backend/internal/domain"
	"github.com/Erfangh-80/Code-analysis/backend/internal/storage"
)

func runGit(t *testing.T, repo string, env []string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repo}, args...)...)
	command.Env = append(os.Environ(), env...)
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return string(out)
}
func commitTestRepo(t *testing.T, repo, date string) {
	t.Helper()
	env := []string{"GIT_AUTHOR_NAME=Demo", "GIT_AUTHOR_EMAIL=demo@test", "GIT_COMMITTER_NAME=Demo", "GIT_COMMITTER_EMAIL=demo@test", "GIT_AUTHOR_DATE=" + date, "GIT_COMMITTER_DATE=" + date}
	runGit(t, repo, env, "add", ".")
	runGit(t, repo, env, "commit", "-m", "initial")
}

func TestManagerAnalyzesTrackedCommitWithoutDirtyFiles(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "demo")
	if err := os.Mkdir(repo, 0755); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, nil, "init")
	if err := os.WriteFile(filepath.Join(repo, "app.py"), []byte("def run(ok):\n    if ok:\n        return 1\n    return 0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	commitTestRepo(t, repo, "2025-01-02T00:00:00Z")
	// Neither unstaged nor untracked source is allowed to enter the snapshot.
	if err := os.WriteFile(filepath.Join(repo, "app.py"), []byte("def broken(:\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "untracked.js"), []byte("const secret = 1"), 0644); err != nil {
		t.Fatal(err)
	}
	store, err := storage.Open(filepath.Join(root, "results.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	manager, err := New(store, Config{AllowedRepoRoot: root, AnalysisTimeout: time.Minute, MaxCommits: 50})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	analysis, err := manager.Submit(domain.AnalysisInput{RepositoryPath: repo, Ref: "HEAD", Languages: []string{"python"}, History: domain.HistoryRange{Since: mustParse(t, "2025-01-01T00:00:00Z"), Until: mustParse(t, "2025-02-01T00:00:00Z")}, Weights: domain.DefaultWeights()})
	if err != nil {
		t.Fatal(err)
	}
	final := waitForAnalysis(t, store, analysis.ID)
	if final.Status != domain.StatusCompleted {
		t.Fatalf("analysis failed/partial: %#v", final)
	}
	if final.CommitSHA == "" {
		t.Fatal("commit SHA was not captured")
	}
	files, total, err := store.ListFiles(analysis.ID, storage.FileQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(files) != 1 || files[0].Path != "app.py" || files[0].Static.AnalysisStatus != "ok" {
		t.Fatalf("snapshot results: %#v / %d", files, total)
	}
}
func waitForAnalysis(t *testing.T, store *storage.Store, id string) domain.Analysis {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		value, err := store.GetAnalysis(id)
		if err != nil {
			t.Fatal(err)
		}
		if value.Status == domain.StatusCompleted || value.Status == domain.StatusPartial || value.Status == domain.StatusFailed {
			return value
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatal("analysis job timed out")
	return domain.Analysis{}
}
func mustParse(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
