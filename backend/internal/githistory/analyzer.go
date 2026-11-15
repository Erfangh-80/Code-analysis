// Package githistory provides read-only local Git-history analysis.
package githistory

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

var (
	// ErrInvalidRepository is returned when an input is not a readable local Git worktree.
	ErrInvalidRepository = errors.New("invalid local Git repository")
	// ErrInvalidHistoryRange is returned when optional history bounds are malformed.
	ErrInvalidHistoryRange = errors.New("invalid Git history range")
	// ErrHistoryUnavailable wraps read-only Git failures other than an empty history.
	ErrHistoryUnavailable = errors.New("Git history is unavailable")
)

const (
	recordSeparator = "\x1e"
	fieldSeparator  = "\x1f"
)

type commandRunner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

type gitCommandRunner struct{}

func (gitCommandRunner) Run(ctx context.Context, directory string, arguments ...string) ([]byte, error) {
	commandArguments := append([]string{"-C", directory, "--no-pager"}, arguments...)
	command := exec.CommandContext(ctx, "git", commandArguments...)
	command.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git command failed: %w", err)
	}
	return output, nil
}

// Analyzer reads Git metadata and diffs without changing the target repository.
type Analyzer struct {
	runner commandRunner
}

// NewAnalyzer creates the production read-only Git analyzer.
func NewAnalyzer() *Analyzer {
	return &Analyzer{runner: gitCommandRunner{}}
}

// Analyze extracts commit and per-file history facts from a local repository.
func (analyzer *Analyzer) Analyze(ctx context.Context, request domain.GitAnalysisRequest) (domain.GitAnalysisReport, error) {
	if err := validateHistoryRange(request.History); err != nil {
		return domain.GitAnalysisReport{}, err
	}

	repositoryRoot, err := analyzer.validateRepository(ctx, request.RepositoryPath)
	if err != nil {
		return domain.GitAnalysisReport{}, err
	}

	report := domain.GitAnalysisReport{History: request.History, Files: make([]domain.GitFileMetrics, 0)}
	noCommits, err := analyzer.hasNoCommits(ctx, repositoryRoot)
	if err != nil {
		return domain.GitAnalysisReport{}, err
	}
	if noCommits {
		report.Diagnostics = append(report.Diagnostics, diagnostic(
			domain.DiagnosticCodeInsufficientHistory,
			"The local repository has no commits to analyze.",
			"warning",
		))
		return report, nil
	}

	if shallow, shallowErr := analyzer.isShallowRepository(ctx, repositoryRoot); shallowErr != nil {
		return domain.GitAnalysisReport{}, shallowErr
	} else if shallow {
		report.Diagnostics = append(report.Diagnostics, diagnostic(
			domain.DiagnosticCodeShallowHistory,
			"The local repository is shallow, so older commits may be unavailable.",
			"warning",
		))
	}

	commits, err := analyzer.commits(ctx, repositoryRoot, request.History)
	if err != nil {
		return domain.GitAnalysisReport{}, err
	}
	report.CommitCount = len(commits)
	if len(commits) < 2 {
		report.Diagnostics = append(report.Diagnostics, diagnostic(
			domain.DiagnosticCodeInsufficientHistory,
			"Fewer than two commits are available in the selected history range.",
			"warning",
		))
	}

	files := make(map[string]*fileAccumulator)
	for _, commit := range commits {
		if err := ctx.Err(); err != nil {
			return domain.GitAnalysisReport{}, err
		}
		changes, err := analyzer.changesForCommit(ctx, repositoryRoot, commit.Hash)
		if err != nil {
			return domain.GitAnalysisReport{}, err
		}
		for _, change := range changes {
			accumulator := applyIdentityChange(files, change)
			accumulator.apply(change, commit)
		}
	}

	report.Files = summarizeFiles(files)
	return report, nil
}

