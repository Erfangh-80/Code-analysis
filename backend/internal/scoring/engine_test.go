package scoring_test

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
	"github.com/Erfangh-80/code-analysis/backend/internal/scoring"
)

func TestEngineNormalizesWeightsScoresAndRanksHotspots(t *testing.T) {
	t.Parallel()

	report, err := scoring.NewEngine().Analyze(domain.RiskAnalysisRequest{
		Files: []domain.HotspotInput{
			{Path: "cmd/low.go", Metrics: domain.HotspotMetrics{LinesOfCode: 10, MaxFunctionLength: 2, MaxClassLength: 0, GitChurn: 10, ChangeFrequency: 1}},
			{Path: "cmd/medium.go", Metrics: domain.HotspotMetrics{LinesOfCode: 20, MaxFunctionLength: 4, MaxClassLength: 0, GitChurn: 20, ChangeFrequency: 2}},
			{Path: "lib/high.go", Metrics: domain.HotspotMetrics{LinesOfCode: 40, MaxFunctionLength: 8, MaxClassLength: 10, GitChurn: 40, ChangeFrequency: 4}},
		},
		Profile: &domain.RiskProfile{
			Version: "test-v1",
			Weights: map[domain.RiskMetric]float64{
				domain.RiskMetricLinesOfCode:       0.20,
				domain.RiskMetricMaxFunctionLength: 0.20,
				domain.RiskMetricMaxClassLength:    0.10,
				domain.RiskMetricGitChurn:          0.30,
				domain.RiskMetricChangeFrequency:   0.20,
			},
			Bands: domain.RiskBands{LowMax: 34, MediumMax: 64},
		},
	})
	if err != nil {
		t.Fatalf("calculate risk: %v", err)
	}

	if len(report.Files) != 3 {
		t.Fatalf("hotspot count = %d, want 3", len(report.Files))
	}
	assertHotspot(t, report.Files[0], "lib/high.go", 100, domain.RiskLevelHigh, 1)
	assertHotspot(t, report.Files[1], "cmd/medium.go", 45, domain.RiskLevelMedium, 2)
	assertHotspot(t, report.Files[2], "cmd/low.go", 23, domain.RiskLevelLow, 3)

	contribution := metricContribution(t, report.Files[1], domain.RiskMetricGitChurn)
	if contribution.NormalizationCap != 40 || math.Abs(contribution.Normalized-0.5) > 0.0001 || math.Abs(contribution.Weighted-0.15) > 0.0001 {
		t.Errorf("git churn contribution = %+v, want cap=40 normalized=.5 weighted=.15", contribution)
	}
	if report.Formula == "" || report.Normalization == "" || len(report.Limitations) == 0 {
		t.Errorf("model documentation = %+v, want formula, normalization, and limitations", report)
	}
}

func TestEngineCapsNearestRank95thPercentileNormalization(t *testing.T) {
	t.Parallel()

	files := make([]domain.HotspotInput, 0, 20)
	for value := 1; value <= 20; value++ {
		files = append(files, domain.HotspotInput{
			Path:    fmt.Sprintf("file-%02d.go", value),
			Metrics: domain.HotspotMetrics{LinesOfCode: value},
		})
	}
	files[len(files)-1].Metrics.LinesOfCode = 1000

	report, err := scoring.NewEngine().Analyze(domain.RiskAnalysisRequest{
		Files: files,
		Profile: &domain.RiskProfile{
			Version: "loc-only",
			Weights: map[domain.RiskMetric]float64{domain.RiskMetricLinesOfCode: 1},
			Bands:   domain.RiskBands{LowMax: 34, MediumMax: 64},
		},
	})
	if err != nil {
		t.Fatalf("calculate capped normalization: %v", err)
	}

	outlierHotspot := hotspotByPath(t, report.Files, "file-20.go")
	outlier := metricContribution(t, outlierHotspot, domain.RiskMetricLinesOfCode)
	if outlier.NormalizationCap != 19 || outlier.Normalized != 1 || outlierHotspot.Score != 100 {
		t.Errorf("outlier = hotspot=%+v contribution=%+v, want 95th cap=19 and score=100", outlierHotspot, outlier)
	}
	atCapHotspot := hotspotByPath(t, report.Files, "file-19.go")
	atCap := metricContribution(t, atCapHotspot, domain.RiskMetricLinesOfCode)
	if atCap.Normalized != 1 || atCapHotspot.Score != 100 {
		t.Errorf("95th-percentile file = hotspot=%+v contribution=%+v, want normalized=1 score=100", atCapHotspot, atCap)
	}
}

