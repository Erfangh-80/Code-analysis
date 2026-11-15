// Package static implements local, AST-based source analysis. It never executes
// the repository source it receives.
package static

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

// Parser is the common language-parser boundary used by the static analyzer.
type Parser interface {
	Parse(source []byte) (*tree_sitter.Tree, error)
}

type treeSitterParser struct {
	language *tree_sitter.Language
}

func (parser treeSitterParser) Parse(source []byte) (*tree_sitter.Tree, error) {
	instance := tree_sitter.NewParser()
	defer instance.Close()

	if err := instance.SetLanguage(parser.language); err != nil {
		return nil, fmt.Errorf("configure tree-sitter language: %w", err)
	}
	return instance.Parse(source, nil), nil
}

// Analyzer calculates static file-level metrics using a parser selected from the file extension.
type Analyzer struct {
	parsers map[domain.Language]Parser
}

// NewAnalyzer builds the supported Go, JavaScript, and Python parser set.
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		parsers: map[domain.Language]Parser{
			domain.LanguageGo: treeSitterParser{
				language: tree_sitter.NewLanguage(tree_sitter_go.Language()),
			},
			domain.LanguageJavaScript: treeSitterParser{
				language: tree_sitter.NewLanguage(tree_sitter_javascript.Language()),
			},
			domain.LanguagePython: treeSitterParser{
				language: tree_sitter.NewLanguage(tree_sitter_python.Language()),
			},
		},
	}
}

// Analyze examines one source byte sequence without evaluating or importing it.
func (analyzer *Analyzer) Analyze(file domain.SourceFile) domain.FileAnalysis {
	language := languageForPath(file.Path)
	result := domain.FileAnalysis{
		Path:     filepath.ToSlash(file.Path),
		Language: language,
	}

	if language == domain.LanguageUnknown {
		result.Status = domain.AnalysisStatusUnsupported
		result.Diagnostics = []domain.Diagnostic{{
			Code:     domain.DiagnosticCodeUnsupported,
			Message:  "This file extension is not supported by the static analyzer.",
			Severity: "warning",
		}}
		return result
	}

	if len(bytes.TrimSpace(file.Content)) == 0 {
		result.Status = domain.AnalysisStatusEmpty
		return result
	}

	lineFlags := executableLineFlags(string(file.Content), language)
	parser, exists := analyzer.parsers[language]
	if !exists {
		result.Status = domain.AnalysisStatusUnsupported
		result.Diagnostics = []domain.Diagnostic{{
			Code:     domain.DiagnosticCodeUnsupported,
			Message:  "No parser is registered for this supported file extension.",
			Severity: "error",
		}}
		return result
	}

	tree, err := parser.Parse(file.Content)
	if err != nil || tree == nil {
		return analyzer.fallbackResult(result, lineFlags, file.Content, fmt.Errorf("parse source: %w", err))
	}
	defer tree.Close()

	root := tree.RootNode()
	if root == nil || root.HasError() {
		return analyzer.fallbackResult(result, lineFlags, file.Content, fmt.Errorf("tree-sitter reported a syntax error"))
	}

	result.Functions, result.Classes = extractDeclarations(root, file.Content, language, lineFlags)
	result.Metrics = metricsFor(lineFlags, result.Functions, result.Classes)
	result.Status = statusFor(result.Metrics)
	return result
}

func (analyzer *Analyzer) fallbackResult(result domain.FileAnalysis, lineFlags []bool, source []byte, parseErr error) domain.FileAnalysis {
	result.Functions, result.Classes = fallbackDeclarations(source, result.Language, lineFlags)
	result.Metrics = metricsFor(lineFlags, result.Functions, result.Classes)
	result.Diagnostics = []domain.Diagnostic{{
		Code:     domain.DiagnosticCodeParseError,
		Message:  "The language parser could not fully parse this file.",
		Severity: "error",
	}}

	if len(result.Functions) > 0 || len(result.Classes) > 0 {
		result.Status = domain.AnalysisStatusFallback
		result.UsedFallback = true
		result.Diagnostics = append(result.Diagnostics, domain.Diagnostic{
			Code:     domain.DiagnosticCodeFallbackUsed,
			Message:  "Fallback declaration extraction was used after a parser failure.",
			Severity: "warning",
		})
		return result
	}

	result.Status = domain.AnalysisStatusParseError
	if parseErr != nil {
		// Do not expose parser internals or source content through the API. The
		// stable diagnostic above is intentionally the only client-facing detail.
		_ = parseErr
	}
	return result
}

