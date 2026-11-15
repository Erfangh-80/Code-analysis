// Package scoring calculates deterministic, explainable risk scores from
// already-collected static and Git metrics.
package scoring

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

var (
	// ErrInvalidProfile indicates weights, bands, or metric identifiers that cannot produce a meaningful score.
	ErrInvalidProfile = errors.New("invalid risk profile")
	// ErrInvalidInput indicates malformed file paths, duplicate paths, or negative metrics.
	ErrInvalidInput = errors.New("invalid risk input")
	// ErrInvalidHeatmap indicates invalid or identical heatmap axes.
	ErrInvalidHeatmap = errors.New("invalid heatmap configuration")
)

var supportedMetrics = []domain.RiskMetric{
	domain.RiskMetricLinesOfCode,
	domain.RiskMetricMaxFunctionLength,
	domain.RiskMetricMaxClassLength,
	domain.RiskMetricGitChurn,
	domain.RiskMetricChangeFrequency,
}

const scoreFormula = "round(100 × sum(weight(metric) × normalized(metric, file)))"
const normalizationDescription = "Each metric is divided by its nearest-rank 95th-percentile value across the requested files and capped at 1."

var modelLimitations = []string{
	"This score prioritizes review and refactoring work; it does not predict defects, incidents, or security vulnerabilities.",
	"Scores are relative to the requested file population, so scores from different runs are not directly comparable.",
	"The v1 model uses only lines of code, maximum function length, maximum class length, Git churn, and change frequency.",
	"A file without a selected-history Git record has zero churn and change frequency; inspect Git-analysis diagnostics and history scope before interpreting that result.",
}

// Engine calculates hotspot scores without filesystem, Git, HTTP, or frontend dependencies.
type Engine struct{}

type normalizationResult struct {
	values map[string]map[domain.RiskMetric]float64
	caps   map[domain.RiskMetric]int
}

// NewEngine creates a risk engine using the v1 metric model.
func NewEngine() *Engine {
	return &Engine{}
}

// DefaultProfile returns the versioned local default. Weights intentionally use
// only metrics currently produced by the implemented analyzers.
func DefaultProfile() domain.RiskProfile {
	return domain.RiskProfile{
		Version: "risk-v1",
		Weights: map[domain.RiskMetric]float64{
			domain.RiskMetricLinesOfCode:       0.15,
			domain.RiskMetricMaxFunctionLength: 0.25,
			domain.RiskMetricMaxClassLength:    0.10,
			domain.RiskMetricGitChurn:          0.25,
			domain.RiskMetricChangeFrequency:   0.25,
		},
		Bands: domain.RiskBands{LowMax: 34, MediumMax: 64},
	}
}

// Analyze normalizes signals across the supplied file population, scores each
// file, ranks hotspots, aggregates module scores, and prepares heatmap cells.
func (engine *Engine) Analyze(request domain.RiskAnalysisRequest) (domain.RiskAnalysisReport, error) {
	profile := DefaultProfile()
	if request.Profile != nil {
		profile = cloneProfile(*request.Profile)
	}
	if err := validateProfile(profile); err != nil {
		return domain.RiskAnalysisReport{}, err
	}
	if err := validateInputs(request.Files); err != nil {
		return domain.RiskAnalysisReport{}, err
	}
	inputs := canonicalizeInputs(request.Files)

	heatmap, err := validateHeatmap(request.Heatmap)
	if err != nil {
		return domain.RiskAnalysisReport{}, err
	}
	normalization := normalize(inputs)

	hotspots := make([]domain.Hotspot, 0, len(inputs))
	for _, input := range inputs {
		hotspot := domain.Hotspot{
			Path:          input.Path,
			Module:        modulePath(input.Path),
			Metrics:       input.Metrics,
			Contributions: make([]domain.RiskContribution, 0, len(supportedMetrics)),
		}
		weightedTotal := 0.0
		for _, metric := range supportedMetrics {
			value := normalization.values[input.Path][metric]
			weight := profile.Weights[metric]
			weighted := value * weight
			weightedTotal += weighted
			hotspot.Contributions = append(hotspot.Contributions, domain.RiskContribution{
				Metric:           metric,
				Raw:              input.Metrics.Value(metric),
				NormalizationCap: normalization.caps[metric],
				Normalized:       round(value, 6),
				Weight:           round(weight, 6),
				Weighted:         round(weighted, 6),
			})
		}
		// The epsilon prevents binary floating-point representation from turning an
		// exact decimal half (for example, 22.5) into a value infinitesimally
		// below the intended conventional rounding boundary.
		hotspot.Score = int(math.Round(weightedTotal*100 + 1e-9))
		hotspot.Level = riskLevel(hotspot.Score, profile.Bands)
		hotspots = append(hotspots, hotspot)
	}

	sort.Slice(hotspots, func(left, right int) bool {
		if hotspots[left].Score == hotspots[right].Score {
			return hotspots[left].Path < hotspots[right].Path
		}
		return hotspots[left].Score > hotspots[right].Score
	})
	for index := range hotspots {
		hotspots[index].Rank = index + 1
	}

	return domain.RiskAnalysisReport{
		Profile:       profile,
		Formula:       scoreFormula,
		Normalization: normalizationDescription,
		Limitations:   append([]string(nil), modelLimitations...),
		Files:         hotspots,
		Modules:       aggregateModules(hotspots, profile.Bands),
		Heatmap:       prepareHeatmap(hotspots, heatmap),
	}, nil
}

