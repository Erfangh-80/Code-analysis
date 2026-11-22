// Package application coordinates use cases without depending on HTTP details.
package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/analysisguard"
	"github.com/Erfangh-80/code-analysis/backend/internal/analyzer/static"
	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

var (
	// ErrInvalidAnalysisRequest indicates a request that cannot be safely read locally.
	ErrInvalidAnalysisRequest = errors.New("invalid static analysis request")
)

// FailureRecorder stores safe metadata about unsuccessful file analysis.
type FailureRecorder interface {
	Record(domain.FailureRecord)
	List() []domain.FailureRecord
}

// StaticAnalysisRequest identifies local files relative to one local root directory.
type StaticAnalysisRequest struct {
	RootPath string
	Files    []string
}

// StaticAnalysisService reads requested local source files and delegates only static parsing.
type StaticAnalysisService struct {
	analyzer   *static.Analyzer
	failures   FailureRecorder
	pathPolicy *analysisguard.PathPolicy
	limits     analysisguard.Limits
}

// NewStaticAnalysisService creates a backwards-compatible service for in-process
// callers. The production server must use NewStaticAnalysisServiceWithGuards.
func NewStaticAnalysisService(analyzer *static.Analyzer, failures FailureRecorder) *StaticAnalysisService {
	limits, _ := analysisguard.DefaultLimits().Normalize()
	return &StaticAnalysisService{analyzer: analyzer, failures: failures, limits: limits}
}

// NewStaticAnalysisServiceWithGuards creates a bounded service that restricts
// root paths to a server-configured policy. It never stores source content.
func NewStaticAnalysisServiceWithGuards(analyzer *static.Analyzer, failures FailureRecorder, policy *analysisguard.PathPolicy, limits analysisguard.Limits) (*StaticAnalysisService, error) {
	if policy == nil {
		return nil, fmt.Errorf("%w: a path policy is required", analysisguard.ErrInvalidAllowedRoot)
	}
	normalized, err := limits.Normalize()
	if err != nil {
		return nil, err
	}
	return &StaticAnalysisService{analyzer: analyzer, failures: failures, pathPolicy: policy, limits: normalized}, nil
}

// Analyze reads files under RootPath, calculates real static results, and aggregates directories.
// It never executes target source or changes the target filesystem.
func (service *StaticAnalysisService) Analyze(ctx context.Context, request StaticAnalysisRequest) (domain.BatchAnalysis, error) {
	return service.analyze(ctx, request, nil)
}

// AnalyzeWithProgress behaves like Analyze and invokes onFile after every file
// attempt. Callers can persist coarse lifecycle progress without inspecting or
// retaining source content.
func (service *StaticAnalysisService) AnalyzeWithProgress(ctx context.Context, request StaticAnalysisRequest, onFile func(completed, total int, analysis domain.FileAnalysis)) (domain.BatchAnalysis, error) {
	return service.analyze(ctx, request, onFile)
}

func (service *StaticAnalysisService) analyze(ctx context.Context, request StaticAnalysisRequest, onFile func(completed, total int, analysis domain.FileAnalysis)) (domain.BatchAnalysis, error) {
	root, err := service.validateRoot(request.RootPath)
	if err != nil {
		return domain.BatchAnalysis{}, err
	}
	if len(request.Files) == 0 {
		return domain.BatchAnalysis{}, fmt.Errorf("%w: at least one file is required", ErrInvalidAnalysisRequest)
	}
	if len(request.Files) > service.limits.MaxFiles {
		return domain.BatchAnalysis{}, fmt.Errorf("%w: requested file count exceeds the configured limit", analysisguard.ErrResourceLimit)
	}

	result := domain.BatchAnalysis{Files: make([]domain.FileAnalysis, 0, len(request.Files))}
	var sourceBytes int64
	for _, requestedPath := range request.Files {
		if err := ctx.Err(); err != nil {
			return domain.BatchAnalysis{}, err
		}

		relativePath, absolutePath, err := pathWithinRoot(root, requestedPath)
		if err != nil {
			return domain.BatchAnalysis{}, err
		}
		analysis, consumed := service.analyzeFile(relativePath, absolutePath, sourceBytes)
		sourceBytes += consumed
		result.Files = append(result.Files, analysis)
		if analysis.HasParserFailure() || analysis.Status == domain.AnalysisStatusReadError {
			service.recordFailure(analysis)
		}
		if onFile != nil {
			onFile(len(result.Files), len(request.Files), analysis)
		}
	}

	sort.Slice(result.Files, func(left, right int) bool {
		return result.Files[left].Path < result.Files[right].Path
	})
	result.Modules = AggregateModules(result.Files)
	return result, nil
}

