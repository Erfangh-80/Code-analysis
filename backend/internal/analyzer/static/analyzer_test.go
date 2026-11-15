package static

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

func TestAnalyzerExtractsGoFunctionsAndMethodsAndCalculatesMetrics(t *testing.T) {
	t.Parallel()

	analysis := analyzeFixture(t, "sample.go")

	if analysis.Language != domain.LanguageGo {
		t.Fatalf("language = %q, want %q", analysis.Language, domain.LanguageGo)
	}
	if analysis.Status != domain.AnalysisStatusAnalyzed {
		t.Fatalf("status = %q, want %q", analysis.Status, domain.AnalysisStatusAnalyzed)
	}
	if analysis.Metrics.LinesOfCode != 11 {
		t.Errorf("LOC = %d, want 11", analysis.Metrics.LinesOfCode)
	}
	if analysis.Metrics.FunctionCount != 2 {
		t.Errorf("function count = %d, want 2", analysis.Metrics.FunctionCount)
	}
	if analysis.Metrics.ClassCount != 0 {
		t.Errorf("class count = %d, want 0", analysis.Metrics.ClassCount)
	}
	if analysis.Metrics.FunctionLength.Max != 6 || analysis.Metrics.FunctionLength.Total != 9 {
		t.Errorf("function lengths = %+v, want max=6 total=9", analysis.Metrics.FunctionLength)
	}
	assertDeclaration(t, analysis.Functions, "NewAccount", domain.DeclarationKindFunction, 3)
	assertDeclaration(t, analysis.Functions, "Credit", domain.DeclarationKindMethod, 6)
}

func TestAnalyzerExtractsJavaScriptClassesAndMethodsAndCalculatesMetrics(t *testing.T) {
	t.Parallel()

	analysis := analyzeFixture(t, "sample.js")

	if analysis.Language != domain.LanguageJavaScript {
		t.Fatalf("language = %q, want %q", analysis.Language, domain.LanguageJavaScript)
	}
	if analysis.Status != domain.AnalysisStatusAnalyzed {
		t.Fatalf("status = %q, want %q", analysis.Status, domain.AnalysisStatusAnalyzed)
	}
	if analysis.Metrics.LinesOfCode != 8 {
		t.Errorf("LOC = %d, want 8", analysis.Metrics.LinesOfCode)
	}
	if analysis.Metrics.FunctionCount != 2 || analysis.Metrics.ClassCount != 1 {
		t.Errorf("counts = functions:%d classes:%d, want functions:2 classes:1", analysis.Metrics.FunctionCount, analysis.Metrics.ClassCount)
	}
	if analysis.Metrics.FunctionLength.Max != 3 || analysis.Metrics.FunctionLength.Total != 6 {
		t.Errorf("function lengths = %+v, want max=3 total=6", analysis.Metrics.FunctionLength)
	}
	if analysis.Metrics.ClassLength.Max != 5 || analysis.Metrics.ClassLength.Total != 5 {
		t.Errorf("class lengths = %+v, want max=5 total=5", analysis.Metrics.ClassLength)
	}
	assertDeclaration(t, analysis.Classes, "Account", domain.DeclarationKindClass, 5)
	assertDeclaration(t, analysis.Functions, "credit", domain.DeclarationKindMethod, 3)
}

func TestAnalyzerExtractsPythonClassesAndMethodsAndCalculatesMetrics(t *testing.T) {
	t.Parallel()

	analysis := analyzeFixture(t, "sample.py")

	if analysis.Language != domain.LanguagePython {
		t.Fatalf("language = %q, want %q", analysis.Language, domain.LanguagePython)
	}
	if analysis.Status != domain.AnalysisStatusAnalyzed {
		t.Fatalf("status = %q, want %q", analysis.Status, domain.AnalysisStatusAnalyzed)
	}
	if analysis.Metrics.LinesOfCode != 5 {
		t.Errorf("LOC = %d, want 5", analysis.Metrics.LinesOfCode)
	}
	if analysis.Metrics.FunctionCount != 2 || analysis.Metrics.ClassCount != 1 {
		t.Errorf("counts = functions:%d classes:%d, want functions:2 classes:1", analysis.Metrics.FunctionCount, analysis.Metrics.ClassCount)
	}
	if analysis.Metrics.FunctionLength.Max != 2 || analysis.Metrics.FunctionLength.Total != 4 {
		t.Errorf("function lengths = %+v, want max=2 total=4", analysis.Metrics.FunctionLength)
	}
	if analysis.Metrics.ClassLength.Max != 3 || analysis.Metrics.ClassLength.Total != 3 {
		t.Errorf("class lengths = %+v, want max=3 total=3", analysis.Metrics.ClassLength)
	}
	assertDeclaration(t, analysis.Classes, "Account", domain.DeclarationKindClass, 3)
	assertDeclaration(t, analysis.Functions, "credit", domain.DeclarationKindMethod, 2)
}

