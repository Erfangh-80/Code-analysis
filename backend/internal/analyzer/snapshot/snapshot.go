// Package snapshot reads only tracked blobs from a selected Git commit.
package snapshot

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path"
	"strings"

	static "github.com/Erfangh-80/Code-analysis/backend/internal/analyzer/static"
)

type File struct {
	Path, Language string
	Size           int64
}

var ignoredDirectories = map[string]struct{}{".git": {}, "node_modules": {}, "vendor": {}, "dist": {}, "build": {}, ".next": {}, "__pycache__": {}}

func List(ctx context.Context, repository, commit string, languages map[string]bool) ([]File, error) {
	output, err := git(ctx, repository, "ls-tree", "-r", "-z", "--full-tree", commit)
	if err != nil {
		return nil, fmt.Errorf("unable to list snapshot files")
	}
	files := []File{}
	for _, record := range bytes.Split(output, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		parts := bytes.SplitN(record, []byte{'\t'}, 2)
		if len(parts) != 2 {
			continue
		}
		metadata := strings.Fields(string(parts[0]))
		if len(metadata) < 3 || metadata[1] != "blob" {
			continue
		}
		filePath := string(parts[1])
		language := static.LanguageForPath(filePath)
		if language == "" || !languages[language] || ignored(filePath) {
			continue
		}
		files = append(files, File{Path: filePath, Language: language})
	}
	return files, nil
}
func Read(ctx context.Context, repository, commit, filePath string) ([]byte, error) {
	// Arguments are passed directly; no shell executes repository-derived text.
	return git(ctx, repository, "show", commit+":"+filePath)
}
func ignored(filePath string) bool {
	for _, segment := range strings.Split(path.Clean(filePath), "/") {
		if _, found := ignoredDirectories[segment]; found {
			return true
		}
	}
	return false
}
func git(ctx context.Context, repository string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repository}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return out, nil
}
