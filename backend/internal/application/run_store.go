package application

import (
	"context"
	"errors"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

var (
	// ErrAnalysisNotFound indicates an unknown locally persisted analysis ID.
	ErrAnalysisNotFound = errors.New("analysis not found")
	// ErrAnalysisAlreadyRunning indicates the same local project/repository has an active run.
	ErrAnalysisAlreadyRunning = errors.New("analysis already running")
	// ErrInvalidRunState indicates an operation incompatible with a terminal run.
	ErrInvalidRunState = errors.New("invalid analysis run state")
)

// RunStore persists local analysis lifecycle state and completed immutable reports.
// Implementations belong to infrastructure; application orchestration depends only
// on this interface.
type RunStore interface {
	CreateQueued(context.Context, domain.AnalysisRun) error
	Get(context.Context, domain.AnalysisID) (domain.AnalysisRun, error)
	List(context.Context, domain.AnalysisListQuery) (domain.AnalysisPage, error)
	SaveRun(context.Context, domain.AnalysisRun) error
	SaveReport(context.Context, domain.AnalysisID, domain.AnalysisReport) error
	GetReport(context.Context, domain.AnalysisID) (domain.AnalysisReport, error)
	RecoverInterruptedRuns(context.Context) error
}
