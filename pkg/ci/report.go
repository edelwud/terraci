package ci

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SaveReport writes a plugin report as {serviceDir}/{plugin}-report.json.
func SaveReport(serviceDir string, report *Report) error {
	return SaveJSON(serviceDir, report.Plugin+"-report.json", report)
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
