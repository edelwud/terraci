package plugin

// CommandPlugin returns the current per-command plugin instance matching the
// fallback plugin's name. It lets cobra commands be registered from prototype
// plugin instances while executing against the fresh command-scoped instance.
func CommandPlugin[T Plugin](ctx *AppContext, fallback T) T {
	if ctx == nil || ctx.resolver == nil {
		return fallback
	}
	current, ok := ctx.resolver.GetPlugin(fallback.Name())
	if !ok {
		return fallback
	}
	typed, ok := current.(T)
	if !ok {
		return fallback
	}
	currentConfig, currentHasConfig := current.(ConfigLoader)
	fallbackConfig, fallbackHasConfig := any(fallback).(ConfigLoader)
	if currentHasConfig && fallbackHasConfig && !currentConfig.IsConfigured() && fallbackConfig.IsConfigured() {
		return fallback
	}
	return typed
}
