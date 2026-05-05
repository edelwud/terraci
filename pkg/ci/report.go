package ci

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// AggregateReportProducer is the canonical producer name for the aggregated
// Terraform plan summary report consumed by local renderers.
const AggregateReportProducer = "summary"

// ReportFilename returns the canonical artifact name for a producer's report.
func ReportFilename(producer string) string {
	return producer + "-report.json"
}

// AggregateReportFilename returns the canonical artifact name for the
// aggregated Terraform plan summary report.
func AggregateReportFilename() string {
	return ReportFilename(AggregateReportProducer)
}

// BuildReport assembles a producer Report with the standard provenance shell.
// Producers (cost / policy / tfupdate) call this instead of constructing
// the &Report{...} literal by hand — drives uniform field ordering and a
// single point to update when the Report shape grows.
func BuildReport(producer, title string, status ReportStatus, summary string, sections ...ReportSection) *Report {
	return &Report{
		Producer:   producer,
		Title:      title,
		Status:     status,
		Summary:    summary,
		Provenance: NewProvenance("", "", ""),
		Sections:   sections,
	}
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

// SaveReport writes a report as {serviceDir}/{producer}-report.json.
func SaveReport(serviceDir string, report *Report) error {
	if err := report.Validate(); err != nil {
		return fmt.Errorf("validate report: %w", err)
	}
	return SaveJSON(serviceDir, ReportFilename(report.Producer), report)
}

// SaveResultsAndReport persists a producer's raw result payload alongside its
// canonical Report. Both writes are attempted independently — failures are
// joined into a single error so callers always know which side broke.
//
// Replaces the recurring 6-line "save results, build report, save report"
// pattern in cost / policy / tfupdate. Producers must build their report up
// front (since report construction returns its own error) and pass it in.
func SaveResultsAndReport(serviceDir, resultsFilename string, results any, report *Report) error {
	if serviceDir == "" {
		return nil
	}

	var errs []error
	if resultsFilename != "" {
		if err := SaveJSON(serviceDir, resultsFilename, results); err != nil {
			errs = append(errs, fmt.Errorf("save results: %w", err))
		}
	}
	if report != nil {
		if err := SaveReport(serviceDir, report); err != nil {
			errs = append(errs, fmt.Errorf("save report: %w", err))
		}
	}
	return errors.Join(errs...)
}

// SaveJSON writes any value as indented JSON to {serviceDir}/{filename}.
func SaveJSON(serviceDir, filename string, v any) error {
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	path := filepath.Join(serviceDir, filename)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// LoadReport reads a single report file from disk.
func LoadReport(path string) (*Report, error) {
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

// LoadReports reads all *-report.json files from the service directory in
// deterministic filename order.
func LoadReports(serviceDir string) ([]*Report, error) {
	pattern := filepath.Join(serviceDir, "*-report.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob reports: %w", err)
	}
	if len(files) == 0 {
		return nil, nil
	}

	sort.Strings(files)
	reports := make([]*Report, 0, len(files))
	for _, file := range files {
		report, err := LoadReport(file)
		if err != nil {
			return nil, fmt.Errorf("load report %s: %w", filepath.Base(file), err)
		}
		reports = append(reports, report)
	}

	return reports, nil
}
