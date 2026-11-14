package api

import (
	"encoding/csv"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/Erfangh-80/Code-analysis/backend/internal/domain"
)

type heatValue struct {
	Name        string
	X, Y, Score float64
}

func buildHeatmap(values []heatValue, bins int) []domain.HeatmapCell {
	cells := make([]domain.HeatmapCell, bins*bins)
	for y := 0; y < bins; y++ {
		for x := 0; x < bins; x++ {
			cells[y*bins+x] = domain.HeatmapCell{X: x, Y: y, Paths: []string{}}
		}
	}
	totals := make([]float64, len(cells))
	for _, value := range values {
		x := bin(value.X, bins)
		y := bin(value.Y, bins)
		index := y*bins + x
		cells[index].Count++
		cells[index].Paths = append(cells[index].Paths, value.Name)
		totals[index] += value.Score
	}
	for i := range cells {
		if cells[i].Count > 0 {
			average := totals[i] / float64(cells[i].Count)
			cells[i].AverageScore = &average
			sort.Strings(cells[i].Paths)
		}
	}
	return cells
}
func bin(value float64, bins int) int {
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}
	index := int(math.Floor(value * float64(bins)))
	if index == bins {
		index = bins - 1
	}
	return index
}
func filterHeatmap(files []domain.FileResult, x, y, bins int) []domain.FileResult {
	out := []domain.FileResult{}
	for _, file := range files {
		if file.HotspotScore == nil || file.Normalized.Churn == nil || file.Normalized.Complexity == nil {
			continue
		}
		if bin(*file.Normalized.Churn, bins) == x && bin(*file.Normalized.Complexity, bins) == y {
			out = append(out, file)
		}
	}
	return out
}
func moduleHeatValues(files []domain.FileResult) []heatValue {
	type acc struct {
		score, x, y float64
		n           int
	}
	groups := map[string]*acc{}
	for _, file := range files {
		if file.HotspotScore == nil || file.Normalized.Churn == nil || file.Normalized.Complexity == nil {
			continue
		}
		group := groups[file.Module]
		if group == nil {
			group = &acc{}
			groups[file.Module] = group
		}
		group.score += *file.HotspotScore
		group.x += *file.Normalized.Churn
		group.y += *file.Normalized.Complexity
		group.n++
	}
	out := []heatValue{}
	for name, group := range groups {
		if group.n > 0 {
			out = append(out, heatValue{Name: name, Score: group.score / float64(group.n), X: group.x / float64(group.n), Y: group.y / float64(group.n)})
		}
	}
	return out
}

func writeFileCSV(w http.ResponseWriter, files []domain.FileResult) int {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="hotspots-files.csv"`)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"path", "module", "language", "loc", "max_function_complexity", "churn", "change_frequency", "author_count", "hotspot_score", "analysis_status"})
	for _, file := range files {
		_ = writer.Write([]string{safeCSV(file.Path), safeCSV(file.Module), safeCSV(file.Static.Language), nullableInt(file.Static.LOC), nullableInt(file.Static.MaxFunctionComplexity), strconv.Itoa(file.Git.Churn), strconv.Itoa(file.Git.ChangeFrequency), strconv.Itoa(file.Git.AuthorCount), nullableFloat(file.HotspotScore), safeCSV(file.Static.AnalysisStatus)})
	}
	writer.Flush()
	return 200
}
func writeModuleCSV(w http.ResponseWriter, modules []domain.ModuleResult) int {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="hotspots-modules.csv"`)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"module", "file_count", "scorable_file_count", "total_loc", "total_churn", "unique_commit_count", "author_count", "average_score", "highest_score"})
	for _, module := range modules {
		_ = writer.Write([]string{safeCSV(module.Name), strconv.Itoa(module.FileCount), strconv.Itoa(module.ScorableFileCount), strconv.Itoa(module.TotalLOC), strconv.Itoa(module.TotalChurn), strconv.Itoa(module.UniqueCommitCount), strconv.Itoa(module.AuthorCount), nullableFloat(module.AverageScore), nullableFloat(module.HighestScore)})
	}
	writer.Flush()
	return 200
}
func nullableInt(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}
func nullableFloat(value *float64) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%.6f", *value)
}

// Formula-looking cells are deliberately escaped to protect spreadsheet users.
func safeCSV(value string) string {
	if value == "" {
		return value
	}
	if strings.ContainsAny(value[:1], "=+-@\t\r") {
		return "'" + value
	}
	return value
}
