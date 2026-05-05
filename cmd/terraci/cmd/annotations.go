package cmd

// Cobra command annotation keys + canonical "true" value used for opt-in
// flags. Keep these centralized so the goconst lint stays quiet and the
// annotation vocabulary doesn't drift across command files.
const (
	annotationSkipConfig    = "skipConfig"
	annotationSkipPreflight = "skipPreflight"
	annotationTrue          = "true"
)
