package render

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/edelwud/terraci/pkg/ci"
)

type SummaryReportLoader interface {
	Load() (*ci.Report, error)
}

type summaryReportLoader struct {
	serviceDir string
}

func NewSummaryReportLoader(serviceDir string) SummaryReportLoader {
	return summaryReportLoader{serviceDir: serviceDir}
}

func (l summaryReportLoader) Load() (*ci.Report, error) {
	if l.serviceDir == "" {
		return nil, nil
	}

	report, err := ci.LoadReport(filepath.Join(l.serviceDir, ci.ReportFilename("summary")))
	if err == nil {
		return report, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return nil, fmt.Errorf("load summary report: %w", err)
}
