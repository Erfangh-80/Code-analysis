package repositoryscan_test

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/analysisguard"
	"github.com/Erfangh-80/code-analysis/backend/internal/repositoryscan"
)

func TestGuardedScannerStopsFilesystemDiscoveryAtFileCountLimit(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.go", "b.go", "c.go"} {
		writeScannerFile(t, filepath.Join(root, name), "package p\n")
	}
	policy, err := analysisguard.NewPathPolicy([]string{root})
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}
	scanner, err := repositoryscan.NewScannerWithGuards(policy, analysisguard.Limits{
		MaxFiles:            1,
		MaxFileBytes:        1024,
		MaxTotalSourceBytes: 4096,
		MaxGitCommits:       10,
		MaxGitOutputBytes:   4096,
		AnalysisTimeout:     time.Second,
	})
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
	if want := []string{"a.go"}; !reflect.DeepEqual(scan.Files, want) {
		t.Errorf("files = %v, want %v", scan.Files, want)
	}
	if !hasScannerDiagnostic(scan.Diagnostics, "RESOURCE_LIMIT_FILE_COUNT") {
		t.Errorf("diagnostics = %+v, want file-count resource diagnostic", scan.Diagnostics)
	}
}
