package application_test

import (
	"testing"

	"github.com/Erfangh-80/code-analysis/backend/internal/application"
	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

func TestBuildHotspotInputsJoinsStaticAndGitMetrics(t *testing.T) {
	t.Parallel()

	inputs := application.BuildHotspotInputs(
		[]domain.FileAnalysis{{
			Path: "pkg/worker.go",
			Metrics: domain.Metrics{
				LinesOfCode:    80,
				FunctionLength: domain.LengthSummary{Max: 24},
				ClassLength:    domain.LengthSummary{Max: 0},
			},
		}},
		[]domain.GitFileMetrics{{Path: "pkg/worker.go", GitChurn: 31, ChangeFrequency: 7}},
	)
	if len(inputs) != 1 {
		t.Fatalf("input count = %d, want 1", len(inputs))
	}
	if inputs[0].Metrics.LinesOfCode != 80 || inputs[0].Metrics.MaxFunctionLength != 24 || inputs[0].Metrics.GitChurn != 31 || inputs[0].Metrics.ChangeFrequency != 7 {
		t.Errorf("joined input = %+v", inputs[0])
	}
}
