package ci

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ReportFilename returns the canonical artifact name for a plugin report.
func ReportFilename(plugin string) string {
	return plugin + "-report.json"
}

// SaveReport writes a plugin report as {serviceDir}/{plugin}-report.json.
func SaveReport(serviceDir string, report *Report) error {
	return SaveJSON(serviceDir, ReportFilename(report.Plugin), report)
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
			continue
		}
		reports = append(reports, report)
	}

	return reports, nil
}
