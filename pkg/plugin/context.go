package plugin

import (
	"sync"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/config"
)

// AppContext is the public API available to plugins.
// Config and WorkDir are resolved lazily via the accessor function
// so they reflect the state after PersistentPreRunE completes.
type AppContext struct {
	Config     *config.Config
	WorkDir    string
	ServiceDir string // resolved absolute path to project service directory
	Version    string

	// Refresh is called before accessing Config/WorkDir to ensure
	// they reflect the current state. Set by the App struct.
	Refresh func()
}

// Ensure refreshes the context from the App state.
// Safe to call multiple times; no-op if Refresh is nil.
func (a *AppContext) Ensure() {
	if a.Refresh != nil {
		a.Refresh()
	}
}

// ExecutionContext holds shared mutable state during command execution.
// Plugins read and write to this during the summary phase.
type ExecutionContext struct {
	mu sync.Mutex

	// PlanResults is the collected plan result data, available for enrichment.
	PlanResults *ci.PlanResultCollection

	// Sections holds additional markdown sections contributed by plugins.
	Sections []CommentSection

	// Data holds arbitrary typed data for inter-plugin communication.
	Data map[string]any
}

// CommentSection is an additional section contributed by a plugin to the PR/MR comment.
type CommentSection struct {
	Order   int    // Lower = rendered first
	Title   string // Section heading
	Content string // Markdown content
}

// NewExecutionContext creates an ExecutionContext from plan results.
func NewExecutionContext(plans *ci.PlanResultCollection) *ExecutionContext {
	return &ExecutionContext{
		PlanResults: plans,
		Data:        make(map[string]any),
	}
}

// SetData stores a value for inter-plugin communication.
func (e *ExecutionContext) SetData(key string, value any) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Data[key] = value
}

// GetData retrieves a value set by another plugin.
func (e *ExecutionContext) GetData(key string) (any, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	v, ok := e.Data[key]
	return v, ok
}

// AddSection adds a comment section from a plugin.
func (e *ExecutionContext) AddSection(section CommentSection) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Sections = append(e.Sections, section)
}
