package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/analysisguard"
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

type pathList []string

func (paths *pathList) String() string {
	return strings.Join(*paths, ",")
}

func (paths *pathList) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("an allowed root cannot be empty")
	}
	*paths = append(*paths, value)
	return nil
}

func defaultDatabasePath() string {
	if dataDirectory, err := os.UserConfigDir(); err == nil && dataDirectory != "" {
		return filepath.Join(dataDirectory, "code-risk-analyzer", "analyses.db")
	}
	return filepath.Join(".", "code-risk-analyzer", "analyses.db")
}

func defaultAllowedRoot() (string, error) {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home, nil
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("determine default allowed analysis root: %w", err)
	}
	return workingDirectory, nil
}

func main() {
	defaults := analysisguard.DefaultLimits()
	address := flag.String("addr", "127.0.0.1:8080", "local HTTP listen address")
	dataPath := flag.String("data", defaultDatabasePath(), "local SQLite analysis database path")
	maxConcurrentRuns := flag.Int("max-concurrent-runs", 2, "maximum local project analyses running at once")
	maxFiles := flag.Int("max-files", defaults.MaxFiles, "maximum source files considered in one analysis")
	maxFileBytes := flag.Int64("max-file-bytes", defaults.MaxFileBytes, "maximum bytes read from one source file")
	maxTotalSourceBytes := flag.Int64("max-total-source-bytes", defaults.MaxTotalSourceBytes, "maximum total source bytes read in one analysis")
	maxGitCommits := flag.Int("max-git-commits", defaults.MaxGitCommits, "maximum commits inspected in one Git analysis")
	maxGitOutputBytes := flag.Int64("max-git-output-bytes", defaults.MaxGitOutputBytes, "maximum output retained from one Git command")
	analysisTimeout := flag.Duration("analysis-timeout", defaults.AnalysisTimeout, "maximum wall-clock duration for one queued or running analysis")
	var allowedRoots pathList
	flag.Var(&allowedRoots, "allowed-root", "absolute directory allowed for local analysis; repeat to allow more roots (defaults to the current user's home directory)")
	flag.Parse()

	if len(allowedRoots) == 0 {
		root, err := defaultAllowedRoot()
		if err != nil {
			log.Fatalf("configure default allowed analysis root: %v", err)
		}
		allowedRoots = append(allowedRoots, root)
	}
	limits, err := (analysisguard.Limits{
		MaxFiles:            *maxFiles,
		MaxFileBytes:        *maxFileBytes,
		MaxTotalSourceBytes: *maxTotalSourceBytes,
		MaxGitCommits:       *maxGitCommits,
		MaxGitOutputBytes:   *maxGitOutputBytes,
		AnalysisTimeout:     *analysisTimeout,
	}).Normalize()
	if err != nil {
		log.Fatalf("configure analysis resource limits: %v", err)
	}
	policy, err := analysisguard.NewPathPolicy(allowedRoots)
	if err != nil {
		log.Fatalf("configure allowed analysis roots: %v", err)
	}

	store, err := sqlite.Open(*dataPath)
	if err != nil {
		log.Fatalf("open local analysis database: %v", err)
	}
	defer store.Close()

	failures := failedfiles.NewRegistry()
	scanner, err := repositoryscan.NewScannerWithGuards(policy, limits)
	if err != nil {
		log.Fatalf("configure repository scanner: %v", err)
	}
	staticAnalysis, err := application.NewStaticAnalysisServiceWithGuards(static.NewAnalyzer(), failures, policy, limits)
	if err != nil {
		log.Fatalf("configure static analysis: %v", err)
	}
	gitAnalysis, err := githistory.NewAnalyzerWithGuards(policy, limits)
	if err != nil {
		log.Fatalf("configure Git analysis: %v", err)
	}
	orchestrator, err := application.NewOrchestratorWithLimits(
		store,
		scanner,
		staticAnalysis,
		gitAnalysis,
		scoring.NewEngine(),
		*maxConcurrentRuns,
		limits,
	)
	if err != nil {
		log.Fatalf("configure analysis orchestrator: %v", err)
	}
	handler := api.NewHandlerWithServices(version, orchestrator, staticAnalysis, gitAnalysis)

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
