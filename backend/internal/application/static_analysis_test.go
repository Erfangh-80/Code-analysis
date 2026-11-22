package application_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Erfangh-80/code-analysis/backend/internal/analysisguard"
	"github.com/Erfangh-80/code-analysis/backend/internal/analyzer/static"
	"github.com/Erfangh-80/code-analysis/backend/internal/application"
	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
	"github.com/Erfangh-80/code-analysis/backend/internal/storage/failedfiles"
)

func TestStaticAnalysisServiceAggregatesModulesAndRecordsParserFailures(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "internal", "billing", "ledger.go"), "package billing\n\nfunc Balance() int { return 1 }\n")
	writeFile(t, filepath.Join(root, "broken.js"), "function broken( {\n")

	registry := failedfiles.NewRegistry()
	service := application.NewStaticAnalysisService(static.NewAnalyzer(), registry)
	result, err := service.Analyze(context.Background(), application.StaticAnalysisRequest{
		RootPath: root,
		Files:    []string{"internal/billing/ledger.go", "broken.js"},
	})
	if err != nil {
		t.Fatalf("analyze files: %v", err)
	}

	if len(result.Files) != 2 {
		t.Fatalf("file results = %d, want 2", len(result.Files))
	}
	if len(result.Modules) != 2 {
		t.Fatalf("module results = %d, want 2", len(result.Modules))
	}
	if result.Modules[0].Path != "." || result.Modules[1].Path != "internal/billing" {
		t.Errorf("module paths = %+v, want [. internal/billing]", result.Modules)
	}
	if result.Modules[0].Metrics.LinesOfCode != 1 || result.Modules[0].FailedFileCount != 1 {
		t.Errorf("root module = %+v, want one LOC and one recorded parser failure", result.Modules[0])
	}
	if result.Modules[1].Metrics.LinesOfCode != 2 || result.Modules[1].Metrics.FunctionCount != 1 {
		t.Errorf("billing module = %+v, want two LOC and one function", result.Modules[1])
	}

	failures := registry.List()
	if len(failures) != 1 {
		t.Fatalf("recorded failures = %d, want 1", len(failures))
	}
	if failures[0].Path != "broken.js" {
		t.Errorf("failure path = %q, want broken.js", failures[0].Path)
	}
	if failures[0].Status != domain.AnalysisStatusFallback || !failures[0].UsedFallback {
		t.Errorf("failure = %+v, want fallback parser failure", failures[0])
	}
}

func TestStaticAnalysisServiceRejectsPathsOutsideRoot(t *testing.T) {
	t.Parallel()

	service := application.NewStaticAnalysisService(static.NewAnalyzer(), failedfiles.NewRegistry())
	_, err := service.Analyze(context.Background(), application.StaticAnalysisRequest{
		RootPath: t.TempDir(),
		Files:    []string{"../outside.go"},
	})
	if err == nil {
		t.Fatal("Analyze() error = nil, want path validation error")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}
