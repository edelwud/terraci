package eval

import (
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestNewContext(t *testing.T) {
	locals := map[string]cty.Value{
		"service": cty.StringVal("platform"),
		"env":     cty.StringVal("stage"),
	}
	variables := map[string]cty.Value{
		"region": cty.StringVal("eu-central-1"),
	}

	ctx := NewContext(locals, variables, "platform/stage/eu-central-1/vpc")

	if ctx == nil {
		t.Fatal("NewContext returned nil")
	}

	// Check local variable
	localVal := ctx.Variables["local"]
	if localVal.IsNull() {
		t.Error("local variable is null")
	}
	if localVal.GetAttr("service").AsString() != "platform" {
		t.Errorf("expected local.service to be 'platform', got %q", localVal.GetAttr("service").AsString())
	}

	// Check var variable
	varVal := ctx.Variables["var"]
	if varVal.IsNull() {
		t.Error("var variable is null")
	}
	if varVal.GetAttr("region").AsString() != "eu-central-1" {
		t.Errorf("expected var.region to be 'eu-central-1', got %q", varVal.GetAttr("region").AsString())
	}

	// Check path.module
	pathVal := ctx.Variables["path"]
	if pathVal.IsNull() {
		t.Error("path variable is null")
	}
	// path.module should be an absolute path
	moduleStr := pathVal.GetAttr("module").AsString()
	if !strings.HasSuffix(moduleStr, "platform/stage/eu-central-1/vpc") {
		t.Errorf("expected path.module to end with 'platform/stage/eu-central-1/vpc', got %q", moduleStr)
	}
	if !strings.HasPrefix(moduleStr, "/") {
		t.Errorf("expected path.module to be absolute, got %q", moduleStr)
	}

	// Check that functions are available
	if ctx.Functions == nil {
		t.Error("Functions map is nil")
	}
	if _, ok := ctx.Functions["lookup"]; !ok {
		t.Error("lookup function not found")
	}
}

func TestFunctions(t *testing.T) {
	funcs := Functions()

	if funcs == nil {
		t.Fatal("Functions returned nil")
	}

	if _, ok := funcs["lookup"]; !ok {
		t.Error("lookup function not found")
	}
}
