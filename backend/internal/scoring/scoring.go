// Package scoring implements the documented relative hotspot formula.
package scoring

import (
	"errors"
	"fmt"
	"math"
	"path"
	"sort"

	"github.com/Erfangh-80/Code-analysis/backend/internal/domain"
)

func ValidateWeights(w domain.Weights) error {
	values := []float64{w.Complexity, w.Churn, w.Frequency}
	sum := 0.0
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			return errors.New("weights must be finite and non-negative")
		}
		sum += v
	}
	if sum <= 0 {
		return errors.New("at least one weight must be greater than zero")
	}
	return nil
}

func normalizedWeights(w domain.Weights) domain.Weights {
	sum := w.Complexity + w.Churn + w.Frequency
	return domain.Weights{Complexity: w.Complexity / sum, Churn: w.Churn / sum, Frequency: w.Frequency / sum}
}

// Apply modifies the slice elements with normalized values, contributions and
// scores. It only compares syntactically valid files within this analysis.
func Apply(files []domain.FileResult, weights domain.Weights) []string {
	if err := ValidateWeights(weights); err != nil {
		return []string{err.Error()}
	}
	eligible := make([]int, 0, len(files))
	for i := range files {
		files[i].Module = ModuleForPath(files[i].Path)
		if files[i].Static.AnalysisStatus == "" || files[i].Static.AnalysisStatus == "ok" {
			if files[i].Static.MaxFunctionComplexity != nil {
				eligible = append(eligible, i)
			}
		}
	}
	if len(eligible) < 2 {
		for _, i := range eligible {
			setReason(&files[i], "کمتر از دو فایل قابل مقایسه برای امتیازدهی وجود دارد.")
		}
		return []string{"fewer than two scorable files; hotspot scores are null"}
	}

	complexity := make([]float64, len(eligible))
	churn := make([]float64, len(eligible))
	frequency := make([]float64, len(eligible))
	for j, i := range eligible {
		complexity[j] = float64(*files[i].Static.MaxFunctionComplexity)
		churn[j] = float64(files[i].Git.Churn)
		frequency[j] = float64(files[i].Git.ChangeFrequency)
	}
	cn, cVar := normalize(complexity)
	hn, hVar := normalize(churn)
	fn, fVar := normalize(frequency)
	warnings := []string{}
	if !cVar {
		warnings = append(warnings, "complexity has no differentiating value; its normalized value is zero")
	}
	if !hVar {
		warnings = append(warnings, "churn has no differentiating value; its normalized value is zero")
	}
	if !fVar {
		warnings = append(warnings, "change frequency has no differentiating value; its normalized value is zero")
	}
	if !cVar && !hVar && !fVar {
		for _, i := range eligible {
			setReason(&files[i], "هیچ‌یک از شاخص‌ها میان فایل‌های قابل مقایسه قدرت تفکیک ندارد.")
		}
		return append(warnings, "no metric differentiates eligible files; hotspot scores are null")
	}

	effective := normalizedWeights(weights)
	for j, i := range eligible {
		f := &files[i]
		f.Normalized = domain.NormalizedMetrics{Complexity: floatp(cn[j]), Churn: floatp(hn[j]), Frequency: floatp(fn[j])}
		f.EffectiveWeight = effective
		cc := 100 * effective.Complexity * cn[j]
		hc := 100 * effective.Churn * hn[j]
		fc := 100 * effective.Frequency * fn[j]
		f.Contributions = domain.Contributions{Complexity: floatp(cc), Churn: floatp(hc), Frequency: floatp(fc)}
		score := cc + hc + fc
		f.HotspotScore = floatp(score)
		f.ScoreReason = nil
	}
	return warnings
}

func normalize(values []float64) ([]float64, bool) {
	if len(values) == 0 {
		return nil, false
	}
	transformed := make([]float64, len(values))
	min, max := math.Inf(1), math.Inf(-1)
	for i, v := range values {
		if v < 0 {
			v = 0
		}
		transformed[i] = math.Log1p(v)
		min = math.Min(min, transformed[i])
		max = math.Max(max, transformed[i])
	}
	result := make([]float64, len(values))
	if max == min {
		return result, false
	}
	for i := range transformed {
		result[i] = (transformed[i] - min) / (max - min)
	}
	return result, true
}

func floatp(v float64) *float64                     { return &v }
func setReason(f *domain.FileResult, reason string) { f.HotspotScore = nil; f.ScoreReason = &reason }

func ModuleForPath(filePath string) string {
	dir := path.Dir(filePath)
	if dir == "." || dir == "" {
		return "."
	}
	first := dir
	for {
		next := path.Dir(first)
		if next == "." || next == "/" {
			return first
		}
		first = next
	}
}

func RankFiles(files []domain.FileResult) {
	sort.SliceStable(files, func(i, j int) bool {
		if files[i].HotspotScore == nil {
			return false
		}
		if files[j].HotspotScore == nil {
			return true
		}
		if *files[i].HotspotScore == *files[j].HotspotScore {
			return files[i].Path < files[j].Path
		}
		return *files[i].HotspotScore > *files[j].HotspotScore
	})
}

func ValidateLanguage(language string) error {
	for _, v := range []string{"go", "python", "javascript"} {
		if language == v {
			return nil
		}
	}
	return fmt.Errorf("unsupported language %q", language)
}
