package application

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

// Summary returns a compact, completed-report overview without requiring a
// client to derive counts or availability from raw artifacts.
func (orchestrator *Orchestrator) Summary(ctx context.Context, id domain.AnalysisID) (domain.AnalysisResultSummary, error) {
	report, err := orchestrator.Report(ctx, id)
	if err != nil {
		return domain.AnalysisResultSummary{}, err
	}
	if report.Analysis.Summary == nil {
		return domain.AnalysisResultSummary{}, ErrResultUnavailable
	}

	result := domain.AnalysisResultSummary{
		Analysis:       cloneAnalysisRun(report.Analysis),
		Summary:        *report.Analysis.Summary,
		RiskCoverage:   cloneRiskCoverage(report.Artifacts.RiskCoverage),
		HasGitArtifact: report.Artifacts.Git != nil,
	}
	if report.Artifacts.Static != nil {
		result.ModuleCount = len(report.Artifacts.Static.Modules)
	}
	if report.Artifacts.Risk != nil {
		for _, hotspot := range report.Artifacts.Risk.Files {
			switch hotspot.Level {
			case domain.RiskLevelLow:
				result.RiskLevels.Low++
			case domain.RiskLevelMedium:
				result.RiskLevels.Medium++
			case domain.RiskLevelHigh:
				result.RiskLevels.High++
			}
		}
	}
	return result, nil
}

// Hotspots returns the completed run's scored files with deterministic
// filtering, ordering, and pagination.
func (orchestrator *Orchestrator) Hotspots(ctx context.Context, id domain.AnalysisID, query domain.ResultListQuery) (domain.AnalysisHotspotPage, error) {
	report, err := orchestrator.Report(ctx, id)
	if err != nil {
		return domain.AnalysisHotspotPage{}, err
	}
	if report.Artifacts.Risk == nil {
		return domain.AnalysisHotspotPage{}, ErrResultUnavailable
	}

	items := make([]domain.Hotspot, 0, len(report.Artifacts.Risk.Files))
	for _, hotspot := range report.Artifacts.Risk.Files {
		if matchesHotspot(hotspot, query) {
			items = append(items, cloneHotspot(hotspot))
		}
	}
	normalizeResultQuery(&query, domain.ResultSortScore)
	sortHotspots(items, query)
	pageItems, page, pageSize := pageHotspots(items, query.Page, query.PageSize)
	return domain.AnalysisHotspotPage{
		Items:        pageItems,
		Page:         page,
		PageSize:     pageSize,
		TotalItems:   len(items),
		RiskCoverage: cloneRiskCoverage(report.Artifacts.RiskCoverage),
	}, nil
}

// Files returns static-file artifacts joined with optional Git and risk facts.
func (orchestrator *Orchestrator) Files(ctx context.Context, id domain.AnalysisID, query domain.ResultListQuery) (domain.AnalysisFilePage, error) {
	report, err := orchestrator.Report(ctx, id)
	if err != nil {
		return domain.AnalysisFilePage{}, err
	}
	if report.Artifacts.Static == nil {
		return domain.AnalysisFilePage{}, ErrResultUnavailable
	}

	items := buildFileResults(report)
	filtered := make([]domain.AnalysisFileResult, 0, len(items))
	for _, item := range items {
		if matchesFile(item, query) {
			filtered = append(filtered, item)
		}
	}
	normalizeResultQuery(&query, domain.ResultSortPath)
	sortFiles(filtered, query)
	pageItems, page, pageSize := pageFiles(filtered, query.Page, query.PageSize)
	return domain.AnalysisFilePage{
		Items:        pageItems,
		Page:         page,
		PageSize:     pageSize,
		TotalItems:   len(filtered),
		RiskCoverage: cloneRiskCoverage(report.Artifacts.RiskCoverage),
	}, nil
}

// File returns one project-relative file result from a completed report.
func (orchestrator *Orchestrator) File(ctx context.Context, id domain.AnalysisID, path string) (domain.AnalysisFileResult, error) {
	report, err := orchestrator.Report(ctx, id)
	if err != nil {
		return domain.AnalysisFileResult{}, err
	}
	if report.Artifacts.Static == nil {
		return domain.AnalysisFileResult{}, ErrResultUnavailable
	}
	for _, item := range buildFileResults(report) {
		if item.Analysis.Path == path {
			return item, nil
		}
	}
	return domain.AnalysisFileResult{}, ErrAnalysisFileNotFound
}

