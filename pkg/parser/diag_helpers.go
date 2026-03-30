package parser

import "github.com/hashicorp/hcl/v2"

func (pm *ParsedModule) addDiags(diags hcl.Diagnostics) {
	pm.Diagnostics = append(pm.Diagnostics, diags...)
}
