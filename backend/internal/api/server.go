// Package api exposes the local HTTP boundary for Code Risk Analyzer.
package api

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Erfangh-80/code-analysis/backend/internal/analysisguard"
	"github.com/Erfangh-80/code-analysis/backend/internal/analyzer/static"
	"github.com/Erfangh-80/code-analysis/backend/internal/application"
	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
	"github.com/Erfangh-80/code-analysis/backend/internal/githistory"
	"github.com/Erfangh-80/code-analysis/backend/internal/scoring"
	"github.com/Erfangh-80/code-analysis/backend/internal/storage/failedfiles"
)

const jsonContentType = "application/json; charset=utf-8"
const yamlContentType = "application/yaml; charset=utf-8"

//go:embed openapi.yaml
var openAPISpecification []byte

// HealthResponse is the contract returned by the local service health endpoint.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// ErrorResponse is the standard safe error envelope for API routes.
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// APIError describes a client-safe API error.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type staticAnalysisRequest struct {
	RootPath string   `json:"rootPath"`
	Files    []string `json:"files"`
}

type staticAnalysisFailureResponse struct {
	Items []domain.FailureRecord `json:"items"`
}

type handler struct {
	version        string
	staticAnalysis *application.StaticAnalysisService
	gitHistory     *githistory.Analyzer
	riskEngine     *scoring.Engine
	analyses       *application.Orchestrator
}

// NewHandler returns the local v1 API handler. The static-analysis API reads
// only requested local files and never executes repository source.
func NewHandler(version string) http.Handler {
	return newHandler(version, nil, nil, nil)
}

// NewHandlerWithAnalysisService composes the local HTTP API with the durable
// asynchronous analysis orchestration service.
func NewHandlerWithAnalysisService(version string, analyses *application.Orchestrator) http.Handler {
	return newHandler(version, analyses, nil, nil)
}

// NewHandlerWithServices composes the HTTP boundary with server-configured,
// guarded static and Git services. The production composition root uses this
// constructor so direct API endpoints obey the same path/resource policy.
func NewHandlerWithServices(version string, analyses *application.Orchestrator, staticAnalysis *application.StaticAnalysisService, gitHistory *githistory.Analyzer) http.Handler {
	return newHandler(version, analyses, staticAnalysis, gitHistory)
}

func newHandler(version string, analyses *application.Orchestrator, staticAnalysis *application.StaticAnalysisService, gitHistory *githistory.Analyzer) http.Handler {
	if staticAnalysis == nil {
		staticAnalysis = application.NewStaticAnalysisService(static.NewAnalyzer(), failedfiles.NewRegistry())
	}
	if gitHistory == nil {
		gitHistory = githistory.NewAnalyzer()
	}
	return &handler{
		version:        version,
		staticAnalysis: staticAnalysis,
		gitHistory:     gitHistory,
		riskEngine:     scoring.NewEngine(),
		analyses:       analyses,
	}
}

func (handler *handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	setSecurityHeaders(response)
	if handler.handleCORS(response, request) {
		return
	}
	if request.URL.Path == "/api/v1/analyses" || strings.HasPrefix(request.URL.Path, "/api/v1/analyses/") {
		handler.handleAnalyses(response, request)
		return
	}

	switch request.URL.Path {
	case "/api/v1/health":
		handler.handleHealth(response, request)
	case "/api/v1/openapi.yaml":
		handler.handleOpenAPI(response, request)
	case "/api/v1/static-analysis":
		handler.handleStaticAnalysis(response, request)
	case "/api/v1/static-analysis/failures":
		handler.handleStaticAnalysisFailures(response, request)
	case "/api/v1/git-analysis":
		handler.handleGitAnalysis(response, request)
	case "/api/v1/risk-analysis":
		handler.handleRiskAnalysis(response, request)
	default:
		if request.URL.Path == "/swagger" || strings.HasPrefix(request.URL.Path, "/swagger/") {
			handler.handleSwaggerUI(response, request)
			return
		}
		writeError(response, http.StatusNotFound, "NOT_FOUND", "The requested API route does not exist.")
	}
}

func (handler *handler) handleHealth(response http.ResponseWriter, request *http.Request) {
	if !requireMethod(response, request, http.MethodGet) {
		return
	}
	writeJSON(response, http.StatusOK, HealthResponse{
		Status:  "ok",
		Version: handler.version,
	})
}

func (handler *handler) handleOpenAPI(response http.ResponseWriter, request *http.Request) {
	if !requireMethod(response, request, http.MethodGet) {
		return
	}
	response.Header().Set("Content-Type", yamlContentType)
	response.WriteHeader(http.StatusOK)
	_, _ = response.Write(openAPISpecification)
}

