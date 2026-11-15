// Package domain contains the backend's framework-independent analysis model.
package domain

import "time"

// Language identifies a source language supported by the static analyzer.
type Language string

const (
	LanguageGo         Language = "go"
	LanguageJavaScript Language = "javascript"
	LanguagePython     Language = "python"
	LanguageUnknown    Language = "unknown"
)

// AnalysisStatus describes how confidently a file was analyzed.
type AnalysisStatus string

const (
	AnalysisStatusAnalyzed     AnalysisStatus = "analyzed"
	AnalysisStatusEmpty        AnalysisStatus = "empty"
	AnalysisStatusNoExecutable AnalysisStatus = "no_executable_code"
	AnalysisStatusFallback     AnalysisStatus = "fallback"
	AnalysisStatusParseError   AnalysisStatus = "parse_error"
	AnalysisStatusUnsupported  AnalysisStatus = "unsupported"
	AnalysisStatusReadError    AnalysisStatus = "read_error"
)

// DeclarationKind distinguishes top-level functions, methods, and classes.
type DeclarationKind string

const (
	DeclarationKindFunction DeclarationKind = "function"
	DeclarationKindMethod   DeclarationKind = "method"
	DeclarationKindClass    DeclarationKind = "class"
)

// DiagnosticCode is a stable machine-readable reason attached to an analysis.
type DiagnosticCode string

const (
	DiagnosticCodeParseError    DiagnosticCode = "PARSE_ERROR"
	DiagnosticCodeFallbackUsed  DiagnosticCode = "FALLBACK_USED"
	DiagnosticCodeUnsupported   DiagnosticCode = "UNSUPPORTED_LANGUAGE"
	DiagnosticCodeReadError     DiagnosticCode = "READ_ERROR"
	DiagnosticCodeInvalidRequest DiagnosticCode = "INVALID_REQUEST"
)

// Diagnostic describes an honest limitation or failure encountered while reading a file.
type Diagnostic struct {
	Code     DiagnosticCode `json:"code"`
	Message  string         `json:"message"`
	Severity string         `json:"severity"`
}

// LengthSummary summarizes non-blank, non-comment LOC over declarations.
type LengthSummary struct {
	Max   int `json:"max"`
	Total int `json:"total"`
}

// Metrics are static, file-level values calculated by the backend.
type Metrics struct {
	LinesOfCode   int           `json:"linesOfCode"`
	FunctionCount int           `json:"functionCount"`
	ClassCount    int           `json:"classCount"`
	FunctionLength LengthSummary `json:"functionLength"`
	ClassLength    LengthSummary `json:"classLength"`
}

// Declaration is a function, method, or class identified by a language parser.
type Declaration struct {
	Name      string          `json:"name"`
	Kind      DeclarationKind `json:"kind"`
	StartLine int             `json:"startLine"`
	EndLine   int             `json:"endLine"`
	Length    int             `json:"length"`
}

// SourceFile is a source byte sequence supplied to the static analyzer.
// Path should be repository-relative when it is presented outside the backend.
type SourceFile struct {
	Path    string
	Content []byte
}

// FileAnalysis is the complete static result for one source file.
type FileAnalysis struct {
	Path         string         `json:"path"`
	Language     Language       `json:"language"`
	Status       AnalysisStatus `json:"status"`
	UsedFallback bool           `json:"usedFallback"`
	Metrics      Metrics        `json:"metrics"`
	Functions    []Declaration  `json:"functions"`
	Classes      []Declaration  `json:"classes"`
	Diagnostics  []Diagnostic   `json:"diagnostics"`
}

// HasParserFailure reports whether parsing failed, including a recovered fallback result.
func (analysis FileAnalysis) HasParserFailure() bool {
	return analysis.Status == AnalysisStatusParseError || analysis.Status == AnalysisStatusFallback
}

// ModuleAnalysis aggregates file metrics for one repository-relative directory.
type ModuleAnalysis struct {
	Path             string  `json:"path"`
	FileCount        int     `json:"fileCount"`
	FailedFileCount  int     `json:"failedFileCount"`
	Metrics          Metrics `json:"metrics"`
}

// BatchAnalysis is a truthful aggregate returned after a set of local files is analyzed.
type BatchAnalysis struct {
	Files   []FileAnalysis   `json:"files"`
	Modules []ModuleAnalysis `json:"modules"`
}

// FailureRecord records a local file whose source could not be parsed or read cleanly.
type FailureRecord struct {
	Path         string         `json:"path"`
	Language     Language       `json:"language"`
	Status       AnalysisStatus `json:"status"`
	UsedFallback bool           `json:"usedFallback"`
	Diagnostics  []Diagnostic   `json:"diagnostics"`
	RecordedAt   time.Time      `json:"recordedAt"`
}
