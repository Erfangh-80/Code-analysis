package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Erfangh-80/Code-analysis/backend/internal/api"
	"github.com/Erfangh-80/Code-analysis/backend/internal/jobs"
	"github.com/Erfangh-80/Code-analysis/backend/internal/storage"
)

func main() {
	root := requiredEnv("ANALYZER_REPO_ROOT")
	token := requiredEnv("ANALYZER_API_TOKEN")
	dataDir := env("ANALYZER_DATA_DIR", "./data")
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		log.Fatal(err)
	}
	store, err := storage.Open(filepath.Join(dataDir, "analyzer.sqlite"))
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	if err := store.RecoverRunning(); err != nil {
		log.Fatal(err)
	}
	manager, err := jobs.New(store, jobs.Config{AllowedRepoRoot: root, MaxCommits: envInt("ANALYZER_MAX_COMMITS", 5000), AnalysisTimeout: envDuration("ANALYZER_ANALYSIS_TIMEOUT", 10*time.Minute), MaxFileBytes: int64(envInt("ANALYZER_MAX_FILE_BYTES", 2*1024*1024))})
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()
	origins := nonEmpty(strings.Split(os.Getenv("ANALYZER_ALLOWED_ORIGINS"), ","))
	apiHandler := api.New(store, manager, api.Config{Token: token, AllowedOrigins: origins})
	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler)
	webDir := env("ANALYZER_WEB_DIR", "")
	if webDir != "" {
		mux.Handle("/", spa(webDir))
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte("Code Hotspot Analyzer API is running. Build the frontend or use /api/v1/health."))
		})
	}
	server := &http.Server{Addr: env("ANALYZER_LISTEN_ADDR", "127.0.0.1:8080"), Handler: securityHeaders(mux), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 60 * time.Second}
	cert, key := os.Getenv("ANALYZER_TLS_CERT"), os.Getenv("ANALYZER_TLS_KEY")
	log.Printf("Code Hotspot Analyzer listening on %s", server.Addr)
	if (cert == "") != (key == "") {
		log.Fatal("both ANALYZER_TLS_CERT and ANALYZER_TLS_KEY must be set together")
	}
	if cert != "" {
		log.Fatal(server.ListenAndServeTLS(cert, key))
	}
	log.Fatal(server.ListenAndServe())
}
func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s is required", name)
	}
	return value
}
func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
func envInt(name string, fallback int) int {
	if value := os.Getenv(name); value != "" {
		if number, err := strconv.Atoi(value); err == nil && number > 0 {
			return number
		}
	}
	return fallback
}
func envDuration(name string, fallback time.Duration) time.Duration {
	if value := os.Getenv(name); value != "" {
		if duration, err := time.ParseDuration(value); err == nil && duration > 0 {
			return duration
		}
	}
	return fallback
}
func nonEmpty(values []string) []string {
	out := []string{}
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
func spa(root string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			candidate := filepath.Join(root, filepath.Clean(r.URL.Path))
			if strings.HasPrefix(candidate, filepath.Clean(root)+string(filepath.Separator)) {
				if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
					http.ServeFile(w, r, candidate)
					return
				}
			}
		}
		index := filepath.Join(root, "index.html")
		if _, err := os.Stat(index); errors.Is(err, os.ErrNotExist) {
			http.Error(w, "frontend not built", http.StatusNotFound)
			return
		}
		http.ServeFile(w, r, index)
	})
}