func cloneProfile(profile domain.RiskProfile) domain.RiskProfile {
	weights := make(map[domain.RiskMetric]float64, len(profile.Weights))
	for metric, weight := range profile.Weights {
		weights[metric] = weight
	}
	profile.Weights = weights
	return profile
}

func validateProfile(profile domain.RiskProfile) error {
	if strings.TrimSpace(profile.Version) == "" {
		return fmt.Errorf("%w: version is required", ErrInvalidProfile)
	}
	if profile.Bands.LowMax < 0 || profile.Bands.LowMax >= profile.Bands.MediumMax || profile.Bands.MediumMax >= 100 {
		return fmt.Errorf("%w: bands must satisfy 0 <= lowMax < mediumMax < 100", ErrInvalidProfile)
	}
	weightTotal := 0.0
	for metric, weight := range profile.Weights {
		if !isSupportedMetric(metric) {
			return fmt.Errorf("%w: unsupported metric %q", ErrInvalidProfile, metric)
		}
		if math.IsNaN(weight) || math.IsInf(weight, 0) || weight < 0 {
			return fmt.Errorf("%w: weight for %q must be finite and non-negative", ErrInvalidProfile, metric)
		}
		weightTotal += weight
	}
	if math.Abs(weightTotal-1) > 0.000001 {
		return fmt.Errorf("%w: weights must sum to 1.0", ErrInvalidProfile)
	}
	return nil
}

func validateInputs(inputs []domain.HotspotInput) error {
	if len(inputs) == 0 {
		return fmt.Errorf("%w: at least one file is required", ErrInvalidInput)
	}
	seenPaths := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		path := filepath.ToSlash(filepath.Clean(input.Path))
		if strings.TrimSpace(input.Path) == "" || path == "." || path == ".." || strings.HasPrefix(path, "../") || filepath.IsAbs(input.Path) {
			return fmt.Errorf("%w: paths must be non-empty repository-relative paths", ErrInvalidInput)
		}
		if _, exists := seenPaths[path]; exists {
			return fmt.Errorf("%w: duplicate file path %q", ErrInvalidInput, path)
		}
		seenPaths[path] = struct{}{}
		if input.Metrics.LinesOfCode < 0 || input.Metrics.MaxFunctionLength < 0 || input.Metrics.MaxClassLength < 0 || input.Metrics.GitChurn < 0 || input.Metrics.ChangeFrequency < 0 {
			return fmt.Errorf("%w: metric values must be non-negative", ErrInvalidInput)
		}
	}
	return nil
}

func canonicalizeInputs(inputs []domain.HotspotInput) []domain.HotspotInput {
	canonical := make([]domain.HotspotInput, len(inputs))
	copy(canonical, inputs)
	for index := range canonical {
		canonical[index].Path = filepath.ToSlash(filepath.Clean(canonical[index].Path))
	}
	return canonical
}

func modulePath(filePath string) string {
	module := filepath.ToSlash(filepath.Dir(filePath))
	if module == "" || module == "." {
		return "."
	}
	return module
}

