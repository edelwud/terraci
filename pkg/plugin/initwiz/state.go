// Package initwiz provides init wizard state management and types for TerraCi.
package initwiz

import (
	"errors"
	"fmt"
	"strings"
)

// StateValue is the set of supported init wizard state value types.
type StateValue interface {
	~string | ~bool
}

// StateKey is a typed key into StateMap.
type StateKey[T StateValue] struct {
	name string
}

// Canonical init wizard state keys owned by TerraCi core.
var (
	ProviderKey       = MustStateKey[string]("provider")
	BinaryKey         = MustStateKey[string]("binary")
	PatternKey        = MustStateKey[string]("pattern")
	PlanEnabledKey    = MustStateKey[bool]("plan_enabled")
	SummaryEnabledKey = MustStateKey[bool]("summary.enabled")
)

// NewStateKey constructs a validated typed state key.
func NewStateKey[T StateValue](name string) (StateKey[T], error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return StateKey[T]{}, errors.New("init state key is required")
	}
	if strings.ContainsAny(name, " \t\r\n") {
		return StateKey[T]{}, fmt.Errorf("init state key %q must not contain whitespace", name)
	}
	return StateKey[T]{name: name}, nil
}

// MustStateKey constructs a typed state key and panics for invalid programmer
// input. Plugin packages should define their init wizard keys once at package
// scope with this helper.
func MustStateKey[T StateValue](name string) StateKey[T] {
	key, err := NewStateKey[T](name)
	if err != nil {
		panic(err)
	}
	return key
}

// Name returns the raw key name used in form rendering and diagnostics.
func (k StateKey[T]) Name() string { return k.name }

// Get returns the typed state value, or the zero value when unset.
func (k StateKey[T]) Get(state *StateMap) T {
	value, _ := k.Lookup(state)
	return value
}

// Lookup returns the typed state value and whether it was set.
func (k StateKey[T]) Lookup(state *StateMap) (T, bool) {
	var zero T
	if state == nil || k.name == "" {
		return zero, false
	}
	switch value := state.values[k.name].(type) {
	case *T:
		if value == nil {
			return zero, false
		}
		return *value, true
	case T:
		return value, true
	default:
		return zero, false
	}
}

// Set stores a typed state value. If a UI binding already exists for this key,
// the binding target is updated in place.
func (k StateKey[T]) Set(state *StateMap, value T) {
	if state == nil || k.name == "" {
		return
	}
	if existing, ok := state.values[k.name].(*T); ok && existing != nil {
		*existing = value
		return
	}
	v := value
	state.values[k.name] = &v
}

// Bind returns a stable typed pointer for TUI form binding.
func (k StateKey[T]) Bind(state *StateMap) *T {
	var value T
	if state == nil || k.name == "" {
		return &value
	}
	if existing, ok := state.values[k.name].(*T); ok && existing != nil {
		return existing
	}
	if existing, ok := state.values[k.name].(T); ok {
		value = existing
	}
	state.values[k.name] = &value
	return &value
}

// StateMap is mutable wizard state backed by typed StateKey slots. It remains
// mutable because TUI form bindings need stable pointers, but plugin code
// reads and writes only through StateKey values.
type StateMap struct {
	values map[string]any
}

// NewStateMap creates an empty StateMap.
func NewStateMap() *StateMap {
	return &StateMap{values: make(map[string]any)}
}
