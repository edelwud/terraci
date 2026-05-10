package domain

import "fmt"

// Action controls how policy decisions are surfaced to CI.
type Action string

const (
	ActionBlock  Action = "block"
	ActionWarn   Action = "warn"
	ActionIgnore Action = "ignore"
)

// ActionPolicy is the effective enforcement policy for one module.
type ActionPolicy struct {
	FailureAction Action
	WarningAction Action
}

func DefaultActionPolicy() ActionPolicy {
	return ActionPolicy{
		FailureAction: ActionBlock,
		WarningAction: ActionWarn,
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

func (p ActionPolicy) CanBlock() bool {
	return p.FailureAction == ActionBlock || p.WarningAction == ActionBlock
}
