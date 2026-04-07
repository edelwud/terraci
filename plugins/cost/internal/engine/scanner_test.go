package engine_test

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edelwud/terraci/plugins/cost/internal/engine"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

func TestModuleScanner_ScanMany_PreservesOrderAndReturnsFirstInputError(t *testing.T) {
	t.Parallel()

	adapter := &countingAdapter{
		results: map[string]adapterResult{
			"mod-a": {plan: &engine.ModulePlan{ModulePath: "mod-a", Region: "us-east-1"}},
			"mod-b": {err: fmt.Errorf("mod-b failed")},
			"mod-c": {err: fmt.Errorf("mod-c failed")},
		},
	}

	scanner := engine.NewModuleScanner(adapter)
	_, err := scanner.ScanMany([]string{"mod-a", "mod-b", "mod-c"}, map[string]string{
		"mod-a": "us-east-1",
		"mod-b": "us-east-1",
		"mod-c": "us-east-1",
	})
	if err == nil || err.Error() != "mod-b failed" {
		t.Fatalf("ScanMany() error = %v, want first input error from mod-b", err)
	}
}

func TestModuleScanner_ScanManyBestEffort_PreservesIndicesAndDefaultRegion(t *testing.T) {
	t.Parallel()

	adapter := &countingAdapter{
		results: map[string]adapterResult{
			"mod-a": {plan: &engine.ModulePlan{ModulePath: "mod-a"}},
			"mod-b": {err: fmt.Errorf("boom")},
		},
	}

	scanner := engine.NewModuleScanner(adapter)
	plans := scanner.ScanManyBestEffort([]string{"mod-a", "mod-b"}, map[string]string{})
	if len(plans) != 2 {
		t.Fatalf("len(plans) = %d, want 2", len(plans))
	}
	if plans[0].Index != 0 || plans[0].ModulePath != "mod-a" || plans[0].Region != model.DefaultRegion {
		t.Fatalf("plans[0] = %+v, want preserved index/path/default region", plans[0])
	}
	if plans[1].Index != 1 || plans[1].Err == nil || plans[1].Region != model.DefaultRegion {
		t.Fatalf("plans[1] = %+v, want preserved index and error", plans[1])
	}
}

func TestModuleScanner_ScanMany_RunsConcurrently(t *testing.T) {
	t.Parallel()

	adapter := &countingAdapter{
		results: map[string]adapterResult{
			"mod-a": {plan: &engine.ModulePlan{ModulePath: "mod-a"}},
			"mod-b": {plan: &engine.ModulePlan{ModulePath: "mod-b"}},
			"mod-c": {plan: &engine.ModulePlan{ModulePath: "mod-c"}},
			"mod-d": {plan: &engine.ModulePlan{ModulePath: "mod-d"}},
		},
		delay: 20 * time.Millisecond,
	}

	scanner := engine.NewModuleScanner(adapter)
	if _, err := scanner.ScanMany([]string{"mod-a", "mod-b", "mod-c", "mod-d"}, map[string]string{}); err != nil {
		t.Fatalf("ScanMany() error = %v", err)
	}

	if got := adapter.maxConcurrent.Load(); got < 2 {
		t.Fatalf("maxConcurrent = %d, want >= 2 to confirm bounded parallel scanning", got)
	}
}

func TestModuleScanner_ScanMany_RespectsConcurrencyLimit(t *testing.T) {
	t.Parallel()

	const moduleCount = 12
	results := make(map[string]adapterResult, moduleCount)
	paths := make([]string, 0, moduleCount)
	for i := range moduleCount {
		path := fmt.Sprintf("mod-%02d", i)
		paths = append(paths, path)
		results[path] = adapterResult{plan: &engine.ModulePlan{ModulePath: path}}
	}

	adapter := &countingAdapter{
		results: results,
		delay:   20 * time.Millisecond,
	}
	scanner := engine.NewModuleScanner(adapter)
	if _, err := scanner.ScanMany(paths, map[string]string{}); err != nil {
		t.Fatalf("ScanMany() error = %v", err)
	}

	if got := adapter.maxConcurrent.Load(); got > 4 {
		t.Fatalf("maxConcurrent = %d, want <= 4", got)
	}
}

type adapterResult struct {
	plan *engine.ModulePlan
	err  error
}

type countingAdapter struct {
	results map[string]adapterResult
	delay   time.Duration

	active        atomic.Int32
	maxConcurrent atomic.Int32
}

func (a *countingAdapter) LoadModule(modulePath, region string) (*engine.ModulePlan, error) {
	current := a.active.Add(1)
	defer a.active.Add(-1)

	for {
		currentMax := a.maxConcurrent.Load()
		if current <= currentMax || a.maxConcurrent.CompareAndSwap(currentMax, current) {
			break
		}
	}

	if a.delay > 0 {
		time.Sleep(a.delay)
	}

	result, ok := a.results[modulePath]
	if !ok {
		return nil, fmt.Errorf("unexpected module %s", modulePath)
	}
	if result.plan != nil {
		planCopy := *result.plan
		planCopy.Region = region
		return &planCopy, result.err
	}
	return nil, result.err
}
