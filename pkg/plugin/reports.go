package plugin

import (
	"sync"

	"github.com/edelwud/terraci/pkg/ci"
)

// ReportRegistry allows plugins to publish and consume reports in-memory.
// In CI (multi-process), reports are still written to JSON files for artifacts.
// The registry provides an in-process fast path for single-process runs.
type ReportRegistry struct {
	mu      sync.RWMutex
	reports map[string]*ci.Report
}

// NewReportRegistry creates a new empty ReportRegistry.
func NewReportRegistry() *ReportRegistry {
	return &ReportRegistry{
		reports: make(map[string]*ci.Report),
	}
}

// Publish stores a report in the registry, keyed by plugin name.
func (r *ReportRegistry) Publish(report *ci.Report) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reports[report.Plugin] = report
}

// Get retrieves a report by plugin name.
func (r *ReportRegistry) Get(pluginName string) (*ci.Report, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rep, ok := r.reports[pluginName]
	return rep, ok
}

// All returns all published reports.
func (r *ReportRegistry) All() []*ci.Report {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ci.Report, 0, len(r.reports))
	for _, rep := range r.reports {
		result = append(result, rep)
	}
	return result
}
