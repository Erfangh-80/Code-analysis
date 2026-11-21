package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSTemporarilyAllowsEveryBrowserOriginAndAppliesSecurityHeaders(t *testing.T) {
	handler := NewHandler("test-version")

	preflight := httptest.NewRecorder()
	preflightRequest := httptest.NewRequest(http.MethodOptions, "/api/v1/analyses", nil)
	preflightRequest.Header.Set("Origin", "https://ui.example.test")
	preflightRequest.Header.Set("Access-Control-Request-Method", http.MethodPost)
	handler.ServeHTTP(preflight, preflightRequest)
	if preflight.Code != http.StatusNoContent {
		t.Fatalf("preflight status=%d body=%s", preflight.Code, preflight.Body.String())
	}
	assertPermissiveCORSHeaders(t, preflight.Header())

	for _, origin := range []string{
		"https://ui.example.test",
		"http://127.0.0.1:5173",
		"null",
	} {
		t.Run(origin, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
			request.Header.Set("Origin", origin)
			handler.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("CORS status=%d body=%s", response.Code, response.Body.String())
			}
			assertPermissiveCORSHeaders(t, response.Header())
			for header, want := range map[string]string{
				"X-Content-Type-Options":       "nosniff",
				"X-Frame-Options":              "DENY",
				"Referrer-Policy":              "no-referrer",
				"Cross-Origin-Resource-Policy": "same-origin",
				"Cache-Control":                "no-store",
			} {
				if got := response.Header().Get(header); got != want {
					t.Errorf("%s = %q, want %q", header, got, want)
				}
			}
			if response.Header().Get("Content-Security-Policy") == "" {
				t.Error("Content-Security-Policy header is missing")
			}
		})
	}
}

func TestCORSHeadersAreNotSentWithoutAnOrigin(t *testing.T) {
	handler := NewHandler("test-version")
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin=%q, want no CORS header", got)
	}
}

func assertPermissiveCORSHeaders(t *testing.T, headers http.Header) {
	t.Helper()
	for header, want := range map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type",
		"Access-Control-Max-Age":       "600",
	} {
		if got := headers.Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}
