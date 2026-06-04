package summaryengine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
)

const defaultEnvironment = "default"

type planStats struct {
	Total     int
	Success   int
	NoChanges int
	Changes   int
	Failed    int
	Pending   int
	Running   int
}

func calculateStats(plans []ci.PlanResult) planStats {
	var stats planStats
	stats.Total = len(plans)
	for i := range plans {
		switch plans[i].Status() {
		case ci.PlanStatusSuccess, ci.PlanStatusNoChanges:
			stats.NoChanges++
		case ci.PlanStatusChanges:
			stats.Changes++
		case ci.PlanStatusFailed:
			stats.Failed++
		case ci.PlanStatusPending:
			stats.Pending++
		case ci.PlanStatusRunning:
			stats.Running++
		}
	}
	stats.Success = stats.NoChanges + stats.Changes
	return stats
}

func renderStats(stats planStats) string {
	var parts []string
	if stats.Changes > 0 {
		parts = append(parts, fmt.Sprintf("**%d** with changes", stats.Changes))
	}
	if stats.NoChanges > 0 {
		parts = append(parts, fmt.Sprintf("**%d** no changes", stats.NoChanges))
	}
	if stats.Failed > 0 {
		parts = append(parts, fmt.Sprintf("**%d** failed", stats.Failed))
	}
	if stats.Running > 0 {
		parts = append(parts, fmt.Sprintf("**%d** running", stats.Running))
	}
	if stats.Pending > 0 {
		parts = append(parts, fmt.Sprintf("**%d** pending", stats.Pending))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("**%d** modules analyzed", stats.Total)
	}
	return fmt.Sprintf("**%d** modules: %s", stats.Total, strings.Join(parts, " | "))
}

func groupByEnvironment(plans []ci.PlanResult) map[string][]ci.PlanResult {
	result := make(map[string][]ci.PlanResult)
	for i := range plans {
		env := plans[i].Component("environment")
		if env == "" {
			env = defaultEnvironment
		}
		result[env] = append(result[env], plans[i])
	}
	return result
}

func sortedKeys(m map[string][]ci.PlanResult) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func visibleEnvironmentPlans(plans []ci.PlanResult) []ci.PlanResult {
	visible := make([]ci.PlanResult, 0, len(plans))
	for i := range plans {
		if plans[i].Status() == ci.PlanStatusChanges || plans[i].Status() == ci.PlanStatusFailed {
			visible = append(visible, plans[i])
		}
	}
	return visible
}
