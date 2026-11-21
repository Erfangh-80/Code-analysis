package api

import "net/http"

func setSecurityHeaders(response http.ResponseWriter) {
	response.Header().Set("X-Content-Type-Options", "nosniff")
	response.Header().Set("X-Frame-Options", "DENY")
	response.Header().Set("Referrer-Policy", "no-referrer")
	response.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	response.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; frame-ancestors 'none'; object-src 'none'; connect-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'")
	response.Header().Set("Cache-Control", "no-store")
}

// handleCORS temporarily permits browser access from every origin while the
// application is being developed. It deliberately does not enable credentials:
// browser pages can call the local API, but cannot attach cross-origin cookies
// or HTTP authentication state. Restrict this policy before public release.
func (handler *handler) handleCORS(response http.ResponseWriter, request *http.Request) bool {
	if request.Header.Get("Origin") == "" {
		return false
	}

	response.Header().Set("Access-Control-Allow-Origin", "*")
	response.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	response.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	response.Header().Set("Access-Control-Max-Age", "600")

	if request.Method == http.MethodOptions {
		response.WriteHeader(http.StatusNoContent)
		return true
	}
	return false
}
