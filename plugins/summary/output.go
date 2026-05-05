package summary

import (
	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
)

func printSummary(collection *ci.PlanResultCollection) {
	var changes, noChanges, failed int
	for i := range collection.Results {
		switch collection.Results[i].Status {
		case ci.PlanStatusChanges:
			changes++
		case ci.PlanStatusNoChanges, ci.PlanStatusSuccess:
			noChanges++
		case ci.PlanStatusFailed:
			failed++
		case ci.PlanStatusPending, ci.PlanStatusRunning:
			// Not counted
		}
	}

	log.Info("summary")
	log.IncreasePadding()
	log.WithField("total", len(collection.Results)).Info("modules")
	if changes > 0 {
		log.WithField("count", changes).Info("with changes")
	}
	if noChanges > 0 {
		log.WithField("count", noChanges).Info("no changes")
	}
	if failed > 0 {
		log.WithField("count", failed).Warn("failed")
	}
	log.DecreasePadding()
}
