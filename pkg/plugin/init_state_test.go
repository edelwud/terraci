package plugin

import "testing"

func TestStateMap_SetGet(t *testing.T) {
	s := NewStateMap()
	s.Set("key", "value")

	got := s.Get("key")
	if got != "value" {
		t.Errorf("Get(key) = %v, want value", got)
	}
}

func TestStateMap_GetMissing(t *testing.T) {
	s := NewStateMap()
	got := s.Get("missing")
	if got != nil {
		t.Errorf("Get(missing) = %v, want nil", got)
	}
}

func TestStateMap_Provider(t *testing.T) {
	s := NewStateMap()
	if s.Provider() != "" {
		t.Errorf("Provider() = %q, want empty", s.Provider())
	}

	s.Set("provider", "gitlab")
	if s.Provider() != "gitlab" {
		t.Errorf("Provider() = %q, want gitlab", s.Provider())
	}
}

func TestStateMap_Binary(t *testing.T) {
	s := NewStateMap()
	if s.Binary() != "" {
		t.Errorf("Binary() = %q, want empty", s.Binary())
	}

	s.Set("binary", "terraform")
	if s.Binary() != "terraform" {
		t.Errorf("Binary() = %q, want terraform", s.Binary())
	}
}

func TestStateMap_StringPtr(t *testing.T) {
	s := NewStateMap()

	// First call creates pointer with empty default
	ptr := s.StringPtr("name")
	if *ptr != "" {
		t.Errorf("StringPtr default = %q, want empty", *ptr)
	}

	// Mutating pointer updates Get
	*ptr = "hello"
	got := s.Get("name")
	if got != "hello" {
		t.Errorf("Get after pointer mutation = %v, want hello", got)
	}

	// Second call returns same pointer
	ptr2 := s.StringPtr("name")
	if ptr != ptr2 {
		t.Error("StringPtr should return stable pointer")
	}
}

func TestStateMap_StringPtr_WithPresetValue(t *testing.T) {
	s := NewStateMap()
	s.Set("name", "preset")

	ptr := s.StringPtr("name")
	if *ptr != "preset" {
		t.Errorf("StringPtr with preset = %q, want preset", *ptr)
	}
}

func TestStateMap_BoolPtr(t *testing.T) {
	s := NewStateMap()

	ptr := s.BoolPtr("enabled")
	if *ptr != false {
		t.Errorf("BoolPtr default = %v, want false", *ptr)
	}

	*ptr = true
	got := s.Get("enabled")
	if got != true {
		t.Errorf("Get after pointer mutation = %v, want true", got)
	}

	// Stable pointer
	ptr2 := s.BoolPtr("enabled")
	if ptr != ptr2 {
		t.Error("BoolPtr should return stable pointer")
	}
}

func TestStateMap_BoolPtr_WithPresetValue(t *testing.T) {
	s := NewStateMap()
	s.Set("enabled", true)

	ptr := s.BoolPtr("enabled")
	if *ptr != true {
		t.Errorf("BoolPtr with preset = %v, want true", *ptr)
	}
}

func TestStateMap_StringPtr_OverridesPlainValue(t *testing.T) {
	s := NewStateMap()
	s.Set("key", "plain")

	ptr := s.StringPtr("key")
	*ptr = "pointer"

	// Get should prefer pointer-backed value
	if s.Get("key") != "pointer" {
		t.Errorf("Get should return pointer value, got %v", s.Get("key"))
	}
}

func TestStateMap_Provider_NonStringValue(t *testing.T) {
	s := NewStateMap()
	s.Set("provider", 42) // not a string

	if s.Provider() != "" {
		t.Errorf("Provider with non-string = %q, want empty", s.Provider())
	}
}

func TestStateMap_Binary_NonStringValue(t *testing.T) {
	s := NewStateMap()
	s.Set("binary", true) // not a string

	if s.Binary() != "" {
		t.Errorf("Binary with non-string = %q, want empty", s.Binary())
	}
}

func TestStateMap_String(t *testing.T) {
	s := NewStateMap()

	if s.String("missing") != "" {
		t.Error("String(missing) should return empty string")
	}

	s.Set("key", "hello")
	if s.String("key") != "hello" {
		t.Errorf("String(key) = %q, want hello", s.String("key"))
	}

	// Non-string value
	s.Set("num", 42)
	if s.String("num") != "" {
		t.Errorf("String(num) = %q, want empty for non-string", s.String("num"))
	}
}

func TestStateMap_Bool(t *testing.T) {
	s := NewStateMap()

	if s.Bool("missing") {
		t.Error("Bool(missing) should return false")
	}

	s.Set("flag", true)
	if !s.Bool("flag") {
		t.Error("Bool(flag) should return true")
	}

	s.Set("flag", false)
	if s.Bool("flag") {
		t.Error("Bool(flag) should return false")
	}

	// Non-bool value
	s.Set("str", "yes")
	if s.Bool("str") {
		t.Error("Bool(str) should return false for non-bool")
	}
}

func TestStateMap_String_PrefersPointerValue(t *testing.T) {
	s := NewStateMap()
	s.Set("key", "plain")

	ptr := s.StringPtr("key")
	*ptr = "pointer"

	if s.String("key") != "pointer" {
		t.Errorf("String should prefer pointer value, got %q", s.String("key"))
	}
}

func TestStateMap_Bool_PrefersPointerValue(t *testing.T) {
	s := NewStateMap()
	s.Set("flag", false)

	ptr := s.BoolPtr("flag")
	*ptr = true

	if !s.Bool("flag") {
		t.Error("Bool should prefer pointer value")
	}
}
