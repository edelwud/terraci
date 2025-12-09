# Kafka ACL module - manages access control for Kafka clusters
# This module depends on the kafka module for cluster information

# Note: This demonstrates transitive library dependencies
# When kafka module changes, modules using kafka_acl should also be affected

resource "kafka_acl" "read" {
  for_each = toset(var.read_topics)

  resource_name       = each.value
  resource_type       = "Topic"
  acl_principal       = var.principal
  acl_host            = "*"
  acl_operation       = "Read"
  acl_permission_type = "Allow"
}

resource "kafka_acl" "write" {
  for_each = toset(var.write_topics)

  resource_name       = each.value
  resource_type       = "Topic"
  acl_principal       = var.principal
  acl_host            = "*"
  acl_operation       = "Write"
  acl_permission_type = "Allow"
}

resource "kafka_acl" "describe" {
  for_each = toset(concat(var.read_topics, var.write_topics))

  resource_name       = each.value
  resource_type       = "Topic"
  acl_principal       = var.principal
  acl_host            = "*"
  acl_operation       = "Describe"
  acl_permission_type = "Allow"
}

resource "kafka_acl" "consumer_group" {
  count = var.consumer_group != null ? 1 : 0

  resource_name       = var.consumer_group
  resource_type       = "Group"
  acl_principal       = var.principal
  acl_host            = "*"
  acl_operation       = "Read"
  acl_permission_type = "Allow"
}
