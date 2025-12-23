# S3 bucket security policies
#
# These policies enforce security best practices for S3 buckets.
# Uses OPA v1 Rego syntax.

package terraform

import rego.v1

# Deny public S3 buckets
deny contains msg if {
	resource := input.resource_changes[_]
	resource.type == "aws_s3_bucket"
	resource.change.actions[_] != "delete"
	resource.change.after.acl == "public-read"
	msg := sprintf("S3 bucket '%s' must not have public-read ACL", [resource.name])
}

deny contains msg if {
	resource := input.resource_changes[_]
	resource.type == "aws_s3_bucket"
	resource.change.actions[_] != "delete"
	resource.change.after.acl == "public-read-write"
	msg := sprintf("S3 bucket '%s' must not have public-read-write ACL", [resource.name])
}

# Deny S3 buckets without encryption
deny contains msg if {
	resource := input.resource_changes[_]
	resource.type == "aws_s3_bucket"
	resource.change.actions[_] == "create"
	not has_encryption(resource)
	msg := sprintf("S3 bucket '%s' must have server-side encryption enabled", [resource.name])
}

# Warn about buckets without versioning
warn contains msg if {
	resource := input.resource_changes[_]
	resource.type == "aws_s3_bucket"
	resource.change.actions[_] != "delete"
	not has_versioning(resource)
	msg := sprintf("S3 bucket '%s' should have versioning enabled", [resource.name])
}

# Helper: check if bucket has encryption configuration
has_encryption(resource) if {
	resource.change.after.server_side_encryption_configuration[_]
}

# Helper: check if bucket has versioning
has_versioning(resource) if {
	resource.change.after.versioning[_].enabled == true
}
