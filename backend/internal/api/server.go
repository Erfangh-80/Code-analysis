// Package api exposes the local HTTP boundary for the core Code Risk Analyzer.
package api

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"net/http"
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
}

// NewHandler returns a local v1 core-analysis handler for in-process callers.
func NewHandler(version string) http.Handler {
	return newHandler(version, nil, nil)
}

// NewHandlerWithCoreServices composes the direct static, Git-history, and risk
// APIs with the production-configured guards. It intentionally contains no
// durable run orchestration, report persistence, or dashboard result routes.
func NewHandlerWithCoreServices(version string, staticAnalysis *application.StaticAnalysisService, gitHistory *githistory.Analyzer) http.Handler {
	return newHandler(version, staticAnalysis, gitHistory)
}

func newHandler(version string, staticAnalysis *application.StaticAnalysisService, gitHistory *githistory.Analyzer) http.Handler {
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
	}
}

func (handler *handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	setSecurityHeaders(response)
	if handler.handleCORS(response, request) {
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
	writeJSON(response, http.StatusOK, HealthResponse{Status: "ok", Version: handler.version})
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
	result, err := handler.staticAnalysis.Analyze(request.Context(), application.StaticAnalysisRequest{RootPath: payload.RootPath, Files: payload.Files})
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

func (handler *handler) handleStaticAnalysisFailures(response http.ResponseWriter, request *http.Request) {
	if !requireMethod(response, request, http.MethodGet) {
		return
	}
	writeJSON(response, http.StatusOK, staticAnalysisFailureResponse{Items: handler.staticAnalysis.FailedFiles()})
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
		if errors.Is(err, analysisguard.ErrResourceLimit) || errors.Is(err, githistory.ErrResourceLimit) {
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
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "Provide one or more complete file metric records and an optional valid scoring profile.")
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

func requireMethod(response http.ResponseWriter, request *http.Request, expected string) bool {
	if request.Method == expected {
		return true
	}
	response.Header().Set("Allow", expected)
	writeError(response, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "This HTTP method is not supported for the requested route.")
	return false
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
	if strings.TrimSpace(payload.RootPath) == "" || len(payload.Files) == 0 {
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
	if strings.TrimSpace(payload.RepositoryPath) == "" {
		return domain.GitAnalysisRequest{}, io.ErrUnexpectedEOF
	}
	if payload.History.From != nil && payload.History.To != nil && payload.History.From.After(*payload.History.To) {
		return domain.GitAnalysisRequest{}, githistory.ErrInvalidHistoryRange
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
	requiredMetrics := []string{"linesOfCode", "maxFunctionLength", "maxClassLength", "gitChurn", "changeFrequency"}
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
	writeJSON(response, status, ErrorResponse{Error: APIError{Code: code, Message: message}})
}

func writeJSON(response http.ResponseWriter, status int, payload any) {
	response.Header().Set("Content-Type", jsonContentType)
	response.WriteHeader(status)
	if err := json.NewEncoder(response).Encode(payload); err != nil {
		return
	}
}