func (handler *handler) handleStaticAnalysis(response http.ResponseWriter, request *http.Request) {
	if !requireMethod(response, request, http.MethodPost) {
		return
	}

	payload, err := decodeStaticAnalysisRequest(request)
	if err != nil {
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "Provide an absolute local rootPath and one or more relative file paths.")
		return
	}
	result, err := handler.staticAnalysis.Analyze(request.Context(), application.StaticAnalysisRequest{
		RootPath: payload.RootPath,
		Files:    payload.Files,
	})
	if err != nil {
		if errors.Is(err, analysisguard.ErrResourceLimit) {
			writeError(response, http.StatusRequestEntityTooLarge, "RESOURCE_LIMIT_EXCEEDED", "The requested local analysis exceeds a configured resource limit.")
			return
		}
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "The requested local analysis paths are invalid or inaccessible.")
		return
	}
	writeJSON(response, http.StatusOK, result)
}

func (handler *handler) handleGitAnalysis(response http.ResponseWriter, request *http.Request) {
	if !requireMethod(response, request, http.MethodPost) {
		return
	}

	payload, err := decodeGitAnalysisRequest(request)
	if err != nil {
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "Provide an absolute local repositoryPath and a valid optional history range.")
		return
	}
	report, err := handler.gitHistory.Analyze(request.Context(), payload)
	if err != nil {
		if errors.Is(err, githistory.ErrResourceLimit) {
			writeError(response, http.StatusRequestEntityTooLarge, "RESOURCE_LIMIT_EXCEEDED", "The requested local Git analysis exceeds a configured resource limit.")
			return
		}
		if errors.Is(err, githistory.ErrInvalidRepository) || errors.Is(err, githistory.ErrInvalidHistoryRange) {
			writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "The local Git repository path or history range is invalid.")
			return
		}
		writeError(response, http.StatusInternalServerError, "GIT_HISTORY_ERROR", "The local Git history could not be analyzed completely.")
		return
	}
	writeJSON(response, http.StatusOK, report)
}

func (handler *handler) handleRiskAnalysis(response http.ResponseWriter, request *http.Request) {
	if !requireMethod(response, request, http.MethodPost) {
		return
	}

	payload, err := decodeRiskAnalysisRequest(request)
	if err != nil {
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "Provide one or more file metric records and an optional valid scoring profile.")
		return
	}
	report, err := handler.riskEngine.Analyze(payload)
	if err != nil {
		if errors.Is(err, scoring.ErrInvalidInput) || errors.Is(err, scoring.ErrInvalidProfile) || errors.Is(err, scoring.ErrInvalidHeatmap) {
			writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "The risk metrics, scoring profile, or heatmap axes are invalid.")
			return
		}
		writeError(response, http.StatusInternalServerError, "RISK_ENGINE_ERROR", "The risk score could not be calculated.")
		return
	}
	writeJSON(response, http.StatusOK, report)
}

func (handler *handler) handleAnalyses(response http.ResponseWriter, request *http.Request) {
	if handler.analyses == nil {
		writeError(response, http.StatusServiceUnavailable, "ANALYSIS_UNAVAILABLE", "Durable analysis orchestration is not configured for this handler.")
		return
	}
	if request.URL.Path == "/api/v1/analyses" {
		switch request.Method {
		case http.MethodPost:
			handler.handleStartAnalysis(response, request)
		case http.MethodGet:
			handler.handleListAnalyses(response, request)
		default:
			response.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
			writeError(response, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "This HTTP method is not supported for the requested route.")
		}
		return
	}

	parts := strings.Split(strings.TrimPrefix(request.URL.Path, "/api/v1/analyses/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" || len(parts) > 3 {
		writeError(response, http.StatusNotFound, "NOT_FOUND", "The requested API route does not exist.")
		return
	}
	id := domain.AnalysisID(parts[0])
	if len(parts) == 1 {
		if !requireMethod(response, request, http.MethodGet) {
			return
		}
		if err := requireOnlyQueryParameters(request, nil); err != nil {
			writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "This analysis status endpoint does not accept query parameters.")
			return
		}
		run, err := handler.analyses.Get(request.Context(), id)
		if err != nil {
			handleAnalysisError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, map[string]domain.AnalysisRun{"analysis": run})
		return
	}

	if len(parts) == 3 {
		if parts[1] == "files" && parts[2] == "detail" {
			handler.handleAnalysisFile(response, request, id)
			return
		}
		writeError(response, http.StatusNotFound, "NOT_FOUND", "The requested API route does not exist.")
		return
	}

	switch parts[1] {
	case "cancel":
		handler.handleCancelAnalysis(response, request, id)
	case "report":
		handler.handleAnalysisReport(response, request, id)
	case "summary":
		handler.handleAnalysisSummary(response, request, id)
	case "hotspots":
		handler.handleAnalysisHotspots(response, request, id)
	case "files":
		handler.handleAnalysisFiles(response, request, id)
	case "modules":
		handler.handleAnalysisModules(response, request, id)
	case "heatmap":
		handler.handleAnalysisHeatmap(response, request, id)
	case "export":
		handler.handleAnalysisExport(response, request, id)
	default:
		writeError(response, http.StatusNotFound, "NOT_FOUND", "The requested API route does not exist.")
	}
}

func (handler *handler) handleStartAnalysis(response http.ResponseWriter, request *http.Request) {
	payload, err := decodeAnalysisRequest(request)
	if err != nil {
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "Provide an absolute local projectPath and valid optional history/profile settings.")
		return
	}
	run, err := handler.analyses.Start(request.Context(), payload)
	if err != nil {
		handleAnalysisError(response, err)
		return
	}
	writeJSON(response, http.StatusAccepted, map[string]domain.AnalysisRun{"analysis": run})
}

