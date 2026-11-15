package domain

import "time"

// GitHistoryRange restricts Git commits by inclusive committer timestamps.
// Nil bounds mean all reachable history is analyzed.
type GitHistoryRange struct {
	From *time.Time `json:"from"`
	To   *time.Time `json:"to"`
}

// GitAnalysisRequest identifies a local repository and optional history window.
type GitAnalysisRequest struct {
	RepositoryPath string          `json:"repositoryPath"`
	History        GitHistoryRange `json:"history"`
	// ScopePath is an internal repository-relative path that limits history to
	// a selected project directory. Public Git API callers omit it for a full
	// worktree report.
	ScopePath string `json:"-"`
}

// GitFileMetrics captures read-only history facts for a repository-relative file identity.
type GitFileMetrics struct {
	Path                string     `json:"path"`
	PreviousPaths       []string   `json:"previousPaths"`
	Deleted             bool       `json:"deleted"`
	GitChurn            int        `json:"gitChurn"`
	ChangeFrequency     int        `json:"changeFrequency"`
	AuthorCount         int        `json:"authorCount"`
	FirstChangedAt      *time.Time `json:"firstChangedAt"`
	LastChangedAt       *time.Time `json:"lastChangedAt"`
	ChangeConcentration float64    `json:"changeConcentration"`
	ChangeDispersion    float64    `json:"changeDispersion"`
}

// GitAnalysisReport is the read-only Git-history result for one local repository.
type GitAnalysisReport struct {
	CommitCount int              `json:"commitCount"`
	History     GitHistoryRange  `json:"history"`
	Files       []GitFileMetrics `json:"files"`
	Diagnostics []Diagnostic     `json:"diagnostics"`
}
