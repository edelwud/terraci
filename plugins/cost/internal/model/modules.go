package model

// GroupByModule groups resources by their Terraform module address into a tree.
func GroupByModule(resources []ResourceCost) []SubmoduleCost {
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

	nodes := make(map[string]*SubmoduleCost, len(order))
	for _, addr := range order {
		g := groups[addr]
		nodes[addr] = &SubmoduleCost{
			ModuleAddr:  addr,
			MonthlyCost: g.cost,
			Resources:   g.resources,
		}
	}

	attached := make(map[string]bool)
	for i := len(order) - 1; i >= 0; i-- {
		addr := order[i]
		parent := FindParentAddr(addr, nodes)
		if parent != "" {
			nodes[parent].Children = append(nodes[parent].Children, *nodes[addr])
			attached[addr] = true
		}
	}

	var roots []SubmoduleCost
	for _, addr := range order {
		if !attached[addr] {
			roots = append(roots, *nodes[addr])
		}
	}

	return roots
}

// FindParentAddr finds the nearest existing parent module address.
func FindParentAddr(addr string, nodes map[string]*SubmoduleCost) string {
	const modulePrefix = "module."

	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] != '.' {
			continue
		}
		candidate := addr[:i]
		rest := addr[len(candidate)+1:]
		if len(rest) >= len(modulePrefix) && rest[:len(modulePrefix)] == modulePrefix {
			if _, ok := nodes[candidate]; ok {
				return candidate
			}
		}
	}
	return ""
}
