package analysisguard_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/analysisguard"
)

func TestPathPolicyCanonicalizesRootsAndRejectsPathsOutsideThem(t *testing.T) {
	allowedRoot := t.TempDir()
	outsideRoot := t.TempDir()
	inside := filepath.Join(allowedRoot, "project")
	if err := os.MkdirAll(inside, 0o700); err != nil {
		t.Fatalf("create project: %v", err)
	}

	policy, err := analysisguard.NewPathPolicy([]string{allowedRoot})
	if err != nil {
		t.Fatalf("new path policy: %v", err)
	}
	resolved, err := policy.ResolveDirectory(inside)
	if err != nil {
		t.Fatalf("resolve allowed directory: %v", err)
	}
	if resolved != inside {
		t.Errorf("resolved path = %q, want %q", resolved, inside)
	}
	if _, err := policy.ResolveDirectory(outsideRoot); !errors.Is(err, analysisguard.ErrPathOutsideAllowedRoots) {
		t.Fatalf("outside root error = %v, want ErrPathOutsideAllowedRoots", err)
	}
}

func TestPathPolicyRejectsSymlinkThatEscapesAllowedRoot(t *testing.T) {
	allowedRoot := t.TempDir()
	outsideRoot := t.TempDir()
	link := filepath.Join(allowedRoot, "escape")
	if err := os.Symlink(outsideRoot, link); err != nil {
		t.Skipf("symbolic links are unavailable on this platform: %v", err)
	}

	policy, err := analysisguard.NewPathPolicy([]string{allowedRoot})
	if err != nil {
		t.Fatalf("new path policy: %v", err)
	}
	if _, err := policy.ResolveDirectory(link); !errors.Is(err, analysisguard.ErrPathOutsideAllowedRoots) {
		t.Fatalf("escaped symlink error = %v, want ErrPathOutsideAllowedRoots", err)
	}
}

func TestLimitsNormalizeDefaultsAndRejectInvalidValues(t *testing.T) {
	limits, err := (analysisguard.Limits{MaxFiles: 7, AnalysisTimeout: 2 * time.Second}).Normalize()
	if err != nil {
		t.Fatalf("normalize limits: %v", err)
	}
	if limits.MaxFiles != 7 || limits.MaxFileBytes <= 0 || limits.MaxTotalSourceBytes <= 0 || limits.MaxGitCommits <= 0 || limits.MaxGitOutputBytes <= 0 || limits.AnalysisTimeout != 2*time.Second {
		t.Errorf("normalized limits = %+v", limits)
	}

	if _, err := (analysisguard.Limits{MaxFileBytes: -1}).Normalize(); !errors.Is(err, analysisguard.ErrInvalidLimits) {
		t.Fatalf("invalid limit error = %v, want ErrInvalidLimits", err)
	}
}
