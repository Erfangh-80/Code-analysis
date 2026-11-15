package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/analyzer/static"
	"github.com/Erfangh-80/code-analysis/backend/internal/application"
	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
	"github.com/Erfangh-80/code-analysis/backend/internal/githistory"
	"github.com/Erfangh-80/code-analysis/backend/internal/infrastructure/sqlite"
	"github.com/Erfangh-80/code-analysis/backend/internal/repositoryscan"
	"github.com/Erfangh-80/code-analysis/backend/internal/scoring"
	"github.com/Erfangh-80/code-analysis/backend/internal/storage/failedfiles"
)

func TestAnalysisEndpointsStartPollAndReturnPersistedReport(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "sample.go"), []byte("package sample\nfunc Value() int { return 1 }\n"), 0o600); err != nil {
		t.Fatalf("write source fixture: %v", err)
	}
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "analyses.db"))
	if err != nil {
		t.Fatalf("open SQLite store: %v", err)
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
		t.Fatalf("marshal start request: %v", err)
	}
	startResponse := httptest.NewRecorder()
	handler.ServeHTTP(startResponse, httptest.NewRequest(http.MethodPost, "/api/v1/analyses", bytes.NewReader(payload)))
	if startResponse.Code != http.StatusAccepted {
		t.Fatalf("start status = %d, want %d; body=%s", startResponse.Code, http.StatusAccepted, startResponse.Body.String())
	}
	if bytes.Contains(startResponse.Body.Bytes(), []byte(project)) {
		t.Fatalf("start response leaked the absolute project path: %s", startResponse.Body.String())
	}
	var started struct {
		Analysis domain.AnalysisRun `json:"analysis"`
	}
	if err := json.NewDecoder(startResponse.Body).Decode(&started); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	if started.Analysis.ID == "" || started.Analysis.Status != domain.AnalysisStatusQueued {
		t.Errorf("started analysis = %+v", started.Analysis)
	}

	status := pollAPIAnalysis(t, handler, started.Analysis.ID)
	if status.Status != domain.AnalysisStatusCompleted {
		t.Fatalf("analysis status = %q, want completed; diagnostics=%+v", status.Status, status.Diagnostics)
	}
	reportResponse := httptest.NewRecorder()
	handler.ServeHTTP(reportResponse, httptest.NewRequest(http.MethodGet, "/api/v1/analyses/"+string(started.Analysis.ID)+"/report", nil))
	if reportResponse.Code != http.StatusOK {
		t.Fatalf("report status = %d, want %d; body=%s", reportResponse.Code, http.StatusOK, reportResponse.Body.String())
	}
	var report domain.AnalysisReport
	if err := json.NewDecoder(reportResponse.Body).Decode(&report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.Artifacts.Static == nil || report.Artifacts.Risk == nil {
		t.Errorf("report artifacts = %+v", report.Artifacts)
	}
}

func TestAnalysisEndpointsRejectDuplicateActiveProject(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "sample.go"), []byte("package sample\nfunc Value() int { return 1 }\n"), 0o600); err != nil {
		t.Fatalf("write source fixture: %v", err)
	}
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "analyses.db"))
	if err != nil {
		t.Fatalf("open SQLite store: %v", err)
	}
	defer store.Close()
	scanner := repositoryscan.NewScanner()
	scope, err := scanner.Resolve(context.Background(), project)
	if err != nil {
		t.Fatalf("resolve fixture project: %v", err)
	}
	if err := store.CreateQueued(context.Background(), domain.AnalysisRun{
		ID:          "analysis_existing",
		Status:      domain.AnalysisStatusQueued,
		CreatedAt:   time.Now().UTC(),
		Stages:      []domain.StageSnapshot{},
		Diagnostics: []domain.AnalysisDiagnostic{},
		Scope:       scope,
	}); err != nil {
		t.Fatalf("create existing run: %v", err)
	}
	handler := NewHandlerWithAnalysisService("test-version", application.NewOrchestrator(store, scanner, application.NewStaticAnalysisService(static.NewAnalyzer(), failedfiles.NewRegistry()), githistory.NewAnalyzer(), scoring.NewEngine(), 1))
	payload, err := json.Marshal(map[string]string{"projectPath": project})
	if err != nil {
		t.Fatalf("marshal start request: %v", err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/analyses", bytes.NewReader(payload)))
	if response.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want %d; body=%s", response.Code, http.StatusConflict, response.Body.String())
	}
	assertErrorCode(t, response, "ANALYSIS_ALREADY_RUNNING")
}

func pollAPIAnalysis(t *testing.T, handler http.Handler, id domain.AnalysisID) domain.AnalysisRun {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/analyses/"+string(id), nil))
		if response.Code != http.StatusOK {
			t.Fatalf("status endpoint = %d, body=%s", response.Code, response.Body.String())
		}
		var payload struct {
			Analysis domain.AnalysisRun `json:"analysis"`
		}
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatalf("decode status response: %v", err)
		}
		if payload.Analysis.Status == domain.AnalysisStatusCompleted || payload.Analysis.Status == domain.AnalysisStatusFailed || payload.Analysis.Status == domain.AnalysisStatusCancelled {
			return payload.Analysis
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("analysis %q did not finish", id)
	return domain.AnalysisRun{}
}
