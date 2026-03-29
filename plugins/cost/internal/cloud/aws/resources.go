package aws

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/aws/ec2"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/aws/eks"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/aws/elasticache"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/aws/elb"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/aws/rds"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/aws/serverless"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/aws/storage"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

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

var Definition = cloud.Definition{
	Manifest: awskit.Manifest,
	FetcherFactory: func() pricing.PriceFetcher {
		return awskit.NewFetcher()
	},
	Resources: []cloud.ResourceRegistration{
		// EC2
		{Type: string(ResourceInstance), Handler: &ec2.InstanceHandler{}},
		{Type: string(ResourceEBSVolume), Handler: &ec2.EBSHandler{}},
		{Type: string(ResourceEIP), Handler: &ec2.EIPHandler{}},
		{Type: string(ResourceNATGateway), Handler: &ec2.NATHandler{}},
		// RDS
		{Type: string(ResourceDBInstance), Handler: &rds.InstanceHandler{}},
		{Type: string(ResourceRDSCluster), Handler: &rds.ClusterHandler{}},
		{Type: string(ResourceRDSClusterInstance), Handler: &rds.ClusterInstanceHandler{}},
		// ELB
		{Type: string(ResourceLoadBalancer), Handler: &elb.ALBHandler{}},
		{Type: string(ResourceApplicationLoadBalancerAlias), Handler: &elb.ALBHandler{}},
		{Type: string(ResourceClassicLoadBalancer), Handler: &elb.ClassicHandler{}},
		// ElastiCache
		{Type: string(ResourceElastiCacheCluster), Handler: &elasticache.ClusterHandler{}},
		{Type: string(ResourceElastiCacheReplicationGroup), Handler: &elasticache.ReplicationGroupHandler{}},
		{Type: string(ResourceElastiCacheServerlessCache), Handler: &elasticache.ServerlessHandler{}},
		// EKS
		{Type: string(ResourceEKSCluster), Handler: &eks.ClusterHandler{}},
		{Type: string(ResourceEKSNodeGroup), Handler: &eks.NodeGroupHandler{}},
		// Serverless
		{Type: string(ResourceLambdaFunction), Handler: &serverless.LambdaHandler{}},
		{Type: string(ResourceDynamoDBTable), Handler: &serverless.DynamoDBHandler{}},
		{Type: string(ResourceSQSQueue), Handler: &serverless.SQSHandler{}},
		{Type: string(ResourceSNSTopic), Handler: &serverless.SNSHandler{}},
		// Storage & misc
		{Type: string(ResourceS3Bucket), Handler: &storage.S3Handler{}},
		{Type: string(ResourceCloudWatchLogGroup), Handler: &storage.LogGroupHandler{}},
		{Type: string(ResourceCloudWatchMetricAlarm), Handler: &storage.AlarmHandler{}},
		{Type: string(ResourceSecretsManagerSecret), Handler: &storage.SecretsManagerHandler{}},
		{Type: string(ResourceKMSKey), Handler: &storage.KMSHandler{}},
		{Type: string(ResourceRoute53Zone), Handler: &storage.Route53Handler{}},
	},
}
