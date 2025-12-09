output "read_acl_ids" {
  description = "IDs of read ACLs"
  value       = [for acl in kafka_acl.read : acl.id]
}

output "write_acl_ids" {
  description = "IDs of write ACLs"
  value       = [for acl in kafka_acl.write : acl.id]
}
