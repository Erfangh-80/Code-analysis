// Package failedfiles provides a local, concurrency-safe registry of analysis failures.
package failedfiles

import (
	"sort"
	"sync"

	"github.com/Erfangh-80/code-analysis/backend/internal/domain"
)

// Registry stores parser and file-read failures observed by the current local process.
type Registry struct {
	mu      sync.RWMutex
	records []domain.FailureRecord
}

// NewRegistry creates an empty local failure registry.
func NewRegistry() *Registry {
	return &Registry{records: make([]domain.FailureRecord, 0)}
}

// Record appends a safe failure record without retaining source content.
func (registry *Registry) Record(record domain.FailureRecord) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	record.Diagnostics = append([]domain.Diagnostic(nil), record.Diagnostics...)
	registry.records = append(registry.records, record)
}

// List returns a stable copy of the current records ordered by path then time.
func (registry *Registry) List() []domain.FailureRecord {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	records := make([]domain.FailureRecord, len(registry.records))
	copy(records, registry.records)
	for index := range records {
		records[index].Diagnostics = append([]domain.Diagnostic(nil), records[index].Diagnostics...)
	}
	sort.SliceStable(records, func(left, right int) bool {
		if records[left].Path == records[right].Path {
			return records[left].RecordedAt.Before(records[right].RecordedAt)
		}
		return records[left].Path < records[right].Path
	})
	return records
}
