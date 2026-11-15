package api

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

func (handler *handler) handleCancelAnalysis(response http.ResponseWriter, request *http.Request, id domain.AnalysisID) {
	if !requireMethod(response, request, http.MethodPost) {
		return
	}
	if err := requireOnlyQueryParameters(request, nil); err != nil {
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "This analysis cancellation endpoint does not accept query parameters.")
		return
	}
	run, err := handler.analyses.Cancel(request.Context(), id)
	if err != nil {
		handleAnalysisError(response, err)
		return
	}
	writeJSON(response, http.StatusAccepted, map[string]domain.AnalysisRun{"analysis": run})
}

func (handler *handler) handleAnalysisReport(response http.ResponseWriter, request *http.Request, id domain.AnalysisID) {
	if !requireMethod(response, request, http.MethodGet) {
		return
	}
	if err := requireOnlyQueryParameters(request, nil); err != nil {
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "This completed report does not accept query parameters.")
		return
	}
	report, err := handler.analyses.Report(request.Context(), id)
	if err != nil {
		handleAnalysisError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, report)
}

func (handler *handler) handleAnalysisSummary(response http.ResponseWriter, request *http.Request, id domain.AnalysisID) {
	if !requireMethod(response, request, http.MethodGet) {
		return
	}
	if err := requireOnlyQueryParameters(request, nil); err != nil {
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "This analysis summary does not accept query parameters.")
		return
	}
	summary, err := handler.analyses.Summary(request.Context(), id)
	if err != nil {
		handleAnalysisError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, summary)
}

func (handler *handler) handleAnalysisHotspots(response http.ResponseWriter, request *http.Request, id domain.AnalysisID) {
	if !requireMethod(response, request, http.MethodGet) {
		return
	}
	query, err := decodeResultListQuery(request, false, hotspotSorts())
	if err != nil {
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "The hotspot filters, sort order, or pagination parameters are invalid.")
		return
	}
	page, err := handler.analyses.Hotspots(request.Context(), id, query)
	if err != nil {
		handleAnalysisError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, page)
}

func (handler *handler) handleAnalysisFiles(response http.ResponseWriter, request *http.Request, id domain.AnalysisID) {
	if !requireMethod(response, request, http.MethodGet) {
		return
	}
	query, err := decodeResultListQuery(request, true, fileSorts())
	if err != nil {
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "The file filters, sort order, or pagination parameters are invalid.")
		return
	}
	page, err := handler.analyses.Files(request.Context(), id, query)
	if err != nil {
		handleAnalysisError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, page)
}

func (handler *handler) handleAnalysisFile(response http.ResponseWriter, request *http.Request, id domain.AnalysisID) {
	if !requireMethod(response, request, http.MethodGet) {
		return
	}
	if err := requireOnlyQueryParameters(request, map[string]struct{}{"path": {}}); err != nil {
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "Provide one project-relative path query parameter.")
		return
	}
	path, err := singleQueryValue(request, "path", true)
	if err != nil {
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "Provide one project-relative path query parameter.")
		return
	}
	path, err = normalizeProjectRelativePath(path, false)
	if err != nil {
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "Provide one project-relative path query parameter.")
		return
	}
	result, err := handler.analyses.File(request.Context(), id, path)
	if err != nil {
		handleAnalysisError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, result)
}

func (handler *handler) handleAnalysisModules(response http.ResponseWriter, request *http.Request, id domain.AnalysisID) {
	if !requireMethod(response, request, http.MethodGet) {
		return
	}
	if err := requireOnlyQueryParameters(request, nil); err != nil {
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "This module view does not accept query parameters.")
		return
	}
	modules, err := handler.analyses.Modules(request.Context(), id)
	if err != nil {
		handleAnalysisError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, modules)
}

func (handler *handler) handleAnalysisHeatmap(response http.ResponseWriter, request *http.Request, id domain.AnalysisID) {
	if !requireMethod(response, request, http.MethodGet) {
		return
	}
	if err := requireOnlyQueryParameters(request, nil); err != nil {
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "The saved heatmap does not accept alternate axes.")
		return
	}
	heatmap, err := handler.analyses.Heatmap(request.Context(), id)
	if err != nil {
		handleAnalysisError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, heatmap)
}

func (handler *handler) handleAnalysisExport(response http.ResponseWriter, request *http.Request, id domain.AnalysisID) {
	if !requireMethod(response, request, http.MethodGet) {
		return
	}
	if err := requireOnlyQueryParameters(request, map[string]struct{}{"format": {}}); err != nil {
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "Provide one export format: json or csv.")
		return
	}
	format, err := singleQueryValue(request, "format", true)
	if err != nil || (format != "json" && format != "csv") {
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "Provide one export format: json or csv.")
		return
	}
	report, err := handler.analyses.Report(request.Context(), id)
	if err != nil {
		handleAnalysisError(response, err)
		return
	}
	if format == "json" {
		writeAnalysisJSONExport(response, report)
		return
	}
	writeAnalysisCSVExport(response, report)
}

