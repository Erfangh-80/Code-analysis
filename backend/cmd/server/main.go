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
	"strings"
	"syscall"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/analysisguard"
	"github.com/Erfangh-80/code-analysis/backend/internal/analyzer/static"
	"github.com/Erfangh-80/code-analysis/backend/internal/api"
	"github.com/Erfangh-80/code-analysis/backend/internal/application"
	"github.com/Erfangh-80/code-analysis/backend/internal/githistory"
	"github.com/Erfangh-80/code-analysis/backend/internal/storage/failedfiles"
)

const version = "0.1.0-core"

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
	maxFiles := flag.Int("max-files", defaults.MaxFiles, "maximum source files accepted in one static-analysis request")
	maxFileBytes := flag.Int64("max-file-bytes", defaults.MaxFileBytes, "maximum bytes read from one source file")
	maxTotalSourceBytes := flag.Int64("max-total-source-bytes", defaults.MaxTotalSourceBytes, "maximum total source bytes read in one static-analysis request")
	maxGitCommits := flag.Int("max-git-commits", defaults.MaxGitCommits, "maximum commits inspected in one Git analysis")
	maxGitOutputBytes := flag.Int64("max-git-output-bytes", defaults.MaxGitOutputBytes, "maximum output retained from one Git command")
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
	}).Normalize()
	if err != nil {
		log.Fatalf("configure analysis resource limits: %v", err)
	}
	policy, err := analysisguard.NewPathPolicy(allowedRoots)
	if err != nil {
		log.Fatalf("configure allowed analysis roots: %v", err)
	}

	failures := failedfiles.NewRegistry()
	staticAnalysis, err := application.NewStaticAnalysisServiceWithGuards(static.NewAnalyzer(), failures, policy, limits)
	if err != nil {
		log.Fatalf("configure static analysis: %v", err)
	}
	gitAnalysis, err := githistory.NewAnalyzerWithGuards(policy, limits)
	if err != nil {
		log.Fatalf("configure Git analysis: %v", err)
	}
	handler := api.NewHandlerWithCoreServices(version, staticAnalysis, gitAnalysis)

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
		log.Printf("Code Risk Analyzer core API %s listening on http://%s", version, *address)
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
		log.Print("Code Risk Analyzer core API stopped")
	}
}
