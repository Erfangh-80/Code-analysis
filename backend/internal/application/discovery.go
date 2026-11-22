package application

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Erfangh-80/code-analysis/backend/internal/analysisguard"
	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

// skippedDiscoveryDirectories are never entered during bounded discovery. The
// fixed list covers VCS metadata, dependency vendors, build output, and caches
// so a plain root path stays fast and honest without .gitignore processing.
var skippedDiscoveryDirectories = map[string]struct{}{
	".git": {}, ".hg": {}, ".svn": {},
	"node_modules": {}, "vendor": {},
	"dist": {}, "build": {}, "out": {}, "target": {}, "bin": {}, "obj": {},
	"coverage": {}, "__pycache__": {}, ".pytest_cache": {}, ".mypy_cache": {}, ".ruff_cache": {},
	".venv": {}, "venv": {}, "env": {},
	".idea": {}, ".vscode": {}, ".next": {}, ".nuxt": {}, ".turbo": {}, ".cache": {},
}

// discoverSourceFiles walks root without following directory symlinks and
// returns sorted slash-normalized relative paths of supported source files.
// File symlinks resolving outside root are skipped, and exceeding maxFiles
// stops the walk with a controlled resource-limit error instead of unbounded
// work. Unreadable entries are skipped here; explicitly requested files still
// surface their read errors through the normal per-file pipeline.
func discoverSourceFiles(root string, maxFiles int) ([]string, error) {
	files := make([]string, 0, 64)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			if path == root {
				return nil
			}
			if _, skipped := skippedDiscoveryDirectories[entry.Name()]; skipped {
				return filepath.SkipDir
			}
			return nil
		}
		if languageFromExtension(entry.Name()) == domain.LanguageUnknown {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(path)
			if err != nil {
				return nil
			}
			relative, err := filepath.Rel(root, target)
			if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
				return nil
			}
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		files = append(files, filepath.ToSlash(relative))
		if len(files) > maxFiles {
			return fmt.Errorf("%w: discovered source files exceed the configured limit", analysisguard.ErrResourceLimit)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}
