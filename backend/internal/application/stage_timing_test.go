package application

import (
	"testing"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

func TestSetStagePersistsTimingForCompletedStaticAndGitStages(t *testing.T) {
	run := domain.AnalysisRun{Stages: initialStages()}
	for _, stage := range []domain.AnalysisStage{domain.AnalysisStageStatic, domain.AnalysisStageGit} {
		setStage(&run, stage, domain.StageStatusRunning, domain.StageProgress{Total: 1})
	}
	time.Sleep(2 * time.Millisecond)
	for _, stage := range []domain.AnalysisStage{domain.AnalysisStageStatic, domain.AnalysisStageGit} {
		setStage(&run, stage, domain.StageStatusCompleted, domain.StageProgress{Total: 1, Completed: 1})
	}

	for _, snapshot := range run.Stages {
		if snapshot.Stage != domain.AnalysisStageStatic && snapshot.Stage != domain.AnalysisStageGit {
			continue
		}
		if snapshot.StartedAt == nil || snapshot.CompletedAt == nil || snapshot.DurationMillis <= 0 {
			t.Errorf("stage %q timing = %+v, want startedAt, completedAt, and positive duration", snapshot.Stage, snapshot)
		}
	}
}
