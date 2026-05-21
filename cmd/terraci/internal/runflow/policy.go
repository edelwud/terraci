package runflow

import "github.com/spf13/cobra"

const (
	annotationSkipConfig    = "terraci.runflow.skipConfig"
	annotationSkipPreflight = "terraci.runflow.skipPreflight"
	annotationTrue          = "true"
)

// CommandPolicy describes which framework lifecycle phases a cobra command
// needs before its RunE executes.
type CommandPolicy struct {
	SkipConfig    bool
	SkipPreflight bool
}

// MarkCommand records policy on cmd. Cobra annotations are kept as a private
// storage detail so command packages use typed policy instead of raw strings.
func MarkCommand(cmd *cobra.Command, policy CommandPolicy) *cobra.Command {
	if cmd == nil {
		return nil
	}
	if cmd.Annotations == nil {
		cmd.Annotations = make(map[string]string)
	}
	if policy.SkipConfig {
		cmd.Annotations[annotationSkipConfig] = annotationTrue
	} else {
		delete(cmd.Annotations, annotationSkipConfig)
	}
	if policy.SkipPreflight {
		cmd.Annotations[annotationSkipPreflight] = annotationTrue
	} else {
		delete(cmd.Annotations, annotationSkipPreflight)
	}
	if len(cmd.Annotations) == 0 {
		cmd.Annotations = nil
	}
	return cmd
}

// PolicyFromCommand returns the typed runflow policy stored on cmd.
func PolicyFromCommand(cmd *cobra.Command) CommandPolicy {
	if cmd == nil {
		return CommandPolicy{}
	}
	return CommandPolicy{
		SkipConfig:    cmd.Annotations[annotationSkipConfig] == annotationTrue,
		SkipPreflight: cmd.Annotations[annotationSkipPreflight] == annotationTrue,
	}
}