func (handler *handler) handleListAnalyses(response http.ResponseWriter, request *http.Request) {
	query, err := decodeAnalysisListQuery(request)
	if err != nil {
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "The requested analysis list filters are invalid.")
		return
	}
	page, err := handler.analyses.List(request.Context(), query)
	if err != nil {
		writeError(response, http.StatusInternalServerError, "ANALYSIS_STORE_ERROR", "Local analysis records could not be listed.")
		return
	}
	writeJSON(response, http.StatusOK, page)
}

func handleAnalysisError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, githistory.ErrInvalidHistoryRange):
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "The requested local analysis history range is invalid.")
	case errors.Is(err, application.ErrAnalysisNotFound):
		writeError(response, http.StatusNotFound, "ANALYSIS_NOT_FOUND", "The requested local analysis does not exist.")
	case errors.Is(err, application.ErrAnalysisFileNotFound):
		writeError(response, http.StatusNotFound, "ANALYSIS_FILE_NOT_FOUND", "The requested project-relative file is not present in this analysis.")
	case errors.Is(err, application.ErrResultUnavailable):
		writeError(response, http.StatusConflict, "RESULT_UNAVAILABLE", "This completed analysis does not contain the requested result artifact.")
	case errors.Is(err, application.ErrAnalysisAlreadyRunning):
		writeError(response, http.StatusConflict, "ANALYSIS_ALREADY_RUNNING", "An analysis is already active for this local project.")
	case errors.Is(err, application.ErrInvalidRunState):
		writeError(response, http.StatusConflict, "INVALID_RUN_STATE", "This operation is not allowed for the analysis in its current state.")
	default:
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "The local analysis request is invalid or inaccessible.")
	}
}

func (handler *handler) handleStaticAnalysisFailures(response http.ResponseWriter, request *http.Request) {
	if !requireMethod(response, request, http.MethodGet) {
		return
	}
	writeJSON(response, http.StatusOK, staticAnalysisFailureResponse{
		Items: handler.staticAnalysis.FailedFiles(),
	})
}

func requireMethod(response http.ResponseWriter, request *http.Request, expected string) bool {
	if request.Method == expected {
		return true
	}
	response.Header().Set("Allow", expected)
	writeError(response, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "This HTTP method is not supported for the requested route.")
	return false
}

