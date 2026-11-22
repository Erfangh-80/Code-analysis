package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/analysisguard"
	"github.com/Erfangh-80/code-analysis/backend/internal/analyzer/static"
	"github.com/Erfangh-80/code-analysis/backend/internal/application"
	"github.com/Erfangh-80/code-analysis/backend/internal/githistory"
	"github.com/Erfangh-80/code-analysis/backend/internal/storage/failedfiles"
)

func TestGuardedHandlerRejectsOutsidePathsWithoutLeakingSourceOrHostPath(t *testing.T) {
	allowedRoot := t.TempDir()
	outsideRoot := t.TempDir()
	secretPath := filepath.Join(outsideRoot, "secret.go")
	secret := "package secret\n// DO_NOT_EXPOSE_THIS_SOURCE\n"
	if err := os.WriteFile(secretPath, []byte(secret), 0o600); err != nil {
		t.Fatalf("write outside source: %v", err)
	}
	policy, err := analysisguard.NewPathPolicy([]string{allowedRoot})
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}
	limits := analysisguard.Limits{
		MaxFiles:            10,
		MaxFileBytes:        1024,
		MaxTotalSourceBytes: 4096,
		MaxGitCommits:       10,
		MaxGitOutputBytes:   4096,
		AnalysisTimeout:     time.Second,
	}
	staticService, err := application.NewStaticAnalysisServiceWithGuards(static.NewAnalyzer(), failedfiles.NewRegistry(), policy, limits)
	if err != nil {
		t.Fatalf("new guarded static service: %v", err)
	}
	gitService, err := githistory.NewAnalyzerWithGuards(policy, limits)
	if err != nil {
		t.Fatalf("new guarded Git service: %v", err)
	}
	handler := NewHandlerWithCoreServices("test-version", staticService, gitService)
	payload, err := json.Marshal(map[string]any{"rootPath": outsideRoot, "files": []string{"secret.go"}})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/static-analysis", bytes.NewReader(payload)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if strings.Contains(body, outsideRoot) || strings.Contains(body, secret) || strings.Contains(body, "DO_NOT_EXPOSE_THIS_SOURCE") {
		t.Fatalf("guarded error leaked sensitive input: %s", body)
	}
}
