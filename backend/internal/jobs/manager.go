// Package jobs provides the intentionally small, persistent-at-boundary analysis queue.
package jobs

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Erfangh-80/Code-analysis/backend/internal/analyzer/snapshot"
	static "github.com/Erfangh-80/Code-analysis/backend/internal/analyzer/static"
	"github.com/Erfangh-80/Code-analysis/backend/internal/domain"
	"github.com/Erfangh-80/Code-analysis/backend/internal/evaluation"
	"github.com/Erfangh-80/Code-analysis/backend/internal/githistory"
	"github.com/Erfangh-80/Code-analysis/backend/internal/scoring"
	"github.com/Erfangh-80/Code-analysis/backend/internal/storage"
)

type Config struct {
	AllowedRepoRoot string
	MaxCommits      int
	AnalysisTimeout time.Duration
	MaxFileBytes    int64
}
type Manager struct {
	store  *storage.Store
	config Config
	queue  chan string
	stop   chan struct{}
	once   sync.Once
}

type JobError struct{ Code, Message string }

func (e *JobError) Error() string { return e.Message }

func New(store *storage.Store, config Config) (*Manager, error) {
	if config.AllowedRepoRoot == "" {
		return nil, fmt.Errorf("allowed repository root is required")
	}
	if config.MaxCommits <= 0 {
		config.MaxCommits = 5000
	}
	if config.AnalysisTimeout <= 0 {
		config.AnalysisTimeout = 10 * time.Minute
	}
	resolved, err := filepath.EvalSymlinks(config.AllowedRepoRoot)
	if err != nil {
		return nil, fmt.Errorf("allowed repository root is unavailable")
	}
	config.AllowedRepoRoot = resolved
	if config.MaxFileBytes > 0 {
		static.MaxFileBytes = config.MaxFileBytes
	}
	m := &Manager{store: store, config: config, queue: make(chan string, 100), stop: make(chan struct{})}
	go m.worker()
	return m, nil
}
func (m *Manager) Close() { m.once.Do(func() { close(m.stop) }) }

func (m *Manager) Submit(input domain.AnalysisInput) (domain.Analysis, error) {
	if err := m.validateInput(&input); err != nil {
		return domain.Analysis{}, err
	}
	now := time.Now().UTC()
	if input.Ref == "" {
		input.Ref = "HEAD"
	}
	if input.History.Until.IsZero() {
		input.History.Until = now
	}
	if input.History.Since.IsZero() {
		input.History.Since = input.History.Until.AddDate(0, 0, -180)
	}
	if !input.History.Since.Before(input.History.Until) {
		return domain.Analysis{}, &JobError{"INVALID_HISTORY_RANGE", "بازه تاریخ باید به صورت [since, until) معتبر باشد."}
	}
	analysis := domain.Analysis{ID: randomID(), Status: domain.StatusQueued, RepositoryPath: input.RepositoryPath, Ref: input.Ref, Languages: input.Languages, History: input.History, Weights: input.Weights, FormulaVersion: domain.FormulaVersion, HistoryComplete: true, CurrentStage: "queued", Warnings: []string{}, CreatedAt: now}
	if err := m.store.CreateAnalysis(analysis); err != nil {
		return domain.Analysis{}, err
	}
	select {
	case m.queue <- analysis.ID:
		return analysis, nil
	default:
		return domain.Analysis{}, &JobError{"QUEUE_FULL", "صف تحلیل موقتاً پر است."}
	}
}
func (m *Manager) validateInput(input *domain.AnalysisInput) error {
	if err := m.validateRepositoryPath(input.RepositoryPath); err != nil {
		return err
	}
	if input.Ref != "" && input.Ref != "HEAD" && !fullSHA(input.Ref) {
		return &JobError{"INVALID_REF", "ref فقط می‌تواند HEAD یا SHA کامل commit باشد."}
	}
	if len(input.Languages) == 0 {
		input.Languages = []string{"go", "python", "javascript"}
	}
	seen := map[string]bool{}
	for _, language := range input.Languages {
		if err := scoring.ValidateLanguage(language); err != nil {
			return &JobError{"INVALID_LANGUAGE", "زبان انتخاب‌شده پشتیبانی نمی‌شود."}
		}
		if seen[language] {
			return &JobError{"INVALID_LANGUAGE", "هر زبان فقط یک بار قابل انتخاب است."}
		}
		seen[language] = true
	}
	if err := scoring.ValidateWeights(input.Weights); err != nil {
		return &JobError{"INVALID_WEIGHTS", "وزن‌ها باید نامنفی و دارای مجموع بزرگ‌تر از صفر باشند."}
	}
	return nil
}
func (m *Manager) validateRepositoryPath(value string) error {
	if value == "" {
		return &JobError{"INVALID_REPO_PATH", "مسیر مخزن الزامی است."}
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return &JobError{"INVALID_REPO_PATH", "مسیر مخزن معتبر نیست."}
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return &JobError{"INVALID_REPO_PATH", "مسیر مخزن قابل خواندن نیست."}
	}
	if !within(m.config.AllowedRepoRoot, resolved) {
		return &JobError{"REPO_NOT_ALLOWED", "مسیر مخزن خارج از محدوده مجاز است."}
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return &JobError{"INVALID_REPO_PATH", "مسیر مخزن یک پوشه قابل خواندن نیست."}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := githistory.IsRepository(ctx, resolved); err != nil {
		return &JobError{"INVALID_REPOSITORY", "مسیر باید یک مخزن Git دارای commit باشد."}
	}
	// Reject Git metadata/object stores outside the configured trust boundary.
	for _, args := range [][]string{{"rev-parse", "--absolute-git-dir"}, {"rev-parse", "--git-path", "objects"}} {
		gitPath, err := gitOutput(ctx, resolved, args...)
		if err != nil {
			return &JobError{"INVALID_REPOSITORY", "متادیتای Git قابل خواندن نیست."}
		}
		gitPath = strings.TrimSpace(gitPath)
		if !filepath.IsAbs(gitPath) {
			gitPath = filepath.Join(resolved, gitPath)
		}
		real, err := filepath.EvalSymlinks(gitPath)
		if err != nil || !within(m.config.AllowedRepoRoot, real) {
			return &JobError{"REPO_NOT_ALLOWED", "متادیتا یا object storage مخزن خارج از محدوده مجاز است."}
		}
	}
	return nil
}
func fullSHA(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, char := range value {
		if !(char >= '0' && char <= '9') && !(char >= 'a' && char <= 'f') && !(char >= 'A' && char <= 'F') {
			return false
		}
	}
	return true
}
func within(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}
func gitOutput(ctx context.Context, repository string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repository}, args...)...)
	out, err := cmd.Output()
	return string(out), err
}

