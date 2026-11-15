// Package repositoryscan discovers analyzable local project files without
// executing project code or requiring a Git repository.
package repositoryscan

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

var ErrInvalidProjectPath = errors.New("invalid local project path")

const maxInspectableBytes = 8192

type commandRunner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

type gitRunner struct{}

func (gitRunner) Run(ctx context.Context, directory string, arguments ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", directory, "--no-pager"}, arguments...)...)
	command.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	return command.Output()
}

// Scanner resolves a local project scope and discovers supported source files.
type Scanner struct {
	runner commandRunner
}

// NewScanner creates the production scanner. It uses Git only for read-only
// scope discovery and ignored-file handling when a local worktree exists.
func NewScanner() *Scanner {
	return &Scanner{runner: gitRunner{}}
}

// Resolve validates an absolute local directory and records an optional parent
// Git worktree. The selected ProjectPath remains the analysis scope.
func (scanner *Scanner) Resolve(ctx context.Context, projectPath string) (domain.ProjectScope, error) {
	if strings.TrimSpace(projectPath) == "" || !filepath.IsAbs(projectPath) {
		return domain.ProjectScope{}, fmt.Errorf("%w: projectPath must be an absolute local directory", ErrInvalidProjectPath)
	}
	resolved, err := filepath.EvalSymlinks(projectPath)
	if err != nil {
		return domain.ProjectScope{}, fmt.Errorf("%w: projectPath is inaccessible", ErrInvalidProjectPath)
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return domain.ProjectScope{}, fmt.Errorf("%w: projectPath is not a directory", ErrInvalidProjectPath)
	}

	scope := domain.ProjectScope{ProjectPath: resolved, ScopeKey: resolved}
	insideWorktree, err := scanner.runner.Run(ctx, resolved, "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(string(insideWorktree)) != "true" {
		return scope, nil
	}
	root, err := scanner.runner.Run(ctx, resolved, "rev-parse", "--show-toplevel")
	if err != nil {
		return scope, nil
	}
	gitRoot := strings.TrimSpace(string(root))
	relative, err := filepath.Rel(gitRoot, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return scope, nil
	}
	if relative == "." {
		relative = ""
	}
	scope.GitAvailable = true
	scope.GitRepositoryRoot = gitRoot
	scope.GitRelativeScope = filepath.ToSlash(relative)
	// A Git repository is the conflict boundary even when a selected project is
	// a subdirectory of that worktree.
	scope.ScopeKey = gitRoot
	return scope, nil
}

// Scan returns supported regular source files only. Every returned path is
// slash-normalized and relative to the selected local project path.
func (scanner *Scanner) Scan(ctx context.Context, scope domain.ProjectScope) (domain.ProjectScan, error) {
	if err := ctx.Err(); err != nil {
		return domain.ProjectScan{}, err
	}
	var files []string
	var diagnostics []domain.AnalysisDiagnostic
	var err error
	if scope.GitAvailable {
		files, diagnostics, err = scanner.scanGitWorktree(ctx, scope)
		if err != nil {
			diagnostics = append(diagnostics, *diagnostic("GIT_DISCOVERY_UNAVAILABLE", "Git file discovery was unavailable; the local filesystem scanner was used instead.", "warning", ""))
			files, diagnostics, err = scanFilesystem(ctx, scope.ProjectPath, diagnostics)
		}
	} else {
		files, diagnostics, err = scanFilesystem(ctx, scope.ProjectPath, diagnostics)
		diagnostics = append(diagnostics, *diagnostic("GIT_UNAVAILABLE", "No local Git worktree was detected; Git metrics are unavailable for this analysis.", "warning", ""))
	}
	if err != nil {
		return domain.ProjectScan{}, err
	}
	sort.Strings(files)
	return domain.ProjectScan{Scope: scope, Files: files, Diagnostics: diagnostics}, nil
}

func (scanner *Scanner) scanGitWorktree(ctx context.Context, scope domain.ProjectScope) ([]string, []domain.AnalysisDiagnostic, error) {
	arguments := []string{"ls-files", "--cached", "--others", "--exclude-standard", "-z", "--"}
	if scope.GitRelativeScope == "" {
		arguments = append(arguments, ".")
	} else {
		arguments = append(arguments, scope.GitRelativeScope)
	}
	output, err := scanner.runner.Run(ctx, scope.GitRepositoryRoot, arguments...)
	if err != nil {
		return nil, nil, err
	}

	files := make([]string, 0)
	diagnostics := make([]domain.AnalysisDiagnostic, 0)
	seen := make(map[string]struct{})
	for _, repositoryPath := range splitNUL(output) {
		if err := ctx.Err(); err != nil {
			return nil, diagnostics, err
		}
		absolute := filepath.Join(scope.GitRepositoryRoot, filepath.FromSlash(repositoryPath))
		relative, err := filepath.Rel(scope.ProjectPath, absolute)
		if err != nil || !safeRelativePath(relative) || hasDefaultExcludedDirectory(relative) {
			continue
		}
		accepted, fileDiagnostic := inspectFile(scope.ProjectPath, relative)
		if fileDiagnostic != nil {
			diagnostics = append(diagnostics, *fileDiagnostic)
		}
		if !accepted {
			continue
		}
		normalized := filepath.ToSlash(relative)
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		files = append(files, normalized)
	}
	return files, diagnostics, nil
}

