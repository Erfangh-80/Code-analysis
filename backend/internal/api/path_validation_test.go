package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/Erfangh-80/code-analysis/backend/internal/analysisguard"
	"github.com/Erfangh-80/code-analysis/backend/internal/analyzer/static"
	"github.com/Erfangh-80/code-analysis/backend/internal/application"
	"github.com/Erfangh-80/code-analysis/backend/internal/githistory"
	"github.com/Erfangh-80/code-analysis/backend/internal/infrastructure/sqlite"
	"github.com/Erfangh-80/code-analysis/backend/internal/repositoryscan"
	"github.com/Erfangh-80/code-analysis/backend/internal/scoring"
	"github.com/Erfangh-80/code-analysis/backend/internal/storage/failedfiles"
)

func TestAnalysisPathValidationEndpointAcceptsAccessibleLocalProjectWithoutStartingRun(t *testing.T) {
	project := t.TempDir()
	store, err := sqlite.OpenMemory()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()
	handler := NewHandlerWithAnalysisService("test-version", application.NewOrchestrator(
		store,
		repositoryscan.NewScanner(),
		application.NewStaticAnalysisService(static.NewAnalyzer(), failedfiles.NewRegistry()),
		githistory.NewAnalyzer(),
		scoring.NewEngine(),
		1,
	))
	payload, err := json.Marshal(map[string]string{"projectPath": project})
	if err != nil {
		t.Fatalf("marshal validation request: %v", err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/analyses/validate-path", bytes.NewReader(payload)))
	if response.Code != http.StatusOK {
		t.Fatalf("validation status=%d body=%s", response.Code, response.Body.String())
	}
	if bytes.Contains(response.Body.Bytes(), []byte(project)) {
		t.Fatalf("validation response leaked absolute project path: %s", response.Body.String())
	}
	var result struct {
		Valid bool `json:"valid"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode validation response: %v", err)
	}
	if !result.Valid {
		t.Error("validation result valid=false, want true")
	}

	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/analyses", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}
	var page struct {
		TotalItems int `json:"totalItems"`
	}
	if err := json.NewDecoder(list.Body).Decode(&page); err != nil {
		t.Fatalf("decode analysis page: %v", err)
	}
	if page.TotalItems != 0 {
		t.Errorf("path validation started a run; totalItems=%d", page.TotalItems)
	}
}

func TestAnalysisPathValidationEndpointRejectsUnauthorizedPathWithoutPathDisclosure(t *testing.T) {
	allowedRoot := t.TempDir()
	unauthorizedProject := t.TempDir()
	policy, err := analysisguard.NewPathPolicy([]string{allowedRoot})
	if err != nil {
		t.Fatalf("create allowed-root policy: %v", err)
	}
	scanner, err := repositoryscan.NewScannerWithGuards(policy, analysisguard.DefaultLimits())
	if err != nil {
		t.Fatalf("create guarded scanner: %v", err)
	}
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "analyses.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()
	handler := NewHandlerWithAnalysisService("test-version", application.NewOrchestrator(
		store,
		scanner,
		application.NewStaticAnalysisService(static.NewAnalyzer(), failedfiles.NewRegistry()),
		githistory.NewAnalyzer(),
		scoring.NewEngine(),
		1,
	))
	payload, err := json.Marshal(map[string]string{"projectPath": unauthorizedProject})
	if err != nil {
		t.Fatalf("marshal validation request: %v", err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/analyses/validate-path", bytes.NewReader(payload)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("validation status=%d body=%s", response.Code, response.Body.String())
	}
	if bytes.Contains(response.Body.Bytes(), []byte(unauthorizedProject)) {
		t.Fatalf("validation error leaked absolute project path: %s", response.Body.String())
	}
	assertErrorCode(t, response, "VALIDATION_ERROR")
}

func TestAnalysisPathValidationEndpointRejectsInvalidPath(t *testing.T) {
	store, err := sqlite.OpenMemory()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()
	handler := NewHandlerWithAnalysisService("test-version", application.NewOrchestrator(
		store,
		repositoryscan.NewScanner(),
		application.NewStaticAnalysisService(static.NewAnalyzer(), failedfiles.NewRegistry()),
		githistory.NewAnalyzer(),
		scoring.NewEngine(),
		1,
	))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/analyses/validate-path", bytes.NewBufferString(`{"projectPath":"../outside"}`)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("validation status=%d body=%s", response.Code, response.Body.String())
	}
	assertErrorCode(t, response, "VALIDATION_ERROR")
}