func TestEngineSupportsConfiguredWeightsAndRejectsInvalidProfiles(t *testing.T) {
	t.Parallel()

	engine := scoring.NewEngine()
	report, err := engine.Analyze(domain.RiskAnalysisRequest{
		Files: []domain.HotspotInput{
			{Path: "slow.go", Metrics: domain.HotspotMetrics{LinesOfCode: 500, GitChurn: 1}},
			{Path: "volatile.go", Metrics: domain.HotspotMetrics{LinesOfCode: 1, GitChurn: 100}},
		},
		Profile: &domain.RiskProfile{
			Version: "churn-only",
			Weights: map[domain.RiskMetric]float64{domain.RiskMetricGitChurn: 1},
			Bands:   domain.RiskBands{LowMax: 34, MediumMax: 64},
		},
	})
	if err != nil {
		t.Fatalf("calculate custom profile: %v", err)
	}
	assertHotspot(t, report.Files[0], "volatile.go", 100, domain.RiskLevelHigh, 1)
	assertHotspot(t, report.Files[1], "slow.go", 1, domain.RiskLevelLow, 2)

	_, err = engine.Analyze(domain.RiskAnalysisRequest{
		Files: []domain.HotspotInput{{Path: "file.go", Metrics: domain.HotspotMetrics{LinesOfCode: 1}}},
		Profile: &domain.RiskProfile{
			Weights: map[domain.RiskMetric]float64{domain.RiskMetricLinesOfCode: 0.5},
			Bands:   domain.RiskBands{LowMax: 34, MediumMax: 64},
		},
	})
	if !errors.Is(err, scoring.ErrInvalidProfile) {
		t.Fatalf("invalid profile error = %v, want ErrInvalidProfile", err)
	}

	_, err = engine.Analyze(domain.RiskAnalysisRequest{
		Files: []domain.HotspotInput{{Path: "file.go", Metrics: domain.HotspotMetrics{LinesOfCode: 1}}},
		Heatmap: &domain.HeatmapConfiguration{
			XMetric: domain.RiskMetricLinesOfCode,
			YMetric: domain.RiskMetricLinesOfCode,
		},
	})
	if !errors.Is(err, scoring.ErrInvalidHeatmap) {
		t.Fatalf("invalid heatmap error = %v, want ErrInvalidHeatmap", err)
	}
}

func TestEngineAggregatesModulesAndPreparesHeatmapCells(t *testing.T) {
	t.Parallel()

	report, err := scoring.NewEngine().Analyze(domain.RiskAnalysisRequest{
		Files: []domain.HotspotInput{
			{Path: "service/critical.go", Metrics: domain.HotspotMetrics{LinesOfCode: 100, GitChurn: 50}},
			{Path: "service/calm.go", Metrics: domain.HotspotMetrics{LinesOfCode: 0, GitChurn: 0}},
			{Path: "root.go", Metrics: domain.HotspotMetrics{LinesOfCode: 50, GitChurn: 20}},
		},
		Profile: &domain.RiskProfile{
			Version: "loc-only",
			Weights: map[domain.RiskMetric]float64{domain.RiskMetricLinesOfCode: 1},
			Bands:   domain.RiskBands{LowMax: 34, MediumMax: 64},
		},
		Heatmap: &domain.HeatmapConfiguration{
			XMetric: domain.RiskMetricGitChurn,
			YMetric: domain.RiskMetricLinesOfCode,
		},
	})
	if err != nil {
		t.Fatalf("calculate module risk: %v", err)
	}

	if len(report.Modules) != 2 {
		t.Fatalf("module count = %d, want 2", len(report.Modules))
	}
	if report.Modules[0].Path != "service" || report.Modules[0].Score != 50 || report.Modules[0].HighestFileScore != 100 {
		t.Errorf("service module = %+v, want mean score=50 and highest=100", report.Modules[0])
	}
	if report.Heatmap.XMetric != domain.RiskMetricGitChurn || report.Heatmap.YMetric != domain.RiskMetricLinesOfCode {
		t.Errorf("heatmap axes = %+v", report.Heatmap)
	}
	if len(report.Heatmap.Cells) != 3 {
		t.Fatalf("heatmap cell count = %d, want 3", len(report.Heatmap.Cells))
	}
	if report.Heatmap.Cells[0].Path != "service/critical.go" || report.Heatmap.Cells[0].XValue != 50 || report.Heatmap.Cells[0].YValue != 100 {
		t.Errorf("first heatmap cell = %+v", report.Heatmap.Cells[0])
	}
}

func assertHotspot(t *testing.T, hotspot domain.Hotspot, path string, score int, level domain.RiskLevel, rank int) {
	t.Helper()
	if hotspot.Path != path || hotspot.Score != score || hotspot.Level != level || hotspot.Rank != rank {
		t.Errorf("hotspot = %+v, want path=%q score=%d level=%q rank=%d", hotspot, path, score, level, rank)
	}
}

func hotspotByPath(t *testing.T, hotspots []domain.Hotspot, path string) domain.Hotspot {
	t.Helper()
	for _, hotspot := range hotspots {
		if hotspot.Path == path {
			return hotspot
		}
	}
	t.Fatalf("hotspot %q missing from %+v", path, hotspots)
	return domain.Hotspot{}
}

func metricContribution(t *testing.T, hotspot domain.Hotspot, metric domain.RiskMetric) domain.RiskContribution {
	t.Helper()
	for _, contribution := range hotspot.Contributions {
		if contribution.Metric == metric {
			return contribution
		}
	}
	t.Fatalf("metric %q missing from contributions: %+v", metric, hotspot.Contributions)
	return domain.RiskContribution{}
}