func scanFilesystem(ctx context.Context, root string, diagnostics []domain.AnalysisDiagnostic) ([]string, []domain.AnalysisDiagnostic, error) {
	matcher, err := newIgnoreMatcher(root)
	if err != nil {
		return nil, diagnostics, err
	}
	files := make([]string, 0)
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			relative, relativeErr := filepath.Rel(root, path)
			if relativeErr == nil && safeRelativePath(relative) {
				diagnostics = append(diagnostics, *diagnostic("FILE_ACCESS_ERROR", "A project path could not be read and was skipped.", "warning", filepath.ToSlash(relative)))
				return nil
			}
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil || !safeRelativePath(relative) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		normalized := filepath.ToSlash(relative)
		if entry.IsDir() {
			if isDefaultExcludedDirectory(entry.Name()) || matcher.ignored(normalized, true) {
				return filepath.SkipDir
			}
			matcher.loadIgnoreFile(path, normalized)
			return nil
		}
		if entry.Name() == ".gitignore" {
			return nil
		}
		if matcher.ignored(normalized, false) {
			return nil
		}
		accepted, fileDiagnostic := inspectFile(root, relative)
		if fileDiagnostic != nil {
			diagnostics = append(diagnostics, *fileDiagnostic)
		}
		if accepted {
			files = append(files, normalized)
		}
		return nil
	})
	if err != nil {
		return nil, diagnostics, err
	}
	return files, diagnostics, nil
}

func inspectFile(root, relative string) (bool, *domain.AnalysisDiagnostic) {
	if !safeRelativePath(relative) || !supportedExtension(relative) {
		return false, nil
	}
	absolute := filepath.Join(root, relative)
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return false, diagnostic("FILE_ACCESS_ERROR", "A project file could not be read and was skipped.", "warning", filepath.ToSlash(relative))
	}
	resolvedRelative, err := filepath.Rel(root, resolved)
	if err != nil || !safeRelativePath(resolvedRelative) {
		return false, diagnostic("UNSAFE_SYMLINK", "A symbolic link outside the selected project was skipped.", "warning", filepath.ToSlash(relative))
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return false, diagnostic("UNSUPPORTED_FILE", "A non-regular project file was skipped.", "warning", filepath.ToSlash(relative))
	}
	file, err := os.Open(resolved)
	if err != nil {
		return false, diagnostic("FILE_ACCESS_ERROR", "A project file could not be read and was skipped.", "warning", filepath.ToSlash(relative))
	}
	defer file.Close()
	buffer := make([]byte, maxInspectableBytes)
	read, readErr := file.Read(buffer)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return false, diagnostic("FILE_ACCESS_ERROR", "A project file could not be read and was skipped.", "warning", filepath.ToSlash(relative))
	}
	if bytes.IndexByte(buffer[:read], 0) >= 0 {
		return false, diagnostic("BINARY_FILE", "A binary project file was skipped.", "warning", filepath.ToSlash(relative))
	}
	return true, nil
}

func supportedExtension(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".js", ".mjs", ".cjs", ".py":
		return true
	default:
		return false
	}
}

func safeRelativePath(path string) bool {
	cleaned := filepath.Clean(path)
	return cleaned != "." && cleaned != ".." && !filepath.IsAbs(cleaned) && !strings.HasPrefix(cleaned, ".."+string(filepath.Separator))
}

func isDefaultExcludedDirectory(name string) bool {
	switch name {
	case ".git", "node_modules", "vendor", ".venv", "venv", "__pycache__", "dist", "build", ".next", ".cache":
		return true
	default:
		return false
	}
}

func hasDefaultExcludedDirectory(relative string) bool {
	for _, segment := range strings.Split(filepath.ToSlash(relative), "/") {
		if isDefaultExcludedDirectory(segment) {
			return true
		}
	}
	return false
}

func splitNUL(output []byte) []string {
	items := strings.Split(string(output), "\x00")
	if len(items) > 0 && items[len(items)-1] == "" {
		return items[:len(items)-1]
	}
	return items
}

func diagnostic(code, message, severity, path string) *domain.AnalysisDiagnostic {
	return &domain.AnalysisDiagnostic{Stage: domain.AnalysisStageDiscovery, Path: path, Code: code, Message: message, Severity: severity}
}
