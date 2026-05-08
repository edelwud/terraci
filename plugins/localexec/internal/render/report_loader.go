package render

import "github.com/edelwud/terraci/pkg/ci"

type SummaryReportLoader interface {
	Reset() error
	Load() (*ci.Report, error)
}

type noopSummaryReportLoader struct{}

func NewSummaryReportLoader(_, _ string, _ []string) SummaryReportLoader {
	return noopSummaryReportLoader{}
}

func (noopSummaryReportLoader) Reset() error {
	return nil
}

func (noopSummaryReportLoader) Load() (*ci.Report, error) {
	return nil, nil
}
