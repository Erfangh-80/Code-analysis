// Package evaluation implements a deliberately approximate, time-safe
// bug-fix proxy evaluation. It never uses future commits in the hotspot score.
package evaluation

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Erfangh-80/Code-analysis/backend/internal/domain"
)

var fixSignal = regexp.MustCompile(`(?i)\b(fix|bug|resolve(?:d|s|ing)?)\b`)

type Options struct {
	Since, Until time.Time
	MaxCommits   int
}

// Run ranks existing snapshot scores against future commit messages that only
// *might* indicate a bug fix. The commit subject is never retained.
func Run(ctx context.Context, repository, snapshot string, files []domain.FileResult, options Options) (sampleSize, fixCommits int, spearman *float64, reason *string, err error) {
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
		return 0, 0, nil, stringp("بازه ارزیابی باید معتبر باشد."), nil
	}
	valid := map[string]float64{}
	for _, file := range files {
		if file.HotspotScore != nil {
			valid[file.Path] = *file.HotspotScore
		}
	}
	if len(valid) < 3 {
		return len(valid), 0, nil, stringp("برای محاسبه همبستگی حداقل سه فایل دارای امتیاز لازم است."), nil
	}
	ancestor, err := ancestorOfHead(ctx, repository, snapshot)
	if err != nil {
		return 0, 0, nil, nil, fmt.Errorf("unable to determine evaluation ancestry")
	}
	if !ancestor {
		return len(valid), 0, nil, stringp("commit snapshot در تاریخچه HEAD فعلی نیست؛ داده آینده قابل ارزیابی نیست."), nil
	}
	log, err := run(ctx, repository, "log", "--format=%H%x00%aI%x00%s", snapshot+"..HEAD")
	if err != nil {
		return 0, 0, nil, nil, fmt.Errorf("unable to inspect future commits")
	}
	counts := map[string]int{}
	for p := range valid {
		counts[p] = 0
	}
	checked := 0
	for _, line := range strings.Split(strings.TrimSuffix(log, "\n"), "\n") {
		if line == "" {
			continue
		}
		if checked >= options.MaxCommits {
			msg := "سقف commitهای ارزیابی رسید؛ ارزیابی ناقص است."
			return len(valid), fixCommits, nil, &msg, nil
		}
		parts := strings.SplitN(line, "\x00", 3)
		if len(parts) != 3 {
			continue
		}
		checked++
		at, parseErr := time.Parse(time.RFC3339, parts[1])
		if parseErr != nil {
			return 0, 0, nil, nil, parseErr
		}
		if at.Before(options.Since) || !at.Before(options.Until) || !fixSignal.MatchString(parts[2]) {
			continue
		}
		diff, err := run(ctx, repository, "diff-tree", "--root", "--no-commit-id", "--no-renames", "--numstat", "-r", parts[0])
		if err != nil {
			return 0, 0, nil, nil, fmt.Errorf("unable to inspect prospective fix diff")
		}
		touched := false
		for _, row := range strings.Split(strings.TrimSuffix(diff, "\n"), "\n") {
			columns := strings.SplitN(row, "\t", 3)
			if len(columns) == 3 {
				if _, ok := counts[columns[2]]; ok {
					counts[columns[2]]++
					touched = true
				}
			}
		}
		if touched {
			fixCommits++
		}
	}
	scores, fixes := make([]float64, 0, len(valid)), make([]float64, 0, len(valid))
	for file, score := range valid {
		scores = append(scores, score)
		fixes = append(fixes, float64(counts[file]))
	}
	if !hasVariance(scores) || !hasVariance(fixes) {
		return len(valid), fixCommits, nil, stringp("امتیازها یا تعداد commitهای احتمالی Bug-fix تنوع کافی ندارند."), nil
	}
	value := spearmanCoefficient(scores, fixes)
	return len(valid), fixCommits, &value, nil, nil
}
func ancestorOfHead(ctx context.Context, repo, snapshot string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repo, "merge-base", "--is-ancestor", snapshot, "HEAD")
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}
func run(ctx context.Context, repo string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
func hasVariance(values []float64) bool {
	if len(values) < 2 {
		return false
	}
	for _, value := range values[1:] {
		if value != values[0] {
			return true
		}
	}
	return false
}
func spearmanCoefficient(left, right []float64) float64 {
	a, b := ranks(left), ranks(right)
	n := float64(len(a))
	sumA, sumB := 0.0, 0.0
	for i := range a {
		sumA += a[i]
		sumB += b[i]
	}
	meanA, meanB := sumA/n, sumB/n
	top, da, db := 0.0, 0.0, 0.0
	for i := range a {
		x, y := a[i]-meanA, b[i]-meanB
		top += x * y
		da += x * x
		db += y * y
	}
	if da == 0 || db == 0 {
		return math.NaN()
	}
	return top / math.Sqrt(da*db)
}
func ranks(values []float64) []float64 {
	type item struct {
		value float64
		index int
	}
	items := make([]item, len(values))
	for i, v := range values {
		items[i] = item{v, i}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].value < items[j].value })
	out := make([]float64, len(values))
	for i := 0; i < len(items); {
		j := i + 1
		for j < len(items) && items[j].value == items[i].value {
			j++
		}
		rank := (float64(i+1) + float64(j)) / 2
		for k := i; k < j; k++ {
			out[items[k].index] = rank
		}
		i = j
	}
	return out
}
func stringp(value string) *string { return &value }