func languageForPath(path string) domain.Language {
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

func extractDeclarations(root *tree_sitter.Node, source []byte, language domain.Language, lineFlags []bool) ([]domain.Declaration, []domain.Declaration) {
	functions := make([]domain.Declaration, 0)
	classes := make([]domain.Declaration, 0)

	var walk func(node *tree_sitter.Node, insideClass bool)
	walk = func(node *tree_sitter.Node, insideClass bool) {
		if node == nil {
			return
		}

		kind := node.Kind()
		isClass := isClassNode(kind, language)
		if isClass {
			classes = append(classes, declarationFromNode(node, source, domain.DeclarationKindClass, lineFlags))
		}
		if isFunctionNode(kind, language) {
			declarationKind := domain.DeclarationKindFunction
			if insideClass || isMethodNode(kind, language) {
				declarationKind = domain.DeclarationKindMethod
			}
			functions = append(functions, declarationFromNode(node, source, declarationKind, lineFlags))
		}

		for index := uint(0); index < node.ChildCount(); index++ {
			walk(node.Child(index), insideClass || isClass)
		}
	}

	walk(root, false)
	return functions, classes
}

func isFunctionNode(kind string, language domain.Language) bool {
	switch language {
	case domain.LanguageGo:
		return kind == "function_declaration" || kind == "method_declaration"
	case domain.LanguageJavaScript:
		switch kind {
		case "function_declaration", "function_expression", "generator_function", "generator_function_declaration", "arrow_function", "method_definition":
			return true
		default:
			return false
		}
	case domain.LanguagePython:
		return kind == "function_definition"
	default:
		return false
	}
}

func isMethodNode(kind string, language domain.Language) bool {
	return (language == domain.LanguageGo && kind == "method_declaration") ||
		(language == domain.LanguageJavaScript && kind == "method_definition")
}

func isClassNode(kind string, language domain.Language) bool {
	return (language == domain.LanguageJavaScript && kind == "class_declaration") ||
		(language == domain.LanguagePython && kind == "class_definition")
}

func declarationFromNode(node *tree_sitter.Node, source []byte, kind domain.DeclarationKind, lineFlags []bool) domain.Declaration {
	startLine := int(node.StartPosition().Row) + 1
	endLine := int(node.EndPosition().Row) + 1
	if endLine < startLine {
		endLine = startLine
	}

	return domain.Declaration{
		Name:      declarationName(node, source),
		Kind:      kind,
		StartLine: startLine,
		EndLine:   endLine,
		Length:    countMarkedLines(lineFlags, startLine, endLine),
	}
}

func declarationName(node *tree_sitter.Node, source []byte) string {
	for _, field := range []string{"name", "property"} {
		if child := node.ChildByFieldName(field); child != nil {
			if name := strings.TrimSpace(child.Utf8Text(source)); name != "" {
				return name
			}
		}
	}

	for index := uint(0); index < node.NamedChildCount(); index++ {
		child := node.NamedChild(index)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "identifier", "property_identifier":
			if name := strings.TrimSpace(child.Utf8Text(source)); name != "" {
				return name
			}
		}
	}
	return "<anonymous>"
}

func statusFor(metrics domain.Metrics) domain.AnalysisStatus {
	if metrics.LinesOfCode == 0 || (metrics.FunctionCount == 0 && metrics.ClassCount == 0) {
		return domain.AnalysisStatusNoExecutable
	}
	return domain.AnalysisStatusAnalyzed
}

func metricsFor(lineFlags []bool, functions, classes []domain.Declaration) domain.Metrics {
	metrics := domain.Metrics{
		LinesOfCode:   countMarkedLines(lineFlags, 1, len(lineFlags)),
		FunctionCount: len(functions),
		ClassCount:    len(classes),
	}
	metrics.FunctionLength = lengthSummary(functions)
	metrics.ClassLength = lengthSummary(classes)
	return metrics
}

func lengthSummary(declarations []domain.Declaration) domain.LengthSummary {
	var summary domain.LengthSummary
	for _, declaration := range declarations {
		summary.Total += declaration.Length
		if declaration.Length > summary.Max {
			summary.Max = declaration.Length
		}
	}
	return summary
}
