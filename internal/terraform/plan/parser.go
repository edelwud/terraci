// Package plan provides terraform plan JSON parsing functionality
package plan

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
)

// ParsedPlan represents a parsed terraform plan with extracted changes
type ParsedPlan struct {
	TerraformVersion string
	FormatVersion    string
	ToAdd            int
	ToChange         int
	ToDestroy        int
	ToImport         int
	Resources        []ResourceChange
}

// ResourceChange represents a single resource change extracted from the plan
type ResourceChange struct {
	Address    string     // Full resource address (e.g., "module.vpc.aws_vpc.main")
	Type       string     // Resource type (e.g., "aws_vpc")
	Name       string     // Resource name (e.g., "main")
	ModuleAddr string     // Module address (e.g., "module.vpc")
	Action     string     // "create", "update", "delete", "replace", "read", "no-op"
	Attributes []AttrDiff // Changed attributes
}

// AttrDiff represents a single attribute change
type AttrDiff struct {
	Path      string // Attribute path (e.g., "instance_type", "tags.Name")
	OldValue  string // Old value (empty for new attributes)
	NewValue  string // New value (empty for removed attributes)
	Sensitive bool   // Whether the attribute is sensitive
	ForceNew  bool   // Whether this change forces resource replacement
	Computed  bool   // Whether the new value is computed (unknown)
}

// ParseJSON parses a terraform plan JSON file and extracts change information
func ParseJSON(path string) (*ParsedPlan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	return ParseJSONData(data)
}

// ParseJSONData parses terraform plan JSON data and extracts change information
func ParseJSONData(data []byte) (*ParsedPlan, error) {
	var plan tfjson.Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan JSON: %w", err)
	}

	// Validate plan format version
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

		// Determine action
		action := determineAction(rc.Change.Actions)
		if action == "no-op" {
			continue // Skip no-op changes
		}

		// Count changes
		switch action {
		case "create":
			parsed.ToAdd++
		case "update":
			parsed.ToChange++
		case "delete":
			parsed.ToDestroy++
		case "replace":
			parsed.ToAdd++
			parsed.ToDestroy++
		}

		// Check for import
		if rc.Change.Importing != nil {
			parsed.ToImport++
		}

		// Extract attribute diffs
		attrs := extractAttributeDiffs(rc.Change)

		parsed.Resources = append(parsed.Resources, ResourceChange{
			Address:    rc.Address,
			Type:       rc.Type,
			Name:       rc.Name,
			ModuleAddr: rc.ModuleAddress,
			Action:     action,
			Attributes: attrs,
		})
	}

	return parsed, nil
}

// HasChanges returns true if the plan has any changes
func (p *ParsedPlan) HasChanges() bool {
	return p.ToAdd > 0 || p.ToChange > 0 || p.ToDestroy > 0
}

// determineAction converts tfjson.Actions to a string action
func determineAction(actions tfjson.Actions) string {
	switch {
	case actions.Create():
		return "create"
	case actions.Update():
		return "update"
	case actions.Delete():
		return "delete"
	case actions.Replace():
		return "replace"
	case actions.Read():
		return "read"
	default:
		return "no-op"
	}
}

// extractAttributeDiffs extracts attribute differences from a Change
func extractAttributeDiffs(change *tfjson.Change) []AttrDiff {
	if change == nil {
		return nil
	}

	// Convert before/after to maps
	beforeMap := toStringMap(change.Before)
	afterMap := toStringMap(change.After)
	afterUnknownMap := toStringMap(change.AfterUnknown)
	beforeSensitiveMap := toStringMap(change.BeforeSensitive)
	afterSensitiveMap := toStringMap(change.AfterSensitive)

	// Get replace paths for force_new detection
	replacePaths := make(map[string]bool)
	for _, rp := range change.ReplacePaths {
		if paths, ok := rp.([]interface{}); ok {
			pathStr := pathToString(paths)
			replacePaths[pathStr] = true
		}
	}

	// Collect all keys
	allKeys := make(map[string]bool)
	collectKeys(beforeMap, "", allKeys)
	collectKeys(afterMap, "", allKeys)

	// Sort keys for consistent output
	sortedKeys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		sortedKeys = append(sortedKeys, k)
	}

	// Pre-allocate diffs slice
	diffs := make([]AttrDiff, 0, len(sortedKeys))
	sort.Strings(sortedKeys)

	// Compare values
	for _, key := range sortedKeys {
		oldVal := getNestedValue(beforeMap, key)
		newVal := getNestedValue(afterMap, key)
		isUnknown := getBoolValue(afterUnknownMap, key)
		isBeforeSensitive := getBoolValue(beforeSensitiveMap, key)
		isAfterSensitive := getBoolValue(afterSensitiveMap, key)

		// Skip if no change
		if reflect.DeepEqual(oldVal, newVal) && !isUnknown {
			continue
		}

		diff := AttrDiff{
			Path:      key,
			OldValue:  formatValue(oldVal),
			NewValue:  formatValue(newVal),
			Sensitive: isBeforeSensitive || isAfterSensitive,
			ForceNew:  replacePaths[key],
			Computed:  isUnknown,
		}

		// Handle sensitive values
		if isBeforeSensitive {
			diff.OldValue = "(sensitive)"
		}
		if isAfterSensitive || (isUnknown && isAfterSensitive) {
			diff.NewValue = "(sensitive)"
		}
		if isUnknown && !isAfterSensitive {
			diff.NewValue = "(known after apply)"
		}

		diffs = append(diffs, diff)
	}

	return diffs
}

// toStringMap converts an interface{} to a map[string]interface{}
func toStringMap(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// collectKeys recursively collects all keys from a nested map
func collectKeys(m map[string]interface{}, prefix string, keys map[string]bool) {
	for k, v := range m {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}

		keys[fullKey] = true

		// Recurse into nested maps
		if nested, ok := v.(map[string]interface{}); ok {
			collectKeys(nested, fullKey, keys)
		}
	}
}

// getNestedValue gets a value from a nested map using dot notation
func getNestedValue(m map[string]interface{}, key string) interface{} {
	if m == nil {
		return nil
	}

	parts := strings.Split(key, ".")
	current := interface{}(m)

	for _, part := range parts {
		if currentMap, ok := current.(map[string]interface{}); ok {
			current = currentMap[part]
		} else {
			return nil
		}
	}

	return current
}

// getBoolValue gets a boolean value from a nested map
func getBoolValue(m map[string]interface{}, key string) bool {
	v := getNestedValue(m, key)
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// formatValue formats a value for display
func formatValue(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case float64:
		// Check if it's an integer
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case []interface{}:
		if len(val) == 0 {
			return "[]"
		}
		return fmt.Sprintf("[%d items]", len(val))
	case map[string]interface{}:
		if len(val) == 0 {
			return "{}"
		}
		return fmt.Sprintf("{%d keys}", len(val))
	default:
		return fmt.Sprintf("%v", v)
	}
}

// pathToString converts a path array to a dot-separated string
func pathToString(paths []interface{}) string {
	parts := make([]string, 0, len(paths))
	for _, p := range paths {
		if s, ok := p.(string); ok {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, ".")
}
