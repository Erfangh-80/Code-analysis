package scoring

import (
	"math"
	"sort"

	"github.com/Erfangh-80/Code-analysis/backend/internal/domain"
)

// AggregateModules uses the fixed MVP module definition: first directory, or
// "." for repository-root files. Identity maps are only inputs in memory.
func AggregateModules(analysisID string, files []domain.FileResult, commits map[string][]string, authors map[string]map[string]int) []domain.ModuleResult {
	type accumulator struct {
		module                                    domain.ModuleResult
		scores, complexities, churns, frequencies []float64
		commits                                   map[string]struct{}
		authors                                   map[string]int
	}
	groups := map[string]*accumulator{}
	for _, file := range files {
		name := ModuleForPath(file.Path)
		file.Module = name
		group := groups[name]
		if group == nil {
			group = &accumulator{module: domain.ModuleResult{ID: analysisID + ":" + name, AnalysisID: analysisID, Name: name}, commits: map[string]struct{}{}, authors: map[string]int{}}
			groups[name] = group
		}
		group.module.FileCount++
		if file.Static.LOC != nil {
			group.module.TotalLOC += *file.Static.LOC
		}
		group.module.TotalChurn += file.Git.Churn
		if file.HotspotScore != nil {
			group.module.ScorableFileCount++
			group.scores = append(group.scores, *file.HotspotScore)
		}
		if file.Static.MaxFunctionComplexity != nil {
			group.complexities = append(group.complexities, float64(*file.Static.MaxFunctionComplexity))
		}
		group.churns = append(group.churns, float64(file.Git.Churn))
		group.frequencies = append(group.frequencies, float64(file.Git.ChangeFrequency))
		for _, sha := range commits[file.Path] {
			group.commits[sha] = struct{}{}
		}
		for author, count := range authors[file.Path] {
			group.authors[author] += count
		}
	}
	output := make([]domain.ModuleResult, 0, len(groups))
	for _, group := range groups {
		m := group.module
		m.UniqueCommitCount = len(group.commits)
		m.AuthorCount = len(group.authors)
		m.AverageScore = average(group.scores)
		m.HighestScore = highest(group.scores)
		m.AverageComplexity = average(group.complexities)
		m.AverageChurn = average(group.churns)
		m.AverageFrequency = average(group.frequencies)
		if len(group.authors) > 0 {
			total := 0
			for _, n := range group.authors {
				total += n
			}
			c := 0.0
			for _, n := range group.authors {
				s := float64(n) / float64(total)
				c += s * s
			}
			d := 1 - c
			m.AuthorConcentration = &c
			m.AuthorDispersion = &d
		}
		output = append(output, m)
	}
	sort.Slice(output, func(i, j int) bool {
		left, right := output[i].HighestScore, output[j].HighestScore
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		if math.Abs(*left-*right) < 1e-12 {
			return output[i].Name < output[j].Name
		}
		return *left > *right
	})
	return output
}
func average(values []float64) *float64 {
	if len(values) == 0 {
		return nil
	}
	total := 0.0
	for _, v := range values {
		total += v
	}
	result := total / float64(len(values))
	return &result
}
func highest(values []float64) *float64 {
	if len(values) == 0 {
		return nil
	}
	result := values[0]
	for _, v := range values[1:] {
		if v > result {
			result = v
		}
	}
	return &result
}
