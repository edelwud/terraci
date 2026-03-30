package parser

import "path/filepath"

func (p *Parser) extractTfvars(index *moduleIndex, pm *ParsedModule) {
	p.extractVariableDefaults(index, pm)

	if tfvars := filepath.Join(pm.Path, "terraform.tfvars"); fileExists(tfvars) {
		p.loadTfvarsFile(index, pm, tfvars)
	}

	autoFiles, _ := filepath.Glob(filepath.Join(pm.Path, "*.auto.tfvars")) //nolint:errcheck
	for _, path := range autoFiles {
		p.loadTfvarsFile(index, pm, path)
	}
}

func (p *Parser) extractVariableDefaults(index *moduleIndex, pm *ParsedModule) {
	for _, block := range index.variableBlocks() {
		if len(block.Labels) < 1 {
			continue
		}

		schema := variableDefaultSchema()
		content, _, diags := block.Body.PartialContent(schema)
		pm.addDiags(diags)
		if content == nil {
			continue
		}
		if attr, ok := content.Attributes["default"]; ok {
			if val, valDiags := attr.Expr.Value(nil); !valDiags.HasErrors() {
				pm.Variables[block.Labels[0]] = val
			}
		}
	}
}

func (p *Parser) loadTfvarsFile(index *moduleIndex, pm *ParsedModule, path string) {
	file, err := index.parseHCLFile(path)
	if err != nil {
		pm.addDiags(tfvarsReadDiagnostic(path, err))
		return
	}
	if file == nil {
		return
	}

	attrs, diags := file.Body.JustAttributes()
	pm.addDiags(diags)
	for name, attr := range attrs {
		if val, valDiags := attr.Expr.Value(nil); !valDiags.HasErrors() {
			pm.Variables[name] = val
		}
	}
}
