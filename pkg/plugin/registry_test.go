package plugin

import (
	"testing"
)

type testPlugin struct {
	name string
	desc string
}

func (p *testPlugin) Name() string        { return p.name }
func (p *testPlugin) Description() string { return p.desc }

type testCommandPlugin struct {
	testPlugin
}

func TestRegisterAndGet(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	p := &testPlugin{name: "test", desc: "A test plugin"}
	Register(p)

	got, ok := Get("test")
	if !ok {
		t.Fatal("expected to find plugin")
	}
	if got.Name() != "test" {
		t.Fatalf("got name %q, want %q", got.Name(), "test")
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	Register(&testPlugin{name: "dup"})

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	Register(&testPlugin{name: "dup"})
}

func TestAll_Order(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	Register(&testPlugin{name: "b"})
	Register(&testPlugin{name: "a"})
	Register(&testPlugin{name: "c"})

	all := All()
	if len(all) != 3 {
		t.Fatalf("got %d plugins, want 3", len(all))
	}
	if all[0].Name() != "b" || all[1].Name() != "a" || all[2].Name() != "c" {
		t.Fatalf("wrong order: %s, %s, %s", all[0].Name(), all[1].Name(), all[2].Name())
	}
}

func TestByCapability(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	Register(&testPlugin{name: "plain"})
	Register(&testCommandPlugin{testPlugin: testPlugin{name: "cmd"}})

	// All plugins
	all := All()
	if len(all) != 2 {
		t.Fatalf("got %d plugins, want 2", len(all))
	}

	// Only command plugins — testCommandPlugin doesn't actually implement CommandProvider,
	// but we can test that ByCapability filters correctly with our test interface
	type hasName interface {
		Plugin
		Name() string
	}
	named := ByCapability[hasName]()
	if len(named) != 2 {
		t.Fatalf("got %d named plugins, want 2", len(named))
	}
}

func TestGetNotFound(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	_, ok := Get("nonexistent")
	if ok {
		t.Fatal("expected not found")
	}
}

func TestResolveProvider_NoPlugins(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	_, err := ResolveProvider()
	if err == nil {
		t.Fatal("expected error with no providers")
	}
}

func TestResolveChangeDetector_None(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	_, err := ResolveChangeDetector()
	if err == nil {
		t.Fatal("expected error with no detectors")
	}
}

func TestCollectContributions_Empty(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	contribs := CollectContributions()
	if len(contribs) != 0 {
		t.Errorf("expected 0 contributions, got %d", len(contribs))
	}
}

func TestAll_Empty(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	all := All()
	if len(all) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(all))
	}
}

func TestByCapability_NoMatch(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	Register(&testPlugin{name: "basic"})

	// VersionProvider is not implemented by testPlugin
	vp := ByCapability[VersionProvider]()
	if len(vp) != 0 {
		t.Errorf("expected 0 VersionProviders, got %d", len(vp))
	}
}

func TestReset(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	Register(&testPlugin{name: "x"})
	if len(All()) != 1 {
		t.Fatal("expected 1 plugin after register")
	}

	Reset()
	if len(All()) != 0 {
		t.Error("expected 0 plugins after reset")
	}
}
