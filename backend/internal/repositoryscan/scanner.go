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

	"github.com/Erfangh-80/code-analysis/backend/internal/analysisguard"
	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

var (
	ErrInvalidProjectPath = errors.New("invalid local project path")
	errDiscoveryFileLimit = errors.New("repository discovery file limit reached")
)

const maxInspectableBytes = 8192

type commandRunner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

type gitRunner struct {
	maxOutputBytes int64
}

func (runner gitRunner) Run(ctx context.Context, directory string, arguments ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", directory, "--no-pager"}, arguments...)...)
	command.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	output := boundedGitOutput{limit: runner.maxOutputBytes}
	command.Stdout = &output
	command.Stderr = io.Discard
	if err := command.Run(); err != nil {
		return nil, err
	}
	if output.exceeded {
		return nil, fmt.Errorf("%w: Git discovery output exceeded the configured limit", analysisguard.ErrResourceLimit)
	}
	return output.buffer.Bytes(), nil
}

type boundedGitOutput struct {
	buffer   bytes.Buffer
	limit    int64
	written  int64
	exceeded bool
}

func (output *boundedGitOutput) Write(value []byte) (int, error) {
	if output.limit <= 0 {
		output.limit = analysisguard.DefaultLimits().MaxGitOutputBytes
	}
	remaining := output.limit - output.written
	if remaining <= 0 {
		output.exceeded = true
		return len(value), nil
	}
	if int64(len(value)) > remaining {
		_, _ = output.buffer.Write(value[:remaining])
		output.written += remaining
		output.exceeded = true
		return len(value), nil
	}
	written, err := output.buffer.Write(value)
	output.written += int64(written)
	return len(value), err
}

// Scanner resolves a local project scope and discovers supported source files.
type Scanner struct {
	runner     commandRunner
	pathPolicy *analysisguard.PathPolicy
	limits     analysisguard.Limits
}

// NewScanner creates a backwards-compatible scanner for in-process callers.
// The production server must use NewScannerWithGuards.
func NewScanner() *Scanner {
	limits, _ := analysisguard.DefaultLimits().Normalize()
	return &Scanner{runner: gitRunner{maxOutputBytes: limits.MaxGitOutputBytes}, limits: limits}
}

// NewScannerWithGuards creates a scanner constrained to server-configured
// local roots and source-file budgets.
func NewScannerWithGuards(policy *analysisguard.PathPolicy, limits analysisguard.Limits) (*Scanner, error) {
	if policy == nil {
		return nil, fmt.Errorf("%w: a path policy is required", analysisguard.ErrInvalidAllowedRoot)
	}
	normalized, err := limits.Normalize()
	if err != nil {
		return nil, err
	}
	return &Scanner{runner: gitRunner{maxOutputBytes: normalized.MaxGitOutputBytes}, pathPolicy: policy, limits: normalized}, nil
}

// Resolve validates an absolute local directory and records an optional parent
// Git worktree. The selected ProjectPath remains the analysis scope.
func (scanner *Scanner) Resolve(ctx context.Context, projectPath string) (domain.ProjectScope, error) {
	resolved, err := scanner.pathPolicy.ResolveDirectory(projectPath)
	if err != nil {
		return domain.ProjectScope{}, fmt.Errorf("%w: projectPath is invalid, inaccessible, or outside the allowed analysis roots", ErrInvalidProjectPath)
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
	// Git history requires reading the enclosing worktree. If it is outside the
	// configured roots, retain safe static analysis for the selected path and
	// deliberately mark Git history unavailable.
	if scanner.pathPolicy != nil && !scanner.pathPolicy.Allows(gitRoot) {
		return scope, nil
	}
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
			files, diagnostics, err = scanFilesystem(ctx, scope.ProjectPath, diagnostics, scanner.limits.MaxFiles, scanner.limits.MaxFileBytes)
		}
	} else {
		files, diagnostics, err = scanFilesystem(ctx, scope.ProjectPath, diagnostics, scanner.limits.MaxFiles, scanner.limits.MaxFileBytes)
		diagnostics = append(diagnostics, *diagnostic("GIT_UNAVAILABLE", "No local Git worktree was detected; Git metrics are unavailable for this analysis.", "warning", ""))
	}
	if err != nil {
		return domain.ProjectScan{}, err
	}
	sort.Strings(files)
	files, diagnostics = scanner.limitFiles(scope, files, diagnostics)
	return domain.ProjectScan{Scope: scope, Files: files, Diagnostics: diagnostics}, nil
}

