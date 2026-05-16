package render

import (
	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/plugins/internal/reportrender"
)

func SummaryReportCLI(report *ci.Report) (string, error) {
	return reportrender.CLIReport(report)
}
