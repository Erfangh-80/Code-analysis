package application

import (
	"context"
	"errors"
	"testing"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

func TestResultQueriesFilterSortPaginateAndCopyCompletedArtifacts(t *testing.T) {
	store := newResultViewStore()
	orchestrator := NewOrchestrator(store, nil, nil, nil, nil, 1)
	ctx := context.Background()

	summary, err := orchestrator.Summary(ctx, store.run.ID)
	if err != nil {
		t.Fatalf("get summary: %v", err)
	}
	if summary.Summary.ScoredFiles != 3 || summary.ModuleCount != 2 || summary.RiskLevels != (domain.RiskLevelCounts{Low: 1, Medium: 1, High: 1}) || !summary.HasGitArtifact {
		t.Errorf("summary = %+v", summary)
	}
	summary.Analysis.Diagnostics[0].Message = "mutated"
	if store.report.Analysis.Diagnostics[0].Message == "mutated" {
		t.Error("summary returned a mutable alias of the persisted analysis run")
	}

	files, err := orchestrator.Files(ctx, store.run.ID, domain.ResultListQuery{
		Module:    "cmd",
		Sort:      domain.ResultSortScore,
		Direction: domain.SortDirectionDescending,
		Page:      1,
		PageSize:  1,
	})
	if err != nil {
		t.Fatalf("get filtered files: %v", err)
	}
	if files.TotalItems != 2 || len(files.Items) != 1 || files.Items[0].Analysis.Path != "cmd/beta.go" {
		t.Fatalf("filtered files = %+v", files)
	}
	if files.Items[0].Git == nil || files.Items[0].Risk == nil {
		t.Fatalf("joined file = %+v, want Git and risk data", files.Items[0])
	}
	minimum, maximum := 80, 95
	filteredByLanguageAndScore, err := orchestrator.Files(ctx, store.run.ID, domain.ResultListQuery{
		Languages: []domain.Language{domain.LanguageGo},
		MinScore:  &minimum,
		MaxScore:  &maximum,
		Page:      1,
		PageSize:  50,
	})
	if err != nil {
		t.Fatalf("filter files by language and score: %v", err)
	}
	if filteredByLanguageAndScore.TotalItems != 1 || len(filteredByLanguageAndScore.Items) != 1 || filteredByLanguageAndScore.Items[0].Analysis.Path != "cmd/beta.go" {
		t.Errorf("language/score filtered files = %+v", filteredByLanguageAndScore)
	}
	files.Items[0].Analysis.Functions[0].Name = "mutated"
	files.Items[0].Git.PreviousPaths[0] = "mutated"
	files.Items[0].Risk.Contributions[0].Weight = 0

	again, err := orchestrator.Files(ctx, store.run.ID, domain.ResultListQuery{Query: "beta", Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("get copied file again: %v", err)
	}
	if len(again.Items) != 1 || again.Items[0].Analysis.Functions[0].Name != "Beta" || again.Items[0].Git.PreviousPaths[0] != "old/beta.go" || again.Items[0].Risk.Contributions[0].Weight != 1 {
		t.Errorf("result view exposed mutable persisted artifact data: %+v", again.Items)
	}

	outOfRange, err := orchestrator.Files(ctx, store.run.ID, domain.ResultListQuery{Page: 3, PageSize: 2})
	if err != nil {
		t.Fatalf("get out-of-range page: %v", err)
	}
	if outOfRange.TotalItems != 3 || outOfRange.Page != 3 || len(outOfRange.Items) != 0 {
		t.Errorf("out-of-range page = %+v", outOfRange)
	}

	hotspots, err := orchestrator.Hotspots(ctx, store.run.ID, domain.ResultListQuery{
		Sort:      domain.ResultSortScore,
		Direction: domain.SortDirectionDescending,
		Page:      1,
		PageSize:  2,
	})
	if err != nil {
		t.Fatalf("get hotspots: %v", err)
	}
	if len(hotspots.Items) != 2 || hotspots.Items[0].Path != "cmd/beta.go" || hotspots.Items[1].Path != "cmd/alpha.go" {
		t.Errorf("score-sorted hotspots = %+v", hotspots.Items)
	}
	hotspots.Items[0].Contributions[0].Weight = 0
	if store.report.Artifacts.Risk.Files[0].Contributions[0].Weight != 1 {
		t.Error("hotspot page returned a mutable alias of the persisted artifact")
	}
}

func TestResultQueriesReturnSpecificErrorsForMissingArtifactsAndFiles(t *testing.T) {
	store := newResultViewStore()
	orchestrator := NewOrchestrator(store, nil, nil, nil, nil, 1)

	if _, err := orchestrator.File(context.Background(), store.run.ID, "missing.go"); !errors.Is(err, ErrAnalysisFileNotFound) {
		t.Errorf("missing file error = %v, want ErrAnalysisFileNotFound", err)
	}

	store.report.Artifacts.Risk = nil
	if _, err := orchestrator.Hotspots(context.Background(), store.run.ID, domain.ResultListQuery{}); !errors.Is(err, ErrResultUnavailable) {
		t.Errorf("hotspots without risk artifact error = %v, want ErrResultUnavailable", err)
	}
	if _, err := orchestrator.Heatmap(context.Background(), store.run.ID); !errors.Is(err, ErrResultUnavailable) {
		t.Errorf("heatmap without risk artifact error = %v, want ErrResultUnavailable", err)
	}

	store.report.Artifacts.Static = nil
	if _, err := orchestrator.Files(context.Background(), store.run.ID, domain.ResultListQuery{}); !errors.Is(err, ErrResultUnavailable) {
		t.Errorf("files without static artifact error = %v, want ErrResultUnavailable", err)
	}
	if _, err := orchestrator.Modules(context.Background(), store.run.ID); !errors.Is(err, ErrResultUnavailable) {
		t.Errorf("modules without static artifact error = %v, want ErrResultUnavailable", err)
	}
}

type resultViewStore struct {
	run    domain.AnalysisRun
	report domain.AnalysisReport
}

func newResultViewStore() *resultViewStore {
	report := resultViewFixture()
	return &resultViewStore{run: report.Analysis, report: report}
}

func (store *resultViewStore) CreateQueued(context.Context, domain.AnalysisRun) error { return nil }
func (store *resultViewStore) SaveRun(context.Context, domain.AnalysisRun) error      { return nil }
func (store *resultViewStore) SaveReport(context.Context, domain.AnalysisID, domain.AnalysisReport) error {
	return nil
}
func (store *resultViewStore) RecoverInterruptedRuns(context.Context) error { return nil }
func (store *resultViewStore) List(context.Context, domain.AnalysisListQuery) (domain.AnalysisPage, error) {
	return domain.AnalysisPage{}, nil
}
func (store *resultViewStore) Get(_ context.Context, id domain.AnalysisID) (domain.AnalysisRun, error) {
	if id != store.run.ID {
		return domain.AnalysisRun{}, ErrAnalysisNotFound
	}
	return store.run, nil
}
func (store *resultViewStore) GetReport(_ context.Context, id domain.AnalysisID) (domain.AnalysisReport, error) {
	if id != store.run.ID {
		return domain.AnalysisReport{}, ErrAnalysisNotFound
	}
	return store.report, nil
}

func resultViewFixture() domain.AnalysisReport {
	run := domain.AnalysisRun{
		ID:     "analysis_result_view",
		Status: domain.AnalysisStatusCompleted,
		Summary: &domain.AnalysisSummary{
			FilesDiscovered: 3,
			FilesAnalyzed:   3,
			GitStatus:       domain.StageStatusCompleted,
			RiskStatus:      domain.StageStatusCompleted,
			ScoredFiles:     3,
		},
		Diagnostics: []domain.AnalysisDiagnostic{{Code: "FIXTURE", Message: "fixture diagnostic"}},
	}
	return domain.AnalysisReport{
		Analysis: run,
		Artifacts: domain.AnalysisArtifacts{
			Static: &domain.BatchAnalysis{
				Files: []domain.FileAnalysis{
					fixtureFileWithLanguage("cmd/alpha.go", "Alpha", 10, domain.LanguageJavaScript),
					fixtureFile("cmd/beta.go", "Beta", 20),
					fixtureFile("root.go", "Root", 5),
				},
				Modules: []domain.ModuleAnalysis{{Path: "cmd", FileCount: 2}, {Path: ".", FileCount: 1}},
			},
			Git: &domain.GitAnalysisReport{Files: []domain.GitFileMetrics{
				{Path: "cmd/alpha.go", PreviousPaths: []string{"old/alpha.go"}, GitChurn: 3, ChangeFrequency: 1},
				{Path: "cmd/beta.go", PreviousPaths: []string{"old/beta.go"}, GitChurn: 6, ChangeFrequency: 2},
				{Path: "root.go", PreviousPaths: []string{}, GitChurn: 1, ChangeFrequency: 1},
			}},
			Risk: &domain.RiskAnalysisReport{Files: []domain.Hotspot{
				fixtureHotspot("cmd/beta.go", "cmd", 90, domain.RiskLevelHigh),
				fixtureHotspot("cmd/alpha.go", "cmd", 60, domain.RiskLevelMedium),
				fixtureHotspot("root.go", ".", 20, domain.RiskLevelLow),
			}, Heatmap: domain.HeatmapData{XMetric: domain.RiskMetricGitChurn, YMetric: domain.RiskMetricLinesOfCode}},
			RiskCoverage: &domain.RiskCoverage{
				AvailableMetrics: []domain.RiskMetric{domain.RiskMetricLinesOfCode, domain.RiskMetricGitChurn},
				RequestedProfile: domain.RiskProfile{Weights: map[domain.RiskMetric]float64{domain.RiskMetricLinesOfCode: 1}},
				EffectiveProfile: domain.RiskProfile{Weights: map[domain.RiskMetric]float64{domain.RiskMetricLinesOfCode: 1}},
			},
		},
	}
}

func fixtureFile(path, name string, lines int) domain.FileAnalysis {
	return fixtureFileWithLanguage(path, name, lines, domain.LanguageGo)
}

func fixtureFileWithLanguage(path, name string, lines int, language domain.Language) domain.FileAnalysis {
	return domain.FileAnalysis{
		Path:      path,
		Language:  language,
		Status:    domain.AnalysisStatusAnalyzed,
		Metrics:   domain.Metrics{LinesOfCode: lines},
		Functions: []domain.Declaration{{Name: name, Kind: domain.DeclarationKindFunction, StartLine: 1, EndLine: 2, Length: 2}},
	}
}

func fixtureHotspot(path, module string, score int, level domain.RiskLevel) domain.Hotspot {
	return domain.Hotspot{
		Path:          path,
		Module:        module,
		Score:         score,
		Level:         level,
		Contributions: []domain.RiskContribution{{Metric: domain.RiskMetricLinesOfCode, Weight: 1}},
	}
}
