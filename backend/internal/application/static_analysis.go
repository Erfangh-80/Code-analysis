// Package application coordinates use cases without depending on HTTP details.
package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
	analyzer *static.Analyzer
	failures FailureRecorder
}

// NewStaticAnalysisService creates the use case from an analyzer and local failure recorder.
func NewStaticAnalysisService(analyzer *static.Analyzer, failures FailureRecorder) *StaticAnalysisService {
	return &StaticAnalysisService{analyzer: analyzer, failures: failures}
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
	root, err := validateRoot(request.RootPath)
	if err != nil {
		return domain.BatchAnalysis{}, err
	}
	if len(request.Files) == 0 {
		return domain.BatchAnalysis{}, fmt.Errorf("%w: at least one file is required", ErrInvalidAnalysisRequest)
	}

	result := domain.BatchAnalysis{Files: make([]domain.FileAnalysis, 0, len(request.Files))}
	for _, requestedPath := range request.Files {
		if err := ctx.Err(); err != nil {
			return domain.BatchAnalysis{}, err
		}

		relativePath, absolutePath, err := pathWithinRoot(root, requestedPath)
		if err != nil {
			return domain.BatchAnalysis{}, err
		}
		content, err := os.ReadFile(absolutePath)
		if err != nil {
			analysis := domain.FileAnalysis{
				Path:     filepath.ToSlash(relativePath),
				Language: languageFromExtension(relativePath),
				Status:   domain.AnalysisStatusReadError,
				Diagnostics: []domain.Diagnostic{{
					Code:     domain.DiagnosticCodeReadError,
					Message:  "The requested local source file could not be read.",
					Severity: "error",
				}},
			}
			result.Files = append(result.Files, analysis)
			service.recordFailure(analysis)
			if onFile != nil {
				onFile(len(result.Files), len(request.Files), analysis)
			}
			continue
		}

		analysis := service.analyzer.Analyze(domain.SourceFile{Path: filepath.ToSlash(relativePath), Content: content})
		result.Files = append(result.Files, analysis)
		if analysis.HasParserFailure() {
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

func validateRoot(path string) (string, error) {
	if strings.TrimSpace(path) == "" || !filepath.IsAbs(path) {
		return "", fmt.Errorf("%w: rootPath must be an absolute local directory", ErrInvalidAnalysisRequest)
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("%w: rootPath is not accessible", ErrInvalidAnalysisRequest)
	}
	info, err := os.Stat(resolvedPath)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%w: rootPath is not a directory", ErrInvalidAnalysisRequest)
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
	if err == nil {
		relativeResolved, relativeErr := filepath.Rel(root, resolvedPath)
		if relativeErr != nil || relativeResolved == ".." || strings.HasPrefix(relativeResolved, ".."+string(filepath.Separator)) {
			return "", "", fmt.Errorf("%w: file path escapes rootPath through a symlink", ErrInvalidAnalysisRequest)
		}
	}
	return relativePath, absolutePath, nil
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
