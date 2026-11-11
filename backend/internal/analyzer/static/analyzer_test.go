package static

import (
	"strings"
	"testing"
)

func TestAnalyzeGoCountsLOCStringsFunctionsAndDecisions(t *testing.T) {
	source := []byte(`package demo

// only a comment
func simple() { println("// literal") } // trailing comment
func complex(items []int, ok bool) int {
  total := 0
  for _, item := range items {
    if ok && item > 1 { total += item }
  }
  return total
}
`)
	result := Analyze("sample.go", source)
	if result.AnalysisStatus != "ok" {
		t.Fatalf("status=%s: %v", result.AnalysisStatus, result.AnalysisError)
	}
	if *result.TotalLines != 11 || *result.LOC != 9 {
		t.Fatalf("lines total=%d loc=%d", *result.TotalLines, *result.LOC)
	}
	if *result.FunctionCount != 2 || *result.MaxFunctionComplexity != 4 {
		t.Fatalf("functions=%d max complexity=%d", *result.FunctionCount, *result.MaxFunctionComplexity)
	}
	if result.Classes != nil {
		t.Fatal("Go classes must be not-applicable (nil), not an invented zero")
	}
}

func TestAnalyzePythonNestedFunctionIsNotAddedToParent(t *testing.T) {
	source := []byte(`class Engine:
    def run(self, values):
        def helper(v):
            return v if v else 0
        for value in values:
            if value and helper(value):
                return value
        return 0
`)
	result := Analyze("engine.py", source)
	if result.AnalysisStatus != "ok" {
		t.Fatalf("status=%s", result.AnalysisStatus)
	}
	if *result.FunctionCount != 2 || len(result.Classes) != 1 {
		t.Fatalf("unexpected functions/classes: %d/%d", *result.FunctionCount, len(result.Classes))
	}
	byName := map[string]int{}
	for _, fn := range result.Functions {
		byName[fn.Name] = fn.Complexity
	}
	if byName["run"] != 4 {
		t.Fatalf("parent complexity got nested decisions or wrong count: %#v", byName)
	}
	if byName["helper"] != 2 {
		t.Fatalf("helper complexity: %#v", byName)
	}
}

func TestAnalyzeJavaScriptCountsLogicalOperatorsAndClasses(t *testing.T) {
	source := []byte(`class Checkout {
  process(items, enabled) {
    if (enabled && items.length) return items[0]
    return null
  }
}
`)
	result := Analyze("checkout.js", source)
	if result.AnalysisStatus != "ok" {
		t.Fatalf("status=%s", result.AnalysisStatus)
	}
	if *result.MaxFunctionComplexity != 3 || len(result.Classes) != 1 {
		t.Fatalf("complexity/classes: %v/%d", result.MaxFunctionComplexity, len(result.Classes))
	}
}

func TestAnalyzeSyntaxErrorMakesDerivedMetricsNull(t *testing.T) {
	result := Analyze("broken.py", []byte("def nope(:\n  pass\n"))
	if result.AnalysisStatus != "parse_error" {
		t.Fatalf("status=%s", result.AnalysisStatus)
	}
	if result.LOC != nil || result.MaxFunctionComplexity != nil || result.FunctionCount != nil {
		t.Fatalf("derived metrics must be null: %#v", result)
	}
	if result.AnalysisError == nil || !strings.Contains(*result.AnalysisError, "syntax") {
		t.Fatalf("safe parse error missing: %#v", result.AnalysisError)
	}
}

func TestAnalyzeRejectsUnsupportedAndOversizedInput(t *testing.T) {
	invalid := Analyze("broken.go", []byte{0xff, 0xfe})
	if invalid.AnalysisStatus != "skipped" || invalid.AnalysisError == nil {
		t.Fatalf("invalid encoding result %#v", invalid)
	}
	unsupported := Analyze("readme.txt", []byte("hello"))
	if unsupported.AnalysisStatus != "skipped" {
		t.Fatalf("unexpected status %s", unsupported.AnalysisStatus)
	}
	old := MaxFileBytes
	MaxFileBytes = 4
	defer func() { MaxFileBytes = old }()
	large := Analyze("large.go", []byte("package x"))
	if large.AnalysisStatus != "skipped" || large.AnalysisError == nil {
		t.Fatalf("large result %#v", large)
	}
}
