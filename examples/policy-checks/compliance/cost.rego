# Cost compliance policies
#
# These policies enforce cost controls on infrastructure resources.
# Evaluated in the "compliance" namespace.

package compliance

import rego.v1

# METADATA
# description: Deny expensive instance types without explicit approval tag
# entrypoint: true
deny contains msg if {
	some resource in input.resource_changes
	resource.type == "aws_instance"
	"create" in resource.change.actions
	is_expensive_type(resource.change.after.instance_type)
	not has_approval_tag(resource)
	msg := sprintf(
		"Instance '%s' uses expensive type '%s' — add tag 'CostApproved=true' to proceed",
		[resource.name, resource.change.after.instance_type],
	)
}

# Warn about any RDS multi-AZ deployment
warn contains msg if {
	some resource in input.resource_changes
	resource.type == "aws_db_instance"
	not "delete" in resource.change.actions
	resource.change.after.multi_az == true
	msg := sprintf(
		"RDS instance '%s' has multi-AZ enabled — monthly cost doubles",
		[resource.name],
	)
}

# Warn about large EBS volumes
warn contains msg if {
	some resource in input.resource_changes
	resource.type == "aws_ebs_volume"
	not "delete" in resource.change.actions
	resource.change.after.size > 500
	msg := sprintf(
		"EBS volume '%s' is %dGB — consider if this size is necessary",
		[resource.name, resource.change.after.size],
	)
}

# Helper: check if instance type is expensive (GPU, memory-optimized, etc.)
is_expensive_type(instance_type) if {
	expensive_prefixes := ["p3", "p4", "p5", "g4", "g5", "x1", "x2", "r6g.8", "r6g.12", "r6g.16", "i3.8", "i3.16"]
	some prefix in expensive_prefixes
	startswith(instance_type, prefix)
}

# Helper: check if resource has CostApproved tag
has_approval_tag(resource) if {
	resource.change.after.tags.CostApproved == "true"
}
