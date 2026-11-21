package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/analyzer/static"
	"github.com/Erfangh-80/code-analysis/backend/internal/api"
	"github.com/Erfangh-80/code-analysis/backend/internal/application"
	"github.com/Erfangh-80/code-analysis/backend/internal/githistory"
	"github.com/Erfangh-80/code-analysis/backend/internal/infrastructure/sqlite"
	"github.com/Erfangh-80/code-analysis/backend/internal/repositoryscan"
	"github.com/Erfangh-80/code-analysis/backend/internal/scoring"
	"github.com/Erfangh-80/code-analysis/backend/internal/storage/failedfiles"
)

const version = "0.1.0"

func defaultDatabasePath() string {
	if dataDirectory, err := os.UserConfigDir(); err == nil && dataDirectory != "" {
		return filepath.Join(dataDirectory, "code-risk-analyzer", "analyses.db")
	}
	return filepath.Join(".", "code-risk-analyzer", "analyses.db")
}

func main() {
	address := flag.String("addr", "127.0.0.1:8080", "local HTTP listen address")
	dataPath := flag.String("data", defaultDatabasePath(), "local SQLite analysis database path")
	maxConcurrentRuns := flag.Int("max-concurrent-runs", 2, "maximum local project analyses running at once")
	flag.Parse()

	store, err := sqlite.Open(*dataPath)
	if err != nil {
		log.Fatalf("open local analysis database: %v", err)
	}
	defer store.Close()

	failures := failedfiles.NewRegistry()
	orchestrator := application.NewOrchestrator(
		store,
		repositoryscan.NewScanner(),
		application.NewStaticAnalysisService(static.NewAnalyzer(), failures),
		githistory.NewAnalyzer(),
		scoring.NewEngine(),
		*maxConcurrentRuns,
	)
	handler := api.NewHandlerWithAnalysisService(version, orchestrator)

	server := &http.Server{
		Addr:              *address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Code Risk Analyzer API %s listening on http://%s", version, *address)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrors:
		log.Fatalf("serve API: %v", err)
	case <-signalContext.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
			return
		}
		log.Print("Code Risk Analyzer API stopped")
	}
}
