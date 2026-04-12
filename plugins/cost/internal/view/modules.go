package view

import (
	"strings"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

// SubmoduleCost groups resource costs by Terraform module address for presentation.
type SubmoduleCost struct {
	ModuleAddr  string               `json:"module_addr"`
	MonthlyCost float64              `json:"monthly_cost"`
	Resources   []model.ResourceCost `json:"resources,omitempty"`
	Children    []SubmoduleCost      `json:"children,omitempty"`
}

// TotalCost returns MonthlyCost including all nested children recursively.
func (s *SubmoduleCost) TotalCost() float64 {
	total := s.MonthlyCost
	for i := range s.Children {
		total += s.Children[i].TotalCost()
	}
	return total
}

// GroupByModule groups resources by their Terraform module address into a tree.
func GroupByModule(resources []model.ResourceCost) []SubmoduleCost {
	type flatGroup struct {
		addr      string
		resources []model.ResourceCost
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

	attached := make(map[string]struct{})
	for i := len(order) - 1; i >= 0; i-- {
		addr := order[i]
		parent := FindParentAddr(addr, nodes)
		if parent != "" {
			nodes[parent].Children = append(nodes[parent].Children, *nodes[addr])
			attached[addr] = struct{}{}
		}
	}

	var roots []SubmoduleCost
	for _, addr := range order {
		if _, ok := attached[addr]; !ok {
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
		rest := addr[i+1:]
		if strings.HasPrefix(rest, modulePrefix) {
			if _, ok := nodes[candidate]; ok {
				return candidate
			}
		}
	}
	return ""
}

// StripModulePrefix removes the module prefix from a resource address.
func StripModulePrefix(address, moduleAddr string) string {
	if moduleAddr == "" {
		return address
	}
	prefix := moduleAddr + "."
	if len(address) > len(prefix) && address[:len(prefix)] == prefix {
		return address[len(prefix):]
	}
	return address
}
