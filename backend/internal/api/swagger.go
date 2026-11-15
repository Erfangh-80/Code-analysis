package api

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// swaggerUIAssets vendors Swagger UI 5.35.0 (Apache-2.0) so that API
// documentation stays local and does not fetch scripts, styles, or specs from a cloud service.
//
//go:embed swaggerui/*
var swaggerUIAssets embed.FS

var swaggerUIFileSystem = mustSwaggerUIFileSystem()
var swaggerUIFileServer = http.FileServer(http.FS(swaggerUIFileSystem))

func mustSwaggerUIFileSystem() fs.FS {
	assets, err := fs.Sub(swaggerUIAssets, "swaggerui")
	if err != nil {
		panic("embedded Swagger UI assets are unavailable: " + err.Error())
	}
	return assets
}

func (handler *handler) handleSwaggerUI(response http.ResponseWriter, request *http.Request) {
	if !requireMethod(response, request, http.MethodGet) {
		return
	}
	if request.URL.Path == "/swagger" {
		http.Redirect(response, request, "/swagger/", http.StatusPermanentRedirect)
		return
	}

	request.URL.Path = strings.TrimPrefix(request.URL.Path, "/swagger")
	if request.URL.Path == "" {
		request.URL.Path = "/"
	}
	swaggerUIFileServer.ServeHTTP(response, request)
}
