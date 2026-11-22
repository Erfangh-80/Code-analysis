package repositoryscan_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Erfangh-80/code-analysis/backend/internal/repositoryscan"
)

func TestScannerDoesNotFollowGitignoreSymlinkOutsideProject(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeScannerFile(t, filepath.Join(root, "keep.go"), "package keep\n")
	ignorePath := filepath.Join(outside, "outside.gitignore")
	writeScannerFile(t, ignorePath, "keep.go\n")
	if err := os.Symlink(ignorePath, filepath.Join(root, ".gitignore")); err != nil {
		t.Skipf("symbolic links are unavailable on this platform: %v", err)
	}

	scanner := repositoryscan.NewScanner()
	scope, err := scanner.Resolve(context.Background(), root)
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	scan, err := scanner.Scan(context.Background(), scope)
	if err != nil {
		t.Fatalf("scan root: %v", err)
	}
	if want := []string{"keep.go"}; !reflect.DeepEqual(scan.Files, want) {
		t.Errorf("files = %v, want %v; escaping .gitignore must not influence scope", scan.Files, want)
	}
}
