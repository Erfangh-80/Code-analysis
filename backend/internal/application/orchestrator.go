package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
	"github.com/Erfangh-80/code-analysis/backend/internal/githistory"
	"github.com/Erfangh-80/code-analysis/backend/internal/scoring"
)

// ProjectScanner is the application port for safe local-project discovery.
type ProjectScanner interface {
	Resolve(context.Context, string) (domain.ProjectScope, error)
	Scan(context.Context, domain.ProjectScope) (domain.ProjectScan, error)
}

// GitHistoryAnalyzer is the application port for read-only local Git facts.
type GitHistoryAnalyzer interface {
	Analyze(context.Context, domain.GitAnalysisRequest) (domain.GitAnalysisReport, error)
}

// RiskAnalyzer is the application port for deterministic risk scoring.
type RiskAnalyzer interface {
	Analyze(domain.RiskAnalysisRequest) (domain.RiskAnalysisReport, error)
}

// Orchestrator coordinates local scan, static, optional Git, and risk stages.
// It has no direct dependency on SQLite, HTTP, or scanner implementation.
type Orchestrator struct {
	store   RunStore
	scanner ProjectScanner
	static  *StaticAnalysisService
	git     GitHistoryAnalyzer
	risk    RiskAnalyzer

	runSlots chan struct{}

	mu      sync.Mutex
	cancels map[domain.AnalysisID]context.CancelFunc
}

// NewOrchestrator creates a bounded local analysis coordinator.
func NewOrchestrator(store RunStore, scanner ProjectScanner, static *StaticAnalysisService, git GitHistoryAnalyzer, risk RiskAnalyzer, maxConcurrentRuns int) *Orchestrator {
	if maxConcurrentRuns < 1 {
		maxConcurrentRuns = 1
	}
	return &Orchestrator{
		store:    store,
		scanner:  scanner,
		static:   static,
		git:      git,
		risk:     risk,
		runSlots: make(chan struct{}, maxConcurrentRuns),
		cancels:  make(map[domain.AnalysisID]context.CancelFunc),
	}
}

// Start validates the local project scope, persists a queued run, and starts
// background analysis. It never uses the request HTTP context after return.
func (orchestrator *Orchestrator) Start(ctx context.Context, request domain.AnalysisRequest) (domain.AnalysisRun, error) {
	if request.History.From != nil && request.History.To != nil && request.History.From.After(*request.History.To) {
		return domain.AnalysisRun{}, githistory.ErrInvalidHistoryRange
	}
	if request.Profile != nil {
		if err := scoring.ValidateProfile(*request.Profile); err != nil {
			return domain.AnalysisRun{}, err
		}
	}
	if orchestrator.store == nil || orchestrator.scanner == nil || orchestrator.static == nil || orchestrator.risk == nil {
		return domain.AnalysisRun{}, fmt.Errorf("analysis orchestrator is not configured")
	}
	scope, err := orchestrator.scanner.Resolve(ctx, request.ProjectPath)
	if err != nil {
		return domain.AnalysisRun{}, err
	}

	now := time.Now().UTC()
	run := domain.AnalysisRun{
		ID:          newAnalysisID(),
		Status:      domain.AnalysisStatusQueued,
		CreatedAt:   now,
		History:     request.History,
		Stages:      initialStages(),
		Diagnostics: make([]domain.AnalysisDiagnostic, 0),
		Scope:       scope,
		Profile:     cloneRiskProfile(request.Profile),
	}
	if err := orchestrator.store.CreateQueued(ctx, run); err != nil {
		return domain.AnalysisRun{}, err
	}

	runContext, cancel := context.WithCancel(context.Background())
	orchestrator.mu.Lock()
	orchestrator.cancels[run.ID] = cancel
	orchestrator.mu.Unlock()
	go orchestrator.execute(runContext, run)
	return run, nil
}

// Get returns a locally persisted lifecycle snapshot.
func (orchestrator *Orchestrator) Get(ctx context.Context, id domain.AnalysisID) (domain.AnalysisRun, error) {
	return orchestrator.store.Get(ctx, id)
}

