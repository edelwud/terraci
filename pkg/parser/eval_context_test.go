package parser

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestEvalContextBuilder_ExtractPathLocals(t *testing.T) {
	builder := newEvalContextBuilder([]string{"service", "environment", "region", "module"})

	locals := builder.extractPathLocals([]string{"platform", "stage", "eu-central-1", "proxy", "prod"})

	if got := locals["service"].AsString(); got != "platform" {
		t.Fatalf("service = %q, want %q", got, "platform")
	}
	if got := locals["environment"].AsString(); got != "stage" {
		t.Fatalf("environment = %q, want %q", got, "stage")
	}
	if got := locals["region"].AsString(); got != "eu-central-1" {
		t.Fatalf("region = %q, want %q", got, "eu-central-1")
	}
	if got := locals["module"].AsString(); got != "prod" {
		t.Fatalf("module = %q, want %q", got, "prod")
	}
	if got := locals["submodule"].AsString(); got != "prod" {
		t.Fatalf("submodule = %q, want %q", got, "prod")
	}
	if got := locals["scope"].AsString(); got != "proxy" {
		t.Fatalf("scope = %q, want %q", got, "proxy")
	}
}

func TestEvalContextBuilder_Build(t *testing.T) {
	builder := newEvalContextBuilder([]string{"service", "environment", "region", "module"})
	evalCtx := builder.build(
		"platform/stage/eu-central-1/eks",
		map[string]cty.Value{"service": cty.StringVal("override")},
		map[string]cty.Value{"region": cty.StringVal("ignored")},
	)

	local := evalCtx.Variables["local"]
	if got := local.GetAttr("service").AsString(); got != "override" {
		t.Fatalf("local.service = %q, want %q", got, "override")
	}
	if got := local.GetAttr("environment").AsString(); got != "stage" {
		t.Fatalf("local.environment = %q, want %q", got, "stage")
	}
}
