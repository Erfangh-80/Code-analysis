package api

import (
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

func TestCSVExportNeutralizesSpreadsheetFormulaPaths(t *testing.T) {
	report := domain.AnalysisReport{
		Analysis: domain.AnalysisRun{ID: "analysis_export"},
		Artifacts: domain.AnalysisArtifacts{Static: &domain.BatchAnalysis{Files: []domain.FileAnalysis{
			{Path: "=HYPERLINK(\"https://untrusted.example.test\")", Language: domain.LanguageGo, Status: domain.AnalysisStatusAnalyzed},
			{Path: "+formula.go", Language: domain.LanguageGo, Status: domain.AnalysisStatusAnalyzed},
			{Path: "-formula.go", Language: domain.LanguageGo, Status: domain.AnalysisStatusAnalyzed},
			{Path: "@formula.go", Language: domain.LanguageGo, Status: domain.AnalysisStatusAnalyzed},
			{Path: " \t=trimmed-formula.go", Language: domain.LanguageGo, Status: domain.AnalysisStatusAnalyzed},
			{Path: "safe.go", Language: domain.LanguageGo, Status: domain.AnalysisStatusAnalyzed},
		}}},
	}

	response := httptest.NewRecorder()
	writeAnalysisCSVExport(response, report)
	if response.Code != http.StatusOK {
		t.Fatalf("CSV export status=%d body=%s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Disposition"); got != "attachment; filename=\"analysis_export.csv\"" {
		t.Errorf("Content-Disposition=%q", got)
	}
	records, err := csv.NewReader(strings.NewReader(response.Body.String())).ReadAll()
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}
	wantPaths := []string{"'=HYPERLINK(\"https://untrusted.example.test\")", "'+formula.go", "'-formula.go", "'@formula.go", "' \t=trimmed-formula.go", "safe.go"}
	if len(records) != len(wantPaths)+1 {
		t.Fatalf("CSV records=%#v", records)
	}
	for index, want := range wantPaths {
		if got := records[index+1][0]; got != want {
			t.Errorf("CSV path at row %d=%q, want %q", index+1, got, want)
		}
	}
}

func TestExportDispositionRemovesHeaderControlCharacters(t *testing.T) {
	value := exportDisposition(domain.AnalysisID("analysis\"\r\nX-Injected: yes"), "csv")
	if strings.ContainsAny(value, "\r\n") || value != "attachment; filename=\"analysis___X-Injected__yes.csv\"" {
		t.Errorf("unsafe content disposition %q", value)
	}
}