// List returns locally persisted lifecycle snapshots.
func (orchestrator *Orchestrator) List(ctx context.Context, query domain.AnalysisListQuery) (domain.AnalysisPage, error) {
	return orchestrator.store.List(ctx, query)
}

// Report returns an immutable completed report.
func (orchestrator *Orchestrator) Report(ctx context.Context, id domain.AnalysisID) (domain.AnalysisReport, error) {
	run, err := orchestrator.store.Get(ctx, id)
	if err != nil {
		return domain.AnalysisReport{}, err
	}
	if run.Status != domain.AnalysisStatusCompleted {
		return domain.AnalysisReport{}, ErrInvalidRunState
	}
	return orchestrator.store.GetReport(ctx, id)
}

// Cancel cooperatively stops a queued or running analysis.
func (orchestrator *Orchestrator) Cancel(ctx context.Context, id domain.AnalysisID) (domain.AnalysisRun, error) {
	run, err := orchestrator.store.Get(ctx, id)
	if err != nil {
		return domain.AnalysisRun{}, err
	}
	if isTerminal(run.Status) {
		return domain.AnalysisRun{}, ErrInvalidRunState
	}
	orchestrator.mu.Lock()
	cancel := orchestrator.cancels[id]
	orchestrator.mu.Unlock()
	if cancel == nil {
		return domain.AnalysisRun{}, ErrInvalidRunState
	}
	cancel()
	return run, nil
}

