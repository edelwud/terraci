package summaryengine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/edelwud/terraci/pkg/ci"
)

// LoadReports reads all *-report.json files from the service directory.
func LoadReports(serviceDir string) []*ci.Report {
	pattern := filepath.Join(serviceDir, "*-report.json")
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		return nil
	}

	sort.Strings(files)
	reports := make([]*ci.Report, 0, len(files))
	for _, f := range files {
		data, readErr := os.ReadFile(f)
		if readErr != nil {
			continue
		}
		var r ci.Report
		if jsonErr := json.Unmarshal(data, &r); jsonErr != nil {
			continue
		}
		reports = append(reports, &r)
	}
	return reports
}

// EnrichPlans applies per-module data from report modules to plan data.
func EnrichPlans(plans []ci.ModulePlan, modules []ci.ModuleReport) {
	if len(modules) == 0 {
		return
	}
	byPath := make(map[string]*ci.ModuleReport, len(modules))
	for i := range modules {
		byPath[modules[i].ModulePath] = &modules[i]
	}
	for i := range plans {
		if mr, ok := byPath[plans[i].ModulePath]; ok {
			if mr.HasCost {
				plans[i].CostBefore = mr.CostBefore
				plans[i].CostAfter = mr.CostAfter
				plans[i].CostDiff = mr.CostDiff
				plans[i].HasCost = true
			}
		}
	}
}
