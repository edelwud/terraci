package plan

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"

	tfjson "github.com/hashicorp/terraform-json"
)

// ParseJSON parses a terraform plan JSON file.
func ParseJSON(path string) (*ParsedPlan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read plan file: %w", err)
	}
	return ParseJSONData(data)
}

// ParseJSONData parses terraform plan JSON data.
func ParseJSONData(data []byte) (*ParsedPlan, error) {
	var plan tfjson.Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("parse plan JSON: %w", err)
	}
	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("invalid plan format: %w", err)
	}

	parsed := &ParsedPlan{
		TerraformVersion: plan.TerraformVersion,
		FormatVersion:    plan.FormatVersion,
		Resources:        make([]ResourceChange, 0, len(plan.ResourceChanges)),
	}

	for _, rc := range plan.ResourceChanges {
		if rc == nil || rc.Change == nil {
			continue
		}

		action := determineAction(rc.Change.Actions)

		if action != ActionNoOp {
			countAction(parsed, action, rc.Change)
		}

		parsed.Resources = append(parsed.Resources, ResourceChange{
			Address:      rc.Address,
			Type:         rc.Type,
			Name:         rc.Name,
			ModuleAddr:   rc.ModuleAddress,
			Action:       action,
			Attributes:   extractAttributeDiffs(rc.Change),
			BeforeValues: toMap(rc.Change.Before),
			AfterValues:  toMap(rc.Change.After),
		})
	}

	return parsed, nil
}

const sensitiveValue = "(sensitive)"

func countAction(p *ParsedPlan, action string, change *tfjson.Change) {
	switch action {
	case ActionCreate:
		p.ToAdd++
	case ActionUpdate:
		p.ToChange++
	case ActionDelete:
		p.ToDestroy++
	case ActionReplace:
		p.ToAdd++
		p.ToDestroy++
	}
	if change.Importing != nil {
		p.ToImport++
	}
}

func determineAction(actions tfjson.Actions) string {
	switch {
	case actions.Create():
		return ActionCreate
	case actions.Update():
		return ActionUpdate
	case actions.Delete():
		return ActionDelete
	case actions.Replace():
		return ActionReplace
	case actions.Read():
		return ActionRead
	default:
		return ActionNoOp
	}
}

// --- Attribute diff extraction ---

func extractAttributeDiffs(change *tfjson.Change) []AttrDiff {
	if change == nil {
		return nil
	}

	before := toMap(change.Before)
	after := toMap(change.After)
	afterUnknown := toMap(change.AfterUnknown)
	beforeSensitive := toMap(change.BeforeSensitive)
	afterSensitive := toMap(change.AfterSensitive)
	replacePaths := collectReplacePaths(change.ReplacePaths)

	allKeys := collectAllKeys(before, after)

	diffs := make([]AttrDiff, 0, len(allKeys))
	for _, key := range allKeys {
		oldVal := getNestedValue(before, key)
		newVal := getNestedValue(after, key)
		isUnknown := getBool(afterUnknown, key)

		if reflect.DeepEqual(oldVal, newVal) && !isUnknown {
			continue
		}

		diffs = append(diffs, buildAttrDiff(key, oldVal, newVal, attrFlags{
			unknown:         isUnknown,
			beforeSensitive: getBool(beforeSensitive, key),
			afterSensitive:  getBool(afterSensitive, key),
			forceNew:        replacePaths[key],
		}))
	}

	return diffs
}

type attrFlags struct {
	unknown, beforeSensitive, afterSensitive, forceNew bool
}

func buildAttrDiff(key string, oldVal, newVal any, f attrFlags) AttrDiff {
	diff := AttrDiff{
		Path:      key,
		OldValue:  formatValue(oldVal),
		NewValue:  formatValue(newVal),
		Sensitive: f.beforeSensitive || f.afterSensitive,
		ForceNew:  f.forceNew,
		Computed:  f.unknown,
	}

	if f.beforeSensitive {
		diff.OldValue = sensitiveValue
	}
	switch {
	case f.afterSensitive:
		diff.NewValue = sensitiveValue
	case f.unknown:
		diff.NewValue = "(known after apply)"
	}

	return diff
}

func collectReplacePaths(raw []any) map[string]bool {
	result := make(map[string]bool, len(raw))
	for _, rp := range raw {
		if paths, ok := rp.([]any); ok {
			result[pathToString(paths)] = true
		}
	}
	return result
}

func collectAllKeys(maps ...map[string]any) []string {
	keys := make(map[string]bool)
	for _, m := range maps {
		collectKeys(m, "", keys)
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	return sorted
}
