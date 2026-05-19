package ci

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// ReportFilename returns the canonical artifact name for a producer's report.
func ReportFilename(producer string) string {
	return producer + "-report.json"
}

// ResultFilename returns the canonical artifact name for a producer's raw result.
func ResultFilename(producer string) string {
	return producer + "-results.json"
}

// StatusFromCounts returns the strictest status implied by the given fail /
// warn counts: any failure → Fail; any warning → Warn; otherwise Pass.
// Producers can still override (e.g. tfupdate treats "updates available" as
// a warning); this helper covers the common case.
func StatusFromCounts(fail, warn int) ReportStatus {
	switch {
	case fail > 0:
		return ReportStatusFail
	case warn > 0:
		return ReportStatusWarn
	default:
		return ReportStatusPass
	}
}

// LoadReport reads and validates a single report file from disk. It is intended
// for low-level tests and store internals; producers should use ReportStore.
func LoadReport(path string) (*Report, error) {
	return loadReport(context.Background(), path)
}

func loadReport(ctx context.Context, path string) (*Report, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read report: %w", err)
	}

	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("decode report: %w", err)
	}
	if err := report.Validate(); err != nil {
		return nil, fmt.Errorf("validate report: %w", err)
	}

	return &report, nil
}
