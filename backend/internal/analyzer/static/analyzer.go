// Package static provides Tree-sitter based read-only source metrics.
package static

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Erfangh-80/Code-analysis/backend/internal/domain"
	sitter "github.com/smacker/go-tree-sitter"
	golang "github.com/smacker/go-tree-sitter/golang"
	javascript "github.com/smacker/go-tree-sitter/javascript"
	python "github.com/smacker/go-tree-sitter/python"
)

// MaxFileBytes is deliberately configurable in tests and through config at the
// orchestration edge. Source is never executed.
var MaxFileBytes int64 = 2 * 1024 * 1024

func LanguageForPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".mjs", ".cjs", ".jsx":
		return "javascript"
	default:
		return ""
	}
}

func Analyze(path string, source []byte) domain.StaticMetrics {
	language := LanguageForPath(path)
	result := domain.StaticMetrics{Language: language, AnalysisStatus: "ok"}
	if language == "" {
		return skipped(result, "unsupported file extension")
	}
	if int64(len(source)) > MaxFileBytes {
		return skipped(result, fmt.Sprintf("file exceeds configured limit of %d bytes", MaxFileBytes))
	}
	if bytes.IndexByte(source, 0) >= 0 {
		return skipped(result, "binary or NUL-containing file")
	}
	if !utf8.Valid(source) {
		return skipped(result, "invalid UTF-8 encoding")
	}
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(languageGrammar(language))
	tree := parser.Parse(nil, source)
	defer tree.Close()
	root := tree.RootNode()
	if root.HasError() {
		message := "syntax parse error; derived metrics are unavailable"
		result.AnalysisStatus, result.AnalysisError = "parse_error", &message
		return result
	}
	total := physicalLines(source)
	loc := countLOC(root, source)
	functions := collectFunctions(root, source, language)
	classes := collectClasses(root, source, language)
	max := 0
	for _, fn := range functions {
		if fn.Complexity > max {
			max = fn.Complexity
		}
	}
	functionCount := len(functions)
	result.TotalLines, result.LOC, result.FunctionCount = intp(total), intp(loc), intp(functionCount)
	if functionCount > 0 {
		result.MaxFunctionComplexity = intp(max)
	} else {
		result.MaxFunctionComplexity = intp(0)
	}
	// nil has an intentional semantic for Go: it has no class construct.
	if language != "go" {
		result.Classes = classes
	}
	result.Functions = functions
	return result
}

func languageGrammar(language string) *sitter.Language {
	switch language {
	case "go":
		return golang.GetLanguage()
	case "python":
		return python.GetLanguage()
	default:
		return javascript.GetLanguage()
	}
}

func skipped(result domain.StaticMetrics, why string) domain.StaticMetrics {
	result.AnalysisStatus, result.AnalysisError = "skipped", &why
	return result
}
func intp(n int) *int { return &n }

func physicalLines(source []byte) int {
	if len(source) == 0 {
		return 0
	}
	count := bytes.Count(source, []byte{'\n'})
	if source[len(source)-1] != '\n' {
		count++
	}
	return count
}

type interval struct{ start, end int }

func countLOC(root *sitter.Node, source []byte) int {
	comments := []interval{}
	walk(root, func(n *sitter.Node) bool {
		if n.Type() == "comment" {
			comments = append(comments, interval{int(n.StartByte()), int(n.EndByte())})
		}
		return true
	})
	sort.Slice(comments, func(i, j int) bool { return comments[i].start < comments[j].start })
	lines, start, ci := 0, 0, 0
	for start < len(source) {
		end := start
		for end < len(source) && source[end] != '\n' {
			end++
		}
		for ci < len(comments) && comments[ci].end <= start {
			ci++
		}
		hasCode := false
		for pos := start; pos < end; pos++ {
			for ci < len(comments) && comments[ci].end <= pos {
				ci++
			}
			inComment := ci < len(comments) && comments[ci].start <= pos && pos < comments[ci].end
			if !inComment && source[pos] != ' ' && source[pos] != '\t' && source[pos] != '\r' {
				hasCode = true
				break
			}
		}
		if hasCode {
			lines++
		}
		if end == len(source) {
			break
		}
		start = end + 1
	}
	return lines
}