func (scanner *Scanner) limitFiles(scope domain.ProjectScope, files []string, diagnostics []domain.AnalysisDiagnostic) ([]string, []domain.AnalysisDiagnostic) {
	limited := make([]string, 0, min(len(files), scanner.limits.MaxFiles))
	var sourceBytes int64
	fileCountReported := false
	byteBudgetReported := false
	for _, relative := range files {
		if len(limited) >= scanner.limits.MaxFiles {
			if !fileCountReported {
				diagnostics = append(diagnostics, *diagnostic("RESOURCE_LIMIT_FILE_COUNT", "The configured file-count limit was reached; remaining files were skipped.", "warning", ""))
				fileCountReported = true
			}
			continue
		}
		info, err := os.Stat(filepath.Join(scope.ProjectPath, filepath.FromSlash(relative)))
		if err != nil || !info.Mode().IsRegular() {
			diagnostics = append(diagnostics, *diagnostic("FILE_ACCESS_ERROR", "A project file could not be read and was skipped.", "warning", relative))
			continue
		}
		if info.Size() > scanner.limits.MaxFileBytes {
			diagnostics = append(diagnostics, *diagnostic("RESOURCE_LIMIT_FILE_SIZE", "A source file exceeded the configured analysis size limit and was skipped.", "warning", relative))
			continue
		}
		if sourceBytes+info.Size() > scanner.limits.MaxTotalSourceBytes {
			if !byteBudgetReported {
				diagnostics = append(diagnostics, *diagnostic("RESOURCE_LIMIT_SOURCE_BYTES", "The configured total source-byte budget was reached; remaining files were skipped.", "warning", ""))
				byteBudgetReported = true
			}
			continue
		}
		sourceBytes += info.Size()
		limited = append(limited, relative)
	}
	return limited, diagnostics
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

	repositoryPaths := splitNUL(output)
	files := make([]string, 0, min(len(repositoryPaths), scanner.limits.MaxFiles))
	diagnostics := make([]domain.AnalysisDiagnostic, 0)
	seen := make(map[string]struct{})
	fileLimitReported := false
	for _, repositoryPath := range repositoryPaths {
		if err := ctx.Err(); err != nil {
			return nil, diagnostics, err
		}
		absolute := filepath.Join(scope.GitRepositoryRoot, filepath.FromSlash(repositoryPath))
		relative, err := filepath.Rel(scope.ProjectPath, absolute)
		if err != nil || !safeRelativePath(relative) || hasDefaultExcludedDirectory(relative) {
			continue
		}
		if len(files) >= scanner.limits.MaxFiles {
			if !fileLimitReported {
				diagnostics = append(diagnostics, *diagnostic("RESOURCE_LIMIT_FILE_COUNT", "The configured file-count limit was reached; remaining files were skipped.", "warning", ""))
				fileLimitReported = true
			}
			break
		}
		accepted, fileDiagnostic := inspectFile(scope.ProjectPath, relative, scanner.limits.MaxFileBytes)
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

func scanFilesystem(ctx context.Context, root string, diagnostics []domain.AnalysisDiagnostic, maxFiles int, maxFileBytes int64) ([]string, []domain.AnalysisDiagnostic, error) {
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
		accepted, fileDiagnostic := inspectFile(root, relative, maxFileBytes)
		if fileDiagnostic != nil {
			diagnostics = append(diagnostics, *fileDiagnostic)
		}
		if accepted {
			if len(files) >= maxFiles {
				diagnostics = append(diagnostics, *diagnostic("RESOURCE_LIMIT_FILE_COUNT", "The configured file-count limit was reached; remaining files were skipped.", "warning", ""))
				return errDiscoveryFileLimit
			}
			files = append(files, normalized)
		}
		return nil
	})
	if errors.Is(err, errDiscoveryFileLimit) {
		return files, diagnostics, nil
	}
	if err != nil {
		return nil, diagnostics, err
	}
	return files, diagnostics, nil
}

func inspectFile(root, relative string, maxFileBytes int64) (bool, *domain.AnalysisDiagnostic) {
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
	if info.Size() > maxFileBytes {
		return false, diagnostic("RESOURCE_LIMIT_FILE_SIZE", "A source file exceeded the configured analysis size limit and was skipped.", "warning", filepath.ToSlash(relative))
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
