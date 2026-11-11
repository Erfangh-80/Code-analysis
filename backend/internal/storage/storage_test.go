package storage

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Erfangh-80/Code-analysis/backend/internal/domain"
)

func TestStorePersistsAnalysisAndFilesAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "results.sqlite")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	analysis := domain.Analysis{ID: "a-1", Status: domain.StatusQueued, RepositoryPath: "/repos/demo", Ref: "HEAD", Languages: []string{"go"}, History: domain.HistoryRange{Since: now.Add(-time.Hour), Until: now}, Weights: domain.DefaultWeights(), FormulaVersion: domain.FormulaVersion, HistoryComplete: true, CreatedAt: now}
	if err := store.CreateAnalysis(analysis); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveFiles("a-1", []domain.FileResult{{ID: "f-1", AnalysisID: "a-1", Path: "main.go", Module: ".", Static: domain.StaticMetrics{Language: "go", AnalysisStatus: "ok"}}}); err != nil {
		t.Fatal(err)
	}
	store.Close()
	store, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	got, err := store.GetAnalysis("a-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.RepositoryPath != analysis.RepositoryPath || got.Status != domain.StatusQueued {
		t.Fatalf("analysis=%#v", got)
	}
	files, total, err := store.ListFiles("a-1", FileQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(files) != 1 || files[0].Path != "main.go" {
		t.Fatalf("files=%#v total=%d", files, total)
	}
}

func TestRecoverRunningJobsMarksThemFailed(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	analysis := domain.Analysis{ID: "running", Status: domain.StatusRunning, RepositoryPath: "/repos/demo", Ref: "HEAD", Languages: []string{"go"}, Weights: domain.DefaultWeights(), FormulaVersion: domain.FormulaVersion, CreatedAt: time.Now().UTC()}
	if err := store.CreateAnalysis(analysis); err != nil {
		t.Fatal(err)
	}
	if err := store.RecoverRunning(); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetAnalysis("running")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusFailed || got.FailureCode == nil || *got.FailureCode != "SERVER_RESTARTED" {
		t.Fatalf("recovery=%#v", got)
	}
}
