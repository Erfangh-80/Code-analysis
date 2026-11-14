package scoring

import (
	"math"
	"testing"

	"github.com/Erfangh-80/Code-analysis/backend/internal/domain"
)

func intp(v int) *int { return &v }

func TestApplyUsesLogNormalizationAndExplainsScore(t *testing.T) {
	files := []domain.FileResult{
		{Path: "low.go", Static: domain.StaticMetrics{MaxFunctionComplexity: intp(2)}, Git: domain.GitMetrics{Churn: 10, ChangeFrequency: 1}},
		{Path: "high.go", Static: domain.StaticMetrics{MaxFunctionComplexity: intp(10)}, Git: domain.GitMetrics{Churn: 110, ChangeFrequency: 6}},
	}
	warnings := Apply(files, domain.DefaultWeights())
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if files[0].HotspotScore == nil || files[1].HotspotScore == nil {
		t.Fatal("both comparable files must receive a score")
	}
	if *files[0].HotspotScore != 0 || *files[1].HotspotScore != 100 {
		t.Fatalf("expected 0 and 100 scores, got %.2f and %.2f", *files[0].HotspotScore, *files[1].HotspotScore)
	}
	if files[1].Contributions.Complexity == nil || math.Abs(*files[1].Contributions.Complexity-50) > .001 {
		t.Fatalf("expected complexity contribution of 50, got %#v", files[1].Contributions.Complexity)
	}
}

func TestApplyRejectsInvalidWeights(t *testing.T) {
	for _, weights := range []domain.Weights{{Complexity: -1, Churn: 1}, {}, {Complexity: math.NaN()}} {
		if err := ValidateWeights(weights); err == nil {
			t.Fatalf("expected invalid weights for %#v", weights)
		}
	}
}

func TestApplyConstantMetricDoesNotClaimLowRisk(t *testing.T) {
	files := []domain.FileResult{
		{Path: "a.py", Static: domain.StaticMetrics{MaxFunctionComplexity: intp(5)}, Git: domain.GitMetrics{Churn: 1, ChangeFrequency: 1}},
		{Path: "b.py", Static: domain.StaticMetrics{MaxFunctionComplexity: intp(5)}, Git: domain.GitMetrics{Churn: 1, ChangeFrequency: 1}},
	}
	warnings := Apply(files, domain.DefaultWeights())
	if files[0].HotspotScore != nil || files[1].HotspotScore != nil {
		t.Fatal("indistinguishable files must have null score")
	}
	if len(warnings) == 0 {
		t.Fatal("must disclose that no metric differentiates files")
	}
}

func TestApplyExcludesParseFailures(t *testing.T) {
	files := []domain.FileResult{
		{Path: "bad.js", Static: domain.StaticMetrics{AnalysisStatus: "parse_error", MaxFunctionComplexity: nil}},
		{Path: "only.js", Static: domain.StaticMetrics{AnalysisStatus: "ok", MaxFunctionComplexity: intp(3)}},
	}
	Apply(files, domain.DefaultWeights())
	if files[0].HotspotScore != nil || files[1].HotspotScore != nil {
		t.Fatal("parse errors and a sole comparable file must not get a score")
	}
}

func TestModuleForPath(t *testing.T) {
	cases := map[string]string{"main.go": ".", "src/api/a.go": "src", "": "."}
	for path, want := range cases {
		if got := ModuleForPath(path); got != want {
			t.Errorf("ModuleForPath(%q)=%q want %q", path, got, want)
		}
	}
}
