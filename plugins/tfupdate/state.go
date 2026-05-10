package tfupdate

// Reset resets command-scoped plugin state.
func (p *Plugin) Reset() {
	p.BasePlugin.Reset()
	p.registryFactory = nil
}
