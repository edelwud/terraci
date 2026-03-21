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

# METADATA
# entrypoint: true
deny contains msg if {
	some resource in input.resource_changes
	"create" in resource.change.actions
	resource.type in taggable_types
	some tag in required_tags
	not has_tag(resource, tag)
	msg := sprintf(
		"%s '%s' is missing required tag '%s'",
		[resource.type, resource.name, tag],
	)
}

# Warn about resources with empty tag values
warn contains msg if {
	some resource in input.resource_changes
	not "delete" in resource.change.actions
	resource.type in taggable_types
	some tag in required_tags
	has_tag(resource, tag)
	tag_value(resource, tag) == ""
	msg := sprintf(
		"%s '%s' has empty value for tag '%s'",
		[resource.type, resource.name, tag],
	)
}

# Helper: check if resource has a specific tag
has_tag(resource, tag) if {
	resource.change.after.tags[tag]
}

has_tag(resource, tag) if {
	resource.change.after.tags_all[tag]
}

# Helper: get tag value (from tags or tags_all)
tag_value(resource, tag) := resource.change.after.tags[tag]

tag_value(resource, tag) := resource.change.after.tags_all[tag] if {
	not resource.change.after.tags[tag]
}
