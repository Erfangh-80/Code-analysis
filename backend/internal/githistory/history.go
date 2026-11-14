// Package githistory reads a local Git repository through the installed Git
// executable. It never invokes user code and passes arguments without a shell.
// This replaces go-git in this MVP because the build environment is restricted
// to offline-compatible dependencies; Git is an explicit runtime prerequisite.
package githistory

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Erfangh-80/Code-analysis/backend/internal/domain"
)

type Options struct {
	Since      time.Time
	Until      time.Time
	MaxCommits int
}

type Result struct {
	Metrics map[string]domain.GitMetrics
	// CommitPaths is transient aggregation data: SHA values are not persisted.
	CommitPaths map[string][]string
	// AuthorChanges is transient, in-memory aggregation data. It is never
	// persisted or returned from the public API.
	AuthorChanges map[string]map[string]int
	Complete      bool
	Warnings      []string
}

type commitInfo struct {
	sha     string
	parents []string
	at      time.Time
	author  string
}

func ResolveCommit(ctx context.Context, repoPath, ref string) (string, error) {
	if ref == "" {
		ref = "HEAD"
	}
	value, err := command(ctx, repoPath, "rev-parse", "--verify", ref+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("invalid or inaccessible commit reference")
	}
	return strings.TrimSpace(value), nil
}

func IsRepository(ctx context.Context, repoPath string) error {
	value, err := command(ctx, repoPath, "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(value) != "true" {
		return fmt.Errorf("path is not a readable Git working tree")
	}
	_, err = ResolveCommit(ctx, repoPath, "HEAD")
	if err != nil {
		return fmt.Errorf("repository has no readable commit")
	}
	return nil
}

// Collect evaluates [Since, Until) from commits reachable from ref. Merge
// commits are intentionally excluded to avoid duplicate changes.
func Collect(repoPath, ref string, paths map[string]struct{}, options Options) (Result, error) {
	return CollectContext(context.Background(), repoPath, ref, paths, options)
}
func CollectContext(ctx context.Context, repoPath, ref string, paths map[string]struct{}, options Options) (Result, error) {
	if options.MaxCommits <= 0 {
		options.MaxCommits = 5000
	}
	if options.Until.IsZero() {
		options.Until = time.Now().UTC()
	}
	if options.Since.IsZero() {
		options.Since = options.Until.AddDate(0, 0, -180)
	}
	if !options.Since.Before(options.Until) {
		return Result{}, fmt.Errorf("history since must be before until")
	}
	sha, err := ResolveCommit(ctx, repoPath, ref)
	if err != nil {
		return Result{}, err
	}
	output, err := command(ctx, repoPath, "rev-list", "--topo-order", sha)
	if err != nil {
		return Result{}, fmt.Errorf("unable to read commit history")
	}
	result := Result{Metrics: make(map[string]domain.GitMetrics, len(paths)), CommitPaths: make(map[string][]string, len(paths)), AuthorChanges: make(map[string]map[string]int, len(paths)), Complete: true}
	// Author identifiers stay in this local map only and are reduced to counts
	// before the result leaves the analyzer.
	authorsByPath := make(map[string]map[string]int, len(paths))
	for p := range paths {
		result.Metrics[p] = domain.GitMetrics{}
		authorsByPath[p] = map[string]int{}
		result.AuthorChanges[p] = authorsByPath[p]
	}
	shallow, _ := command(ctx, repoPath, "rev-parse", "--is-shallow-repository")
	if strings.TrimSpace(shallow) == "true" {
		result.Complete = false
		result.Warnings = append(result.Warnings, "repository is shallow; reachable history may be incomplete")
	}

	commits := strings.Fields(output)
	seen := 0
	for _, candidate := range commits {
		if seen >= options.MaxCommits {
			result.Complete = false
			result.Warnings = append(result.Warnings, fmt.Sprintf("commit limit of %d reached; history metrics are partial", options.MaxCommits))
			break
		}
		info, err := readCommit(ctx, repoPath, candidate)
		if err != nil {
			return result, fmt.Errorf("unable to read commit metadata")
		}
		seen++
		if !inRange(info.at, options.Since, options.Until) || len(info.parents) > 1 {
			continue
		}
		// --root makes an initial commit differ against an empty tree. No rename
		// detection is requested, so results remain explicitly path-based.
		diff, err := command(ctx, repoPath, "diff-tree", "--root", "--no-commit-id", "--no-renames", "--numstat", "-r", info.sha)
		if err != nil {
			return result, fmt.Errorf("unable to calculate Git diff")
		}
		for _, row := range strings.Split(strings.TrimSuffix(diff, "\n"), "\n") {
			parts := strings.SplitN(row, "\t", 3)
			if len(parts) != 3 {
				continue
			}
			if _, ok := paths[parts[2]]; !ok {
				continue
			}
			added, aOK := parseNumStat(parts[0])
			deleted, dOK := parseNumStat(parts[1])
			if !aOK || !dOK {
				result.Complete = false
				result.Warnings = append(result.Warnings, "binary diff encountered; its line churn is unavailable")
				continue
			}
			metric := result.Metrics[parts[2]]
			metric.Churn += added + deleted
			metric.ChangeFrequency++
			result.CommitPaths[parts[2]] = append(result.CommitPaths[parts[2]], info.sha)
			authorsByPath[parts[2]][info.author]++
			updateTimes(&metric, info.at)
			result.Metrics[parts[2]] = metric
		}
	}
	// Author IDs are only transient in the local map and are reduced to
	// descriptive counts here; names and email addresses are never persisted.
	for filePath, authors := range authorsByPath {
		metric := result.Metrics[filePath]
		if metric.ChangeFrequency == 0 {
			continue
		}
		metric.AuthorCount = len(authors)
		concentration := 0.0
		for _, changes := range authors {
			share := float64(changes) / float64(metric.ChangeFrequency)
			concentration += share * share
		}
		dispersion := 1 - concentration
		metric.AuthorConcentration, metric.AuthorDispersion = &concentration, &dispersion
		result.Metrics[filePath] = metric
	}
	return result, nil
}

func parseNumStat(value string) (int, bool) {
	if value == "-" {
		return 0, false
	}
	v, err := strconv.Atoi(value)
	return v, err == nil && v >= 0
}
func inRange(value, since, until time.Time) bool { return !value.Before(since) && value.Before(until) }

func readCommit(ctx context.Context, repoPath, sha string) (commitInfo, error) {
	// NUL separators make fields unambiguous and avoid retaining commit subject.
	out, err := command(ctx, repoPath, "show", "-s", "--format=%H%x00%P%x00%aI%x00%aE", sha)
	if err != nil {
		return commitInfo{}, err
	}
	parts := strings.Split(strings.TrimSuffix(out, "\n"), "\x00")
	if len(parts) != 4 {
		return commitInfo{}, fmt.Errorf("unexpected commit metadata")
	}
	at, err := time.Parse(time.RFC3339, parts[2])
	if err != nil {
		return commitInfo{}, err
	}
	return commitInfo{sha: parts[0], parents: strings.Fields(parts[1]), at: at.UTC(), author: parts[3]}, nil
}
func updateTimes(m *domain.GitMetrics, at time.Time) {
	if m.FirstChange == nil || at.Before(*m.FirstChange) {
		value := at
		m.FirstChange = &value
	}
	if m.LastChange == nil || at.After(*m.LastChange) {
		value := at
		m.LastChange = &value
	}
}
func command(ctx context.Context, repoPath string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repoPath}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git command failed")
	}
	return string(out), nil
}
