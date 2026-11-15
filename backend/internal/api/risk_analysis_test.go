package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Erfangh-80/code-analysis/backend/internal/application"
	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

func TestRiskAnalysisEndpointScoresActualStaticAndGitResults(t *testing.T) {
	t.Parallel()

	repository := createAPIGitFixture(t)
	handler := NewHandler("test-version")

	staticPayload, err := json.Marshal(map[string]any{
		"rootPath": repository,
		"files":    []string{"sample.go"},
	})
	if err != nil {
		t.Fatalf("marshal static request: %v", err)
	}
	staticResponse := httptest.NewRecorder()
	handler.ServeHTTP(staticResponse, httptest.NewRequest(http.MethodPost, "/api/v1/static-analysis", bytes.NewReader(staticPayload)))
	if staticResponse.Code != http.StatusOK {
		t.Fatalf("static status = %d, want %d; body=%s", staticResponse.Code, http.StatusOK, staticResponse.Body.String())
	}
	var staticReport domain.BatchAnalysis
	if err := json.NewDecoder(staticResponse.Body).Decode(&staticReport); err != nil {
		t.Fatalf("decode static report: %v", err)
	}

	gitPayload, err := json.Marshal(map[string]string{"repositoryPath": repository})
	if err != nil {
		t.Fatalf("marshal Git request: %v", err)
	}
	gitResponse := httptest.NewRecorder()
	handler.ServeHTTP(gitResponse, httptest.NewRequest(http.MethodPost, "/api/v1/git-analysis", bytes.NewReader(gitPayload)))
	if gitResponse.Code != http.StatusOK {
		t.Fatalf("Git status = %d, want %d; body=%s", gitResponse.Code, http.StatusOK, gitResponse.Body.String())
	}
	var gitReport domain.GitAnalysisReport
	if err := json.NewDecoder(gitResponse.Body).Decode(&gitReport); err != nil {
		t.Fatalf("decode Git report: %v", err)
	}

	payload, err := json.Marshal(domain.RiskAnalysisRequest{Files: application.BuildHotspotInputs(staticReport.Files, gitReport.Files)})
	if err != nil {
		t.Fatalf("marshal risk request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/risk-analysis", bytes.NewReader(payload))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	var report domain.RiskAnalysisReport
	if err := json.NewDecoder(response.Body).Decode(&report); err != nil {
		t.Fatalf("decode risk report: %v", err)
	}
	if report.Profile.Version != "risk-v1" || report.Formula == "" || report.Normalization == "" || len(report.Limitations) == 0 {
		t.Errorf("model explanation = %+v, want default profile and documentation", report)
	}
	if len(report.Files) != 1 {
		t.Fatalf("hotspot count = %d, want 1; hotspots=%+v", len(report.Files), report.Files)
	}
	if report.Files[0].Path != "sample.go" || report.Files[0].Rank != 1 || report.Files[0].Score <= 0 {
		t.Errorf("hotspot = %+v, want a scored sample.go", report.Files[0])
	}
	if len(report.Files[0].Contributions) != 5 || report.Files[0].Contributions[0].NormalizationCap <= 0 {
		t.Errorf("contributions = %+v, want raw, normalized, weighted, and cap facts", report.Files[0].Contributions)
	}
	if len(report.Heatmap.Cells) != 1 {
		t.Fatalf("heatmap cell count = %d, want 1; heatmap=%+v", len(report.Heatmap.Cells), report.Heatmap)
	}
	if report.Heatmap.XMetric != domain.RiskMetricGitChurn || report.Heatmap.Cells[0].Module != "." {
		t.Errorf("heatmap = %+v", report.Heatmap)
	}
}

func TestRiskAnalysisEndpointRejectsMissingMetrics(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"files":[{"path":"file.go","metrics":{"linesOfCode":1,"maxFunctionLength":0,"gitChurn":0,"changeFrequency":0}}]
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/risk-analysis", bytes.NewReader(payload))
	response := httptest.NewRecorder()
	NewHandler("test-version").ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	assertErrorCode(t, response, "VALIDATION_ERROR")
}

func TestRiskAnalysisEndpointRejectsInvalidWeights(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"files":[{"path":"file.go","metrics":{"linesOfCode":1,"maxFunctionLength":0,"maxClassLength":0,"gitChurn":0,"changeFrequency":0}}],
		"profile":{"version":"invalid","weights":{"lines_of_code":0.5},"bands":{"lowMax":34,"mediumMax":64}}
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/risk-analysis", bytes.NewReader(payload))
	response := httptest.NewRecorder()
	NewHandler("test-version").ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	assertErrorCode(t, response, "VALIDATION_ERROR")
}
