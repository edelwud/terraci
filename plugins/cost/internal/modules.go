package costengine

// groupByModule groups resources by their Terraform module address into a tree.
// Child modules (e.g., module.eks.module.node_group) are nested under parents (module.eks).
// Parent MonthlyCost reflects only direct resources; use TotalCost() for recursive totals.
func groupByModule(resources []ResourceCost) []SubmoduleCost {
	// Step 1: group resources by exact ModuleAddr
	type flatGroup struct {
		addr      string
		resources []ResourceCost
		cost      float64
	}
	groups := make(map[string]*flatGroup)
	var order []string

	for i := range resources {
		rc := &resources[i]
		addr := rc.ModuleAddr
		if groups[addr] == nil {
			groups[addr] = &flatGroup{addr: addr}
			order = append(order, addr)
		}
		groups[addr].resources = append(groups[addr].resources, *rc)
		groups[addr].cost += rc.MonthlyCost
	}

	// Step 2: build tree nodes — each with only its direct resource cost
	nodes := make(map[string]*SubmoduleCost, len(order))
	for _, addr := range order {
		g := groups[addr]
		nodes[addr] = &SubmoduleCost{
			ModuleAddr:  addr,
			MonthlyCost: g.cost, // direct resources only
			Resources:   g.resources,
		}
	}

	// Step 3: attach children to parents, collect roots.
	// Do NOT add child cost to parent — TotalCost() computes recursive totals.
	var roots []SubmoduleCost
	attached := make(map[string]bool)

	for _, addr := range order {
		parent := findParentAddr(addr, nodes)
		if parent != "" {
			nodes[parent].Children = append(nodes[parent].Children, *nodes[addr])
			attached[addr] = true
		}
	}

	for _, addr := range order {
		if !attached[addr] {
			roots = append(roots, *nodes[addr])
		}
	}

	return roots
}

// findParentAddr finds the nearest existing parent module address.
// For "module.eks.module.node_group", checks "module.eks" in the nodes map.
func findParentAddr(addr string, nodes map[string]*SubmoduleCost) string {
	const modulePrefix = "module."

	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] != '.' {
			continue
		}
		candidate := addr[:i]
		// The rest after candidate + "." must start with "module." to be a valid boundary
		rest := addr[len(candidate)+1:]
		if len(rest) >= len(modulePrefix) && rest[:len(modulePrefix)] == modulePrefix {
			if _, ok := nodes[candidate]; ok {
				return candidate
			}
		}
	}
	return ""
}
