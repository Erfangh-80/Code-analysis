// Package domain contains the stable data contracts shared by the analyzer,
// persistence layer and HTTP API.
package domain

import "time"

const FormulaVersion = "hotspot-v1-log1p-minmax"

type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusPartial   Status = "partial"
	StatusFailed    Status = "failed"
)

type Weights struct {
	Complexity float64 `json:"complexity"`
	Churn      float64 `json:"churn"`
	Frequency  float64 `json:"frequency"`
}

func DefaultWeights() Weights { return Weights{Complexity: .5, Churn: .3, Frequency: .2} }

type HistoryRange struct {
	Since time.Time `json:"since"`
	Until time.Time `json:"until"`
}

type AnalysisInput struct {
	RepositoryPath string       `json:"repository_path"`
	Ref            string       `json:"ref"`
	Languages      []string     `json:"languages"`
	History        HistoryRange `json:"history"`
	Weights        Weights      `json:"weights"`
}

type FunctionMetric struct {
	Name       string `json:"name"`
	StartLine  int    `json:"start_line"`
	EndLine    int    `json:"end_line"`
	Length     int    `json:"length"`
	Complexity int    `json:"complexity"`
}

type ClassMetric struct {
	Name      string `json:"name"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Length    int    `json:"length"`
}

type StaticMetrics struct {
	Language string `json:"language"`
	// Nil scalar metrics mean parsing was not valid enough to calculate them.
	TotalLines            *int             `json:"total_lines"`
	LOC                   *int             `json:"loc"`
	FunctionCount         *int             `json:"function_count"`
	MaxFunctionComplexity *int             `json:"max_function_complexity"`
	Functions             []FunctionMetric `json:"functions"`
	// Classes is nil for Go, which explicitly represents “not applicable”; an
	// empty slice means the language supports classes but none were found.
	Classes        []ClassMetric `json:"classes"`
	AnalysisStatus string        `json:"analysis_status"`
	AnalysisError  *string       `json:"analysis_error"`
}

type GitMetrics struct {
	Churn               int        `json:"churn"`
	ChangeFrequency     int        `json:"change_frequency"`
	AuthorCount         int        `json:"author_count"`
	AuthorConcentration *float64   `json:"author_concentration"`
	AuthorDispersion    *float64   `json:"author_dispersion"`
	FirstChange         *time.Time `json:"first_change"`
	LastChange          *time.Time `json:"last_change"`
}

type NormalizedMetrics struct {
	Complexity *float64 `json:"complexity"`
	Churn      *float64 `json:"churn"`
	Frequency  *float64 `json:"frequency"`
}

type Contributions struct {
	Complexity *float64 `json:"complexity"`
	Churn      *float64 `json:"churn"`
	Frequency  *float64 `json:"frequency"`
}

type FileResult struct {
	ID              string            `json:"id"`
	AnalysisID      string            `json:"analysis_id"`
	Path            string            `json:"path"`
	Module          string            `json:"module"`
	Static          StaticMetrics     `json:"static"`
	Git             GitMetrics        `json:"git"`
	Normalized      NormalizedMetrics `json:"normalized"`
	EffectiveWeight Weights           `json:"effective_weight"`
	Contributions   Contributions     `json:"contributions"`
	HotspotScore    *float64          `json:"hotspot_score"`
	ScoreReason     *string           `json:"score_reason"`
}

type ModuleResult struct {
	ID                  string   `json:"id"`
	AnalysisID          string   `json:"analysis_id"`
	Name                string   `json:"name"`
	FileCount           int      `json:"file_count"`
	ScorableFileCount   int      `json:"scorable_file_count"`
	TotalLOC            int      `json:"total_loc"`
	TotalChurn          int      `json:"total_churn"`
	UniqueCommitCount   int      `json:"unique_commit_count"`
	AuthorCount         int      `json:"author_count"`
	AuthorConcentration *float64 `json:"author_concentration"`
	AuthorDispersion    *float64 `json:"author_dispersion"`
	AverageScore        *float64 `json:"average_score"`
	HighestScore        *float64 `json:"highest_score"`
	AverageComplexity   *float64 `json:"average_complexity"`
	AverageChurn        *float64 `json:"average_churn"`
	AverageFrequency    *float64 `json:"average_frequency"`
}

type Analysis struct {
	ID              string       `json:"id"`
	Status          Status       `json:"status"`
	RepositoryPath  string       `json:"repository_path"`
	Ref             string       `json:"ref"`
	CommitSHA       string       `json:"commit_sha"`
	Languages       []string     `json:"languages"`
	History         HistoryRange `json:"history"`
	Weights         Weights      `json:"weights"`
	FormulaVersion  string       `json:"formula_version"`
	HistoryComplete bool         `json:"history_complete"`
	CurrentStage    string       `json:"current_stage"`
	FilesProcessed  int          `json:"files_processed"`
	FilesDiscovered int          `json:"files_discovered"`
	Warnings        []string     `json:"warnings"`
	FailureCode     *string      `json:"failure_code"`
	FailureMessage  *string      `json:"failure_message"`
	CreatedAt       time.Time    `json:"created_at"`
	StartedAt       *time.Time   `json:"started_at"`
	FinishedAt      *time.Time   `json:"finished_at"`
}

type HeatmapCell struct {
	X            int      `json:"x"`
	Y            int      `json:"y"`
	Count        int      `json:"count"`
	AverageScore *float64 `json:"average_score"`
	Paths        []string `json:"paths,omitempty"`
}

type Evaluation struct {
	ID                string     `json:"id"`
	AnalysisID        string     `json:"analysis_id"`
	Status            Status     `json:"status"`
	Since             time.Time  `json:"since"`
	Until             time.Time  `json:"until"`
	SampleSize        int        `json:"sample_size"`
	Spearman          *float64   `json:"spearman"`
	Reason            *string    `json:"reason"`
	PotentialFixCount int        `json:"potential_fix_count"`
	CreatedAt         time.Time  `json:"created_at"`
	FinishedAt        *time.Time `json:"finished_at"`
}
