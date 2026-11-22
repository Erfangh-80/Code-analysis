package api

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestCompletedAnalysisResultEndpointsFilterPaginateAndExport(t *testing.T) {
	project := createResultAPIFixture(t)
	handler, closeHandler := newResultAPIHandler(t)
	defer closeHandler()
	id := startCompletedResultAnalysis(t, handler, project)

	summaryResponse := getResultAPI(t, handler, "/api/v1/analyses/"+string(id)+"/summary")
	if summaryResponse.Code != http.StatusOK {
		t.Fatalf("summary status = %d; body=%s", summaryResponse.Code, summaryResponse.Body.String())
	}
	var summary domain.AnalysisResultSummary
	if err := json.NewDecoder(summaryResponse.Body).Decode(&summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if summary.Analysis.ID != id || summary.Summary.ScoredFiles != 2 || summary.ModuleCount != 2 || summary.RiskCoverage == nil {
		t.Errorf("summary = %+v", summary)
	}

	hotspotsFirst := getResultAPI(t, handler, "/api/v1/analyses/"+string(id)+"/hotspots?sort=path&direction=asc&pageSize=1")
	if hotspotsFirst.Code != http.StatusOK {
		t.Fatalf("hotspots status = %d; body=%s", hotspotsFirst.Code, hotspotsFirst.Body.String())
	}
	var firstHotspots domain.AnalysisHotspotPage
	if err := json.NewDecoder(hotspotsFirst.Body).Decode(&firstHotspots); err != nil {
		t.Fatalf("decode hotspot page: %v", err)
	}
	if firstHotspots.TotalItems != 2 || len(firstHotspots.Items) != 1 || firstHotspots.Items[0].Path != "cmd/beta.go" || firstHotspots.RiskCoverage == nil {
		t.Errorf("first hotspot page = %+v", firstHotspots)
	}
	hotspotsSecond := getResultAPI(t, handler, "/api/v1/analyses/"+string(id)+"/hotspots?sort=path&direction=asc&page=2&pageSize=1")
	var secondHotspots domain.AnalysisHotspotPage
	if err := json.NewDecoder(hotspotsSecond.Body).Decode(&secondHotspots); err != nil {
		t.Fatalf("decode second hotspot page: %v", err)
	}
	if hotspotsSecond.Code != http.StatusOK || len(secondHotspots.Items) != 1 || secondHotspots.Items[0].Path != "pkg/alpha.go" {
		t.Errorf("second hotspot response=%d page=%+v", hotspotsSecond.Code, secondHotspots)
	}

	filesResponse := getResultAPI(t, handler, "/api/v1/analyses/"+string(id)+"/files?module=cmd&language=go&minScore=0&maxScore=100&analysisStatus=analyzed&sort=path&direction=asc")
	if filesResponse.Code != http.StatusOK {
		t.Fatalf("files status = %d; body=%s", filesResponse.Code, filesResponse.Body.String())
	}
	var files domain.AnalysisFilePage
	if err := json.NewDecoder(filesResponse.Body).Decode(&files); err != nil {
		t.Fatalf("decode file page: %v", err)
	}
	if files.TotalItems != 1 || len(files.Items) != 1 || files.Items[0].Analysis.Path != "cmd/beta.go" || files.Items[0].Risk == nil || files.Items[0].Git == nil {
		t.Errorf("filtered files = %+v", files)
	}

	fileResponse := getResultAPI(t, handler, "/api/v1/analyses/"+string(id)+"/files/detail?path=cmd/beta.go")
	if fileResponse.Code != http.StatusOK {
		t.Fatalf("file detail status = %d; body=%s", fileResponse.Code, fileResponse.Body.String())
	}
	var file domain.AnalysisFileResult
	if err := json.NewDecoder(fileResponse.Body).Decode(&file); err != nil {
		t.Fatalf("decode file detail: %v", err)
	}
	if file.Analysis.Path != "cmd/beta.go" || file.Risk == nil || file.Git == nil {
		t.Errorf("file detail = %+v", file)
	}

	modulesResponse := getResultAPI(t, handler, "/api/v1/analyses/"+string(id)+"/modules")
	if modulesResponse.Code != http.StatusOK {
		t.Fatalf("modules status = %d; body=%s", modulesResponse.Code, modulesResponse.Body.String())
	}
	var modules domain.AnalysisModules
	if err := json.NewDecoder(modulesResponse.Body).Decode(&modules); err != nil {
		t.Fatalf("decode modules: %v", err)
	}
	if len(modules.Static) != 2 || len(modules.Risk) != 2 || modules.RiskCoverage == nil {
		t.Errorf("modules = %+v", modules)
	}

	heatmapResponse := getResultAPI(t, handler, "/api/v1/analyses/"+string(id)+"/heatmap")
	if heatmapResponse.Code != http.StatusOK {
		t.Fatalf("heatmap status = %d; body=%s", heatmapResponse.Code, heatmapResponse.Body.String())
	}
	var heatmap domain.HeatmapData
	if err := json.NewDecoder(heatmapResponse.Body).Decode(&heatmap); err != nil {
		t.Fatalf("decode heatmap: %v", err)
	}
	if len(heatmap.Cells) != 2 || heatmap.XMetric != domain.RiskMetricGitChurn {
		t.Errorf("heatmap = %+v", heatmap)
	}

	jsonExport := getResultAPI(t, handler, "/api/v1/analyses/"+string(id)+"/export?format=json")
	if jsonExport.Code != http.StatusOK || !strings.HasPrefix(jsonExport.Header().Get("Content-Type"), "application/json") || !strings.Contains(jsonExport.Header().Get("Content-Disposition"), ".json") {
		t.Fatalf("JSON export status=%d headers=%v body=%s", jsonExport.Code, jsonExport.Header(), jsonExport.Body.String())
	}
	var exportedReport domain.AnalysisReport
	if err := json.NewDecoder(jsonExport.Body).Decode(&exportedReport); err != nil {
		t.Fatalf("decode JSON export: %v", err)
	}
	if exportedReport.Analysis.ID != id || exportedReport.Artifacts.Risk == nil {
		t.Errorf("JSON export report = %+v", exportedReport)
	}

	csvExport := getResultAPI(t, handler, "/api/v1/analyses/"+string(id)+"/export?format=csv")
	if csvExport.Code != http.StatusOK || !strings.HasPrefix(csvExport.Header().Get("Content-Type"), "text/csv") || !strings.Contains(csvExport.Header().Get("Content-Disposition"), ".csv") {
		t.Fatalf("CSV export status=%d headers=%v body=%s", csvExport.Code, csvExport.Header(), csvExport.Body.String())
	}
	records, err := csv.NewReader(strings.NewReader(csvExport.Body.String())).ReadAll()
	if err != nil {
		t.Fatalf("parse CSV export: %v", err)
	}
	wantHeader := []string{"path", "language", "analysis_status", "risk_score", "risk_level", "lines_of_code", "function_count", "class_count", "max_function_length", "total_function_length", "max_class_length", "total_class_length", "git_churn", "change_frequency"}
	if len(records) != 3 || !reflect.DeepEqual(records[0], wantHeader) || records[1][0] == project || records[2][0] == project {
		t.Errorf("CSV records = %#v", records)
	}
}

func TestFullAnalysisFlowOverRealHTTPValidatesStartsPollsAndReadsResults(t *testing.T) {
	project := createResultAPIFixture(t)
	handler, closeHandler := newResultAPIHandler(t)
	defer closeHandler()
	server := httptest.NewServer(handler)
	defer server.Close()

	id := startCompletedResultAnalysisOverHTTP(t, server.Client(), server.URL, project)
	paths := []string{
		"/api/v1/analyses/" + string(id) + "/summary",
		"/api/v1/analyses/" + string(id) + "/hotspots?page=1&pageSize=1",
		"/api/v1/analyses/" + string(id) + "/files?page=1&pageSize=1",
		"/api/v1/analyses/" + string(id) + "/files/detail?path=cmd%2Fbeta.go",
		"/api/v1/analyses/" + string(id) + "/modules",
		"/api/v1/analyses/" + string(id) + "/heatmap",
		"/api/v1/analyses/" + string(id) + "/report",
		"/api/v1/analyses/" + string(id) + "/export?format=json",
		"/api/v1/analyses/" + string(id) + "/export?format=csv",
	}
	for _, path := range paths {
		response, err := server.Client().Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("read GET %s: read=%v close=%v", path, readErr, closeErr)
		}
		if response.StatusCode != http.StatusOK {
			t.Fatalf("GET %s status=%d body=%s", path, response.StatusCode, string(body))
		}
		if got := response.Header.Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("GET %s X-Content-Type-Options=%q, want nosniff", path, got)
		}
	}
}

