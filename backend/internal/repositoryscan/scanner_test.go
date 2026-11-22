package repositoryscan_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Erfangh-80/code-analysis/backend/internal/repositoryscan"
)

func TestScannerFindsOnlySupportedNonIgnoredFilesWithoutGit(t *testing.T) {
	root := t.TempDir()
	writeScannerFile(t, filepath.Join(root, ".gitignore"), "ignored.py\ncache/\n/root-only.py\n")
	writeScannerFile(t, filepath.Join(root, "keep.go"), "package sample\n")
	writeScannerFile(t, filepath.Join(root, "ignored.py"), "print('ignored')\n")
	writeScannerFile(t, filepath.Join(root, "root-only.py"), "print('ignored at root')\n")
	writeScannerFile(t, filepath.Join(root, "nested", "root-only.py"), "print('not root ignored')\n")
	writeScannerFile(t, filepath.Join(root, "cache", "skip.js"), "console.log('ignored')\n")
	writeScannerFile(t, filepath.Join(root, "notes.txt"), "not source\n")
	if err := os.WriteFile(filepath.Join(root, "binary.go"), []byte{'p', 'k', 0}, 0o600); err != nil {
		t.Fatalf("write binary fixture: %v", err)
	}

	scanner := repositoryscan.NewScanner()
	scope, err := scanner.Resolve(context.Background(), root)
	if err != nil {
		t.Fatalf("resolve project: %v", err)
	}
	if scope.GitAvailable {
		t.Fatal("non-Git fixture was unexpectedly identified as a Git worktree")
	}
	scan, err := scanner.Scan(context.Background(), scope)
	if err != nil {
		t.Fatalf("scan project: %v", err)
	}
	if want := []string{"keep.go", "nested/root-only.py"}; !reflect.DeepEqual(scan.Files, want) {
		t.Errorf("files = %v, want %v", scan.Files, want)
	}
	if len(scan.Diagnostics) == 0 {
		t.Error("expected non-Git or binary-file diagnostic")
	}
}

func TestScannerLimitsGitDiscoveryToSelectedSubdirectory(t *testing.T) {
	repository := t.TempDir()
	runScannerGit(t, repository, "init")
	runScannerGit(t, repository, "config", "user.name", "Scanner Fixture")
	runScannerGit(t, repository, "config", "user.email", "scanner@example.test")
	writeScannerFile(t, filepath.Join(repository, "backend", "worker.go"), "package backend\nfunc Work() {}\n")
	writeScannerFile(t, filepath.Join(repository, "backend", "vendor", "generated.go"), "package vendor\n")
	writeScannerFile(t, filepath.Join(repository, "frontend", "app.js"), "function app() {}\n")
	writeScannerFile(t, filepath.Join(repository, ".gitignore"), "ignored.py\n")
	writeScannerFile(t, filepath.Join(repository, "backend", "ignored.py"), "print('ignored')\n")
	runScannerGit(t, repository, "add", ".")
	runScannerGit(t, repository, "commit", "-m", "fixture")

	projectPath := filepath.Join(repository, "backend")
	scanner := repositoryscan.NewScanner()
	scope, err := scanner.Resolve(context.Background(), projectPath)
	if err != nil {
		t.Fatalf("resolve Git subdirectory: %v", err)
	}
	if !scope.GitAvailable || scope.GitRepositoryRoot != repository || scope.GitRelativeScope != "backend" {
		t.Errorf("scope = %+v, want Git root and backend scope", scope)
	}
	scan, err := scanner.Scan(context.Background(), scope)
	if err != nil {
		t.Fatalf("scan Git project: %v", err)
	}
	if want := []string{"worker.go"}; !reflect.DeepEqual(scan.Files, want) {
		t.Errorf("files = %v, want %v", scan.Files, want)
	}
}

func writeScannerFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
}

func runScannerGit(t *testing.T, repository string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repository}, arguments...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
}
