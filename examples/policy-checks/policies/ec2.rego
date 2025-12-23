# EC2 instance compliance policies
#
# These policies enforce security and compliance rules for EC2 instances.
# Uses OPA v1 Rego syntax.

package terraform

import rego.v1

# Deny instances with public IP in production
deny contains msg if {
	resource := input.resource_changes[_]
	resource.type == "aws_instance"
	resource.change.actions[_] != "delete"
	resource.change.after.associate_public_ip_address == true
	is_production_module
	msg := sprintf("EC2 instance '%s' must not have public IP in production", [resource.name])
}

# Deny instances without IMDSv2
deny contains msg if {
	resource := input.resource_changes[_]
	resource.type == "aws_instance"
	resource.change.actions[_] == "create"
	not uses_imdsv2(resource)
	msg := sprintf("EC2 instance '%s' must use IMDSv2 (metadata_options.http_tokens = required)", [resource.name])
}

# Warn about instances using default VPC
warn contains msg if {
	resource := input.resource_changes[_]
	resource.type == "aws_instance"
	resource.change.actions[_] != "delete"
	not resource.change.after.subnet_id
	msg := sprintf("EC2 instance '%s' should specify a subnet_id (avoid default VPC)", [resource.name])
}

# Warn about large instance types
warn contains msg if {
	resource := input.resource_changes[_]
	resource.type == "aws_instance"
	resource.change.actions[_] != "delete"
	is_large_instance(resource.change.after.instance_type)
	msg := sprintf("EC2 instance '%s' uses large instance type '%s' - verify this is needed", [resource.name, resource.change.after.instance_type])
}

# Helper: check if module path contains "prod"
is_production_module if {
	contains(input.configuration.root_module.module_calls[_].source, "prod")
}

# Helper: check if IMDSv2 is enabled
uses_imdsv2(resource) if {
	resource.change.after.metadata_options[_].http_tokens == "required"
}

# Helper: check if instance type is large
is_large_instance(instance_type) if {
	large_types := ["x1", "x2", "p3", "p4", "g4", "g5", "inf1", "trn1"]
	some prefix in large_types
	startswith(instance_type, prefix)
}
