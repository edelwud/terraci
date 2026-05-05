package render

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/planresults"
)

// summaryProducerName is the canonical Producer field of summary plugin
// reports. Centralized so file paths and the validate guard use the same
// literal.
const summaryProducerName = "summary"

type SummaryReportLoader interface {
	Reset() error
	Load() (*ci.Report, error)
}

type summaryReportLoader struct {
	serviceDir string
	workDir    string
	segments   []string
}

func NewSummaryReportLoader(serviceDir, workDir string, segments []string) SummaryReportLoader {
	return summaryReportLoader{serviceDir: serviceDir, workDir: workDir, segments: append([]string(nil), segments...)}
}

func (l summaryReportLoader) Reset() error {
	if l.serviceDir == "" {
		return nil
	}

	err := os.Remove(filepath.Join(l.serviceDir, ci.ReportFilename(summaryProducerName)))
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("remove stale summary report: %w", err)
}

func (l summaryReportLoader) Load() (*ci.Report, error) {
	if l.serviceDir == "" {
		return nil, nil
	}

	report, err := ci.LoadReport(filepath.Join(l.serviceDir, ci.ReportFilename(summaryProducerName)))
	if err == nil {
		if validateErr := l.validate(report); validateErr != nil {
			return nil, validateErr
		}
		return report, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return nil, fmt.Errorf("load summary report: %w", err)
}

func (l summaryReportLoader) validate(report *ci.Report) error {
	if report == nil {
		return nil
	}
	if report.Producer != "" && report.Producer != summaryProducerName {
		return fmt.Errorf("summary report producer mismatch: %q", report.Producer)
	}
	if report.Provenance == nil {
		return nil
	}
	if report.Provenance.PlanResultsFingerprint == "" || l.workDir == "" {
		return nil
	}

	collection, err := planresults.Scan(l.workDir, l.segments)
	if err != nil {
		return fmt.Errorf("validate summary report provenance: %w", err)
	}
	current := collection.Fingerprint()
	if current == "" || current != report.Provenance.PlanResultsFingerprint {
		return errors.New("summary report provenance mismatch")
	}

	return nil
}
