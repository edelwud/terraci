package extract

import (
	"path/filepath"
)

func extractTfvars(ctx *Context) {
	extractVariableDefaults(ctx)

	if tfvars := filepath.Join(ctx.Sink.Path(), "terraform.tfvars"); fileExists(tfvars) {
		loadTfvarsFile(ctx, tfvars)
	}

	autoFiles, _ := filepath.Glob(filepath.Join(ctx.Sink.Path(), "*.auto.tfvars")) //nolint:errcheck
	for _, path := range autoFiles {
		loadTfvarsFile(ctx, path)
	}
}

func extractVariableDefaults(ctx *Context) {
	for _, variable := range ctx.Source.VariableBlockViews() {
		val, ok, diags := variable.DefaultValue()
		ctx.Sink.AddDiags(diags)
		if !ok {
			continue
		}
		ctx.Sink.SetVariable(variable.Name(), val)
	}
}

func loadTfvarsFile(ctx *Context, path string) {
	file, err := ctx.Source.ParseHCLFile(path)
	if err != nil {
		ctx.Sink.AddDiags(tfvarsReadDiagnostic(path, err))
		return
	}
	if file == nil {
		return
	}

	attrs, diags := file.Body.JustAttributes()
	ctx.Sink.AddDiags(diags)
	for name, attr := range attrs {
		if val, valDiags := attr.Expr.Value(nil); !valDiags.HasErrors() {
			ctx.Sink.SetVariable(name, val)
		}
	}
}
