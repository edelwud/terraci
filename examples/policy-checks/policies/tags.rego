# Resource tagging policies
#
# These policies enforce required tags on all resources.
# Uses OPA v1 Rego syntax.

package terraform

import rego.v1

# Required tags for all taggable resources
required_tags := ["Environment", "Project", "Owner"]

# Taggable AWS resource types
taggable_types := [
	"aws_instance",
	"aws_s3_bucket",
	"aws_rds_cluster",
	"aws_rds_instance",
	"aws_elasticache_cluster",
	"aws_elasticsearch_domain",
	"aws_lambda_function",
	"aws_ecs_cluster",
	"aws_ecs_service",
	"aws_eks_cluster",
	"aws_lb",
	"aws_lb_target_group",
	"aws_security_group",
	"aws_vpc",
	"aws_subnet",
]

# Deny resources missing required tags
deny contains msg if {
	resource := input.resource_changes[_]
	resource.change.actions[_] == "create"
	is_taggable(resource.type)
	tag := required_tags[_]
	not has_tag(resource, tag)
	msg := sprintf("%s '%s' is missing required tag '%s'", [resource.type, resource.name, tag])
}

# Warn about resources with empty tag values
warn contains msg if {
	resource := input.resource_changes[_]
	resource.change.actions[_] != "delete"
	is_taggable(resource.type)
	tag := required_tags[_]
	has_tag(resource, tag)
	get_tag_value(resource, tag) == ""
	msg := sprintf("%s '%s' has empty value for tag '%s'", [resource.type, resource.name, tag])
}

# Helper: check if resource type is taggable
is_taggable(resource_type) if {
	resource_type == taggable_types[_]
}

# Helper: check if resource has a specific tag
has_tag(resource, tag) if {
	resource.change.after.tags[tag]
}

has_tag(resource, tag) if {
	resource.change.after.tags_all[tag]
}

# Helper: get tag value
get_tag_value(resource, tag) := value if {
	value := resource.change.after.tags[tag]
}

get_tag_value(resource, tag) := value if {
	not resource.change.after.tags[tag]
	value := resource.change.after.tags_all[tag]
}
