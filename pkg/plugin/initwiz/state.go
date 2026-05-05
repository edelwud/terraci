// Package initwiz provides init wizard state management and types for TerraCi.
// It contains the StateMap for huh form binding and all init-related type
// definitions (InitContributor, InitGroupSpec, InitField, etc.).
package initwiz

// StateMap is the typed init state backed by a single map[string]any. It
// exposes typed accessors for plugins (String/Bool) and stable pointers for
// huh form field binding (StringPtr/BoolPtr).
//
// Lifecycle:
//  1. Core/plugins call Set("key", value) to populate defaults.
//  2. Plugins call StringPtr("key") / BoolPtr("key") to obtain stable
//     pointers for huh fields. The first call upgrades the slot from the
//     plain value to a typed pointer; subsequent Set/Get/String/Bool keep
//     working through that pointer.
//  3. huh forms mutate values through the pointers during the wizard.
//  4. After the form completes, BuildInitConfig reads back via Get/String/Bool.
//
// All accessors flow through the same map, so Set after StringPtr now
// updates the value backing that pointer (no more silent staleness).
type StateMap struct {
	values map[string]any
}

// NewStateMap creates an empty StateMap.
func NewStateMap() *StateMap {
	return &StateMap{values: make(map[string]any)}
}

// Set stores a value, transparently honoring any *string / *bool slot
// previously installed by StringPtr / BoolPtr so existing form bindings
// keep observing the new value.
func (s *StateMap) Set(key string, val any) {
	switch existing := s.values[key].(type) {
	case *string:
		if v, ok := val.(string); ok {
			*existing = v
			return
		}
	case *bool:
		if v, ok := val.(bool); ok {
			*existing = v
			return
		}
	}
	s.values[key] = val
}

// Get retrieves a value, transparently dereferencing pointer-backed slots.
func (s *StateMap) Get(key string) any {
	switch v := s.values[key].(type) {
	case *string:
		return *v
	case *bool:
		return *v
	default:
		return v
	}
}

// String returns the value at key as a string, or "" if missing or of a
// non-string type.
func (s *StateMap) String(key string) string {
	v, ok := s.Get(key).(string)
	if !ok {
		return ""
	}
	return v
}

// Bool returns the value at key as a bool, or false if missing or of a
// non-bool type.
func (s *StateMap) Bool(key string) bool {
	v, ok := s.Get(key).(bool)
	if !ok {
		return false
	}
	return v
}

// Provider returns the current provider name.
func (s *StateMap) Provider() string { return s.String("provider") }

// Binary returns the current terraform binary name.
func (s *StateMap) Binary() string { return s.String("binary") }

// StringPtr returns a stable *string pointer for huh form binding. If the
// slot already holds a *string it is returned unchanged; if it held a plain
// string the slot is upgraded to a pointer initialized with that string.
func (s *StateMap) StringPtr(key string) *string {
	if p, ok := s.values[key].(*string); ok {
		return p
	}
	var v string
	if existing, ok := s.values[key].(string); ok {
		v = existing
	}
	p := &v
	s.values[key] = p
	return p
}

// BoolPtr returns a stable *bool pointer for huh form binding. If the slot
// already holds a *bool it is returned unchanged; if it held a plain bool
// the slot is upgraded to a pointer initialized with that bool.
func (s *StateMap) BoolPtr(key string) *bool {
	if p, ok := s.values[key].(*bool); ok {
		return p
	}
	var v bool
	if existing, ok := s.values[key].(bool); ok {
		v = existing
	}
	p := &v
	s.values[key] = p
	return p
}
