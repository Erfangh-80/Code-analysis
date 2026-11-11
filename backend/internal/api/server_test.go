package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/Erfangh-80/Code-analysis/backend/internal/domain"
	"github.com/Erfangh-80/Code-analysis/backend/internal/jobs"
	"github.com/Erfangh-80/Code-analysis/backend/internal/storage"
)

func TestHealthIsPublicButAnalysisAPIRequiresToken(t *testing.T) {
	handler := testServer(t)
	public := httptest.NewRecorder()
	handler.ServeHTTP(public, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if public.Code != 200 {
		t.Fatalf("health=%d", public.Code)
	}
	blocked := httptest.NewRecorder()
	handler.ServeHTTP(blocked, httptest.NewRequest(http.MethodGet, "/api/v1/analyses", nil))
	if blocked.Code != 401 {
		t.Fatalf("unauthorized=%d body=%s", blocked.Code, blocked.Body.String())
	}
}
func TestCSVReportEscapesFormulaLookingPath(t *testing.T) {
	handler, store := testServerWithStore(t)
	analysis := domain.Analysis{ID: "a", Status: domain.StatusCompleted, RepositoryPath: "/safe", Ref: "HEAD", Languages: []string{"go"}, Weights: domain.DefaultWeights(), FormulaVersion: domain.FormulaVersion, HistoryComplete: true, CreatedAt: time.Now().UTC()}
	if err := store.CreateAnalysis(analysis); err != nil {
		t.Fatal(err)
	}
	files := []domain.FileResult{{ID: "f", AnalysisID: "a", Path: "=DANGEROUS.csv", Module: ".", Static: domain.StaticMetrics{Language: "go", AnalysisStatus: "ok"}}}
	if err := store.SaveFiles("a", files); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/analyses/a/reports?format=csv&level=file", nil)
	request.Header.Set("X-Analyzer-Token", "test-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 || !bytes.Contains(response.Body.Bytes(), []byte("'=DANGEROUS.csv")) {
		t.Fatalf("csv response=%d %s", response.Code, response.Body.String())
	}
}
func TestMalformedAnalysisRequestGetsControlledError(t *testing.T) {
	handler := testServer(t)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/analyses", bytes.NewBufferString(`{"repository_path":`))
	request.Header.Set("X-Analyzer-Token", "test-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 400 || !bytes.Contains(response.Body.Bytes(), []byte("INVALID_JSON")) {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
}
func testServer(t *testing.T) http.Handler { handler, _ := testServerWithStore(t); return handler }
func testServerWithStore(t *testing.T) (http.Handler, *storage.Store) {
	t.Helper()
	root := t.TempDir()
	store, err := storage.Open(filepath.Join(root, "data.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	manager, err := jobs.New(store, jobs.Config{AllowedRepoRoot: root, AnalysisTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(manager.Close)
	return New(store, manager, Config{Token: "test-token"}), store
}
