package plugin

// Source exposes a command-scoped set of plugin instances.
type Source interface {
	All() []Plugin
	GetPlugin(name string) (Plugin, bool)
}