func (orchestrator *Orchestrator) execute(ctx context.Context, run domain.AnalysisRun) {
	defer orchestrator.unregister(run.ID)
	select {
	case orchestrator.runSlots <- struct{}{}:
		defer func() { <-orchestrator.runSlots }()
	case <-ctx.Done():
		orchestrator.cancelRun(&run)
		return
	}

	now := time.Now().UTC()
	run.Status = domain.AnalysisStatusRunning
	run.StartedAt = &now
	setCurrentStage(&run, domain.AnalysisCurrentStageValidation)
	setStage(&run, domain.AnalysisStageValidation, domain.StageStatusRunning, domain.StageProgress{})
	if !orchestrator.saveRun(&run) {
		return
	}
	setStage(&run, domain.AnalysisStageValidation, domain.StageStatusCompleted, domain.StageProgress{Total: 1, Completed: 1})
	setCurrentStage(&run, domain.AnalysisCurrentStageDiscovery)
	setStage(&run, domain.AnalysisStageDiscovery, domain.StageStatusRunning, domain.StageProgress{})
	if !orchestrator.saveRun(&run) {
		return
	}

	scan, err := orchestrator.scanner.Scan(ctx, run.Scope)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			orchestrator.cancelRun(&run)
			return
		}
		orchestrator.failRun(&run, domain.AnalysisStageDiscovery, "DISCOVERY_FAILED", "The local project files could not be discovered.")
		return
	}
	appendDiagnostics(&run, scan.Diagnostics)
	run.Progress.FilesDiscovered = len(scan.Files)
	run.Progress.FilesEligible = len(scan.Files)
	setStage(&run, domain.AnalysisStageDiscovery, domain.StageStatusCompleted, domain.StageProgress{Total: len(scan.Files), Completed: len(scan.Files)})
	setCurrentStage(&run, domain.AnalysisCurrentStageAnalysis)
	setStage(&run, domain.AnalysisStageStatic, domain.StageStatusRunning, domain.StageProgress{Total: len(scan.Files)})
	if scan.Scope.GitAvailable {
		setStage(&run, domain.AnalysisStageGit, domain.StageStatusRunning, domain.StageProgress{})
	} else {
		setStage(&run, domain.AnalysisStageGit, domain.StageStatusUnavailable, domain.StageProgress{})
	}
	if !orchestrator.saveRun(&run) {
		return
	}

	type staticResult struct {
		result domain.BatchAnalysis
		err    error
	}
	staticResults := make(chan staticResult, 1)
	if len(scan.Files) == 0 {
		staticResults <- staticResult{result: domain.BatchAnalysis{Files: []domain.FileAnalysis{}, Modules: []domain.ModuleAnalysis{}}}
	} else {
		go func() {
			result, analysisErr := orchestrator.static.AnalyzeWithProgress(ctx, StaticAnalysisRequest{RootPath: scan.Scope.ProjectPath, Files: scan.Files}, func(completed, total int, file domain.FileAnalysis) {
				orchestrator.recordStaticProgress(&run, completed, total, file)
			})
			staticResults <- staticResult{result: result, err: analysisErr}
		}()
	}

	type gitResult struct {
		report domain.GitAnalysisReport
		err    error
	}
	gitResults := make(chan gitResult, 1)
	if scan.Scope.GitAvailable && orchestrator.git != nil {
		go func() {
			report, analysisErr := orchestrator.git.Analyze(ctx, domain.GitAnalysisRequest{
				RepositoryPath: scan.Scope.GitRepositoryRoot,
				ScopePath:      scan.Scope.GitRelativeScope,
				History:        run.History,
			})
			gitResults <- gitResult{report: rebaseGitReport(report, scan.Scope.GitRelativeScope), err: analysisErr}
		}()
	} else {
		gitResults <- gitResult{}
	}

	var staticOutcome staticResult
	var gitOutcome gitResult
	for completed := 0; completed < 2; {
		select {
		case staticOutcome = <-staticResults:
			completed++
			staticResults = nil
		case gitOutcome = <-gitResults:
			completed++
			gitResults = nil
		case <-ctx.Done():
			orchestrator.cancelRun(&run)
			return
		}
	}
	if ctx.Err() != nil {
		orchestrator.cancelRun(&run)
		return
	}
	if staticOutcome.err != nil {
		orchestrator.failRun(&run, domain.AnalysisStageStatic, "STATIC_ANALYSIS_FAILED", "Static analysis could not complete for the selected project.")
		return
	}

	staticReport := staticOutcome.result
	setStage(&run, domain.AnalysisStageStatic, domain.StageStatusCompleted, domain.StageProgress{Total: len(scan.Files), Completed: len(staticReport.Files)})

	artifacts := domain.AnalysisArtifacts{Static: &staticReport}
	gitReportAvailable := scan.Scope.GitAvailable && gitOutcome.err == nil
	gitAvailable := gitReportAvailable && gitOutcome.report.CommitCount > 0
	if gitReportAvailable {
		artifacts.Git = &gitOutcome.report
		for _, diagnostic := range gitOutcome.report.Diagnostics {
			appendDiagnostic(&run, domain.AnalysisStageGit, "", string(diagnostic.Code), diagnostic.Severity, diagnostic.Message)
		}
	}
	if gitAvailable {
		run.Progress.CommitsTotal = gitOutcome.report.CommitCount
		run.Progress.CommitsScanned = gitOutcome.report.CommitCount
		setStage(&run, domain.AnalysisStageGit, domain.StageStatusCompleted, domain.StageProgress{Total: gitOutcome.report.CommitCount, Completed: gitOutcome.report.CommitCount})
	} else if scan.Scope.GitAvailable {
		setStage(&run, domain.AnalysisStageGit, domain.StageStatusUnavailable, domain.StageProgress{})
		appendDiagnostic(&run, domain.AnalysisStageGit, "", "GIT_UNAVAILABLE", "warning", "Git history is unavailable for the selected project; static metrics remain available for risk scoring.")
	}

	setCurrentStage(&run, domain.AnalysisCurrentStageScoring)
	setStage(&run, domain.AnalysisStageRisk, domain.StageStatusRunning, domain.StageProgress{})
	if !orchestrator.saveRun(&run) {
		return
	}

	inputs := buildScorableInputs(staticReport.Files, artifacts.Git)
	availability := domain.MetricAvailability{LinesOfCode: true, MaxFunctionLength: true, MaxClassLength: true}
	if gitAvailable {
		availability.GitChurn = true
		availability.ChangeFrequency = true
	}
	if len(inputs) == 0 {
		setStage(&run, domain.AnalysisStageRisk, domain.StageStatusUnavailable, domain.StageProgress{})
		appendDiagnostic(&run, domain.AnalysisStageRisk, "", "NO_SCORABLE_FILES", "warning", "No file has a complete, valid static analysis for risk scoring.")
	} else {
		riskReport, riskErr := orchestrator.risk.Analyze(domain.RiskAnalysisRequest{
			Files:            inputs,
			Profile:          cloneRiskProfile(run.Profile),
			AvailableMetrics: availability.Metrics(),
		})
		if riskErr != nil {
			if errors.Is(riskErr, scoring.ErrNoAvailableMetrics) || errors.Is(riskErr, scoring.ErrInvalidHeatmap) {
				setStage(&run, domain.AnalysisStageRisk, domain.StageStatusUnavailable, domain.StageProgress{})
				appendDiagnostic(&run, domain.AnalysisStageRisk, "", "RISK_UNAVAILABLE", "warning", "No valid risk model could be calculated from the available metrics.")
			} else {
				orchestrator.failRun(&run, domain.AnalysisStageRisk, "RISK_SCORING_FAILED", "Risk scoring could not complete for the available metrics.")
				return
			}
		} else {
			artifacts.Risk = &riskReport
			artifacts.RiskCoverage = &domain.RiskCoverage{
				AvailableMetrics:   append([]domain.RiskMetric(nil), riskReport.AvailableMetrics...),
				UnavailableMetrics: append([]domain.RiskMetric(nil), riskReport.UnavailableMetrics...),
				RequestedProfile:   riskReport.RequestedProfile,
				EffectiveProfile:   riskReport.Profile,
			}
			setStage(&run, domain.AnalysisStageRisk, domain.StageStatusCompleted, domain.StageProgress{Total: len(inputs), Completed: len(inputs)})
		}
	}

	setCurrentStage(&run, domain.AnalysisCurrentStageFinalizing)
	if ctx.Err() != nil {
		orchestrator.cancelRun(&run)
		return
	}
	scoredFiles := 0
	if artifacts.Risk != nil {
		scoredFiles = len(artifacts.Risk.Files)
	}
	run.Summary = &domain.AnalysisSummary{
		FilesDiscovered: run.Progress.FilesDiscovered,
		FilesAnalyzed:   run.Progress.FilesAnalyzed,
		FilesWithErrors: run.Progress.FilesWithErrors,
		GitStatus:       stageStatus(run, domain.AnalysisStageGit),
		RiskStatus:      stageStatus(run, domain.AnalysisStageRisk),
		ScoredFiles:     scoredFiles,
		UnscoredFiles:   len(staticReport.Files) - scoredFiles,
	}
	completed := time.Now().UTC()
	run.Status = domain.AnalysisStatusCompleted
	run.CompletedAt = &completed
	run.CurrentStage = nil
	run.Progress.Percent = 100
	if err := orchestrator.store.SaveReport(context.Background(), run.ID, domain.AnalysisReport{Analysis: run, Artifacts: artifacts}); err != nil {
		orchestrator.failRun(&run, domain.AnalysisStageRisk, "PERSISTENCE_FAILED", "The completed local analysis report could not be stored.")
	}
}

