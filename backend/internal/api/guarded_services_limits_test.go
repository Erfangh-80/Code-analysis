package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/analysisguard"
	"github.com/Erfangh-80/code-analysis/backend/internal/analyzer/static"
	"github.com/Erfangh-80/code-analysis/backend/internal/application"
	"github.com/Erfangh-80/code-analysis/backend/internal/githistory"
	"github.com/Erfangh-80/code-analysis/backend/internal/storage/failedfiles"
)

func TestGuardedHandlerReportsStaticResourceLimitAsRequestTooLarge(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"one.go", "two.go"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("package p\n"), 0o600); err != nil {
			t.Fatalf("write source fixture: %v", err)
		}
	}
	policy, err := analysisguard.NewPathPolicy([]string{root})
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}
	limits := analysisguard.Limits{
		MaxFiles:            1,
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
	handler := NewHandlerWithServices("test-version", nil, staticService, gitService)
	payload, err := json.Marshal(map[string]any{"rootPath": root, "files": []string{"one.go", "two.go"}})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/static-analysis", bytes.NewReader(payload)))
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	assertErrorCode(t, response, "RESOURCE_LIMIT_EXCEEDED")
}