func TestAnalysisResultEndpointsRejectInvalidQueriesAndUnknownFiles(t *testing.T) {
	project := createResultAPIFixture(t)
	handler, closeHandler := newResultAPIHandler(t)
	defer closeHandler()
	id := startCompletedResultAnalysis(t, handler, project)

	cases := []string{
		"/api/v1/analyses/" + string(id) + "/hotspots?page=0",
		"/api/v1/analyses/" + string(id) + "/hotspots?page=",
		"/api/v1/analyses/" + string(id) + "/hotspots?page=1&page=2",
		"/api/v1/analyses/" + string(id) + "/hotspots?sort=language",
		"/api/v1/analyses/" + string(id) + "/hotspots?direction=",
		"/api/v1/analyses/" + string(id) + "/files?unknown=value",
		"/api/v1/analyses/" + string(id) + "/files?language=ruby",
		"/api/v1/analyses/" + string(id) + "/files?minScore=101",
		"/api/v1/analyses/" + string(id) + "/files?minScore=80&maxScore=20",
		"/api/v1/analyses/" + string(id) + "/files?module=../../outside",
		"/api/v1/analyses/" + string(id) + "/files/detail?path=../outside.go",
		"/api/v1/analyses/" + string(id) + "/export?format=xml",
		"/api/v1/analyses/" + string(id) + "/export?format=json&format=csv",
		"/api/v1/analyses/" + string(id) + "/summary?unexpected=true",
		"/api/v1/analyses/" + string(id) + "/report?unexpected=true",
	}
	for _, path := range cases {
		response := getResultAPI(t, handler, path)
		if response.Code != http.StatusBadRequest {
			t.Errorf("path %q status=%d body=%s, want 400", path, response.Code, response.Body.String())
		}
		assertErrorCode(t, response, "VALIDATION_ERROR")
	}

	missing := getResultAPI(t, handler, "/api/v1/analyses/"+string(id)+"/files/detail?path=pkg/missing.go")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing file status=%d body=%s", missing.Code, missing.Body.String())
	}
	assertErrorCode(t, missing, "ANALYSIS_FILE_NOT_FOUND")
}