// Modules returns static directory aggregates and optional risk-ranked module
// scores from the same immutable completed report.
func (orchestrator *Orchestrator) Modules(ctx context.Context, id domain.AnalysisID) (domain.AnalysisModules, error) {
	report, err := orchestrator.Report(ctx, id)
	if err != nil {
		return domain.AnalysisModules{}, err
	}
	if report.Artifacts.Static == nil {
		return domain.AnalysisModules{}, ErrResultUnavailable
	}
	result := domain.AnalysisModules{
		Static:       make([]domain.ModuleAnalysis, 0, len(report.Artifacts.Static.Modules)),
		Risk:         make([]domain.ModuleHotspot, 0),
		RiskCoverage: cloneRiskCoverage(report.Artifacts.RiskCoverage),
	}
	result.Static = append(result.Static, report.Artifacts.Static.Modules...)
	if report.Artifacts.Risk != nil {
		result.Risk = append(result.Risk, report.Artifacts.Risk.Modules...)
	}
	return result, nil
}

// Heatmap returns the backend-prepared coordinates from the immutable scoring
// artifact. It never recalculates a score or permits unavailable axes.
func (orchestrator *Orchestrator) Heatmap(ctx context.Context, id domain.AnalysisID) (domain.HeatmapData, error) {
	report, err := orchestrator.Report(ctx, id)
	if err != nil {
		return domain.HeatmapData{}, err
	}
	if report.Artifacts.Risk == nil {
		return domain.HeatmapData{}, ErrResultUnavailable
	}
	heatmap := report.Artifacts.Risk.Heatmap
	heatmap.Cells = append(make([]domain.HeatmapCell, 0, len(heatmap.Cells)), heatmap.Cells...)
	return heatmap, nil
}

func buildFileResults(report domain.AnalysisReport) []domain.AnalysisFileResult {
	if report.Artifacts.Static == nil {
		return make([]domain.AnalysisFileResult, 0)
	}
	gitByPath := make(map[string]domain.GitFileMetrics)
	if report.Artifacts.Git != nil {
		for _, file := range report.Artifacts.Git.Files {
			gitByPath[file.Path] = file
		}
	}
	riskByPath := make(map[string]domain.Hotspot)
	if report.Artifacts.Risk != nil {
		for _, hotspot := range report.Artifacts.Risk.Files {
			riskByPath[hotspot.Path] = hotspot
		}
	}

	items := make([]domain.AnalysisFileResult, 0, len(report.Artifacts.Static.Files))
	for _, analysis := range report.Artifacts.Static.Files {
		item := domain.AnalysisFileResult{Analysis: cloneFileAnalysis(analysis)}
		if file, exists := gitByPath[analysis.Path]; exists {
			clone := cloneGitFileMetrics(file)
			item.Git = &clone
		}
		if hotspot, exists := riskByPath[analysis.Path]; exists {
			clone := cloneHotspot(hotspot)
			item.Risk = &clone
		}
		items = append(items, item)
	}
	return items
}

func matchesHotspot(hotspot domain.Hotspot, query domain.ResultListQuery) bool {
	if !matchesPathAndModule(hotspot.Path, hotspot.Module, query.Query, query.Module) {
		return false
	}
	return matchesRiskLevel(hotspot.Level, query.RiskLevels)
}

func matchesFile(item domain.AnalysisFileResult, query domain.ResultListQuery) bool {
	module := moduleForPath(item.Analysis.Path)
	if item.Risk != nil {
		module = item.Risk.Module
	}
	if !matchesPathAndModule(item.Analysis.Path, module, query.Query, query.Module) {
		return false
	}
	if len(query.Languages) > 0 && !containsLanguage(query.Languages, item.Analysis.Language) {
		return false
	}
	if len(query.AnalysisStatuses) > 0 && !containsAnalysisStatus(query.AnalysisStatuses, item.Analysis.Status) {
		return false
	}
	if len(query.RiskLevels) > 0 {
		if item.Risk == nil || !matchesRiskLevel(item.Risk.Level, query.RiskLevels) {
			return false
		}
	}
	if query.MinScore != nil && (item.Risk == nil || item.Risk.Score < *query.MinScore) {
		return false
	}
	if query.MaxScore != nil && (item.Risk == nil || item.Risk.Score > *query.MaxScore) {
		return false
	}
	return true
}

func matchesPathAndModule(path, module, query, requiredModule string) bool {
	if query != "" && !strings.Contains(strings.ToLower(path), strings.ToLower(query)) {
		return false
	}
	if requiredModule == "" || requiredModule == "." {
		return true
	}
	return module == requiredModule || strings.HasPrefix(module, requiredModule+"/")
}