func TestAnalyzerHandlesEmptyAndNonExecutableFiles(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()

	empty := analyzer.Analyze(domain.SourceFile{Path: "empty.py", Content: []byte("\n\t \n")})
	if empty.Status != domain.AnalysisStatusEmpty || empty.Metrics.LinesOfCode != 0 {
		t.Errorf("empty file = status:%q metrics:%+v", empty.Status, empty.Metrics)
	}

	comments := analyzeFixture(t, "comments.go")
	if comments.Status != domain.AnalysisStatusNoExecutable {
		t.Errorf("comment-only status = %q, want %q", comments.Status, domain.AnalysisStatusNoExecutable)
	}
	if comments.Metrics.LinesOfCode != 0 || comments.Metrics.FunctionCount != 0 || comments.Metrics.ClassCount != 0 {
		t.Errorf("comment-only metrics = %+v, want zero metrics", comments.Metrics)
	}

	declarationsOnly := analyzer.Analyze(domain.SourceFile{Path: "metadata.go", Content: []byte("package metadata\n")})
	if declarationsOnly.Status != domain.AnalysisStatusNoExecutable || declarationsOnly.Metrics.LinesOfCode != 1 {
		t.Errorf("declaration-only result = %+v, want no-executable status with one LOC", declarationsOnly)
	}
}

func TestAnalyzerUsesFallbackAndReportsParserFailures(t *testing.T) {
	t.Parallel()

	analysis := analyzeFixture(t, "broken.js")

	if analysis.Status != domain.AnalysisStatusFallback {
		t.Fatalf("status = %q, want %q", analysis.Status, domain.AnalysisStatusFallback)
	}
	if !analysis.UsedFallback {
		t.Error("UsedFallback = false, want true")
	}
	if len(analysis.Diagnostics) == 0 || analysis.Diagnostics[0].Code != domain.DiagnosticCodeParseError {
		t.Errorf("diagnostics = %+v, want parse-error diagnostic", analysis.Diagnostics)
	}
	if analysis.Metrics.FunctionCount != 1 {
		t.Errorf("function count = %d, want 1 from fallback", analysis.Metrics.FunctionCount)
	}

	unrecoverable := NewAnalyzer().Analyze(domain.SourceFile{Path: "unrecoverable.js", Content: []byte("function (\n")})
	if unrecoverable.Status != domain.AnalysisStatusParseError || unrecoverable.UsedFallback {
		t.Errorf("unrecoverable result = %+v, want parse error without fallback", unrecoverable)
	}
}

func TestAnalyzerMarksUnsupportedFilesWithoutFabricatingMetrics(t *testing.T) {
	t.Parallel()

	analysis := NewAnalyzer().Analyze(domain.SourceFile{Path: "notes.txt", Content: []byte("plain text")})

	if analysis.Status != domain.AnalysisStatusUnsupported {
		t.Fatalf("status = %q, want %q", analysis.Status, domain.AnalysisStatusUnsupported)
	}
	if analysis.Metrics.LinesOfCode != 0 || len(analysis.Functions) != 0 || len(analysis.Classes) != 0 {
		t.Errorf("unsupported result = %+v, want no fabricated metrics", analysis)
	}
}

func analyzeFixture(t *testing.T, name string) domain.FileAnalysis {
	t.Helper()

	path := filepath.Join("testdata", name)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return NewAnalyzer().Analyze(domain.SourceFile{Path: path, Content: content})
}

func assertDeclaration(t *testing.T, declarations []domain.Declaration, name string, kind domain.DeclarationKind, length int) {
	t.Helper()
	for _, declaration := range declarations {
		if declaration.Name == name && declaration.Kind == kind {
			if declaration.Length != length {
				t.Errorf("%s length = %d, want %d", name, declaration.Length, length)
			}
			return
		}
	}
	t.Errorf("missing declaration %q with kind %q in %+v", name, kind, declarations)
}
