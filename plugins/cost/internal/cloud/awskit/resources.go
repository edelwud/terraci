package awskit

// ResourceKey is a typed Terraform resource identifier supported by the AWS cost provider.
type ResourceKey string

const (
	ResourceInstance                     ResourceKey = "aws_instance"
	ResourceEBSVolume                    ResourceKey = "aws_ebs_volume"
	ResourceEIP                          ResourceKey = "aws_eip"
	ResourceNATGateway                   ResourceKey = "aws_nat_gateway"
	ResourceDBInstance                   ResourceKey = "aws_db_instance"
	ResourceRDSCluster                   ResourceKey = "aws_rds_cluster"
	ResourceRDSClusterInstance           ResourceKey = "aws_rds_cluster_instance"
	ResourceLoadBalancer                 ResourceKey = "aws_lb"
	ResourceApplicationLoadBalancerAlias ResourceKey = "aws_alb"
	ResourceClassicLoadBalancer          ResourceKey = "aws_elb"
	ResourceElastiCacheCluster           ResourceKey = "aws_elasticache_cluster"
	ResourceElastiCacheReplicationGroup  ResourceKey = "aws_elasticache_replication_group"
	ResourceElastiCacheServerlessCache   ResourceKey = "aws_elasticache_serverless_cache"
	ResourceEKSCluster                   ResourceKey = "aws_eks_cluster"
	ResourceEKSNodeGroup                 ResourceKey = "aws_eks_node_group"
	ResourceLambdaFunction               ResourceKey = "aws_lambda_function"
	ResourceDynamoDBTable                ResourceKey = "aws_dynamodb_table"
	ResourceSQSQueue                     ResourceKey = "aws_sqs_queue"
	ResourceSNSTopic                     ResourceKey = "aws_sns_topic"
	ResourceS3Bucket                     ResourceKey = "aws_s3_bucket"
	ResourceCloudWatchLogGroup           ResourceKey = "aws_cloudwatch_log_group"
	ResourceCloudWatchMetricAlarm        ResourceKey = "aws_cloudwatch_metric_alarm"
	ResourceSecretsManagerSecret         ResourceKey = "aws_secretsmanager_secret"
	ResourceKMSKey                       ResourceKey = "aws_kms_key"
	ResourceRoute53Zone                  ResourceKey = "aws_route53_zone"
)
