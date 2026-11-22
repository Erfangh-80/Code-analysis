package githistory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/analysisguard"
	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
	"github.com/Erfangh-80/code-analysis/backend/internal/githistory"
)

func TestGuardedAnalyzerRejectsRepositoryOutsideAllowedRoots(t *testing.T) {
	repository := createHistoryFixture(t)
	policy, err := analysisguard.NewPathPolicy([]string{t.TempDir()})
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}
	analyzer, err := githistory.NewAnalyzerWithGuards(policy, analysisguard.DefaultLimits())
	if err != nil {
		t.Fatalf("new guarded analyzer: %v", err)
	}

	_, err = analyzer.Analyze(context.Background(), domain.GitAnalysisRequest{RepositoryPath: repository})
	if !errors.Is(err, githistory.ErrInvalidRepository) {
		t.Fatalf("Analyze error = %v, want ErrInvalidRepository", err)
	}
}

func TestGuardedAnalyzerStopsWhenCommitBudgetIsExceeded(t *testing.T) {
	repository := createHistoryFixture(t)
	policy, err := analysisguard.NewPathPolicy([]string{repository})
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}
	limits := analysisguard.Limits{
		MaxFiles:            10,
		MaxFileBytes:        1024,
		MaxTotalSourceBytes: 4096,
		MaxGitCommits:       2,
		MaxGitOutputBytes:   4096,
		AnalysisTimeout:     time.Second,
	}
	analyzer, err := githistory.NewAnalyzerWithGuards(policy, limits)
	if err != nil {
		t.Fatalf("new guarded analyzer: %v", err)
	}

	_, err = analyzer.Analyze(context.Background(), domain.GitAnalysisRequest{RepositoryPath: repository})
	if !errors.Is(err, githistory.ErrResourceLimit) {
		t.Fatalf("Analyze error = %v, want ErrResourceLimit", err)
	}
}
