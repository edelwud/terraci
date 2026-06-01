package initwiz

import "testing"

func TestStateKeyValidation(t *testing.T) {
	t.Parallel()

	if _, err := NewStateKey[string](""); err == nil {
		t.Fatal("NewStateKey(empty) error = nil, want error")
	}
	if _, err := NewStateKey[string]("bad key"); err == nil {
		t.Fatal("NewStateKey(whitespace) error = nil, want error")
	}
	key, err := NewStateKey[string]("feature.enabled")
	if err != nil {
		t.Fatalf("NewStateKey() error = %v", err)
	}
	if key.Name() != "feature.enabled" {
		t.Fatalf("Name() = %q, want feature.enabled", key.Name())
	}
}

func TestStateKeyGetSetLookup(t *testing.T) {
	t.Parallel()

	state := NewStateMap()
	nameKey := MustStateKey[string]("name")
	flagKey := MustStateKey[bool]("flag")

	if got := nameKey.Get(state); got != "" {
		t.Fatalf("Get(unset string) = %q, want empty", got)
	}
	if _, ok := flagKey.Lookup(state); ok {
		t.Fatal("Lookup(unset bool) ok = true, want false")
	}

	nameKey.Set(state, "terraform")
	flagKey.Set(state, true)

	if got := nameKey.Get(state); got != "terraform" {
		t.Fatalf("Get(name) = %q, want terraform", got)
	}
	if got, ok := flagKey.Lookup(state); !ok || !got {
		t.Fatalf("Lookup(flag) = %v, %v; want true, true", got, ok)
	}
}

func TestStateKeyBindIsStable(t *testing.T) {
	t.Parallel()

	state := NewStateMap()
	key := MustStateKey[string]("name")
	key.Set(state, "initial")

	ptr := key.Bind(state)
	if *ptr != "initial" {
		t.Fatalf("Bind() = %q, want initial", *ptr)
	}
	*ptr = "from-ui"
	if got := key.Get(state); got != "from-ui" {
		t.Fatalf("Get() after pointer mutation = %q, want from-ui", got)
	}
	if ptr2 := key.Bind(state); ptr2 != ptr {
		t.Fatal("Bind() did not return stable pointer")
	}

	key.Set(state, "from-code")
	if *ptr != "from-code" {
		t.Fatalf("Set() did not update bound pointer: %q", *ptr)
	}
}

func TestStateKeyWrongTypeIsolation(t *testing.T) {
	t.Parallel()

	state := NewStateMap()
	stringKey := MustStateKey[string]("shared")
	boolKey := MustStateKey[bool]("shared")

	stringKey.Set(state, "yes")
	if _, ok := boolKey.Lookup(state); ok {
		t.Fatal("Lookup(bool over string slot) ok = true, want false")
	}

	boolPtr := boolKey.Bind(state)
	*boolPtr = true
	if got := boolKey.Get(state); !got {
		t.Fatal("Get(bool) = false, want true")
	}
	if _, ok := stringKey.Lookup(state); ok {
		t.Fatal("Lookup(string over bool slot) ok = true, want false")
	}
}

func TestInitFieldDefaultsAndDefensiveOptions(t *testing.T) {
	t.Parallel()

	state := NewStateMap()
	key := MustStateKey[string]("mode")
	field := NewSelectField(SelectFieldOptions{
		Key:     key,
		Title:   "Mode",
		Default: "all",
		Options: []InitOption{{Label: "All", Value: "all"}},
	})

	field.ApplyDefault(state)
	if got := key.Get(state); got != "all" {
		t.Fatalf("default = %q, want all", got)
	}

	options := field.Options()
	options[0].Value = "changed"
	if got := field.Options()[0].Value; got != "all" {
		t.Fatalf("Options leaked mutation: %q", got)
	}
	if field.Key() != "mode" || field.Type() != FieldSelect || field.Title() != "Mode" {
		t.Fatalf("field getters returned unexpected values")
	}
}

func TestInitFieldConstructorValidation(t *testing.T) {
	t.Parallel()

	assertPanics(t, func() {
		_ = NewStringField(StringFieldOptions{Title: "Missing key"})
	})
	assertPanics(t, func() {
		_ = NewBoolField(BoolFieldOptions{Key: MustStateKey[bool]("enabled")})
	})
	assertPanics(t, func() {
		_ = NewSelectField(SelectFieldOptions{
			Key:   MustStateKey[string]("mode"),
			Title: "Mode",
		})
	})
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}
