package model

// EstimateAction is the provider-neutral Terraform plan action model used across
// the cost engine, result assembly, and pipeline scanning layers.
type EstimateAction string

const (
	ActionCreate  EstimateAction = "create"
	ActionDelete  EstimateAction = "delete"
	ActionUpdate  EstimateAction = "update"
	ActionReplace EstimateAction = "replace"
	ActionNoOp    EstimateAction = "no-op"
)
