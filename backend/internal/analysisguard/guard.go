// Package analysisguard centralizes local-path and resource boundaries for
// analysis. It never reads project source or executes project code.
package analysisguard

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	// ErrInvalidLocalPath indicates a non-absolute, inaccessible, or non-directory input.
	ErrInvalidLocalPath = errors.New("invalid local path")
	// ErrInvalidAllowedRoot indicates an unusable configured analysis root.
	ErrInvalidAllowedRoot = errors.New("invalid allowed analysis root")
	// ErrPathOutsideAllowedRoots indicates an otherwise valid local path outside policy.
	ErrPathOutsideAllowedRoots = errors.New("path is outside the allowed analysis roots")
	// ErrInvalidLimits indicates an invalid resource-limit configuration.
	ErrInvalidLimits = errors.New("invalid analysis resource limits")
	// ErrResourceLimit indicates controlled work was stopped before exhausting host resources.
	ErrResourceLimit = errors.New("analysis resource limit exceeded")
)

const (
	defaultMaxFiles            = 10_000
	defaultMaxFileBytes  int64 = 2 << 20
	defaultMaxTotalBytes int64 = 128 << 20
	defaultMaxGitCommits       = 20_000
	defaultMaxGitOutput  int64 = 16 << 20
	defaultTimeout             = 10 * time.Minute
)

// Limits bounds CPU-adjacent work and source/output memory retained by one run.
// Zero values are replaced with conservative defaults by Normalize.
type Limits struct {
	MaxFiles            int
	MaxFileBytes        int64
	MaxTotalSourceBytes int64
	MaxGitCommits       int
	MaxGitOutputBytes   int64
	AnalysisTimeout     time.Duration
}

// DefaultLimits returns the limits used by the production server unless its
// operator supplies a stricter command-line value.
func DefaultLimits() Limits {
	return Limits{
		MaxFiles:            defaultMaxFiles,
		MaxFileBytes:        defaultMaxFileBytes,
		MaxTotalSourceBytes: defaultMaxTotalBytes,
		MaxGitCommits:       defaultMaxGitCommits,
		MaxGitOutputBytes:   defaultMaxGitOutput,
		AnalysisTimeout:     defaultTimeout,
	}
}

// Normalize applies defaults and rejects contradictory or negative limits.
func (limits Limits) Normalize() (Limits, error) {
	defaults := DefaultLimits()
	if limits.MaxFiles == 0 {
		limits.MaxFiles = defaults.MaxFiles
	}
	if limits.MaxFileBytes == 0 {
		limits.MaxFileBytes = defaults.MaxFileBytes
	}
	if limits.MaxTotalSourceBytes == 0 {
		limits.MaxTotalSourceBytes = defaults.MaxTotalSourceBytes
	}
	if limits.MaxGitCommits == 0 {
		limits.MaxGitCommits = defaults.MaxGitCommits
	}
	if limits.MaxGitOutputBytes == 0 {
		limits.MaxGitOutputBytes = defaults.MaxGitOutputBytes
	}
	if limits.AnalysisTimeout == 0 {
		limits.AnalysisTimeout = defaults.AnalysisTimeout
	}
	if limits.MaxFiles < 1 || limits.MaxFileBytes < 1 || limits.MaxTotalSourceBytes < limits.MaxFileBytes || limits.MaxGitCommits < 1 || limits.MaxGitOutputBytes < 1 || limits.AnalysisTimeout < time.Second {
		return Limits{}, fmt.Errorf("%w: values must be positive and max total source bytes must cover one file", ErrInvalidLimits)
	}
	return limits, nil
}

// PathPolicy confines analysis inputs to canonical local directories supplied
// by the server operator. A nil policy is intentionally unrestricted for
// backwards-compatible in-process tests; production code must use NewPathPolicy.
type PathPolicy struct {
	allowedRoots []string
}

// NewPathPolicy canonicalizes configured roots once at startup. An empty root
// list is rejected so production deployments cannot accidentally be unbounded.
func NewPathPolicy(roots []string) (*PathPolicy, error) {
	if len(roots) == 0 {
		return nil, fmt.Errorf("%w: at least one absolute directory is required", ErrInvalidAllowedRoot)
	}
	allowed := make([]string, 0, len(roots))
	seen := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		resolved, err := resolveDirectory(root)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidAllowedRoot, err)
		}
		if _, exists := seen[resolved]; exists {
			continue
		}
		seen[resolved] = struct{}{}
		allowed = append(allowed, resolved)
	}
	return &PathPolicy{allowedRoots: allowed}, nil
}

// ResolveDirectory returns a canonical directory only when it is inside one of
// the configured roots. Symlink targets are checked after canonicalization.
func (policy *PathPolicy) ResolveDirectory(path string) (string, error) {
	resolved, err := resolveDirectory(path)
	if err != nil {
		return "", err
	}
	if policy == nil {
		return resolved, nil
	}
	for _, root := range policy.allowedRoots {
		if isWithin(root, resolved) {
			return resolved, nil
		}
	}
	return "", fmt.Errorf("%w", ErrPathOutsideAllowedRoots)
}

// Allows reports whether a canonical or non-canonical local path is within a
// configured root. It is useful for derived Git roots and does not leak paths.
func (policy *PathPolicy) Allows(path string) bool {
	if policy == nil {
		return true
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	for _, root := range policy.allowedRoots {
		if isWithin(root, resolved) {
			return true
		}
	}
	return false
}

// Roots returns a copy of canonical roots for safe server configuration tests.
func (policy *PathPolicy) Roots() []string {
	if policy == nil {
		return nil
	}
	return append([]string(nil), policy.allowedRoots...)
}

func resolveDirectory(path string) (string, error) {
	if strings.TrimSpace(path) == "" || !filepath.IsAbs(path) {
		return "", fmt.Errorf("%w: an absolute path is required", ErrInvalidLocalPath)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("%w: path is inaccessible", ErrInvalidLocalPath)
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%w: path is not a directory", ErrInvalidLocalPath)
	}
	return resolved, nil
}

func isWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative))
}
