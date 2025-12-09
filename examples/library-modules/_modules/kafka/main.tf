# Reusable Kafka/MSK configuration module
# This is a library module - no providers or remote state here

resource "aws_msk_configuration" "this" {
  name           = var.cluster_name
  kafka_versions = var.kafka_versions

  server_properties = <<PROPERTIES
auto.create.topics.enable = ${var.auto_create_topics}
default.replication.factor = ${var.replication_factor}
min.insync.replicas = ${var.min_insync_replicas}
num.partitions = ${var.default_partitions}
log.retention.hours = ${var.log_retention_hours}
PROPERTIES
}

resource "aws_msk_cluster" "this" {
  cluster_name           = var.cluster_name
  kafka_version          = var.kafka_versions[0]
  number_of_broker_nodes = var.broker_count

  broker_node_group_info {
    instance_type   = var.instance_type
    client_subnets  = var.subnet_ids
    security_groups = var.security_group_ids

    storage_info {
      ebs_storage_info {
        volume_size = var.ebs_volume_size
      }
    }
  }

  configuration_info {
    arn      = aws_msk_configuration.this.arn
    revision = aws_msk_configuration.this.latest_revision
  }

  encryption_info {
    encryption_in_transit {
      client_broker = "TLS"
      in_cluster    = true
    }
  }

  tags = var.tags
}
