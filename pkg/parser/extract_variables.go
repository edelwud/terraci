package parser

import "path/filepath"

func extractTfvars(ctx *extractContext) {
	extractVariableDefaults(ctx)

	if tfvars := filepath.Join(ctx.parsed.Path, "terraform.tfvars"); fileExists(tfvars) {
		loadTfvarsFile(ctx, tfvars)
	}

	autoFiles, _ := filepath.Glob(filepath.Join(ctx.parsed.Path, "*.auto.tfvars")) //nolint:errcheck
	for _, path := range autoFiles {
		loadTfvarsFile(ctx, path)
	}
}

func extractVariableDefaults(ctx *extractContext) {
	for _, block := range ctx.index.variableBlocks() {
		if len(block.Labels) < 1 {
			continue
		}

		schema := variableDefaultSchema()
		content, _, diags := block.Body.PartialContent(schema)
		ctx.addDiags(diags)
		if content == nil {
			continue
		}
		if attr, ok := content.Attributes["default"]; ok {
			if val, valDiags := attr.Expr.Value(nil); !valDiags.HasErrors() {
				ctx.parsed.Variables[block.Labels[0]] = val
			}
		}
	}
}

func loadTfvarsFile(ctx *extractContext, path string) {
	file, err := ctx.index.parseHCLFile(path)
	if err != nil {
		ctx.addDiags(tfvarsReadDiagnostic(path, err))
		return
	}
	if file == nil {
		return
	}

	attrs, diags := file.Body.JustAttributes()
	ctx.addDiags(diags)
	for name, attr := range attrs {
		if val, valDiags := attr.Expr.Value(nil); !valDiags.HasErrors() {
			ctx.parsed.Variables[name] = val
		}
	}
}