func newResultAPIHandler(t *testing.T) (http.Handler, func()) {
	t.Helper()
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "analyses.db"))
	if err != nil {
		t.Fatalf("open result API store: %v", err)
	}
	orchestrator := application.NewOrchestrator(
		store,
		repositoryscan.NewScanner(),
		application.NewStaticAnalysisService(static.NewAnalyzer(), failedfiles.NewRegistry()),
		githistory.NewAnalyzer(),
		scoring.NewEngine(),
		2,
	)
	return NewHandlerWithAnalysisService("test-version", orchestrator), func() { _ = store.Close() }
}

func startCompletedResultAnalysis(t *testing.T, handler http.Handler, project string) domain.AnalysisID {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"projectPath": project})
	if err != nil {
		t.Fatalf("marshal analysis start payload: %v", err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/analyses", bytes.NewReader(payload)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("start analysis status=%d body=%s", response.Code, response.Body.String())
	}
	var envelope struct {
		Analysis domain.AnalysisRun `json:"analysis"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode started analysis: %v", err)
	}
	completed := pollAPIAnalysis(t, handler, envelope.Analysis.ID)
	if completed.Status != domain.AnalysisStatusCompleted {
		t.Fatalf("completed status=%q diagnostics=%+v", completed.Status, completed.Diagnostics)
	}
	return envelope.Analysis.ID
}

func startCompletedResultAnalysisOverHTTP(t *testing.T, client *http.Client, baseURL, project string) domain.AnalysisID {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"projectPath": project})
	if err != nil {
		t.Fatalf("marshal HTTP analysis request payload: %v", err)
	}
	validationResponse, err := client.Post(baseURL+"/api/v1/analyses/validate-path", jsonContentType, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("POST path validation: %v", err)
	}
	var validation struct {
		Valid bool `json:"valid"`
	}
	decodeValidationErr := json.NewDecoder(validationResponse.Body).Decode(&validation)
	closeValidationErr := validationResponse.Body.Close()
	if decodeValidationErr != nil || closeValidationErr != nil {
		t.Fatalf("decode HTTP path validation: decode=%v close=%v", decodeValidationErr, closeValidationErr)
	}
	if validationResponse.StatusCode != http.StatusOK || !validation.Valid {
		t.Fatalf("POST path validation status=%d valid=%t", validationResponse.StatusCode, validation.Valid)
	}

	response, err := client.Post(baseURL+"/api/v1/analyses", jsonContentType, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("POST analysis: %v", err)
	}
	if response.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		t.Fatalf("POST analysis status=%d body=%s", response.StatusCode, body)
	}
	var envelope struct {
		Analysis domain.AnalysisRun `json:"analysis"`
	}
	decodeErr := json.NewDecoder(response.Body).Decode(&envelope)
	closeErr := response.Body.Close()
	if decodeErr != nil || closeErr != nil {
		t.Fatalf("decode HTTP started analysis: decode=%v close=%v", decodeErr, closeErr)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		statusResponse, err := client.Get(baseURL + "/api/v1/analyses/" + string(envelope.Analysis.ID))
		if err != nil {
			t.Fatalf("GET HTTP analysis status: %v", err)
		}
		var statusEnvelope struct {
			Analysis domain.AnalysisRun `json:"analysis"`
		}
		decodeErr := json.NewDecoder(statusResponse.Body).Decode(&statusEnvelope)
		closeErr := statusResponse.Body.Close()
		if decodeErr != nil || closeErr != nil {
			t.Fatalf("decode HTTP analysis status: decode=%v close=%v", decodeErr, closeErr)
		}
		if statusResponse.StatusCode != http.StatusOK {
			t.Fatalf("GET HTTP analysis status=%d", statusResponse.StatusCode)
		}
		if statusEnvelope.Analysis.Status == domain.AnalysisStatusCompleted {
			return envelope.Analysis.ID
		}
		if statusEnvelope.Analysis.Status == domain.AnalysisStatusFailed || statusEnvelope.Analysis.Status == domain.AnalysisStatusCancelled {
			t.Fatalf("HTTP analysis status=%q diagnostics=%+v", statusEnvelope.Analysis.Status, statusEnvelope.Analysis.Diagnostics)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("HTTP analysis %q did not finish", envelope.Analysis.ID)
	return ""
}

func getResultAPI(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	return response
}

func createResultAPIFixture(t *testing.T) string {
	t.Helper()
	repository := t.TempDir()
	runAPIGit(t, repository, "init")
	runAPIGit(t, repository, "config", "user.name", "Result API Fixture")
	runAPIGit(t, repository, "config", "user.email", "result-api@example.test")
	writeResultAPIFile(t, filepath.Join(repository, "cmd", "beta.go"), "package cmd\n\nfunc Beta() int {\n\treturn 2\n}\n")
	writeResultAPIFile(t, filepath.Join(repository, "pkg", "alpha.go"), "package pkg\n\nfunc Alpha() int {\n\treturn 1\n}\n")
	runAPIGit(t, repository, "add", ".")
	runAPIGit(t, repository, "commit", "-m", "result fixture")
	return repository
}

func writeResultAPIFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture source: %v", err)
	}
}

func TestResultEndpointsReportUnavailableOptionalArtifacts(t *testing.T) {
	project := t.TempDir()
	handler, closeHandler := newResultAPIHandler(t)
	defer closeHandler()
	id := startCompletedResultAnalysis(t, handler, project)

	for _, endpoint := range []string{"hotspots", "heatmap"} {
		response := getResultAPI(t, handler, "/api/v1/analyses/"+string(id)+"/"+endpoint)
		if response.Code != http.StatusConflict {
			t.Errorf("%s without risk artifact status=%d body=%s, want 409", endpoint, response.Code, response.Body.String())
		}
		assertErrorCode(t, response, "RESULT_UNAVAILABLE")
	}
}

func TestResultEndpointsRejectNonCompletedAnalysis(t *testing.T) {
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "analyses.db"))
	if err != nil {
		t.Fatalf("open queued-result store: %v", err)
	}
	defer store.Close()
	queued := domain.AnalysisRun{
		ID:          "analysis_queued_result",
		Status:      domain.AnalysisStatusQueued,
		CreatedAt:   time.Now().UTC(),
		Stages:      []domain.StageSnapshot{},
		Diagnostics: []domain.AnalysisDiagnostic{},
		Scope:       domain.ProjectScope{ProjectPath: "/local/project", ScopeKey: "queued-result"},
	}
	if err := store.CreateQueued(context.Background(), queued); err != nil {
		t.Fatalf("create queued analysis: %v", err)
	}
	handler := NewHandlerWithAnalysisService("test-version", application.NewOrchestrator(store, repositoryscan.NewScanner(), application.NewStaticAnalysisService(static.NewAnalyzer(), failedfiles.NewRegistry()), githistory.NewAnalyzer(), scoring.NewEngine(), 1))
	response := getResultAPI(t, handler, "/api/v1/analyses/analysis_queued_result/summary")
	if response.Code != http.StatusConflict {
		t.Fatalf("queued summary status=%d body=%s", response.Code, response.Body.String())
	}
	assertErrorCode(t, response, "INVALID_RUN_STATE")
}