func (analyzer *Analyzer) validateRepository(ctx context.Context, path string) (string, error) {
	if strings.TrimSpace(path) == "" || !filepath.IsAbs(path) {
		return "", fmt.Errorf("%w: repositoryPath must be an absolute local path", ErrInvalidRepository)
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("%w: repositoryPath is inaccessible", ErrInvalidRepository)
	}
	info, err := os.Stat(resolvedPath)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%w: repositoryPath is not a directory", ErrInvalidRepository)
	}

	insideWorktree, err := analyzer.git(ctx, resolvedPath, "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(string(insideWorktree)) != "true" {
		return "", fmt.Errorf("%w: path is not a Git worktree", ErrInvalidRepository)
	}
	root, err := analyzer.git(ctx, resolvedPath, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(root)), nil
}

func (analyzer *Analyzer) hasNoCommits(ctx context.Context, repositoryRoot string) (bool, error) {
	_, err := analyzer.runner.Run(ctx, repositoryRoot, "rev-parse", "--verify", "--quiet", "HEAD")
	if err == nil {
		return false, nil
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
		return true, nil
	}
	return false, fmt.Errorf("%w: %v", ErrHistoryUnavailable, err)
}

func (analyzer *Analyzer) isShallowRepository(ctx context.Context, repositoryRoot string) (bool, error) {
	output, err := analyzer.git(ctx, repositoryRoot, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(output)) == "true", nil
}

func (analyzer *Analyzer) commits(ctx context.Context, repositoryRoot string, history domain.GitHistoryRange) ([]commit, error) {
	arguments := []string{"log", "--reverse", "--format=%x1e%H%x1f%an%x1f%ae%x1f%cI"}
	if history.From != nil {
		arguments = append(arguments, "--since="+history.From.UTC().Format(time.RFC3339))
	}
	if history.To != nil {
		arguments = append(arguments, "--until="+history.To.UTC().Format(time.RFC3339))
	}
	arguments = append(arguments, "HEAD")

	output, err := analyzer.git(ctx, repositoryRoot, arguments...)
	if err != nil {
		return nil, err
	}
	return parseCommits(output)
}

func (analyzer *Analyzer) changesForCommit(ctx context.Context, repositoryRoot, hash string) ([]fileChange, error) {
	nameStatus, err := analyzer.git(ctx, repositoryRoot, "diff-tree", "--root", "--no-commit-id", "--name-status", "-z", "--find-renames", "-r", hash)
	if err != nil {
		return nil, err
	}
	numstat, err := analyzer.git(ctx, repositoryRoot, "diff-tree", "--root", "--no-commit-id", "--numstat", "-z", "--find-renames", "-r", hash)
	if err != nil {
		return nil, err
	}

	statuses, err := parseNameStatus(nameStatus)
	if err != nil {
		return nil, err
	}
	statistics, err := parseNumstat(numstat)
	if err != nil {
		return nil, err
	}
	return joinChanges(statuses, statistics), nil
}

func (analyzer *Analyzer) git(ctx context.Context, repositoryRoot string, arguments ...string) ([]byte, error) {
	output, err := analyzer.runner.Run(ctx, repositoryRoot, arguments...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHistoryUnavailable, err)
	}
	return output, nil
}

type commit struct {
	Hash        string
	AuthorEmail string
	At          time.Time
}

type fileStatus struct {
	Code    string
	OldPath string
	NewPath string
}

type numstat struct {
	OldPath    string
	NewPath    string
	Additions  int
	Deletions  int
	HasBinary  bool
}

type fileChange struct {
	Code      string
	OldPath   string
	NewPath   string
	Additions int
	Deletions int
	HasBinary bool
}

func parseCommits(output []byte) ([]commit, error) {
	entries := strings.Split(string(output), recordSeparator)
	commits := make([]commit, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		fields := strings.Split(entry, fieldSeparator)
		if len(fields) != 4 {
			return nil, fmt.Errorf("%w: malformed commit metadata", ErrHistoryUnavailable)
		}
		instant, err := time.Parse(time.RFC3339, strings.TrimSpace(fields[3]))
		if err != nil {
			return nil, fmt.Errorf("%w: malformed commit timestamp", ErrHistoryUnavailable)
		}
		author := strings.TrimSpace(fields[2])
		if author == "" {
			author = strings.TrimSpace(fields[1])
		}
		commits = append(commits, commit{Hash: fields[0], AuthorEmail: author, At: instant.UTC()})
	}
	return commits, nil
}

