package discovery

import (
	"path/filepath"
	"testing"
)

func TestModule_ID(t *testing.T) {
	m := TestModule("platform", "stage", "eu-central-1", "vpc")
	if got := m.ID(); got != "platform/stage/eu-central-1/vpc" {
		t.Errorf("ID() = %q, want platform/stage/eu-central-1/vpc", got)
	}

	sub := TestModule("platform", "stage", "eu-central-1", "ec2")
	sub.SetComponent("submodule", "rabbitmq")
	sub.RelativePath = filepath.Join("platform", "stage", "eu-central-1", "ec2", "rabbitmq")
	if got := sub.ID(); got != "platform/stage/eu-central-1/ec2/rabbitmq" {
		t.Errorf("submodule ID() = %q", got)
	}
}

func TestModule_Name(t *testing.T) {
	base := NewModule(
		[]string{"service", "environment", "region", "module"},
		[]string{"platform", "stage", "eu-central-1", "vpc"},
		"", "platform/stage/eu-central-1/vpc",
	)
	if got := base.Name(); got != "vpc" {
		t.Errorf("Name() = %q, want vpc", got)
	}

	sub := NewModule(
		[]string{"service", "environment", "region", "module"},
		[]string{"platform", "stage", "eu-central-1", "ec2"},
		"", "platform/stage/eu-central-1/ec2/rabbitmq",
	)
	sub.SetComponent("submodule", "rabbitmq")
	if got := sub.Name(); got != "ec2/rabbitmq" {
		t.Errorf("submodule Name() = %q, want ec2/rabbitmq", got)
	}
}

func TestModule_IsSubmodule(t *testing.T) {
	base := TestModule("platform", "stage", "eu-central-1", "vpc")
	if base.IsSubmodule() {
		t.Error("base module should not be submodule")
	}

	sub := TestModule("platform", "stage", "eu-central-1", "ec2")
	sub.SetComponent("submodule", "rabbitmq")
	if !sub.IsSubmodule() {
		t.Error("should be submodule")
	}
}

func TestModule_Get(t *testing.T) {
	m := TestModule("platform", "stage", "eu-central-1", "vpc")

	tests := []struct{ key, want string }{
		{"service", "platform"},
		{"environment", "stage"},
		{"region", "eu-central-1"},
		{"module", "vpc"},
		{"nonexistent", ""},
	}

	for _, tt := range tests {
		if got := m.Get(tt.key); got != tt.want {
			t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestModule_LeafValue(t *testing.T) {
	m := TestModule("platform", "stage", "eu-central-1", "vpc")
	if got := m.LeafValue(); got != "vpc" {
		t.Errorf("LeafValue() = %q, want vpc", got)
	}

	empty := &Module{}
	if got := empty.LeafValue(); got != "" {
		t.Errorf("empty LeafValue() = %q, want empty", got)
	}
}

func TestModule_ContextPrefix(t *testing.T) {
	m := TestModule("platform", "stage", "eu-central-1", "vpc")
	if got := m.ContextPrefix(); got != "platform/stage/eu-central-1" {
		t.Errorf("ContextPrefix() = %q, want platform/stage/eu-central-1", got)
	}

	single := NewModule([]string{"module"}, []string{"vpc"}, "", "vpc")
	if got := single.ContextPrefix(); got != "" {
		t.Errorf("single segment ContextPrefix() = %q, want empty", got)
	}
}

func TestModule_Components(t *testing.T) {
	m := TestModule("platform", "stage", "eu-central-1", "vpc")
	cp := m.Components()

	if len(cp) != 4 {
		t.Errorf("Components len = %d, want 4", len(cp))
	}

	// Modifying copy shouldn't affect original
	cp["service"] = "changed"
	if m.Get("service") != "platform" {
		t.Error("Components() should return a copy")
	}
}

func TestModule_String(t *testing.T) {
	m := TestModule("platform", "stage", "eu-central-1", "vpc")
	if m.String() != m.ID() {
		t.Errorf("String() = %q, want %q", m.String(), m.ID())
	}
}
