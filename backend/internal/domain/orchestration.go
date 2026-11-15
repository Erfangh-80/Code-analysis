package domain

import "time"

// AnalysisID identifies one locally persisted analysis run.
type AnalysisID string

// AnalysisRunStatus is the lifecycle state of a run. Detailed work is tracked
// by CurrentStage and the independent stage snapshots. It is intentionally
// distinct from AnalysisStatus, which describes a single static file result.
type AnalysisRunStatus string

const (
	AnalysisStatusQueued    AnalysisRunStatus = "queued"
	AnalysisStatusRunning   AnalysisRunStatus = "running"
	AnalysisStatusCompleted AnalysisRunStatus = "completed"
	AnalysisStatusFailed    AnalysisRunStatus = "failed"
	AnalysisStatusCancelled AnalysisRunStatus = "cancelled"
)

// AnalysisCurrentStage is the high-level work currently performed by a run.
type AnalysisCurrentStage string

const (
	AnalysisCurrentStageValidation AnalysisCurrentStage = "validation"
	AnalysisCurrentStageDiscovery  AnalysisCurrentStage = "discovery"
	AnalysisCurrentStageAnalysis   AnalysisCurrentStage = "analysis"
	AnalysisCurrentStageScoring    AnalysisCurrentStage = "risk_scoring"
	AnalysisCurrentStageFinalizing AnalysisCurrentStage = "finalizing"
)

// AnalysisStage identifies independently observable parts of a run.
type AnalysisStage string

const (
	AnalysisStageValidation AnalysisStage = "validation"
	AnalysisStageDiscovery  AnalysisStage = "discovery"
	AnalysisStageStatic     AnalysisStage = "static_analysis"
	AnalysisStageGit        AnalysisStage = "git_analysis"
	AnalysisStageRisk       AnalysisStage = "risk_scoring"
)

// StageStatus is the lifecycle state of one analysis stage.
type StageStatus string

const (
	StageStatusPending     StageStatus = "pending"
	StageStatusRunning     StageStatus = "running"
	StageStatusCompleted   StageStatus = "completed"
	StageStatusUnavailable StageStatus = "unavailable"
	StageStatusFailed      StageStatus = "failed"
	StageStatusCancelled   StageStatus = "cancelled"
	StageStatusSkipped     StageStatus = "skipped"
)

// AnalysisRequest starts one local project analysis. ProjectPath is used only
// on the backend host and is never exposed in public result responses.
type AnalysisRequest struct {
	ProjectPath string          `json:"projectPath"`
	History     GitHistoryRange `json:"history"`
	Profile     *RiskProfile    `json:"profile,omitempty"`
}

// ProjectScope describes the validated local project and, when available, its
// enclosing Git worktree. All paths here are internal-only storage data.
type ProjectScope struct {
	ProjectPath       string `json:"-"`
	ScopeKey          string `json:"-"`
	GitRepositoryRoot string `json:"-"`
	GitRelativeScope  string `json:"-"`
	GitAvailable      bool   `json:"-"`
}

// ProjectScan is the safe working-tree snapshot passed from a scanner to the
// static-analysis stage. File paths are normalized relative to ProjectPath.
type ProjectScan struct {
	Scope       ProjectScope
	Files       []string
	Diagnostics []AnalysisDiagnostic
}

// AnalysisDiagnostic is a safe, stage-scoped diagnostic. Path is always
// repository/project relative when set; it is never an absolute host path.
type AnalysisDiagnostic struct {
	Stage      AnalysisStage `json:"stage"`
	Path       string        `json:"path,omitempty"`
	Code       string        `json:"code"`
	Severity   string        `json:"severity"`
	Message    string        `json:"message"`
	RecordedAt time.Time     `json:"recordedAt"`
}

// StageProgress is the measurable progress for a stage.
type StageProgress struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
}

// StageSnapshot records one stage independently so a client can display
// static, Git, and scoring states without deriving them itself.
type StageSnapshot struct {
	Stage       AnalysisStage `json:"stage"`
	Status      StageStatus   `json:"status"`
	StartedAt   *time.Time    `json:"startedAt,omitempty"`
	CompletedAt *time.Time    `json:"completedAt,omitempty"`
	Progress    StageProgress `json:"progress"`
}

// AnalysisProgress is an aggregate, monotonic view of work completed by a run.
type AnalysisProgress struct {
	FilesDiscovered int `json:"filesDiscovered"`
	FilesEligible   int `json:"filesEligible"`
	FilesAnalyzed   int `json:"filesAnalyzed"`
	FilesWithErrors int `json:"filesWithErrors"`
	CommitsTotal    int `json:"commitsTotal"`
	CommitsScanned  int `json:"commitsScanned"`
	Percent         int `json:"percent"`
}

