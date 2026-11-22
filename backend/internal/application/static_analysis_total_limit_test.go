package application_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/analysisguard"
	"github.com/Erfangh-80/code-analysis/backend/internal/analyzer/static"
	"github.com/Erfangh-80/code-analysis/backend/internal/application"
	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
	"github.com/Erfangh-80/code-analysis/backend/internal/storage/failedfiles"
)

func TestGuardedStaticAnalysisRetainsEarlierFilesWhenTotalSourceBudgetIsReached(t *testing.T) {
	root := t.TempDir()
	content := "package p\nfunc Value() int { return 1 }\n"
	writeFile(t, filepath.Join(root, "first.go"), content)
	writeFile(t, filepath.Join(root, "second.go"), content)
	policy, err := analysisguard.NewPathPolicy([]string{root})
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}
	limits := analysisguard.Limits{
		MaxFiles:            5,
		MaxFileBytes:        int64(len(content)),
		MaxTotalSourceBytes: int64(len(content) + 1),
		MaxGitCommits:       5,
		MaxGitOutputBytes:   1024,
		AnalysisTimeout:     time.Second,
	}
	service, err := application.NewStaticAnalysisServiceWithGuards(static.NewAnalyzer(), failedfiles.NewRegistry(), policy, limits)
	if err != nil {
		t.Fatalf("new guarded service: %v", err)
	}

	result, err := service.Analyze(context.Background(), application.StaticAnalysisRequest{RootPath: root, Files: []string{"first.go", "second.go"}})
	if err != nil {
		t.Fatalf("Analyze error = %v", err)
	}
	if len(result.Files) != 2 || result.Files[0].Status != domain.AnalysisStatusAnalyzed || result.Files[1].Status != domain.AnalysisStatusReadError {
		t.Fatalf("file results = %+v", result.Files)
	}
	if got := result.Files[1].Diagnostics; len(got) != 1 || got[0].Code != domain.DiagnosticCodeResourceLimit {
		t.Errorf("second file diagnostics = %+v, want resource-limit diagnostic", got)
	}
}
