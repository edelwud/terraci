package summary

import (
	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
)

func printSummary(collection *ci.PlanResultCollection) {
	if collection == nil || collection.Len() == 0 {
		return
	}

	var changes, noChanges, failed int
	results := collection.Results()
	for i := range results {
		switch results[i].Status() {
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
	log.WithField("total", collection.Len()).Info("modules")
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