func (m *Manager) worker() {
	for {
		select {
		case <-m.stop:
			return
		case id := <-m.queue:
			m.run(id)
		}
	}
}
func (m *Manager) run(id string) {
	analysis, err := m.store.GetAnalysis(id)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	analysis.Status = domain.StatusRunning
	analysis.CurrentStage = "validating snapshot"
	analysis.StartedAt = &now
	if m.store.UpdateAnalysis(analysis) != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), m.config.AnalysisTimeout)
	defer cancel()
	fail := func(code, message string) {
		finished := time.Now().UTC()
		analysis.Status = domain.StatusFailed
		analysis.CurrentStage = "failed"
		analysis.FailureCode = &code
		analysis.FailureMessage = &message
		analysis.FinishedAt = &finished
		_ = m.store.UpdateAnalysis(analysis)
	}
	commit, err := githistory.ResolveCommit(ctx, analysis.RepositoryPath, analysis.Ref)
	if err != nil {
		fail("INVALID_REF", "commit انتخاب‌شده معتبر یا قابل دسترس نیست.")
		return
	}
	analysis.CommitSHA = commit
	languages := map[string]bool{}
	for _, v := range analysis.Languages {
		languages[v] = true
	}
	files, err := snapshot.List(ctx, analysis.RepositoryPath, commit, languages)
	if err != nil {
		fail("SNAPSHOT_READ_FAILED", "خواندن snapshot ثبت‌شده در commit با خطا مواجه شد.")
		return
	}
	analysis.FilesDiscovered = len(files)
	analysis.CurrentStage = "analyzing source files"
	_ = m.store.UpdateAnalysis(analysis)
	results := make([]domain.FileResult, 0, len(files))
	paths := map[string]struct{}{}
	hasStaticIssue := false
	for _, item := range files {
		content, readErr := snapshot.Read(ctx, analysis.RepositoryPath, commit, item.Path)
		var metrics domain.StaticMetrics
		if readErr != nil {
			message := "خواندن فایل از snapshot ممکن نبود."
			metrics = domain.StaticMetrics{Language: item.Language, AnalysisStatus: "read_error", AnalysisError: &message}
		} else {
			metrics = static.Analyze(item.Path, content)
		}
		results = append(results, domain.FileResult{ID: fileID(analysis.ID, item.Path), AnalysisID: analysis.ID, Path: item.Path, Module: scoring.ModuleForPath(item.Path), Static: metrics})
		paths[item.Path] = struct{}{}
		analysis.FilesProcessed++
		if metrics.AnalysisStatus != "ok" {
			hasStaticIssue = true
			analysis.Warnings = append(analysis.Warnings, item.Path+": "+safeError(metrics.AnalysisError))
		}
		_ = m.store.UpdateAnalysis(analysis)
	}
	analysis.CurrentStage = "analyzing Git history"
	_ = m.store.UpdateAnalysis(analysis)
	history, err := githistory.CollectContext(ctx, analysis.RepositoryPath, commit, paths, githistory.Options{Since: analysis.History.Since, Until: analysis.History.Until, MaxCommits: m.config.MaxCommits})
	if err != nil {
		fail("GIT_HISTORY_FAILED", "تحلیل تاریخچه Git با خطا مواجه شد؛ نتیجه نهایی تولید نشد.")
		return
	}
	analysis.HistoryComplete = history.Complete
	analysis.Warnings = append(analysis.Warnings, history.Warnings...)
	for i := range results {
		results[i].Git = history.Metrics[results[i].Path]
	}
	analysis.CurrentStage = "scoring hotspots"
	_ = m.store.UpdateAnalysis(analysis)
	analysis.Warnings = append(analysis.Warnings, scoring.Apply(results, analysis.Weights)...)
	scoring.RankFiles(results)
	modules := scoring.AggregateModules(analysis.ID, results, history.CommitPaths, history.AuthorChanges)
	if err := m.store.SaveFiles(analysis.ID, results); err != nil {
		fail("STORAGE_FAILED", "ذخیره نتایج فایل‌ها با خطا مواجه شد.")
		return
	}
	if err := m.store.SaveModules(analysis.ID, modules); err != nil {
		fail("STORAGE_FAILED", "ذخیره نتایج ماژول‌ها با خطا مواجه شد.")
		return
	}
	finished := time.Now().UTC()
	analysis.Status = domain.StatusCompleted
	if !analysis.HistoryComplete || hasStaticIssue {
		analysis.Status = domain.StatusPartial
	}
	analysis.CurrentStage = "completed"
	analysis.FinishedAt = &finished
	_ = m.store.UpdateAnalysis(analysis)
}
func randomID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
func fileID(analysis, path string) string {
	value := sha256.Sum256([]byte(analysis + "\x00" + path))
	return hex.EncodeToString(value[:12])
}
func safeError(value *string) string {
	if value == nil {
		return "analysis unavailable"
	}
	return *value
}