func parseNameStatus(output []byte) ([]fileStatus, error) {
	parts := splitNUL(output)
	statuses := make([]fileStatus, 0)
	for index := 0; index < len(parts); {
		code := parts[index]
		index++
		if code == "" {
			continue
		}
		if index >= len(parts) {
			return nil, fmt.Errorf("%w: incomplete name-status record", ErrHistoryUnavailable)
		}
		baseCode := string(code[0])
		if baseCode == "R" || baseCode == "C" {
			if index+1 >= len(parts) {
				return nil, fmt.Errorf("%w: incomplete rename record", ErrHistoryUnavailable)
			}
			statuses = append(statuses, fileStatus{Code: baseCode, OldPath: normalizePath(parts[index]), NewPath: normalizePath(parts[index+1])})
			index += 2
			continue
		}
		path := normalizePath(parts[index])
		index++
		status := fileStatus{Code: baseCode, NewPath: path}
		if baseCode == "D" {
			status.OldPath = path
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func parseNumstat(output []byte) ([]numstat, error) {
	parts := splitNUL(output)
	statistics := make([]numstat, 0)
	for index := 0; index < len(parts); {
		record := parts[index]
		index++
		if record == "" {
			continue
		}
		fields := strings.SplitN(record, "\t", 3)
		if len(fields) != 3 {
			return nil, fmt.Errorf("%w: malformed numstat record", ErrHistoryUnavailable)
		}
		additions, deletions, binary, err := parseNumstatCounts(fields[0], fields[1])
		if err != nil {
			return nil, err
		}
		statistic := numstat{Additions: additions, Deletions: deletions, HasBinary: binary}
		if fields[2] == "" {
			if index+1 >= len(parts) {
				return nil, fmt.Errorf("%w: incomplete renamed numstat record", ErrHistoryUnavailable)
			}
			statistic.OldPath = normalizePath(parts[index])
			statistic.NewPath = normalizePath(parts[index+1])
			index += 2
		} else {
			statistic.NewPath = normalizePath(fields[2])
		}
		statistics = append(statistics, statistic)
	}
	return statistics, nil
}

func parseNumstatCounts(additions, deletions string) (int, int, bool, error) {
	if additions == "-" || deletions == "-" {
		return 0, 0, true, nil
	}
	added, addedErr := strconv.Atoi(additions)
	removed, removedErr := strconv.Atoi(deletions)
	if addedErr != nil || removedErr != nil || added < 0 || removed < 0 {
		return 0, 0, false, fmt.Errorf("%w: invalid numstat counts", ErrHistoryUnavailable)
	}
	return added, removed, false, nil
}

func joinChanges(statuses []fileStatus, statistics []numstat) []fileChange {
	changes := make([]fileChange, 0, len(statuses))
	used := make([]bool, len(statistics))
	for _, status := range statuses {
		change := fileChange{Code: status.Code, OldPath: status.OldPath, NewPath: status.NewPath}
		for index, statistic := range statistics {
			if used[index] || !matches(status, statistic) {
				continue
			}
			used[index] = true
			change.Additions = statistic.Additions
			change.Deletions = statistic.Deletions
			change.HasBinary = statistic.HasBinary
			break
		}
		changes = append(changes, change)
	}
	return changes
}

func matches(status fileStatus, statistic numstat) bool {
	if status.Code == "R" || status.Code == "C" {
		return status.OldPath == statistic.OldPath && status.NewPath == statistic.NewPath
	}
	return status.NewPath == statistic.NewPath
}

type fileAccumulator struct {
	path          string
	previousPaths map[string]struct{}
	deleted       bool
	churn         int
	changes       int
	authors       map[string]struct{}
	firstChanged  *time.Time
	lastChanged   *time.Time
	monthlyChange map[string]int
}

func applyIdentityChange(files map[string]*fileAccumulator, change fileChange) *fileAccumulator {
	if change.Code == "R" {
		accumulator := files[change.OldPath]
		if accumulator == nil {
			accumulator = newFileAccumulator(change.NewPath)
		}
		if change.OldPath != "" && change.OldPath != change.NewPath {
			accumulator.previousPaths[change.OldPath] = struct{}{}
			delete(files, change.OldPath)
		}
		accumulator.path = change.NewPath
		accumulator.deleted = false
		files[change.NewPath] = accumulator
		return accumulator
	}

	path := change.NewPath
	if change.Code == "D" {
		path = change.OldPath
	}
	accumulator := files[path]
	if accumulator == nil {
		accumulator = newFileAccumulator(path)
		files[path] = accumulator
	}
	if change.Code == "C" && change.OldPath != "" {
		accumulator.previousPaths[change.OldPath] = struct{}{}
	}
	accumulator.deleted = change.Code == "D"
	return accumulator
}

func newFileAccumulator(path string) *fileAccumulator {
	return &fileAccumulator{
		path:          path,
		previousPaths: make(map[string]struct{}),
		authors:       make(map[string]struct{}),
		monthlyChange: make(map[string]int),
	}
}

func (accumulator *fileAccumulator) apply(change fileChange, commit commit) {
	accumulator.churn += change.Additions + change.Deletions
	accumulator.changes++
	accumulator.authors[commit.AuthorEmail] = struct{}{}
	accumulator.monthlyChange[commit.At.UTC().Format("2006-01")]++

	instant := commit.At.UTC()
	if accumulator.firstChanged == nil || instant.Before(*accumulator.firstChanged) {
		accumulator.firstChanged = &instant
	}
	if accumulator.lastChanged == nil || instant.After(*accumulator.lastChanged) {
		accumulator.lastChanged = &instant
	}
}

func summarizeFiles(files map[string]*fileAccumulator) []domain.GitFileMetrics {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	result := make([]domain.GitFileMetrics, 0, len(paths))
	for _, path := range paths {
		accumulator := files[path]
		previousPaths := make([]string, 0, len(accumulator.previousPaths))
		for previousPath := range accumulator.previousPaths {
			previousPaths = append(previousPaths, previousPath)
		}
		sort.Strings(previousPaths)
		concentration := changeConcentration(accumulator.monthlyChange, accumulator.changes)
		result = append(result, domain.GitFileMetrics{
			Path:                accumulator.path,
			PreviousPaths:       previousPaths,
			Deleted:             accumulator.deleted,
			GitChurn:            accumulator.churn,
			ChangeFrequency:     accumulator.changes,
			AuthorCount:         len(accumulator.authors),
			FirstChangedAt:      accumulator.firstChanged,
			LastChangedAt:       accumulator.lastChanged,
			ChangeConcentration: concentration,
			ChangeDispersion:    1 - concentration,
		})
	}
	return result
}

func changeConcentration(monthlyChanges map[string]int, totalChanges int) float64 {
	if totalChanges == 0 {
		return 0
	}
	concentration := 0.0
	for _, changes := range monthlyChanges {
		share := float64(changes) / float64(totalChanges)
		concentration += share * share
	}
	return math.Round(concentration*10000) / 10000
}

func splitNUL(output []byte) []string {
	parts := strings.Split(string(output), "\x00")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		return parts[:len(parts)-1]
	}
	return parts
}

func normalizePath(path string) string {
	return filepath.ToSlash(path)
}

func validateHistoryRange(history domain.GitHistoryRange) error {
	if history.From != nil && history.To != nil && history.From.After(*history.To) {
		return fmt.Errorf("%w: from must be before or equal to to", ErrInvalidHistoryRange)
	}
	return nil
}

func diagnostic(code domain.DiagnosticCode, message, severity string) domain.Diagnostic {
	return domain.Diagnostic{Code: code, Message: message, Severity: severity}
}
