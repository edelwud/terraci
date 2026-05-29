package diagnosticlog

import (
	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/diagnostic"
)

// Log writes diagnostics through the shared TerraCi logger.
func Log(diags diagnostic.List) {
	for _, diag := range diags.All() {
		entry := log.WithField("severity", diag.Severity())
		if diag.Source() != "" {
			entry = entry.WithField("source", diag.Source())
		}
		if diag.Module() != "" {
			entry = entry.WithField("module", diag.Module())
		}
		if diag.Cause() != nil {
			entry = entry.WithError(diag.Cause())
		}
		switch diag.Severity() {
		case diagnostic.SeverityError:
			entry.Error(diag.Message())
		case diagnostic.SeverityWarning:
			entry.Warn(diag.Message())
		case diagnostic.SeverityInfo:
			entry.Info(diag.Message())
		default:
			entry.Info(diag.Message())
		}
	}
}