func matchesRiskLevel(level domain.RiskLevel, allowed []domain.RiskLevel) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, candidate := range allowed {
		if candidate == level {
			return true
		}
	}
	return false
}

func containsLanguage(allowed []domain.Language, language domain.Language) bool {
	for _, candidate := range allowed {
		if candidate == language {
			return true
		}
	}
	return false
}

func containsAnalysisStatus(allowed []domain.AnalysisStatus, status domain.AnalysisStatus) bool {
	for _, candidate := range allowed {
		if candidate == status {
			return true
		}
	}
	return false
}

func sortHotspots(items []domain.Hotspot, query domain.ResultListQuery) {
	sort.Slice(items, func(left, right int) bool {
		comparison := compareHotspots(items[left], items[right], query.Sort)
		if comparison == 0 {
			return items[left].Path < items[right].Path
		}
		if query.Direction == domain.SortDirectionDescending {
			return comparison > 0
		}
		return comparison < 0
	})
}

func compareHotspots(left, right domain.Hotspot, sortBy domain.ResultSort) int {
	switch sortBy {
	case domain.ResultSortPath:
		return strings.Compare(left.Path, right.Path)
	case domain.ResultSortModule:
		return strings.Compare(left.Module, right.Module)
	case domain.ResultSortLinesOfCode:
		return compareInt(left.Metrics.LinesOfCode, right.Metrics.LinesOfCode)
	case domain.ResultSortMaxFunctionLength:
		return compareInt(left.Metrics.MaxFunctionLength, right.Metrics.MaxFunctionLength)
	case domain.ResultSortMaxClassLength:
		return compareInt(left.Metrics.MaxClassLength, right.Metrics.MaxClassLength)
	case domain.ResultSortGitChurn:
		return compareInt(left.Metrics.GitChurn, right.Metrics.GitChurn)
	case domain.ResultSortChangeFrequency:
		return compareInt(left.Metrics.ChangeFrequency, right.Metrics.ChangeFrequency)
	default:
		return compareInt(left.Score, right.Score)
	}
}

func sortFiles(items []domain.AnalysisFileResult, query domain.ResultListQuery) {
	sort.Slice(items, func(left, right int) bool {
		comparison := compareFiles(items[left], items[right], query.Sort)
		if comparison == 0 {
			return items[left].Analysis.Path < items[right].Analysis.Path
		}
		if query.Direction == domain.SortDirectionDescending {
			return comparison > 0
		}
		return comparison < 0
	})
}

func compareFiles(left, right domain.AnalysisFileResult, sortBy domain.ResultSort) int {
	switch sortBy {
	case domain.ResultSortScore:
		return compareInt(riskScore(left), riskScore(right))
	case domain.ResultSortModule:
		return strings.Compare(moduleForFile(left), moduleForFile(right))
	case domain.ResultSortLanguage:
		return strings.Compare(string(left.Analysis.Language), string(right.Analysis.Language))
	case domain.ResultSortStatus:
		return strings.Compare(string(left.Analysis.Status), string(right.Analysis.Status))
	case domain.ResultSortLinesOfCode:
		return compareInt(left.Analysis.Metrics.LinesOfCode, right.Analysis.Metrics.LinesOfCode)
	case domain.ResultSortMaxFunctionLength:
		return compareInt(left.Analysis.Metrics.FunctionLength.Max, right.Analysis.Metrics.FunctionLength.Max)
	case domain.ResultSortMaxClassLength:
		return compareInt(left.Analysis.Metrics.ClassLength.Max, right.Analysis.Metrics.ClassLength.Max)
	case domain.ResultSortGitChurn:
		return compareInt(gitChurn(left), gitChurn(right))
	case domain.ResultSortChangeFrequency:
		return compareInt(changeFrequency(left), changeFrequency(right))
	default:
		return strings.Compare(left.Analysis.Path, right.Analysis.Path)
	}
}

func moduleForFile(item domain.AnalysisFileResult) string {
	if item.Risk != nil {
		return item.Risk.Module
	}
	return moduleForPath(item.Analysis.Path)
}

func moduleForPath(path string) string {
	index := strings.LastIndex(path, "/")
	if index < 0 {
		return "."
	}
	return path[:index]
}

func riskScore(item domain.AnalysisFileResult) int {
	if item.Risk == nil {
		return -1
	}
	return item.Risk.Score
}

func gitChurn(item domain.AnalysisFileResult) int {
	if item.Git == nil {
		return -1
	}
	return item.Git.GitChurn
}

func changeFrequency(item domain.AnalysisFileResult) int {
	if item.Git == nil {
		return -1
	}
	return item.Git.ChangeFrequency
}

