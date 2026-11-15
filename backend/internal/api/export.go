package api

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

const csvContentType = "text/csv; charset=utf-8"

func writeAnalysisJSONExport(response http.ResponseWriter, report domain.AnalysisReport) {
	payload, err := json.Marshal(report)
	if err != nil {
		writeError(response, http.StatusInternalServerError, "EXPORT_ERROR", "The completed analysis report could not be exported.")
		return
	}
	response.Header().Set("Content-Type", jsonContentType)
	response.Header().Set("Content-Disposition", exportDisposition(report.Analysis.ID, "json"))
	response.WriteHeader(http.StatusOK)
	_, _ = response.Write(payload)
}

func writeAnalysisCSVExport(response http.ResponseWriter, report domain.AnalysisReport) {
	response.Header().Set("Content-Type", csvContentType)
	response.Header().Set("Content-Disposition", exportDisposition(report.Analysis.ID, "csv"))
	response.WriteHeader(http.StatusOK)

	writer := csv.NewWriter(response)
	_ = writer.Write([]string{
		"path",
		"language",
		"analysis_status",
		"risk_score",
		"risk_level",
		"lines_of_code",
		"function_count",
		"class_count",
		"max_function_length",
		"total_function_length",
		"max_class_length",
		"total_class_length",
		"git_churn",
		"change_frequency",
	})
	if report.Artifacts.Static == nil {
		writer.Flush()
		return
	}

	gitByPath := make(map[string]domain.GitFileMetrics)
	if report.Artifacts.Git != nil {
		for _, file := range report.Artifacts.Git.Files {
			gitByPath[file.Path] = file
		}
	}
	riskByPath := make(map[string]domain.Hotspot)
	if report.Artifacts.Risk != nil {
		for _, hotspot := range report.Artifacts.Risk.Files {
			riskByPath[hotspot.Path] = hotspot
		}
	}
	for _, file := range report.Artifacts.Static.Files {
		risk, hasRisk := riskByPath[file.Path]
		git, hasGit := gitByPath[file.Path]
		record := []string{
			spreadsheetSafe(file.Path),
			string(file.Language),
			string(file.Status),
			"",
			"",
			strconv.Itoa(file.Metrics.LinesOfCode),
			strconv.Itoa(file.Metrics.FunctionCount),
			strconv.Itoa(file.Metrics.ClassCount),
			strconv.Itoa(file.Metrics.FunctionLength.Max),
			strconv.Itoa(file.Metrics.FunctionLength.Total),
			strconv.Itoa(file.Metrics.ClassLength.Max),
			strconv.Itoa(file.Metrics.ClassLength.Total),
			"",
			"",
		}
		if hasRisk {
			record[3] = strconv.Itoa(risk.Score)
			record[4] = string(risk.Level)
		}
		if hasGit {
			record[12] = strconv.Itoa(git.GitChurn)
			record[13] = strconv.Itoa(git.ChangeFrequency)
		}
		_ = writer.Write(record)
	}
	writer.Flush()
}

func exportDisposition(id domain.AnalysisID, extension string) string {
	identifier := strings.Map(func(character rune) rune {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '_' || character == '-' {
			return character
		}
		return '_'
	}, string(id))
	if identifier == "" {
		identifier = "analysis"
	}
	return "attachment; filename=\"" + identifier + "." + extension + "\""
}

func spreadsheetSafe(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed == "" {
		return value
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		return "'" + value
	default:
		return value
	}
}