func (orchestrator *Orchestrator) recordStaticProgress(run *domain.AnalysisRun, completed, total int, file domain.FileAnalysis) {
	run.Progress.FilesAnalyzed = completed
	status := domain.StageStatusRunning
	if completed == total {
		status = domain.StageStatusCompleted
	}
	setStage(run, domain.AnalysisStageStatic, status, domain.StageProgress{Total: total, Completed: completed})
	if file.Status == domain.AnalysisStatusReadError || file.HasParserFailure() || file.Status == domain.AnalysisStatusUnsupported {
		run.Progress.FilesWithErrors++
		if len(file.Diagnostics) == 0 {
			appendDiagnostic(run, domain.AnalysisStageStatic, file.Path, "STATIC_FILE_ERROR", "warning", "A project file could not be analyzed completely.")
		} else {
			for _, diagnostic := range file.Diagnostics {
				appendDiagnostic(run, domain.AnalysisStageStatic, file.Path, string(diagnostic.Code), diagnostic.Severity, diagnostic.Message)
			}
		}
	}
	_ = orchestrator.saveRun(run)
}

func (orchestrator *Orchestrator) saveRun(run *domain.AnalysisRun) bool {
	run.Progress.Percent = progressPercent(*run)
	if err := orchestrator.store.SaveRun(context.Background(), *run); err != nil {
		return false
	}
	return true
}