func compareInt(left, right int) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func normalizeResultQuery(query *domain.ResultListQuery, defaultSort domain.ResultSort) {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 50
	}
	if query.PageSize > 200 {
		query.PageSize = 200
	}
	if query.Sort == "" {
		query.Sort = defaultSort
	}
	if query.Direction == "" {
		if query.Sort == domain.ResultSortScore {
			query.Direction = domain.SortDirectionDescending
		} else {
			query.Direction = domain.SortDirectionAscending
		}
	}
}

func pageHotspots(items []domain.Hotspot, page, pageSize int) ([]domain.Hotspot, int, int) {
	start, end := pageBounds(len(items), page, pageSize)
	if start == end {
		return make([]domain.Hotspot, 0), page, pageSize
	}
	return append([]domain.Hotspot(nil), items[start:end]...), page, pageSize
}

func pageFiles(items []domain.AnalysisFileResult, page, pageSize int) ([]domain.AnalysisFileResult, int, int) {
	start, end := pageBounds(len(items), page, pageSize)
	if start == end {
		return make([]domain.AnalysisFileResult, 0), page, pageSize
	}
	return append([]domain.AnalysisFileResult(nil), items[start:end]...), page, pageSize
}

func pageBounds(total, page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	lastPage := total / pageSize
	if total%pageSize != 0 {
		lastPage++
	}
	if page > lastPage {
		return total, total
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	return start, end
}

func cloneAnalysisRun(run domain.AnalysisRun) domain.AnalysisRun {
	clone := run
	clone.History = cloneGitHistoryRange(run.History)
	clone.Stages = append([]domain.StageSnapshot(nil), run.Stages...)
	for index := range clone.Stages {
		clone.Stages[index].StartedAt = cloneTime(run.Stages[index].StartedAt)
		clone.Stages[index].CompletedAt = cloneTime(run.Stages[index].CompletedAt)
	}
	clone.Diagnostics = append([]domain.AnalysisDiagnostic(nil), run.Diagnostics...)
	clone.CurrentStage = cloneCurrentStage(run.CurrentStage)
	clone.StartedAt = cloneTime(run.StartedAt)
	clone.CompletedAt = cloneTime(run.CompletedAt)
	if run.Summary != nil {
		summary := *run.Summary
		clone.Summary = &summary
	}
	if run.Profile != nil {
		profile := cloneProfile(*run.Profile)
		clone.Profile = &profile
	}
	return clone
}

func cloneFileAnalysis(analysis domain.FileAnalysis) domain.FileAnalysis {
	clone := analysis
	clone.Functions = append([]domain.Declaration(nil), analysis.Functions...)
	clone.Classes = append([]domain.Declaration(nil), analysis.Classes...)
	clone.Diagnostics = append([]domain.Diagnostic(nil), analysis.Diagnostics...)
	return clone
}

func cloneGitFileMetrics(file domain.GitFileMetrics) domain.GitFileMetrics {
	clone := file
	clone.PreviousPaths = append([]string(nil), file.PreviousPaths...)
	clone.FirstChangedAt = cloneTime(file.FirstChangedAt)
	clone.LastChangedAt = cloneTime(file.LastChangedAt)
	return clone
}

func cloneHotspot(hotspot domain.Hotspot) domain.Hotspot {
	clone := hotspot
	clone.Contributions = append([]domain.RiskContribution(nil), hotspot.Contributions...)
	return clone
}

func cloneGitHistoryRange(history domain.GitHistoryRange) domain.GitHistoryRange {
	return domain.GitHistoryRange{From: cloneTime(history.From), To: cloneTime(history.To)}
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func cloneCurrentStage(value *domain.AnalysisCurrentStage) *domain.AnalysisCurrentStage {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func cloneRiskCoverage(coverage *domain.RiskCoverage) *domain.RiskCoverage {
	if coverage == nil {
		return nil
	}
	clone := *coverage
	clone.AvailableMetrics = append([]domain.RiskMetric(nil), coverage.AvailableMetrics...)
	clone.UnavailableMetrics = append([]domain.RiskMetric(nil), coverage.UnavailableMetrics...)
	clone.RequestedProfile = cloneProfile(coverage.RequestedProfile)
	clone.EffectiveProfile = cloneProfile(coverage.EffectiveProfile)
	return &clone
}

func cloneProfile(profile domain.RiskProfile) domain.RiskProfile {
	clone := profile
	clone.Weights = make(map[domain.RiskMetric]float64, len(profile.Weights))
	for metric, weight := range profile.Weights {
		clone.Weights[metric] = weight
	}
	return clone
}
