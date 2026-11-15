// Package api exposes the local HTTP boundary for Code Risk Analyzer.
package api

import (
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/Erfangh-80/code-analysis/backend/internal/analyzer/static"
	"github.com/Erfangh-80/code-analysis/backend/internal/application"
	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
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
}

// NewHandler returns the local v1 API handler. The static-analysis API reads
// only requested local files and never executes repository source.
func NewHandler(version string) http.Handler {
	failures := failedfiles.NewRegistry()
	return &handler{
		version:        version,
		staticAnalysis: application.NewStaticAnalysisService(static.NewAnalyzer(), failures),
	}
}

func (handler *handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/api/v1/health":
		handler.handleHealth(response, request)
	case "/api/v1/openapi.yaml":
		handler.handleOpenAPI(response, request)
	case "/api/v1/static-analysis":
		handler.handleStaticAnalysis(response, request)
	case "/api/v1/static-analysis/failures":
		handler.handleStaticAnalysisFailures(response, request)
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
		writeError(response, http.StatusBadRequest, "VALIDATION_ERROR", "The requested local analysis paths are invalid or inaccessible.")
		return
	}
	writeJSON(response, http.StatusOK, result)
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