// StartEvaluation persists a queued evaluation then executes it away from the
// request goroutine. Evaluation never changes the original hotspot score.
func (m *Manager) StartEvaluation(analysisID string, history domain.HistoryRange) (domain.Evaluation, error) {
	analysis, err := m.store.GetAnalysis(analysisID)
	if err != nil {
		return domain.Evaluation{}, err
	}
	if analysis.Status != domain.StatusCompleted && analysis.Status != domain.StatusPartial {
		return domain.Evaluation{}, &JobError{"ANALYSIS_NOT_READY", "ارزیابی فقط پس از پایان تحلیل قابل اجرا است."}
	}
	now := time.Now().UTC()
	if history.Until.IsZero() {
		history.Until = now
	}
	if history.Since.IsZero() {
		history.Since = analysis.CreatedAt
	}
	if !history.Since.Before(history.Until) {
		return domain.Evaluation{}, &JobError{"INVALID_EVALUATION_RANGE", "بازه ارزیابی باید به صورت [since, until) معتبر باشد."}
	}
	result := domain.Evaluation{ID: randomID(), AnalysisID: analysisID, Status: domain.StatusQueued, Since: history.Since, Until: history.Until, CreatedAt: now}
	if err := m.store.SaveEvaluation(result); err != nil {
		return domain.Evaluation{}, err
	}
	go m.runEvaluation(result, analysis)
	return result, nil
}
func (m *Manager) runEvaluation(result domain.Evaluation, analysis domain.Analysis) {
	result.Status = domain.StatusRunning
	_ = m.store.SaveEvaluation(result)
	ctx, cancel := context.WithTimeout(context.Background(), m.config.AnalysisTimeout)
	defer cancel()
	files, err := m.store.AllFiles(analysis.ID)
	if err == nil {
		result.SampleSize, result.PotentialFixCount, result.Spearman, result.Reason, err = evaluation.Run(ctx, analysis.RepositoryPath, analysis.CommitSHA, files, evaluation.Options{Since: result.Since, Until: result.Until, MaxCommits: m.config.MaxCommits})
	}
	finished := time.Now().UTC()
	result.FinishedAt = &finished
	if err != nil {
		result.Status = domain.StatusFailed
		result.Reason = stringPtr("ارزیابی تاریخچه Git با خطا مواجه شد.")
	} else {
		result.Status = domain.StatusCompleted
	}
	_ = m.store.SaveEvaluation(result)
}
func stringPtr(value string) *string { return &value }