func decodeResultListQuery(request *http.Request, allowAnalysisStatus bool, allowedSorts map[domain.ResultSort]struct{}) (domain.ResultListQuery, error) {
	allowedParameters := map[string]struct{}{
		"page":      {},
		"pageSize":  {},
		"q":         {},
		"module":    {},
		"riskLevel": {},
		"sort":      {},
		"direction": {},
	}
	if allowAnalysisStatus {
		allowedParameters["analysisStatus"] = struct{}{}
	}
	if err := requireOnlyQueryParameters(request, allowedParameters); err != nil {
		return domain.ResultListQuery{}, err
	}

	query := domain.ResultListQuery{Page: 1, PageSize: 50}
	var err error
	if query.Query, err = singleQueryValue(request, "q", false); err != nil || len(query.Query) > 512 {
		return domain.ResultListQuery{}, io.ErrUnexpectedEOF
	}
	module, err := singleQueryValue(request, "module", false)
	if err != nil || len(module) > 512 {
		return domain.ResultListQuery{}, io.ErrUnexpectedEOF
	}
	if module != "" {
		module, err = normalizeProjectRelativePath(module, true)
		if err != nil {
			return domain.ResultListQuery{}, err
		}
		query.Module = module
	}
	if _, exists := request.URL.Query()["page"]; exists {
		value, valueErr := singleQueryValue(request, "page", true)
		if valueErr != nil || !parseResultPage(value, &query.Page, 1, 2147483647) {
			return domain.ResultListQuery{}, io.ErrUnexpectedEOF
		}
	}
	if _, exists := request.URL.Query()["pageSize"]; exists {
		value, valueErr := singleQueryValue(request, "pageSize", true)
		if valueErr != nil || !parseResultPage(value, &query.PageSize, 1, 200) {
			return domain.ResultListQuery{}, io.ErrUnexpectedEOF
		}
	}
	if _, exists := request.URL.Query()["sort"]; exists {
		value, valueErr := singleQueryValue(request, "sort", true)
		if valueErr != nil || value == "" {
			return domain.ResultListQuery{}, io.ErrUnexpectedEOF
		}
		query.Sort = domain.ResultSort(value)
		if _, exists := allowedSorts[query.Sort]; !exists {
			return domain.ResultListQuery{}, io.ErrUnexpectedEOF
		}
	}
	if _, exists := request.URL.Query()["direction"]; exists {
		value, valueErr := singleQueryValue(request, "direction", true)
		if valueErr != nil || value == "" {
			return domain.ResultListQuery{}, io.ErrUnexpectedEOF
		}
		query.Direction = domain.SortDirection(value)
		if query.Direction != domain.SortDirectionAscending && query.Direction != domain.SortDirectionDescending {
			return domain.ResultListQuery{}, io.ErrUnexpectedEOF
		}
	}

	seenRiskLevels := make(map[domain.RiskLevel]struct{})
	for _, value := range request.URL.Query()["riskLevel"] {
		level := domain.RiskLevel(value)
		if (level != domain.RiskLevelLow && level != domain.RiskLevelMedium && level != domain.RiskLevelHigh) || value == "" {
			return domain.ResultListQuery{}, io.ErrUnexpectedEOF
		}
		if _, exists := seenRiskLevels[level]; !exists {
			seenRiskLevels[level] = struct{}{}
			query.RiskLevels = append(query.RiskLevels, level)
		}
	}
	if allowAnalysisStatus {
		seenStatuses := make(map[domain.AnalysisStatus]struct{})
		for _, value := range request.URL.Query()["analysisStatus"] {
			status := domain.AnalysisStatus(value)
			switch status {
			case domain.AnalysisStatusAnalyzed, domain.AnalysisStatusEmpty, domain.AnalysisStatusNoExecutable, domain.AnalysisStatusFallback, domain.AnalysisStatusParseError, domain.AnalysisStatusUnsupported, domain.AnalysisStatusReadError:
			default:
				return domain.ResultListQuery{}, io.ErrUnexpectedEOF
			}
			if _, exists := seenStatuses[status]; !exists {
				seenStatuses[status] = struct{}{}
				query.AnalysisStatuses = append(query.AnalysisStatuses, status)
			}
		}
	}
	return query, nil
}

func parseResultPage(value string, destination *int, minimum, maximum int) bool {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minimum || parsed > maximum {
		return false
	}
	*destination = parsed
	return true
}

func requireOnlyQueryParameters(request *http.Request, allowed map[string]struct{}) error {
	for key := range request.URL.Query() {
		if _, exists := allowed[key]; !exists {
			return errors.New("unknown query parameter")
		}
	}
	return nil
}

func singleQueryValue(request *http.Request, key string, required bool) (string, error) {
	values, exists := request.URL.Query()[key]
	if !exists {
		if required {
			return "", io.ErrUnexpectedEOF
		}
		return "", nil
	}
	if len(values) != 1 {
		return "", io.ErrUnexpectedEOF
	}
	return values[0], nil
}

func normalizeProjectRelativePath(value string, allowRoot bool) (string, error) {
	if value == "" || strings.ContainsRune(value, 0) || strings.Contains(value, "\\") {
		return "", io.ErrUnexpectedEOF
	}
	cleaned := filepath.ToSlash(filepath.Clean(value))
	if allowRoot && cleaned == "." {
		return cleaned, nil
	}
	if cleaned == "." || cleaned == ".." || filepath.IsAbs(value) || strings.HasPrefix(cleaned, "../") {
		return "", io.ErrUnexpectedEOF
	}
	return cleaned, nil
}

func hotspotSorts() map[domain.ResultSort]struct{} {
	return map[domain.ResultSort]struct{}{
		domain.ResultSortScore:             {},
		domain.ResultSortPath:              {},
		domain.ResultSortModule:            {},
		domain.ResultSortLinesOfCode:       {},
		domain.ResultSortMaxFunctionLength: {},
		domain.ResultSortMaxClassLength:    {},
		domain.ResultSortGitChurn:          {},
		domain.ResultSortChangeFrequency:   {},
	}
}

func fileSorts() map[domain.ResultSort]struct{} {
	result := hotspotSorts()
	result[domain.ResultSortLanguage] = struct{}{}
	result[domain.ResultSortStatus] = struct{}{}
	return result
}
