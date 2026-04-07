package cloud

// ResetForTesting clears the provider registry.
// Compiled only for tests; not available in production code.
func ResetForTesting() {
	cpMu.Lock()
	defer cpMu.Unlock()
	providers = make(map[string]Provider)
	cpOrder = nil
}
