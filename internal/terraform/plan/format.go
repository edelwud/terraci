package plan

import (
	"fmt"
	"strings"
)

// Constants for plan summary formatting.
const (
	maxResourcesInSummary = 10
	maxAddressLength      = 80
)

// Summary returns a compact summary line (e.g., "+2 ~1 -1").
func (p *ParsedPlan) Summary() string {
	if !p.HasChanges() {
		return "No changes"
	}

	var counts []string
	if p.ToAdd > 0 {
		counts = append(counts, fmt.Sprintf("+%d", p.ToAdd))
	}
	if p.ToChange > 0 {
		counts = append(counts, fmt.Sprintf("~%d", p.ToChange))
	}
	if p.ToDestroy > 0 {
		counts = append(counts, fmt.Sprintf("-%d", p.ToDestroy))
	}
	if p.ToImport > 0 {
		counts = append(counts, fmt.Sprintf("↓%d", p.ToImport))
	}
	return strings.Join(counts, " ")
}

// Details returns a markdown resource list grouped by action.
func (p *ParsedPlan) Details() string {
	if !p.HasChanges() {
		return ""
	}

	groups := make(map[string][]string)
	for _, r := range p.Resources {
		addr := formatResourceAddress(r)
		groups[r.Action] = append(groups[r.Action], addr)
	}

	var sb strings.Builder

	actionOrder := []struct {
		action string
		label  string
	}{
		{ActionCreate, "Create"},
		{ActionUpdate, "Update"},
		{ActionReplace, "Replace"},
		{ActionDelete, "Delete"},
		{ActionRead, "Read"},
	}

	for _, a := range actionOrder {
		resources := groups[a.action]
		if len(resources) == 0 {
			continue
		}

		fmt.Fprintf(&sb, "**%s:**\n", a.label)
		shown := 0
		for _, addr := range resources {
			if shown >= maxResourcesInSummary {
				fmt.Fprintf(&sb, "- ... +%d more\n", len(resources)-shown)
				break
			}
			fmt.Fprintf(&sb, "- `%s`\n", addr)
			shown++
		}
		sb.WriteString("\n")
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// ExitCode returns 0 for no changes, 2 for changes.
func (p *ParsedPlan) ExitCode() int {
	if !p.HasChanges() {
		return 0
	}
	return 2
}

// FilterPlanOutput extracts only the diff portion from terraform plan output.
func FilterPlanOutput(rawOutput string) string {
	lines := strings.Split(rawOutput, "\n")
	var result []string
	inDiff := false
	skipUntilEmpty := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "Refreshing state...") ||
			strings.HasPrefix(trimmed, "Reading...") ||
			strings.HasPrefix(trimmed, "data.") && strings.Contains(trimmed, "Reading...") {
			skipUntilEmpty = true
			continue
		}

		if skipUntilEmpty {
			if trimmed == "" {
				skipUntilEmpty = false
			}
			continue
		}

		if strings.HasPrefix(trimmed, "# ") ||
			strings.HasPrefix(trimmed, "Plan:") ||
			strings.HasPrefix(trimmed, "Changes to Outputs:") ||
			strings.HasPrefix(trimmed, "Terraform will perform") ||
			strings.HasPrefix(trimmed, "No changes.") {
			inDiff = true
		}

		if strings.HasPrefix(line, "  + ") ||
			strings.HasPrefix(line, "  - ") ||
			strings.HasPrefix(line, "  ~ ") ||
			strings.HasPrefix(line, "  # ") {
			inDiff = true
		}

		if inDiff {
			result = append(result, line)
		}
	}

	if len(result) == 0 {
		return rawOutput
	}

	return strings.Join(result, "\n")
}

func formatResourceAddress(r ResourceChange) string {
	addr := r.Address
	if r.ModuleAddr == "" && r.Type != "" && r.Name != "" {
		addr = r.Type + "." + r.Name
	}
	if len(addr) > maxAddressLength {
		addr = addr[:maxAddressLength-3] + "..."
	}
	return addr
}