func collectFunctions(root *sitter.Node, source []byte, language string) []domain.FunctionMetric {
	functions := []domain.FunctionMetric{}
	walk(root, func(n *sitter.Node) bool {
		if !isFunction(n.Type(), language) {
			return true
		}
		name := functionName(n, source, language)
		start, end := int(n.StartPoint().Row)+1, int(n.EndPoint().Row)+1
		functions = append(functions, domain.FunctionMetric{Name: name, StartLine: start, EndLine: end, Length: end - start + 1, Complexity: complexity(n, source, language)})
		// Descendant functions become their own metrics too. complexity() itself
		// stops at nested function boundaries so they cannot inflate the parent.
		return true
	})
	return functions
}

func collectClasses(root *sitter.Node, source []byte, language string) []domain.ClassMetric {
	classes := []domain.ClassMetric{}
	walk(root, func(n *sitter.Node) bool {
		if (language == "python" && n.Type() == "class_definition") || (language == "javascript" && n.Type() == "class_declaration") {
			start, end := int(n.StartPoint().Row)+1, int(n.EndPoint().Row)+1
			name := childContent(n.ChildByFieldName("name"), source)
			classes = append(classes, domain.ClassMetric{Name: name, StartLine: start, EndLine: end, Length: end - start + 1})
		}
		return true
	})
	return classes
}

func isFunction(kind, language string) bool {
	switch language {
	case "go":
		return kind == "function_declaration" || kind == "method_declaration"
	case "python":
		return kind == "function_definition"
	default:
		switch kind {
		case "function_declaration", "function_expression", "arrow_function", "method_definition", "generator_function_declaration":
			return true
		}
	}
	return false
}

func functionName(n *sitter.Node, source []byte, language string) string {
	if name := childContent(n.ChildByFieldName("name"), source); name != "" {
		return name
	}
	if language == "javascript" && n.Type() == "method_definition" {
		return childContent(n.ChildByFieldName("name"), source)
	}
	return "<anonymous>"
}
func childContent(n *sitter.Node, source []byte) string {
	if n == nil {
		return ""
	}
	return n.Content(source)
}

func complexity(fn *sitter.Node, source []byte, language string) int {
	value := 1
	walkChildren(fn, func(n *sitter.Node) bool {
		if n != fn && isFunction(n.Type(), language) {
			return false
		}
		if decision(n, source, language) {
			value++
		}
		return true
	})
	return value
}

func decision(n *sitter.Node, source []byte, language string) bool {
	t := n.Type()
	switch language {
	case "go":
		if t == "if_statement" || t == "for_statement" || t == "expression_switch_case" || t == "select_case" || t == "communication_case" {
			return true
		}
		return t == "binary_expression" && (strings.Contains(n.Content(source), "&&") || strings.Contains(n.Content(source), "||"))
	case "python":
		if t == "if_statement" || t == "for_statement" || t == "while_statement" || t == "conditional_expression" || t == "except_clause" {
			return true
		}
		return t == "boolean_operator" && (strings.Contains(n.Content(source), " and ") || strings.Contains(n.Content(source), " or "))
	default:
		if t == "if_statement" || t == "for_statement" || t == "for_in_statement" || t == "while_statement" || t == "do_statement" || t == "switch_case" || t == "catch_clause" || t == "ternary_expression" {
			return true
		}
		return t == "binary_expression" && (strings.Contains(n.Content(source), "&&") || strings.Contains(n.Content(source), "||"))
	}
}

func walk(n *sitter.Node, fn func(*sitter.Node) bool) {
	if !fn(n) {
		return
	}
	for i := 0; i < int(n.ChildCount()); i++ {
		walk(n.Child(i), fn)
	}
}
func walkChildren(n *sitter.Node, fn func(*sitter.Node) bool) {
	for i := 0; i < int(n.ChildCount()); i++ {
		walk(n.Child(i), fn)
	}
}
