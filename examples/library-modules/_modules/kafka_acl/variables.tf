variable "principal" {
  description = "Principal (user/service) to grant access"
  type        = string
}

variable "read_topics" {
  description = "Topics to grant read access"
  type        = list(string)
  default     = []
}

variable "write_topics" {
  description = "Topics to grant write access"
  type        = list(string)
  default     = []
}

variable "consumer_group" {
  description = "Consumer group to grant access (optional)"
  type        = string
  default     = null
}

variable "cluster_arn" {
  description = "ARN of the MSK cluster"
  type        = string
}