func (orchestrator *Orchestrator) failRun(run *domain.AnalysisRun, stage domain.AnalysisStage, code, message string) {
	setStage(run, stage, domain.StageStatusFailed, stageProgress(*run, stage))
	skipNonTerminalStages(run)
	appendDiagnostic(run, stage, "", code, "error", message)
	finished := time.Now().UTC()
	run.Status = domain.AnalysisStatusFailed
	run.CurrentStage = nil
	run.CompletedAt = &finished
	_ = orchestrator.store.SaveRun(context.Background(), *run)
}

func (orchestrator *Orchestrator) cancelRun(run *domain.AnalysisRun) {
	now := time.Now().UTC()
	for index := range run.Stages {
		switch run.Stages[index].Status {
		case domain.StageStatusRunning:
			run.Stages[index].Status = domain.StageStatusCancelled
			run.Stages[index].CompletedAt = &now
		case domain.StageStatusPending:
			run.Stages[index].Status = domain.StageStatusSkipped
			run.Stages[index].CompletedAt = &now
		}
	}
	appendDiagnostic(run, domain.AnalysisStageValidation, "", "ANALYSIS_CANCELLED", "warning", "The local analysis was cancelled before completion.")
	finished := time.Now().UTC()
	run.Status = domain.AnalysisStatusCancelled
	run.CurrentStage = nil
	run.CompletedAt = &finished
	_ = orchestrator.store.SaveRun(context.Background(), *run)
}

func skipNonTerminalStages(run *domain.AnalysisRun) {
	now := time.Now().UTC()
	for index := range run.Stages {
		if run.Stages[index].Status == domain.StageStatusRunning || run.Stages[index].Status == domain.StageStatusPending {
			run.Stages[index].Status = domain.StageStatusSkipped
			run.Stages[index].CompletedAt = &now
		}
	}
}

func (orchestrator *Orchestrator) unregister(id domain.AnalysisID) {
	orchestrator.mu.Lock()
	defer orchestrator.mu.Unlock()
	delete(orchestrator.cancels, id)
}

func initialStages() []domain.StageSnapshot {
	return []domain.StageSnapshot{
		{Stage: domain.AnalysisStageValidation, Status: domain.StageStatusPending},
		{Stage: domain.AnalysisStageDiscovery, Status: domain.StageStatusPending},
		{Stage: domain.AnalysisStageStatic, Status: domain.StageStatusPending},
		{Stage: domain.AnalysisStageGit, Status: domain.StageStatusPending},
		{Stage: domain.AnalysisStageRisk, Status: domain.StageStatusPending},
	}
}

func setCurrentStage(run *domain.AnalysisRun, stage domain.AnalysisCurrentStage) {
	run.CurrentStage = &stage
}

func setStage(run *domain.AnalysisRun, stage domain.AnalysisStage, status domain.StageStatus, progress domain.StageProgress) {
	for index := range run.Stages {
		if run.Stages[index].Stage != stage {
			continue
		}
		now := time.Now().UTC()
		if status == domain.StageStatusRunning && run.Stages[index].StartedAt == nil {
			run.Stages[index].StartedAt = &now
		}
		if isTerminalStage(status) {
			run.Stages[index].CompletedAt = &now
		}
		run.Stages[index].Status = status
		run.Stages[index].Progress = progress
		return
	}
}

func stageStatus(run domain.AnalysisRun, stage domain.AnalysisStage) domain.StageStatus {
	for _, snapshot := range run.Stages {
		if snapshot.Stage == stage {
			return snapshot.Status
		}
	}
	return domain.StageStatusPending
}

func stageProgress(run domain.AnalysisRun, stage domain.AnalysisStage) domain.StageProgress {
	for _, snapshot := range run.Stages {
		if snapshot.Stage == stage {
			return snapshot.Progress
		}
	}
	return domain.StageProgress{}
}

