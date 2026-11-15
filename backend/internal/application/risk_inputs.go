package application

import (
	"path/filepath"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

// BuildHotspotInputs joins already-collected static and Git facts by their
// repository-relative path. Static files without a history record retain zero
// Git facts; callers can surface history availability separately.
//
// Joining is an application concern. The scoring package receives only these
// normalized domain inputs and does not depend on static or Git implementations.
func BuildHotspotInputs(staticFiles []domain.FileAnalysis, gitFiles []domain.GitFileMetrics) []domain.HotspotInput {
	gitByPath := make(map[string]domain.GitFileMetrics, len(gitFiles))
	for _, gitFile := range gitFiles {
		gitByPath[filepath.ToSlash(gitFile.Path)] = gitFile
	}

	inputs := make([]domain.HotspotInput, 0, len(staticFiles))
	for _, staticFile := range staticFiles {
		path := filepath.ToSlash(staticFile.Path)
		gitFile := gitByPath[path]
		inputs = append(inputs, domain.HotspotInput{
			Path: path,
			Metrics: domain.HotspotMetrics{
				LinesOfCode:       staticFile.Metrics.LinesOfCode,
				MaxFunctionLength: staticFile.Metrics.FunctionLength.Max,
				MaxClassLength:    staticFile.Metrics.ClassLength.Max,
				GitChurn:          gitFile.GitChurn,
				ChangeFrequency:   gitFile.ChangeFrequency,
			},
		})
	}
	return inputs
}
