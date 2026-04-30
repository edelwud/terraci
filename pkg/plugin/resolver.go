package plugin

// Resolver is the minimal command-scoped plugin lookup surface.
type Resolver interface {
	GetPlugin(name string) (Plugin, bool)
}