func (service *StaticAnalysisService) analyzeFile(relativePath, absolutePath string, usedBytes int64) (domain.FileAnalysis, int64) {
	filePath := filepath.ToSlash(relativePath)
	language := languageFromExtension(relativePath)
	info, err := os.Stat(absolutePath)
	if err != nil || !info.Mode().IsRegular() {
		return unreadableAnalysis(filePath, language, domain.DiagnosticCodeReadError, "The requested local source file could not be read."), 0
	}
	if info.Size() > service.limits.MaxFileBytes {
		return unreadableAnalysis(filePath, language, domain.DiagnosticCodeResourceLimit, "The source file exceeds the configured analysis size limit."), 0
	}
	if usedBytes+info.Size() > service.limits.MaxTotalSourceBytes {
		return unreadableAnalysis(filePath, language, domain.DiagnosticCodeResourceLimit, "The analysis source-byte budget was reached before this file could be read."), 0
	}
	content, err := readBoundedSourceFile(absolutePath, service.limits.MaxFileBytes)
	if err != nil {
		if errors.Is(err, analysisguard.ErrResourceLimit) {
			return unreadableAnalysis(filePath, language, domain.DiagnosticCodeResourceLimit, "The source file exceeds the configured analysis size limit."), 0
		}
		return unreadableAnalysis(filePath, language, domain.DiagnosticCodeReadError, "The requested local source file could not be read."), 0
	}
	if usedBytes+int64(len(content)) > service.limits.MaxTotalSourceBytes {
		return unreadableAnalysis(filePath, language, domain.DiagnosticCodeResourceLimit, "The analysis source-byte budget was reached before this file could be analyzed."), 0
	}
	return service.analyzer.Analyze(domain.SourceFile{Path: filePath, Content: content}), int64(len(content))
}

func readBoundedSourceFile(path string, maxBytes int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maxBytes {
		return nil, fmt.Errorf("%w: source file is too large", analysisguard.ErrResourceLimit)
	}
	return content, nil
}

func unreadableAnalysis(path string, language domain.Language, code domain.DiagnosticCode, message string) domain.FileAnalysis {
	return domain.FileAnalysis{
		Path:     path,
		Language: language,
		Status:   domain.AnalysisStatusReadError,
		Diagnostics: []domain.Diagnostic{{
			Code:     code,
			Message:  message,
			Severity: "error",
		}},
	}
}

// FailedFiles returns failures observed by this local process without source contents.
func (service *StaticAnalysisService) FailedFiles() []domain.FailureRecord {
	return service.failures.List()
}

func (service *StaticAnalysisService) recordFailure(analysis domain.FileAnalysis) {
	service.failures.Record(domain.FailureRecord{
		Path:         analysis.Path,
		Language:     analysis.Language,
		Status:       analysis.Status,
		UsedFallback: analysis.UsedFallback,
		Diagnostics:  analysis.Diagnostics,
		RecordedAt:   time.Now().UTC(),
	})
}

// AggregateModules combines real file-level static metrics by repository-relative directory.
func AggregateModules(files []domain.FileAnalysis) []domain.ModuleAnalysis {
	modules := make(map[string]*domain.ModuleAnalysis)
	for _, file := range files {
		modulePath := filepath.ToSlash(filepath.Dir(file.Path))
		if modulePath == "" || modulePath == "." {
			modulePath = "."
		}
		module, exists := modules[modulePath]
		if !exists {
			module = &domain.ModuleAnalysis{Path: modulePath}
			modules[modulePath] = module
		}

		module.FileCount++
		if file.HasParserFailure() || file.Status == domain.AnalysisStatusReadError {
			module.FailedFileCount++
		}
		addMetrics(&module.Metrics, file.Metrics)
	}

	paths := make([]string, 0, len(modules))
	for path := range modules {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	result := make([]domain.ModuleAnalysis, 0, len(paths))
	for _, path := range paths {
		result = append(result, *modules[path])
	}
	return result
}

func addMetrics(target *domain.Metrics, source domain.Metrics) {
	target.LinesOfCode += source.LinesOfCode
	target.FunctionCount += source.FunctionCount
	target.ClassCount += source.ClassCount
	target.FunctionLength.Total += source.FunctionLength.Total
	target.ClassLength.Total += source.ClassLength.Total
	if source.FunctionLength.Max > target.FunctionLength.Max {
		target.FunctionLength.Max = source.FunctionLength.Max
	}
	if source.ClassLength.Max > target.ClassLength.Max {
		target.ClassLength.Max = source.ClassLength.Max
	}
}

func (service *StaticAnalysisService) validateRoot(path string) (string, error) {
	resolvedPath, err := service.pathPolicy.ResolveDirectory(path)
	if err != nil {
		return "", fmt.Errorf("%w: rootPath is invalid, inaccessible, or outside the allowed analysis roots", ErrInvalidAnalysisRequest)
	}
	return resolvedPath, nil
}

func pathWithinRoot(root, requestedPath string) (string, string, error) {
	if strings.TrimSpace(requestedPath) == "" || filepath.IsAbs(requestedPath) {
		return "", "", fmt.Errorf("%w: files must be non-empty relative paths", ErrInvalidAnalysisRequest)
	}
	relativePath := filepath.Clean(requestedPath)
	if relativePath == "." || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("%w: file path escapes rootPath", ErrInvalidAnalysisRequest)
	}
	absolutePath := filepath.Join(root, relativePath)
	resolvedPath, err := filepath.EvalSymlinks(absolutePath)
	if err != nil {
		return relativePath, absolutePath, nil
	}
	relativeResolved, relativeErr := filepath.Rel(root, resolvedPath)
	if relativeErr != nil || relativeResolved == ".." || strings.HasPrefix(relativeResolved, ".."+string(filepath.Separator)) || filepath.IsAbs(relativeResolved) {
		return "", "", fmt.Errorf("%w: file path escapes rootPath through a symlink", ErrInvalidAnalysisRequest)
	}
	return relativePath, resolvedPath, nil
}

func languageFromExtension(path string) domain.Language {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return domain.LanguageGo
	case ".js", ".mjs", ".cjs":
		return domain.LanguageJavaScript
	case ".py":
		return domain.LanguagePython
	default:
		return domain.LanguageUnknown
	}
}
