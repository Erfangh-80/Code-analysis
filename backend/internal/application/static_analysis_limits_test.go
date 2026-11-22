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
	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
	"github.com/Erfangh-80/code-analysis/backend/internal/storage/failedfiles"
)

func TestGuardedStaticAnalysisRejectsRootOutsideAllowedRoots(t *testing.T) {
	allowedRoot := t.TempDir()
	outsideRoot := t.TempDir()
	writeFile(t, filepath.Join(outsideRoot, "outside.go"), "package outside\n")
	policy, err := analysisguard.NewPathPolicy([]string{allowedRoot})
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}
	service, err := application.NewStaticAnalysisServiceWithGuards(static.NewAnalyzer(), failedfiles.NewRegistry(), policy, analysisguard.DefaultLimits())
	if err != nil {
		t.Fatalf("new guarded service: %v", err)
	}

	_, err = service.Analyze(context.Background(), application.StaticAnalysisRequest{RootPath: outsideRoot, Files: []string{"outside.go"}})
	if !errors.Is(err, application.ErrInvalidAnalysisRequest) {
		t.Fatalf("Analyze error = %v, want ErrInvalidAnalysisRequest", err)
	}
}

func TestGuardedStaticAnalysisBoundsSourceBytesWithoutLeakingContent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ok.go"), "package p\nfunc X() {}\n")
	secret := "package p\n// SECRET_TOKEN_DO_NOT_EXPOSE_AND_NEVER_RETURN_THIS_SOURCE\n"
	writeFile(t, filepath.Join(root, "too-large.go"), secret)
	policy, err := analysisguard.NewPathPolicy([]string{root})
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}
	limits := analysisguard.Limits{
		MaxFiles:            5,
		MaxFileBytes:        32,
		MaxTotalSourceBytes: 64,
		MaxGitCommits:       5,
		MaxGitOutputBytes:   1024,
		AnalysisTimeout:     time.Second,
	}
	service, err := application.NewStaticAnalysisServiceWithGuards(static.NewAnalyzer(), failedfiles.NewRegistry(), policy, limits)
	if err != nil {
		t.Fatalf("new guarded service: %v", err)
	}

	result, err := service.Analyze(context.Background(), application.StaticAnalysisRequest{RootPath: root, Files: []string{"ok.go", "too-large.go"}})
	if err != nil {
		t.Fatalf("Analyze error = %v", err)
	}
	if len(result.Files) != 2 || result.Files[0].Status != domain.AnalysisStatusAnalyzed || result.Files[1].Status != domain.AnalysisStatusReadError {
		t.Fatalf("file results = %+v", result.Files)
	}
	diagnostics := result.Files[1].Diagnostics
	if len(diagnostics) != 1 || diagnostics[0].Code != domain.DiagnosticCodeResourceLimit {
		t.Errorf("resource diagnostics = %+v", diagnostics)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Message == secret {
			t.Fatal("source content was exposed by resource-limit diagnostic")
	}
	}
}

func TestGuardedStaticAnalysisRejectsEscapingSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "secret.go"), "package secret\n")
	if err := os.Symlink(filepath.Join(outside, "secret.go"), filepath.Join(root, "escape.go")); err != nil {
		t.Skipf("symbolic links are unavailable on this platform: %v", err)
	}
	policy, err := analysisguard.NewPathPolicy([]string{root})
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}
	service, err := application.NewStaticAnalysisServiceWithGuards(static.NewAnalyzer(), failedfiles.NewRegistry(), policy, analysisguard.DefaultLimits())
	if err != nil {
		t.Fatalf("new guarded service: %v", err)
	}

	_, err = service.Analyze(context.Background(), application.StaticAnalysisRequest{RootPath: root, Files: []string{"escape.go"}})
	if !errors.Is(err, application.ErrInvalidAnalysisRequest) {
		t.Fatalf("escaped symlink error = %v, want ErrInvalidAnalysisRequest", err)
	}
}