func appendDiagnostics(run *domain.AnalysisRun, diagnostics []domain.AnalysisDiagnostic) {
	for _, diagnostic := range diagnostics {
		if diagnostic.RecordedAt.IsZero() {
			diagnostic.RecordedAt = time.Now().UTC()
		}
		run.Diagnostics = append(run.Diagnostics, diagnostic)
	}
}

func appendDiagnostic(run *domain.AnalysisRun, stage domain.AnalysisStage, path, code, severity, message string) {
	appendDiagnostics(run, []domain.AnalysisDiagnostic{{Stage: stage, Path: path, Code: code, Severity: severity, Message: message}})
}

func progressPercent(run domain.AnalysisRun) int {
	weights := map[domain.AnalysisStage]int{
		domain.AnalysisStageValidation: 5,
		domain.AnalysisStageDiscovery:  15,
		domain.AnalysisStageStatic:     40,
		domain.AnalysisStageGit:        25,
		domain.AnalysisStageRisk:       15,
	}
	percent := 0
	for _, stage := range run.Stages {
		weight := weights[stage.Stage]
		switch stage.Status {
		case domain.StageStatusCompleted, domain.StageStatusUnavailable, domain.StageStatusSkipped:
			percent += weight
		case domain.StageStatusRunning:
			if stage.Progress.Total > 0 {
				percent += weight * stage.Progress.Completed / stage.Progress.Total
			}
		}
	}
	if percent > 100 {
		return 100
	}
	return percent
}

func isTerminal(status domain.AnalysisRunStatus) bool {
	return status == domain.AnalysisStatusCompleted || status == domain.AnalysisStatusFailed || status == domain.AnalysisStatusCancelled
}

func isTerminalStage(status domain.StageStatus) bool {
	return status == domain.StageStatusCompleted || status == domain.StageStatusUnavailable || status == domain.StageStatusFailed || status == domain.StageStatusCancelled || status == domain.StageStatusSkipped
}

func newAnalysisID() domain.AnalysisID {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return domain.AnalysisID(fmt.Sprintf("analysis_%d", time.Now().UTC().UnixNano()))
	}
	return domain.AnalysisID("analysis_" + hex.EncodeToString(bytes))
}

func cloneRiskProfile(profile *domain.RiskProfile) *domain.RiskProfile {
	if profile == nil {
		return nil
	}
	copy := *profile
	copy.Weights = make(map[domain.RiskMetric]float64, len(profile.Weights))
	for metric, weight := range profile.Weights {
		copy.Weights[metric] = weight
	}
	return &copy
}

func buildScorableInputs(staticFiles []domain.FileAnalysis, gitReport *domain.GitAnalysisReport) []domain.HotspotInput {
	eligible := make([]domain.FileAnalysis, 0, len(staticFiles))
	for _, file := range staticFiles {
		if file.Status == domain.AnalysisStatusAnalyzed {
			eligible = append(eligible, file)
		}
	}
	var gitFiles []domain.GitFileMetrics
	if gitReport != nil {
		gitFiles = gitReport.Files
	}
	return BuildHotspotInputs(eligible, gitFiles)
}

func rebaseGitReport(report domain.GitAnalysisReport, gitRelativeScope string) domain.GitAnalysisReport {
	if gitRelativeScope == "" {
		return report
	}
	prefix := strings.Trim(strings.ReplaceAll(gitRelativeScope, "\\", "/"), "/")
	files := make([]domain.GitFileMetrics, 0, len(report.Files))
	for _, file := range report.Files {
		path := strings.TrimPrefix(file.Path, prefix+"/")
		if path == file.Path || path == "" {
			continue
		}
		file.Path = path
		previous := make([]string, 0, len(file.PreviousPaths))
		for _, previousPath := range file.PreviousPaths {
			if rebased := strings.TrimPrefix(previousPath, prefix+"/"); rebased != previousPath && rebased != "" {
				previous = append(previous, rebased)
			}
		}
		file.PreviousPaths = previous
		files = append(files, file)
	}
	sort.Slice(files, func(left, right int) bool { return files[left].Path < files[right].Path })
	report.Files = files
	return report
}
