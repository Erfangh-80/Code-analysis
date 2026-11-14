// Package api exposes the local HTTP boundary for Code Risk Analyzer.
package api

import (
	"encoding/json"
	"net/http"
)

const jsonContentType = "application/json; charset=utf-8"

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

// NewHandler returns the local v1 API handler. Analysis endpoints are added as
// their application services are implemented; this foundation intentionally
// exposes only the truthful service-health capability.
func NewHandler(version string) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v1/health":
			handleHealth(response, request, version)
		default:
			writeError(response, http.StatusNotFound, "NOT_FOUND", "The requested API route does not exist.")
		}
	})
}

func handleHealth(response http.ResponseWriter, request *http.Request, version string) {
	if request.Method != http.MethodGet {
		response.Header().Set("Allow", http.MethodGet)
		writeError(response, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET is supported for this route.")
		return
	}

	writeJSON(response, http.StatusOK, HealthResponse{
		Status:  "ok",
		Version: version,
	})
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
