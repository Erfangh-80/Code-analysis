package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

func TestStaticAnalysisEndpointUsesLocalFilesAndExposesRecordedFailures(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeAPIFixture(t, filepath.Join(root, "pkg", "sample.py"), "def value():\n    return 1\n")
	writeAPIFixture(t, filepath.Join(root, "broken.js"), "function broken( {\n")

	payload, err := json.Marshal(map[string]any{
		"rootPath": root,
		"files":    []string{"pkg/sample.py", "broken.js"},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	handler := NewHandler("test-version")
	request := httptest.NewRequest(http.MethodPost, "/api/v1/static-analysis", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	var result domain.BatchAnalysis
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(result.Files) != 2 || len(result.Modules) != 2 {
		t.Errorf("result = %+v, want 2 file results and 2 module results", result)
	}

	failuresRequest := httptest.NewRequest(http.MethodGet, "/api/v1/static-analysis/failures", nil)
	failuresResponse := httptest.NewRecorder()
	handler.ServeHTTP(failuresResponse, failuresRequest)
	if failuresResponse.Code != http.StatusOK {
		t.Fatalf("failure status = %d, want %d", failuresResponse.Code, http.StatusOK)
	}
	var failures struct {
		Items []domain.FailureRecord `json:"items"`
	}
	if err := json.NewDecoder(failuresResponse.Body).Decode(&failures); err != nil {
		t.Fatalf("decode failures: %v", err)
	}
	if len(failures.Items) != 1 || failures.Items[0].Path != "broken.js" {
		t.Errorf("failure items = %+v, want recorded broken.js", failures.Items)
	}
}

func TestStaticAnalysisEndpointRejectsUnsafeInput(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"rootPath":"relative","files":["../outside.go"]}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/static-analysis", bytes.NewReader(payload))
	response := httptest.NewRecorder()

	NewHandler("test-version").ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	assertErrorCode(t, response, "VALIDATION_ERROR")
}

func TestOpenAPISpecificationIsServedLocally(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil)
	response := httptest.NewRecorder()
	NewHandler("test-version").ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/yaml; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/yaml; charset=utf-8", contentType)
	}
	if !bytes.Contains(response.Body.Bytes(), []byte("openapi: 3.1.0")) {
		t.Error("OpenAPI specification does not contain an OpenAPI version")
	}
	if !bytes.Contains(response.Body.Bytes(), []byte("/git-analysis:")) {
		t.Error("OpenAPI specification does not document the Git-analysis endpoint")
	}
	if !bytes.Contains(response.Body.Bytes(), []byte("/risk-analysis:")) {
		t.Error("OpenAPI specification does not document the Risk Engine endpoint")
	}
}

func writeAPIFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}
