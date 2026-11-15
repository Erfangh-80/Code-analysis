package domain

// ResultSort identifies a deterministic order for persisted result views.
type ResultSort string

const (
	ResultSortScore             ResultSort = "score"
	ResultSortPath              ResultSort = "path"
	ResultSortModule            ResultSort = "module"
	ResultSortLanguage          ResultSort = "language"
	ResultSortStatus            ResultSort = "status"
	ResultSortLinesOfCode       ResultSort = "linesOfCode"
	ResultSortMaxFunctionLength ResultSort = "maxFunctionLength"
	ResultSortMaxClassLength    ResultSort = "maxClassLength"
	ResultSortGitChurn          ResultSort = "gitChurn"
	ResultSortChangeFrequency   ResultSort = "changeFrequency"
)

// SortDirection controls ascending or descending result ordering.
type SortDirection string

const (
	SortDirectionAscending  SortDirection = "asc"
	SortDirectionDescending SortDirection = "desc"
)

// ResultListQuery applies only to a completed persisted report. Filters use
// display-safe project-relative path and module values, never host paths.
type ResultListQuery struct {
	Query            string
	Module           string
	RiskLevels       []RiskLevel
	AnalysisStatuses []AnalysisStatus
	Sort             ResultSort
	Direction        SortDirection
	Page             int
	PageSize         int
}

// RiskLevelCounts summarizes completed scored-file distribution.
type RiskLevelCounts struct {
	Low    int `json:"low"`
	Medium int `json:"medium"`
	High   int `json:"high"`
}

// AnalysisResultSummary is the compact completed-report overview. It retains
// availability metadata so clients do not interpret omitted metrics as zero.
type AnalysisResultSummary struct {
	Analysis       AnalysisRun     `json:"analysis"`
	Summary        AnalysisSummary `json:"summary"`
	RiskLevels     RiskLevelCounts `json:"riskLevels"`
	ModuleCount    int             `json:"moduleCount"`
	RiskCoverage   *RiskCoverage   `json:"riskCoverage,omitempty"`
	HasGitArtifact bool            `json:"hasGitArtifact"`
}

// AnalysisHotspotPage is a paginated, filtered view of immutable hotspot data.
type AnalysisHotspotPage struct {
	Items        []Hotspot     `json:"items"`
	Page         int           `json:"page"`
	PageSize     int           `json:"pageSize"`
	TotalItems   int           `json:"totalItems"`
	RiskCoverage *RiskCoverage `json:"riskCoverage,omitempty"`
}

// AnalysisFileResult joins one static file artifact with its optional Git and
// risk records. The static analysis remains the source of parser diagnostics.
type AnalysisFileResult struct {
	Analysis FileAnalysis    `json:"analysis"`
	Git      *GitFileMetrics `json:"git,omitempty"`
	Risk     *Hotspot        `json:"risk,omitempty"`
}

// AnalysisFilePage is a paginated, filtered view of static-file artifacts.
type AnalysisFilePage struct {
	Items        []AnalysisFileResult `json:"items"`
	Page         int                  `json:"page"`
	PageSize     int                  `json:"pageSize"`
	TotalItems   int                  `json:"totalItems"`
	RiskCoverage *RiskCoverage        `json:"riskCoverage,omitempty"`
}

// AnalysisModules contains static module facts and optional risk-ranked module
// scores from one immutable completed report.
type AnalysisModules struct {
	Static       []ModuleAnalysis `json:"static"`
	Risk         []ModuleHotspot  `json:"risk"`
	RiskCoverage *RiskCoverage    `json:"riskCoverage,omitempty"`
}
