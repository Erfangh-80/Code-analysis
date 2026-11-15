package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSUsesExactAllowlistAndAppliesSecurityHeaders(t *testing.T) {
	handler, err := NewHandlerWithAnalysisServiceAndCORS("test-version", nil, []string{"https://ui.example.test"})
	if err != nil {
		t.Fatalf("create CORS handler: %v", err)
	}

	preflight := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/analyses", nil)
	request.Header.Set("Origin", "https://ui.example.test")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	handler.ServeHTTP(preflight, request)
	if preflight.Code != http.StatusNoContent || preflight.Header().Get("Access-Control-Allow-Origin") != "https://ui.example.test" || preflight.Header().Get("Access-Control-Allow-Methods") != "GET, POST, OPTIONS" {
		t.Errorf("preflight status=%d headers=%v", preflight.Code, preflight.Header())
	}

	allowed := httptest.NewRecorder()
	allowedRequest := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	allowedRequest.Header.Set("Origin", "https://ui.example.test")
	handler.ServeHTTP(allowed, allowedRequest)
	if allowed.Code != http.StatusOK || allowed.Header().Get("Access-Control-Allow-Origin") != "https://ui.example.test" {
		t.Errorf("allowed CORS response status=%d headers=%v", allowed.Code, allowed.Header())
	}
	for header, want := range map[string]string{
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"Referrer-Policy":              "no-referrer",
		"Cross-Origin-Resource-Policy": "same-origin",
		"Cache-Control":                "no-store",
	} {
		if got := allowed.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
	if allowed.Header().Get("Content-Security-Policy") == "" {
		t.Error("Content-Security-Policy header is missing")
	}

	denied := httptest.NewRecorder()
	deniedRequest := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	deniedRequest.Header.Set("Origin", "https://untrusted.example.test")
	handler.ServeHTTP(denied, deniedRequest)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("denied CORS status=%d body=%s", denied.Code, denied.Body.String())
	}
	assertErrorCode(t, denied, "CORS_ORIGIN_DENIED")
}

func TestCORSDefaultsToSameOriginOnly(t *testing.T) {
	handler := NewHandler("test-version")

	sameOrigin := httptest.NewRecorder()
	sameOriginRequest := httptest.NewRequest(http.MethodGet, "http://api.example.test/api/v1/health", nil)
	sameOriginRequest.Host = "api.example.test"
	sameOriginRequest.Header.Set("Origin", "http://api.example.test")
	handler.ServeHTTP(sameOrigin, sameOriginRequest)
	if sameOrigin.Code != http.StatusOK {
		t.Fatalf("same-origin status=%d body=%s", sameOrigin.Code, sameOrigin.Body.String())
	}
	if got := sameOrigin.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("same-origin Access-Control-Allow-Origin=%q, want no CORS header", got)
	}

	crossOrigin := httptest.NewRecorder()
	crossOriginRequest := httptest.NewRequest(http.MethodGet, "http://api.example.test/api/v1/health", nil)
	crossOriginRequest.Host = "api.example.test"
	crossOriginRequest.Header.Set("Origin", "http://ui.example.test")
	handler.ServeHTTP(crossOrigin, crossOriginRequest)
	if crossOrigin.Code != http.StatusForbidden {
		t.Fatalf("cross-origin status=%d body=%s", crossOrigin.Code, crossOrigin.Body.String())
	}
	assertErrorCode(t, crossOrigin, "CORS_ORIGIN_DENIED")
}

func TestCORSConfigurationRejectsUnsafeOrigins(t *testing.T) {
	invalid := []string{
		"*",
		"https://ui.example.test/path",
		"ftp://ui.example.test",
		"https://user@ui.example.test",
		"https://:443",
		"https://ui.example.test:",
		"https://ui.example.test?",
		"https://ui.example.test#",
	}
	for _, origin := range invalid {
		if _, err := ParseAllowedOrigins(origin); err == nil {
			t.Errorf("ParseAllowedOrigins(%q) succeeded, want error", origin)
		}
	}
	origins, err := ParseAllowedOrigins("https://ui.example.test, https://ui.example.test")
	if err != nil || len(origins) != 1 || origins[0] != "https://ui.example.test" {
		t.Errorf("parsed origins=%v err=%v", origins, err)
	}
}
