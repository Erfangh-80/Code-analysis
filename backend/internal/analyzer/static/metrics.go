package static

import (
	"regexp"
	"strings"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

var (
	goFunctionPattern = regexp.MustCompile(`^\s*func\s*(\([^)]*\)\s*)?([A-Za-z_][A-Za-z0-9_]*)`)
	jsFunctionPattern = regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s*\*?\s*([A-Za-z_$][A-Za-z0-9_$]*)`)
	jsClassPattern    = regexp.MustCompile(`^\s*(?:export\s+)?class\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	pythonFunctionPattern = regexp.MustCompile(`^\s*(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)`)
	pythonClassPattern    = regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)`)
)

// executableLineFlags determines physical LOC without blank-only or comment-only lines.
// It is deliberately lexical and never evaluates source text.
func executableLineFlags(source string, language domain.Language) []bool {
	lines := strings.Split(source, "\n")
	flags := make([]bool, len(lines))
	inBlockComment := false

	for index, line := range lines {
		candidate := line
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}

		if language == domain.LanguagePython {
			flags[index] = !strings.HasPrefix(trimmed, "#")
			continue
		}

		candidate = stripCStyleComments(candidate, &inBlockComment)
		if strings.TrimSpace(candidate) != "" {
			flags[index] = true
		}
	}
	return flags
}

func stripCStyleComments(line string, inBlockComment *bool) string {
	var builder strings.Builder
	for position := 0; position < len(line); {
		if *inBlockComment {
			end := strings.Index(line[position:], "*/")
			if end < 0 {
				return builder.String()
			}
			position += end + 2
			*inBlockComment = false
			continue
		}
		if strings.HasPrefix(line[position:], "//") {
			break
		}
		if strings.HasPrefix(line[position:], "/*") {
			*inBlockComment = true
			position += 2
			continue
		}
		builder.WriteByte(line[position])
		position++
	}
	return builder.String()
}

func countMarkedLines(flags []bool, startLine, endLine int) int {
	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(flags) {
		endLine = len(flags)
	}
	if startLine > endLine || len(flags) == 0 {
		return 0
	}

	count := 0
	for index := startLine - 1; index < endLine; index++ {
		if flags[index] {
			count++
		}
	}
	return count
}

func fallbackDeclarations(source []byte, language domain.Language, lineFlags []bool) ([]domain.Declaration, []domain.Declaration) {
	lines := strings.Split(string(source), "\n")
	functions := make([]domain.Declaration, 0)
	classes := make([]domain.Declaration, 0)
	classIndent := -1

	for index, line := range lines {
		lineNumber := index + 1
		indentation := len(line) - len(strings.TrimLeft(line, " \t"))
		trimmed := strings.TrimSpace(line)

		if language == domain.LanguagePython && classIndent >= 0 && trimmed != "" && indentation <= classIndent && !strings.HasPrefix(trimmed, "#") {
			classIndent = -1
		}

		switch language {
		case domain.LanguageGo:
			if matches := goFunctionPattern.FindStringSubmatch(line); len(matches) == 3 {
				kind := domain.DeclarationKindFunction
				if strings.TrimSpace(matches[1]) != "" {
					kind = domain.DeclarationKindMethod
				}
				functions = append(functions, fallbackDeclaration(matches[2], kind, lineNumber, lines, lineFlags, language, indentation))
			}
		case domain.LanguageJavaScript:
			if matches := jsClassPattern.FindStringSubmatch(line); len(matches) == 2 {
				classes = append(classes, fallbackDeclaration(matches[1], domain.DeclarationKindClass, lineNumber, lines, lineFlags, language, indentation))
			}
			if matches := jsFunctionPattern.FindStringSubmatch(line); len(matches) == 2 {
				functions = append(functions, fallbackDeclaration(matches[1], domain.DeclarationKindFunction, lineNumber, lines, lineFlags, language, indentation))
			}
		case domain.LanguagePython:
			if matches := pythonClassPattern.FindStringSubmatch(line); len(matches) == 2 {
				classes = append(classes, fallbackDeclaration(matches[1], domain.DeclarationKindClass, lineNumber, lines, lineFlags, language, indentation))
				classIndent = indentation
			}
			if matches := pythonFunctionPattern.FindStringSubmatch(line); len(matches) == 2 {
				kind := domain.DeclarationKindFunction
				if classIndent >= 0 && indentation > classIndent {
					kind = domain.DeclarationKindMethod
				}
				functions = append(functions, fallbackDeclaration(matches[1], kind, lineNumber, lines, lineFlags, language, indentation))
			}
		}
	}
	return functions, classes
}

func fallbackDeclaration(name string, kind domain.DeclarationKind, startLine int, lines []string, lineFlags []bool, language domain.Language, indentation int) domain.Declaration {
	endLine := fallbackEndLine(startLine, lines, language, indentation)
	return domain.Declaration{
		Name:      name,
		Kind:      kind,
		StartLine: startLine,
		EndLine:   endLine,
		Length:    countMarkedLines(lineFlags, startLine, endLine),
	}
}

func fallbackEndLine(startLine int, lines []string, language domain.Language, indentation int) int {
	if language == domain.LanguagePython {
		for index := startLine; index < len(lines); index++ {
			trimmed := strings.TrimSpace(lines[index])
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			currentIndentation := len(lines[index]) - len(strings.TrimLeft(lines[index], " \t"))
			if currentIndentation <= indentation {
				return index
			}
		}
		return len(lines)
	}

	depth := 0
	seenBrace := false
	for index := startLine - 1; index < len(lines); index++ {
		line := stripQuotedSections(lines[index])
		for _, character := range line {
			switch character {
			case '{':
				depth++
				seenBrace = true
			case '}':
				depth--
				if seenBrace && depth <= 0 {
					return index + 1
				}
			}
		}
	}
	return len(lines)
}

func stripQuotedSections(line string) string {
	var builder strings.Builder
	quote := rune(0)
	escaped := false
	for _, character := range line {
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if character == '\\' {
				escaped = true
				continue
			}
			if character == quote {
				quote = 0
			}
			continue
		}
		if character == '\'' || character == '"' || character == '`' {
			quote = character
			continue
		}
		builder.WriteRune(character)
	}
	return builder.String()
}
