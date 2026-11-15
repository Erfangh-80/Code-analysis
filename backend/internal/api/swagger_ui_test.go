package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSwaggerUIIsServedLocally(t *testing.T) {
	t.Parallel()

	handler := NewHandler("test-version")
	request := httptest.NewRequest(http.MethodGet, "/swagger/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", contentType)
	}
	if !bytes.Contains(response.Body.Bytes(), []byte("SwaggerUIBundle")) {
		t.Error("Swagger UI page does not initialize SwaggerUIBundle")
	}
	if bytes.Contains(response.Body.Bytes(), []byte("https://")) {
		t.Error("Swagger UI page must not load a remote asset or specification")
	}
}

func TestSwaggerUIAssetsAndMethodsAreHandledSafely(t *testing.T) {
	t.Parallel()

	handler := NewHandler("test-version")
	assetRequest := httptest.NewRequest(http.MethodGet, "/swagger/swagger-ui-bundle.js", nil)
	assetResponse := httptest.NewRecorder()
	handler.ServeHTTP(assetResponse, assetRequest)
	if assetResponse.Code != http.StatusOK {
		t.Fatalf("asset status = %d, want %d", assetResponse.Code, http.StatusOK)
	}
	if !bytes.Contains(assetResponse.Body.Bytes(), []byte("SwaggerUIBundle")) {
		t.Error("Swagger UI bundle was not served")
	}

	methodRequest := httptest.NewRequest(http.MethodPost, "/swagger/", nil)
	methodResponse := httptest.NewRecorder()
	handler.ServeHTTP(methodResponse, methodRequest)
	if methodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method status = %d, want %d", methodResponse.Code, http.StatusMethodNotAllowed)
	}
	assertErrorCode(t, methodResponse, "METHOD_NOT_ALLOWED")
}