// AnalysisSummary is a safe completed-run overview.
type AnalysisSummary struct {
	FilesDiscovered int         `json:"filesDiscovered"`
	FilesAnalyzed   int         `json:"filesAnalyzed"`
	FilesWithErrors int         `json:"filesWithErrors"`
	GitStatus       StageStatus `json:"gitStatus"`
	RiskStatus      StageStatus `json:"riskStatus"`
	ScoredFiles     int         `json:"scoredFiles"`
	UnscoredFiles   int         `json:"unscoredFiles"`
}

// AnalysisRun holds persisted lifecycle, progress, and safe metadata. Internal
// local paths are intentionally excluded from JSON responses.
type AnalysisRun struct {
	ID           AnalysisID            `json:"id"`
	Status       AnalysisRunStatus     `json:"status"`
	CurrentStage *AnalysisCurrentStage `json:"currentStage,omitempty"`
	CreatedAt    time.Time             `json:"createdAt"`
	StartedAt    *time.Time            `json:"startedAt,omitempty"`
	CompletedAt  *time.Time            `json:"completedAt,omitempty"`
	History      GitHistoryRange       `json:"history"`
	Stages       []StageSnapshot       `json:"stages"`
	Progress     AnalysisProgress      `json:"progress"`
	Diagnostics  []AnalysisDiagnostic  `json:"diagnostics"`
	Summary      *AnalysisSummary      `json:"summary,omitempty"`

	Scope   ProjectScope `json:"-"`
	Profile *RiskProfile `json:"-"`
}

// MetricAvailability explicitly distinguishes unavailable data from a valid
// zero metric value.
type MetricAvailability struct {
	LinesOfCode       bool `json:"linesOfCode"`
	MaxFunctionLength bool `json:"maxFunctionLength"`
	MaxClassLength    bool `json:"maxClassLength"`
	GitChurn          bool `json:"gitChurn"`
	ChangeFrequency   bool `json:"changeFrequency"`
}

// Metrics returns the stable identifiers whose values are valid.
func (availability MetricAvailability) Metrics() []RiskMetric {
	metrics := make([]RiskMetric, 0, 5)
	if availability.LinesOfCode {
		metrics = append(metrics, RiskMetricLinesOfCode)
	}
	if availability.MaxFunctionLength {
		metrics = append(metrics, RiskMetricMaxFunctionLength)
	}
	if availability.MaxClassLength {
		metrics = append(metrics, RiskMetricMaxClassLength)
	}
	if availability.GitChurn {
		metrics = append(metrics, RiskMetricGitChurn)
	}
	if availability.ChangeFrequency {
		metrics = append(metrics, RiskMetricChangeFrequency)
	}
	return metrics
}

// RiskCoverage explains the effective model used when Git or static metrics
// are unavailable.
type RiskCoverage struct {
	AvailableMetrics   []RiskMetric `json:"availableMetrics"`
	UnavailableMetrics []RiskMetric `json:"unavailableMetrics"`
	RequestedProfile   RiskProfile  `json:"requestedProfile"`
	EffectiveProfile   RiskProfile  `json:"effectiveProfile"`
}

// AnalysisArtifacts are persisted only on the backend. A missing Git or Risk
// result is meaningful and is explained by stage status and diagnostics.
type AnalysisArtifacts struct {
	Static       *BatchAnalysis      `json:"static,omitempty"`
	Git          *GitAnalysisReport  `json:"git,omitempty"`
	Risk         *RiskAnalysisReport `json:"risk,omitempty"`
	RiskCoverage *RiskCoverage       `json:"riskCoverage,omitempty"`
}

// AnalysisReport is the immutable completed report restored from local SQLite.
type AnalysisReport struct {
	Analysis  AnalysisRun       `json:"analysis"`
	Artifacts AnalysisArtifacts `json:"artifacts"`
}

// AnalysisListQuery controls local persisted-run listing.
type AnalysisListQuery struct {
	Status   *AnalysisRunStatus
	Page     int
	PageSize int
}

// AnalysisPage is a paginated persisted-run listing.
type AnalysisPage struct {
	Items      []AnalysisRun `json:"items"`
	Page       int           `json:"page"`
	PageSize   int           `json:"pageSize"`
	TotalItems int           `json:"totalItems"`
}
