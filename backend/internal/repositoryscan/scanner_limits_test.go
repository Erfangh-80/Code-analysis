package repositoryscan_test

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/analysisguard"
	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
	"github.com/Erfangh-80/code-analysis/backend/internal/repositoryscan"
)

func TestGuardedScannerRejectsProjectOutsideAllowedRoots(t *testing.T) {
	allowedRoot := t.TempDir()
	outsideRoot := t.TempDir()
	policy, err := analysisguard.NewPathPolicy([]string{allowedRoot})
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}
	scanner, err := repositoryscan.NewScannerWithGuards(policy, analysisguard.DefaultLimits())
	if err != nil {
		t.Fatalf("new scanner: %v", err)
	}

	_, err = scanner.Resolve(context.Background(), outsideRoot)
	if !errors.Is(err, repositoryscan.ErrInvalidProjectPath) {
		t.Fatalf("resolve error = %v, want ErrInvalidProjectPath", err)
	}
}

func TestGuardedScannerSkipsFilesBeyondConfiguredResourceBudgets(t *testing.T) {
	root := t.TempDir()
	writeScannerFile(t, filepath.Join(root, "a.go"), "package a\n")
	writeScannerFile(t, filepath.Join(root, "b.go"), "package b\n")
	writeScannerFile(t, filepath.Join(root, "too-large.go"), "package too_large\n")
	policy, err := analysisguard.NewPathPolicy([]string{root})
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}
	limits := analysisguard.Limits{
		MaxFiles:            3,
		MaxFileBytes:        16,
		MaxTotalSourceBytes: 24,
		MaxGitCommits:       5,
		MaxGitOutputBytes:   1024,
		AnalysisTimeout:     time.Second,
	}
	scanner, err := repositoryscan.NewScannerWithGuards(policy, limits)
	if err != nil {
		t.Fatalf("new scanner: %v", err)
	}
	scope, err := scanner.Resolve(context.Background(), root)
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	scan, err := scanner.Scan(context.Background(), scope)
	if err != nil {
		t.Fatalf("scan root: %v", err)
	}
	if want := []string{"a.go", "b.go"}; !reflect.DeepEqual(scan.Files, want) {
		t.Errorf("files = %v, want %v", scan.Files, want)
	}
	if !hasScannerDiagnostic(scan.Diagnostics, "RESOURCE_LIMIT_FILE_SIZE") {
		t.Errorf("diagnostics = %+v, want file-size resource diagnostic", scan.Diagnostics)
	}
}

func hasScannerDiagnostic(diagnostics []domain.AnalysisDiagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
