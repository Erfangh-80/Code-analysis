package repositoryscan

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ignoreMatcher implements the Git-ignore features needed by the filesystem
// fallback: comments, negation, directory rules, rooted rules, and * / **
// globs. Git worktrees use Git itself for exact ignore semantics.
const (
	maxIgnoreFileBytes = 256 << 10
	maxIgnoreRules     = 10_000
)

type ignoreMatcher struct {
	root  string
	rules []ignoreRule
}

type ignoreRule struct {
	base          string
	pattern       string
	negated       bool
	directoryOnly bool
	anchored      bool
	hasSlash      bool
}

func newIgnoreMatcher(root string) (*ignoreMatcher, error) {
	matcher := &ignoreMatcher{root: root, rules: make([]ignoreRule, 0)}
	matcher.loadIgnoreFile(root, "")
	return matcher, nil
}

func (matcher *ignoreMatcher) loadIgnoreFile(directory, base string) {
	if len(matcher.rules) >= maxIgnoreRules {
		return
	}
	ignorePath := filepath.Join(directory, ".gitignore")
	resolved, err := filepath.EvalSymlinks(ignorePath)
	if err != nil {
		return
	}
	relative, err := filepath.Rel(matcher.root, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return
	}
	file, err := os.Open(resolved)
	if err != nil {
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxIgnoreFileBytes))
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(content), "\n") {
		if len(matcher.rules) >= maxIgnoreRules {
			return
		}
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rule := ignoreRule{base: strings.Trim(filepath.ToSlash(base), "/")}
		if strings.HasPrefix(line, "!") {
			rule.negated = true
			line = strings.TrimPrefix(line, "!")
		}
		rule.anchored = strings.HasPrefix(line, "/")
		line = strings.TrimPrefix(line, "/")
		rule.directoryOnly = strings.HasSuffix(line, "/")
		line = strings.TrimSuffix(line, "/")
		if line == "" {
			continue
		}
		rule.pattern = filepath.ToSlash(line)
		rule.hasSlash = strings.Contains(rule.pattern, "/")
		matcher.rules = append(matcher.rules, rule)
	}
}

func (matcher *ignoreMatcher) ignored(relative string, isDirectory bool) bool {
	relative = strings.Trim(filepath.ToSlash(relative), "/")
	ignored := false
	for _, rule := range matcher.rules {
		if rule.matches(relative, isDirectory) {
			ignored = !rule.negated
		}
	}
	return ignored
}

func (rule ignoreRule) matches(relative string, isDirectory bool) bool {
	candidate := relative
	if rule.base != "" {
		if candidate == rule.base {
			candidate = ""
		} else if strings.HasPrefix(candidate, rule.base+"/") {
			candidate = strings.TrimPrefix(candidate, rule.base+"/")
		} else {
			return false
		}
	}
	if candidate == "" {
		return false
	}

	segments := strings.Split(candidate, "/")
	if rule.directoryOnly {
		for index := range segments {
			if rule.matchesPath(strings.Join(segments[:index+1], "/")) {
				return true
			}
		}
		return false
	}
	if isDirectory && rule.matchesPath(candidate) {
		return true
	}
	return rule.matchesPath(candidate)
}

func (rule ignoreRule) matchesPath(candidate string) bool {
	if rule.anchored || rule.hasSlash {
		return globMatch(rule.pattern, candidate)
	}
	for _, component := range strings.Split(candidate, "/") {
		if globMatch(rule.pattern, component) {
			return true
		}
	}
	return false
}

func globMatch(pattern, candidate string) bool {
	if !strings.Contains(pattern, "**") {
		matched, err := path.Match(pattern, candidate)
		return err == nil && matched
	}
	return matchGlobRunes([]rune(pattern), []rune(candidate))
}

func matchGlobRunes(pattern, candidate []rune) bool {
	if len(pattern) == 0 {
		return len(candidate) == 0
	}
	switch pattern[0] {
	case '*':
		if len(pattern) > 1 && pattern[1] == '*' {
			next := pattern[2:]
			if len(next) > 0 && next[0] == '/' {
				next = next[1:]
			}
			for index := 0; index <= len(candidate); index++ {
				if matchGlobRunes(next, candidate[index:]) {
					return true
				}
			}
			return false
		}
		for index := 0; ; index++ {
			if matchGlobRunes(pattern[1:], candidate[index:]) {
				return true
			}
			if index == len(candidate) || candidate[index] == '/' {
				return false
			}
		}
	case '?':
		return len(candidate) > 0 && candidate[0] != '/' && matchGlobRunes(pattern[1:], candidate[1:])
	default:
		return len(candidate) > 0 && pattern[0] == candidate[0] && matchGlobRunes(pattern[1:], candidate[1:])
	}
}