func decodeAnalysisRequest(request *http.Request) (domain.AnalysisRequest, error) {
	var payload domain.AnalysisRequest
	decoder := json.NewDecoder(io.LimitReader(request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return domain.AnalysisRequest{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return domain.AnalysisRequest{}, io.ErrUnexpectedEOF
	}
	if strings.TrimSpace(payload.ProjectPath) == "" {
		return domain.AnalysisRequest{}, io.ErrUnexpectedEOF
	}
	return payload, nil
}

func decodeAnalysisListQuery(request *http.Request) (domain.AnalysisListQuery, error) {
	allowed := map[string]struct{}{"status": {}, "page": {}, "pageSize": {}}
	if err := requireOnlyQueryParameters(request, allowed); err != nil {
		return domain.AnalysisListQuery{}, err
	}

	query := domain.AnalysisListQuery{Page: 1, PageSize: 50}
	if _, exists := request.URL.Query()["status"]; exists {
		value, err := singleQueryValue(request, "status", true)
		if err != nil || value == "" {
			return domain.AnalysisListQuery{}, io.ErrUnexpectedEOF
		}
		status := domain.AnalysisRunStatus(value)
		switch status {
		case domain.AnalysisStatusQueued, domain.AnalysisStatusRunning, domain.AnalysisStatusCompleted, domain.AnalysisStatusFailed, domain.AnalysisStatusCancelled:
			query.Status = &status
		default:
			return domain.AnalysisListQuery{}, io.ErrUnexpectedEOF
		}
	}
	if _, exists := request.URL.Query()["page"]; exists {
		value, err := singleQueryValue(request, "page", true)
		if err != nil {
			return domain.AnalysisListQuery{}, err
		}
		query.Page, err = strconv.Atoi(value)
		if err != nil || query.Page < 1 {
			return domain.AnalysisListQuery{}, io.ErrUnexpectedEOF
		}
	}
	if _, exists := request.URL.Query()["pageSize"]; exists {
		value, err := singleQueryValue(request, "pageSize", true)
		if err != nil {
			return domain.AnalysisListQuery{}, err
		}
		query.PageSize, err = strconv.Atoi(value)
		if err != nil || query.PageSize < 1 || query.PageSize > 200 {
			return domain.AnalysisListQuery{}, io.ErrUnexpectedEOF
		}
	}
	return query, nil
}

func decodeStaticAnalysisRequest(request *http.Request) (staticAnalysisRequest, error) {
	var payload staticAnalysisRequest
	decoder := json.NewDecoder(io.LimitReader(request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return staticAnalysisRequest{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return staticAnalysisRequest{}, io.ErrUnexpectedEOF
	}
	if payload.RootPath == "" || len(payload.Files) == 0 {
		return staticAnalysisRequest{}, io.ErrUnexpectedEOF
	}
	return payload, nil
}

func decodeGitAnalysisRequest(request *http.Request) (domain.GitAnalysisRequest, error) {
	var payload domain.GitAnalysisRequest
	decoder := json.NewDecoder(io.LimitReader(request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return domain.GitAnalysisRequest{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return domain.GitAnalysisRequest{}, io.ErrUnexpectedEOF
	}
	if payload.RepositoryPath == "" {
		return domain.GitAnalysisRequest{}, io.ErrUnexpectedEOF
	}
	return payload, nil
}

func decodeRiskAnalysisRequest(request *http.Request) (domain.RiskAnalysisRequest, error) {
	body, err := io.ReadAll(io.LimitReader(request.Body, 1<<20))
	if err != nil {
		return domain.RiskAnalysisRequest{}, err
	}

	var payload domain.RiskAnalysisRequest
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return domain.RiskAnalysisRequest{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return domain.RiskAnalysisRequest{}, io.ErrUnexpectedEOF
	}
	if len(payload.Files) == 0 {
		return domain.RiskAnalysisRequest{}, io.ErrUnexpectedEOF
	}
	if err := requireRiskMetrics(body, payload.AvailableMetrics); err != nil {
		return domain.RiskAnalysisRequest{}, err
	}
	return payload, nil
}

func requireRiskMetrics(body []byte, available []domain.RiskMetric) error {
	var envelope struct {
		Files []struct {
			Metrics map[string]json.RawMessage `json:"metrics"`
		} `json:"files"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return err
	}

	requiredMetrics := []string{
		"linesOfCode",
		"maxFunctionLength",
		"maxClassLength",
		"gitChurn",
		"changeFrequency",
	}
	if available != nil {
		metricFields := map[domain.RiskMetric]string{
			domain.RiskMetricLinesOfCode:       "linesOfCode",
			domain.RiskMetricMaxFunctionLength: "maxFunctionLength",
			domain.RiskMetricMaxClassLength:    "maxClassLength",
			domain.RiskMetricGitChurn:          "gitChurn",
			domain.RiskMetricChangeFrequency:   "changeFrequency",
		}
		requiredMetrics = make([]string, 0, len(available))
		for _, metric := range available {
			field, exists := metricFields[metric]
			if !exists {
				return io.ErrUnexpectedEOF
			}
			requiredMetrics = append(requiredMetrics, field)
		}
	}
	for _, file := range envelope.Files {
		for _, metric := range requiredMetrics {
			if _, exists := file.Metrics[metric]; !exists {
				return io.ErrUnexpectedEOF
			}
		}
	}
	return nil
}

func writeError(response http.ResponseWriter, status int, code, message string) {
	writeJSON(response, status, ErrorResponse{
		Error: APIError{
			Code:    code,
			Message: message,
		},
	})
}

func writeJSON(response http.ResponseWriter, status int, payload any) {
	response.Header().Set("Content-Type", jsonContentType)
	response.WriteHeader(status)
	if err := json.NewEncoder(response).Encode(payload); err != nil {
		// The response is already committed. This payload only contains in-memory
		// server values, so encoding failure is not recoverable at this boundary.
		return
	}
}
