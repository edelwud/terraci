package policyengine

import "fmt"

// Action controls how OPA decisions are surfaced to CI.
type Action string

const (
	ActionBlock  Action = "block"
	ActionWarn   Action = "warn"
	ActionIgnore Action = "ignore"
)

// Decisions configures the effective enforcement behavior for OPA decisions.
type Decisions struct {
	Deny Action `yaml:"deny,omitempty" json:"deny,omitempty" jsonschema:"description=Action for OPA deny decisions,enum=block,enum=warn,enum=ignore,default=block"`
	Warn Action `yaml:"warn,omitempty" json:"warn,omitempty" jsonschema:"description=Action for OPA warn decisions,enum=block,enum=warn,enum=ignore,default=warn"`
}

func DefaultDecisions() Decisions {
	return Decisions{
		Deny: ActionBlock,
		Warn: ActionWarn,
	}
}

func (a Action) Valid() bool {
	switch a {
	case ActionBlock, ActionWarn, ActionIgnore:
		return true
	default:
		return false
	}
}

func (a Action) String() string {
	return string(a)
}

func ValidateAction(name string, action Action) error {
	if action == "" {
		return nil
	}
	if !action.Valid() {
		return fmt.Errorf("%s must be one of: block, warn, ignore", name)
	}
	return nil
}

func (d Decisions) Normalize() Decisions {
	if d.Deny == "" {
		d.Deny = ActionBlock
	}
	if d.Warn == "" {
		d.Warn = ActionWarn
	}
	return d
}

func (d Decisions) Validate() error {
	if err := ValidateAction("decisions.deny", d.Deny); err != nil {
		return err
	}
	return ValidateAction("decisions.warn", d.Warn)
}

func (d Decisions) CanBlock() bool {
	normalized := d.Normalize()
	return normalized.Deny == ActionBlock || normalized.Warn == ActionBlock
}
