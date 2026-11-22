package application_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/analysisguard"
	"github.com/Erfangh-80/code-analysis/backend/internal/analyzer/static"
	"github.com/Erfangh-80/code-analysis/backend/internal/application"
	"github.com/Erfangh-80/code-analysis/backend/internal/storage/failedfiles"
)

func TestStaticAnalysisServiceDiscoversSupportedFilesWhenNoneSpecified(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "cmd", "main.go"), "package main\n\nfunc main() {}\n")
	writeFile(t, filepath.Join(root, "web", "app.js"), "function boot() { return 1; }\n")
	writeFile(t, filepath.Join(root, "scripts", "tool.py"), "def run():\n    return 1\n")
	writeFile(t, filepath.Join(root, "README.md"), "# docs\n")
	writeFile(t, filepath.Join(root, "node_modules", "pkg", "index.js"), "module.exports = 1;\n")
	writeFile(t, filepath.Join(root, ".git", "hooks", "sample.py"), "def hook():\n    pass\n")
	writeFile(t, filepath.Join(root, "dist", "bundle.js"), "function bundled() {}\n")

	service := application.NewStaticAnalysisService(static.NewAnalyzer(), failedfiles.NewRegistry())
	result, err := service.Analyze(context.Background(), application.StaticAnalysisRequest{RootPath: root})
	if err != nil {
		t.Fatalf("analyze discovered files: %v", err)
	}

	if len(result.Files) != 3 {
		t.Fatalf("discovered files = %d, want 3: %+v", len(result.Files), result.Files)
	}
	wantPaths := []string{"cmd/main.go", "scripts/tool.py", "web/app.js"}
	for index, want := range wantPaths {
		if result.Files[index].Path != want {
			t.Errorf("files[%d].Path = %q, want %q", index, result.Files[index].Path, want)
		}
	}
	for _, file := range result.Files {
		if file.Status == "read_error" {
			t.Errorf("file %q unexpectedly unreadable: %+v", file.Path, file.Diagnostics)
		}
	}
}

func TestStaticAnalysisServiceDiscoveryHonorsConfiguredFileLimit(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, name := range []string{"a.go", "b.go", "c.go", "d.go"} {
		writeFile(t, filepath.Join(root, name), "package fixture\n")
	}

	policy, err := analysisguard.NewPathPolicy([]string{root})
	if err != nil {
		t.Fatalf("create path policy: %v", err)
	}
	limits, err := (analysisguard.Limits{
		MaxFiles:            3,
		MaxFileBytes:        1 << 20,
		MaxTotalSourceBytes: 8 << 20,
		MaxGitCommits:       10,
		MaxGitOutputBytes:   1 << 20,
		AnalysisTimeout:     time.Minute,
	}).Normalize()
	if err != nil {
		t.Fatalf("normalize limits: %v", err)
	}
	service, err := application.NewStaticAnalysisServiceWithGuards(static.NewAnalyzer(), failedfiles.NewRegistry(), policy, limits)
	if err != nil {
		t.Fatalf("create guarded service: %v", err)
	}

	_, err = service.Analyze(context.Background(), application.StaticAnalysisRequest{RootPath: root})
	if !errors.Is(err, analysisguard.ErrResourceLimit) {
		t.Fatalf("Analyze() error = %v, want resource limit error", err)
	}
}

func TestStaticAnalysisServiceDiscoverySkipsSymlinksEscapingRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "secret.go"), "package secret\n")
	writeFile(t, filepath.Join(root, "ok.go"), "package fixture\n")
	if err := os.Symlink(filepath.Join(outside, "secret.go"), filepath.Join(root, "linked.go")); err != nil {
		t.Skipf("symlinks unavailable on this host: %v", err)
	}

	service := application.NewStaticAnalysisService(static.NewAnalyzer(), failedfiles.NewRegistry())
	result, err := service.Analyze(context.Background(), application.StaticAnalysisRequest{RootPath: root})
	if err != nil {
		t.Fatalf("analyze discovered files: %v", err)
	}
	if len(result.Files) != 1 || result.Files[0].Path != "ok.go" {
		t.Fatalf("discovered files = %+v, want only ok.go", result.Files)
	}
}

func TestStaticAnalysisServiceDiscoveryReturnsEmptyBatchForSourcelessRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), "# docs\n")

	service := application.NewStaticAnalysisService(static.NewAnalyzer(), failedfiles.NewRegistry())
	result, err := service.Analyze(context.Background(), application.StaticAnalysisRequest{RootPath: root})
	if err != nil {
		t.Fatalf("analyze sourceless root: %v", err)
	}
	if len(result.Files) != 0 || len(result.Modules) != 0 {
		t.Errorf("result = %+v, want an empty batch", result)
	}
}
