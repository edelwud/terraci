# S3 bucket security policies
#
# These policies enforce security best practices for S3 buckets.
# Uses OPA v1 Rego syntax.

package terraform

import rego.v1

# Deny public S3 buckets
deny contains msg if {
	some resource in input.resource_changes
	resource.type == "aws_s3_bucket"
	not "delete" in resource.change.actions
	resource.change.after.acl == "public-read"
	msg := sprintf(
		"S3 bucket '%s' must not have public-read ACL",
		[resource.name],
	)
}

deny contains msg if {
	some resource in input.resource_changes
	resource.type == "aws_s3_bucket"
	not "delete" in resource.change.actions
	resource.change.after.acl == "public-read-write"
	msg := sprintf(
		"S3 bucket '%s' must not have public-read-write ACL",
		[resource.name],
	)
}

# Deny S3 buckets without encryption
deny contains msg if {
	some resource in input.resource_changes
	resource.type == "aws_s3_bucket"
	"create" in resource.change.actions
	not has_encryption(resource)
	msg := sprintf(
		"S3 bucket '%s' must have server-side encryption enabled",
		[resource.name],
	)
}

# Warn about buckets without versioning
warn contains msg if {
	some resource in input.resource_changes
	resource.type == "aws_s3_bucket"
	not "delete" in resource.change.actions
	not has_versioning(resource)
	msg := sprintf(
		"S3 bucket '%s' should have versioning enabled",
		[resource.name],
	)
}

# Helper: check if bucket has encryption configuration
has_encryption(resource) if {
	some _ in resource.change.after.server_side_encryption_configuration
}

# Helper: check if bucket has versioning
has_versioning(resource) if {
	some v in resource.change.after.versioning
	v.enabled == true
}
