package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ParseAllowedOrigins parses a comma-separated allowlist of exact trusted
// origins for explicit cross-origin local UI development. Empty input disables
// cross-origin access; same-origin requests remain available.
func ParseAllowedOrigins(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return []string{}, nil
	}
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		origin, err := canonicalOrigin(part)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[origin]; exists {
			continue
		}
		seen[origin] = struct{}{}
		origins = append(origins, origin)
	}
	return origins, nil
}

func canonicalOrigin(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Hostname() == "" || strings.HasSuffix(parsed.Host, ":") {
		return "", fmt.Errorf("CORS origin must be an absolute http or https origin")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("CORS origin must use http or https")
	}
	if parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || strings.Contains(trimmed, "#") {
		return "", fmt.Errorf("CORS origin must not include credentials, a path, query, or fragment")
	}
	return scheme + "://" + strings.ToLower(parsed.Host), nil
}

func allowedOriginSet(origins []string) (map[string]struct{}, error) {
	allowed := make(map[string]struct{}, len(origins))
	for _, value := range origins {
		origin, err := canonicalOrigin(value)
		if err != nil {
			return nil, err
		}
		allowed[origin] = struct{}{}
	}
	return allowed, nil
}

func setSecurityHeaders(response http.ResponseWriter) {
	response.Header().Set("X-Content-Type-Options", "nosniff")
	response.Header().Set("X-Frame-Options", "DENY")
	response.Header().Set("Referrer-Policy", "no-referrer")
	response.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	response.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; frame-ancestors 'none'; object-src 'none'; connect-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'")
	response.Header().Set("Cache-Control", "no-store")
}

// handleCORS writes and completes rejected/preflight requests. It returns false
// when normal API routing should continue.
func (handler *handler) handleCORS(response http.ResponseWriter, request *http.Request) bool {
	originHeader := request.Header.Get("Origin")
	if originHeader == "" {
		return false
	}
	origin, err := canonicalOrigin(originHeader)
	if err != nil {
		writeError(response, http.StatusForbidden, "CORS_ORIGIN_DENIED", "This browser origin is not allowed to access the local API.")
		return true
	}

	_, explicitlyAllowed := handler.allowedOrigins[origin]
	if !explicitlyAllowed && !sameOrigin(request, origin) {
		writeError(response, http.StatusForbidden, "CORS_ORIGIN_DENIED", "This browser origin is not allowed to access the local API.")
		return true
	}
	if explicitlyAllowed {
		response.Header().Set("Access-Control-Allow-Origin", origin)
		response.Header().Add("Vary", "Origin")
		response.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		response.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		response.Header().Set("Access-Control-Max-Age", "600")
	}
	if request.Method == http.MethodOptions {
		response.WriteHeader(http.StatusNoContent)
		return true
	}
	return false
}

func sameOrigin(request *http.Request, origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	return parsed.Scheme == scheme && strings.EqualFold(parsed.Host, request.Host)
}
