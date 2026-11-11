package jobs

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Erfangh-80/Code-analysis/backend/internal/storage"
)

func TestValidateRepositoryPathRejectsEscapeFromAllowedRoot(t *testing.T) {
	root := t.TempDir()
	store, err := storage.Open(filepath.Join(root, "results.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	manager, err := New(store, Config{AllowedRepoRoot: root, AnalysisTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	err = manager.validateRepositoryPath(filepath.Join(root, ".."))
	jobError, ok := err.(*JobError)
	if !ok || jobError.Code != "REPO_NOT_ALLOWED" {
		t.Fatalf("expected controlled root escape rejection, got %#v", err)
	}
}
