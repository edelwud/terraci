package ciplugin

import (
	log "github.com/caarlos0/log"
)

// ConfigValidator is the optional contract a CI provider config must satisfy
// for Preflight to validate it. nil configs and configs that do not
// implement Validate() are treated as valid.
type ConfigValidator interface {
	Validate() error
}

// PreflightLog describes how to surface CI environment detection in logs.
// providerName is the human label ("gitlab", "github"); contextLabel is the
// short marker for MR/PR context ("MR", "PR"). detectInContext returns
// (active, identifier, ok) — when ok is false the helper logs the
// "no MR/PR" branch.
type PreflightLog struct {
	ProviderName    string
	ContextLabel    string
	DetectInContext func() (id any, active bool)
}

// Preflight runs the standard CI provider preflight: optional config
// validation, then env/context detection logging. It replaces the 17-line
// near-identical Preflight bodies in gitlab/lifecycle.go and
// github/lifecycle.go.
//
// Pass cfg = nil when the plugin has no config, or when the loaded config
// is nil (BasePlugin[C].Config() may return nil before configure runs).
// detectEnv reports whether the runtime is inside the provider's CI; when
// false the helper short-circuits with a successful return.
func Preflight(cfg ConfigValidator, detectEnv func() bool, lg PreflightLog) error {
	if cfg != nil {
		if err := cfg.Validate(); err != nil {
			return err
		}
	}

	if !detectEnv() {
		return nil
	}

	if id, active := lg.DetectInContext(); active {
		log.WithField(lowerCase(lg.ContextLabel), id).
			Debugf("%s: %s context detected", lg.ProviderName, lg.ContextLabel)
	} else {
		log.Debugf("%s: CI detected but not in %s pipeline", lg.ProviderName, lg.ContextLabel)
	}

	return nil
}

// lowerCase returns the lower-case form of a short label like "MR" → "mr".
// Used so log fields stay snake_case ("mr", "pr") even when callers pass
// uppercase context names.
func lowerCase(s string) string {
	out := make([]byte, len(s))
	for i := range len(s) {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}
