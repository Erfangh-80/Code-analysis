package domain

// RiskMetric identifies one analytical signal that can contribute to a hotspot score.
type RiskMetric string

const (
	RiskMetricLinesOfCode       RiskMetric = "lines_of_code"
	RiskMetricMaxFunctionLength RiskMetric = "max_function_length"
	RiskMetricMaxClassLength    RiskMetric = "max_class_length"
	RiskMetricGitChurn          RiskMetric = "git_churn"
	RiskMetricChangeFrequency   RiskMetric = "change_frequency"
)

// RiskLevel is an explainable priority band, not a defect prediction.
type RiskLevel string

const (
	RiskLevelLow    RiskLevel = "low"
	RiskLevelMedium RiskLevel = "medium"
	RiskLevelHigh   RiskLevel = "high"
)

// HotspotMetrics contains source and Git facts used by the v1 risk model.
type HotspotMetrics struct {
	LinesOfCode       int `json:"linesOfCode"`
	MaxFunctionLength int `json:"maxFunctionLength"`
	MaxClassLength    int `json:"maxClassLength"`
	GitChurn          int `json:"gitChurn"`
	ChangeFrequency   int `json:"changeFrequency"`
}

// Value returns a metric value by its stable model identifier.
func (metrics HotspotMetrics) Value(metric RiskMetric) int {
	switch metric {
	case RiskMetricLinesOfCode:
		return metrics.LinesOfCode
	case RiskMetricMaxFunctionLength:
		return metrics.MaxFunctionLength
	case RiskMetricMaxClassLength:
		return metrics.MaxClassLength
	case RiskMetricGitChurn:
		return metrics.GitChurn
	case RiskMetricChangeFrequency:
		return metrics.ChangeFrequency
	default:
		return 0
	}
}

// HotspotInput supplies one file's already-collected analytical facts to the risk engine.
type HotspotInput struct {
	Path    string         `json:"path"`
	Metrics HotspotMetrics `json:"metrics"`
}

// RiskBands defines inclusive score ceilings. Scores above MediumMax are high.
type RiskBands struct {
	LowMax    int `json:"lowMax"`
	MediumMax int `json:"mediumMax"`
}

// RiskProfile is a versioned, local scoring configuration.
type RiskProfile struct {
	Version string                 `json:"version"`
	Weights map[RiskMetric]float64 `json:"weights"`
	Bands   RiskBands              `json:"bands"`
}

// RiskContribution makes one file score interpretable. NormalizationCap is
// the nearest-rank 95th-percentile value used to cap this metric for the run.
type RiskContribution struct {
	Metric           RiskMetric `json:"metric"`
	Raw              int        `json:"raw"`
	NormalizationCap int        `json:"normalizationCap"`
	Normalized       float64    `json:"normalized"`
	Weight           float64    `json:"weight"`
	Weighted         float64    `json:"weighted"`
}

// Hotspot is a ranked, explainable risk result for one repository-relative file.
type Hotspot struct {
	Path          string             `json:"path"`
	Module        string             `json:"module"`
	Metrics       HotspotMetrics     `json:"metrics"`
	Score         int                `json:"score"`
	Level         RiskLevel          `json:"level"`
	Rank          int                `json:"rank"`
	Contributions []RiskContribution `json:"contributions"`
}

// ModuleHotspot is an aggregate score for a repository-relative directory.
type ModuleHotspot struct {
	Path             string    `json:"path"`
	FileCount        int       `json:"fileCount"`
	Score            int       `json:"score"`
	Level            RiskLevel `json:"level"`
	HighestFileScore int       `json:"highestFileScore"`
}

// HeatmapConfiguration selects two metrics for a two-dimensional risk visualization.
type HeatmapConfiguration struct {
	XMetric RiskMetric `json:"xMetric"`
	YMetric RiskMetric `json:"yMetric"`
}

// HeatmapCell is the backend-prepared data for one plotted file.
type HeatmapCell struct {
	Path   string    `json:"path"`
	Module string    `json:"module"`
	XValue int       `json:"xValue"`
	YValue int       `json:"yValue"`
	Score  int       `json:"score"`
	Level  RiskLevel `json:"level"`
	Rank   int       `json:"rank"`
}

// HeatmapData contains axis metadata and scored cells. The frontend only renders it.
type HeatmapData struct {
	XMetric RiskMetric    `json:"xMetric"`
	YMetric RiskMetric    `json:"yMetric"`
	Cells   []HeatmapCell `json:"cells"`
}

// RiskAnalysisRequest is the deterministic input for the risk engine.
type RiskAnalysisRequest struct {
	Files            []HotspotInput        `json:"files"`
	Profile          *RiskProfile          `json:"profile,omitempty"`
	Heatmap          *HeatmapConfiguration `json:"heatmap,omitempty"`
	AvailableMetrics []RiskMetric          `json:"availableMetrics,omitempty"`
}

// RiskAnalysisReport holds the model explanation, ranked hotspots, module aggregates, and heatmap data.
type RiskAnalysisReport struct {
	Profile            RiskProfile     `json:"profile"`
	RequestedProfile   RiskProfile     `json:"requestedProfile"`
	AvailableMetrics   []RiskMetric    `json:"availableMetrics"`
	UnavailableMetrics []RiskMetric    `json:"unavailableMetrics"`
	Formula            string          `json:"formula"`
	Normalization      string          `json:"normalization"`
	Limitations        []string        `json:"limitations"`
	Files              []Hotspot       `json:"files"`
	Modules            []ModuleHotspot `json:"modules"`
	Heatmap            HeatmapData     `json:"heatmap"`
}
