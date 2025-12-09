variable "cluster_name" {
  description = "Name of the MSK cluster"
  type        = string
}

variable "kafka_versions" {
  description = "Kafka versions to support"
  type        = list(string)
  default     = ["3.5.1"]
}

variable "broker_count" {
  description = "Number of broker nodes"
  type        = number
  default     = 3
}

variable "instance_type" {
  description = "Instance type for brokers"
  type        = string
  default     = "kafka.m5.large"
}

variable "subnet_ids" {
  description = "Subnet IDs for broker nodes"
  type        = list(string)
}

variable "security_group_ids" {
  description = "Security group IDs for broker nodes"
  type        = list(string)
}

variable "ebs_volume_size" {
  description = "EBS volume size in GB"
  type        = number
  default     = 100
}

variable "auto_create_topics" {
  description = "Enable auto topic creation"
  type        = bool
  default     = false
}

variable "replication_factor" {
  description = "Default replication factor"
  type        = number
  default     = 3
}

variable "min_insync_replicas" {
  description = "Minimum in-sync replicas"
  type        = number
  default     = 2
}

variable "default_partitions" {
  description = "Default number of partitions"
  type        = number
  default     = 3
}

variable "log_retention_hours" {
  description = "Log retention in hours"
  type        = number
  default     = 168
}

variable "tags" {
  description = "Tags to apply to resources"
  type        = map(string)
  default     = {}
}
