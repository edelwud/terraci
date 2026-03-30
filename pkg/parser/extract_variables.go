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
	for _, variable := range ctx.index.variableBlockViews() {
		val, ok, diags := variable.DefaultValue()
		ctx.addDiags(diags)
		if !ok {
			continue
		}
		ctx.parsed.Variables[variable.Name()] = val
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
