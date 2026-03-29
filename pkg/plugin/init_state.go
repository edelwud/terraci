package plugin

// StateMap is the typed init state backed by maps.
// It provides both typed accessors and stable pointers for huh form field binding.
type StateMap struct {
	values  map[string]any
	strings map[string]*string
	bools   map[string]*bool
}

// NewStateMap creates a new empty StateMap.
func NewStateMap() *StateMap {
	return &StateMap{
		values:  make(map[string]any),
		strings: make(map[string]*string),
		bools:   make(map[string]*bool),
	}
}

// Set stores a value in the state.
func (s *StateMap) Set(key string, val any) { s.values[key] = val }

// Get retrieves a value, preferring pointer-backed values from StringPtr/BoolPtr.
func (s *StateMap) Get(key string) any {
	if p, ok := s.strings[key]; ok {
		return *p
	}
	if p, ok := s.bools[key]; ok {
		return *p
	}
	return s.values[key]
}

// String returns a string value for the key, or empty string if not found.
func (s *StateMap) String(key string) string {
	v, ok := s.Get(key).(string)
	if !ok {
		return ""
	}
	return v
}

// Bool returns a bool value for the key, or false if not found.
func (s *StateMap) Bool(key string) bool {
	v, ok := s.Get(key).(bool)
	if !ok {
		return false
	}
	return v
}

// Provider returns the current provider name.
func (s *StateMap) Provider() string { return s.String("provider") }

// Binary returns the current binary name.
func (s *StateMap) Binary() string { return s.String("binary") }

// StringPtr returns a stable *string pointer for huh form binding.
// If a value was previously Set for the key, it initializes the pointer with that value.
func (s *StateMap) StringPtr(key string) *string {
	if p, ok := s.strings[key]; ok {
		return p
	}
	v, ok := s.values[key].(string)
	if !ok {
		v = ""
	}
	s.strings[key] = &v
	return s.strings[key]
}

// BoolPtr returns a stable *bool pointer for huh form binding.
// If a value was previously Set for the key, it initializes the pointer with that value.
func (s *StateMap) BoolPtr(key string) *bool {
	if p, ok := s.bools[key]; ok {
		return p
	}
	v, ok := s.values[key].(bool)
	if !ok {
		v = false
	}
	s.bools[key] = &v
	return s.bools[key]
}