func validateHeatmap(configuration *domain.HeatmapConfiguration) (domain.HeatmapConfiguration, error) {
	if configuration == nil {
		return domain.HeatmapConfiguration{
			XMetric: domain.RiskMetricGitChurn,
			YMetric: domain.RiskMetricLinesOfCode,
		}, nil
	}
	if !isSupportedMetric(configuration.XMetric) || !isSupportedMetric(configuration.YMetric) || configuration.XMetric == configuration.YMetric {
		return domain.HeatmapConfiguration{}, fmt.Errorf("%w: axes must be distinct supported metrics", ErrInvalidHeatmap)
	}
	return *configuration, nil
}

func normalize(inputs []domain.HotspotInput) normalizationResult {
	result := normalizationResult{
		values: make(map[string]map[domain.RiskMetric]float64, len(inputs)),
		caps:   make(map[domain.RiskMetric]int, len(supportedMetrics)),
	}
	for _, metric := range supportedMetrics {
		values := make([]int, 0, len(inputs))
		for _, input := range inputs {
			values = append(values, input.Metrics.Value(metric))
		}
		ceiling := percentile95(values)
		result.caps[metric] = ceiling
		for _, input := range inputs {
			if result.values[input.Path] == nil {
				result.values[input.Path] = make(map[domain.RiskMetric]float64, len(supportedMetrics))
			}
			if ceiling == 0 {
				result.values[input.Path][metric] = 0
				continue
			}
			result.values[input.Path][metric] = math.Min(float64(input.Metrics.Value(metric))/float64(ceiling), 1)
		}
	}
	return result
}

func percentile95(values []int) int {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int(nil), values...)
	sort.Ints(sorted)
	index := int(math.Ceil(0.95*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	return sorted[index]
}

func aggregateModules(hotspots []domain.Hotspot, bands domain.RiskBands) []domain.ModuleHotspot {
	type accumulator struct {
		path    string
		count   int
		total   int
		highest int
	}
	modules := make(map[string]*accumulator)
	for _, hotspot := range hotspots {
		path := hotspot.Module
		module := modules[path]
		if module == nil {
			module = &accumulator{path: path}
			modules[path] = module
		}
		module.count++
		module.total += hotspot.Score
		if hotspot.Score > module.highest {
			module.highest = hotspot.Score
		}
	}

	result := make([]domain.ModuleHotspot, 0, len(modules))
	for _, module := range modules {
		score := int(math.Round(float64(module.total) / float64(module.count)))
		result = append(result, domain.ModuleHotspot{
			Path:             module.path,
			FileCount:        module.count,
			Score:            score,
			Level:            riskLevel(score, bands),
			HighestFileScore: module.highest,
		})
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].Score != result[right].Score {
			return result[left].Score > result[right].Score
		}
		if result[left].HighestFileScore != result[right].HighestFileScore {
			return result[left].HighestFileScore > result[right].HighestFileScore
		}
		return result[left].Path < result[right].Path
	})
	return result
}

func prepareHeatmap(hotspots []domain.Hotspot, configuration domain.HeatmapConfiguration) domain.HeatmapData {
	cells := make([]domain.HeatmapCell, 0, len(hotspots))
	for _, hotspot := range hotspots {
		cells = append(cells, domain.HeatmapCell{
			Path:   hotspot.Path,
			Module: hotspot.Module,
			XValue: hotspot.Metrics.Value(configuration.XMetric),
			YValue: hotspot.Metrics.Value(configuration.YMetric),
			Score:  hotspot.Score,
			Level:  hotspot.Level,
			Rank:   hotspot.Rank,
		})
	}
	return domain.HeatmapData{XMetric: configuration.XMetric, YMetric: configuration.YMetric, Cells: cells}
}

func riskLevel(score int, bands domain.RiskBands) domain.RiskLevel {
	if score <= bands.LowMax {
		return domain.RiskLevelLow
	}
	if score <= bands.MediumMax {
		return domain.RiskLevelMedium
	}
	return domain.RiskLevelHigh
}

func isSupportedMetric(metric domain.RiskMetric) bool {
	for _, supported := range supportedMetrics {
		if metric == supported {
			return true
		}
	}
	return false
}

func round(value float64, decimals int) float64 {
	factor := math.Pow10(decimals)
	return math.Round(value*factor) / factor
}
