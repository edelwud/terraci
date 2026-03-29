// Package aws implements the AWS cloud provider for cost estimation.
// Registers itself via init() into the cloud provider registry.
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
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

func init() {
	cloud.Register(&provider{})
}

// provider implements cloud.Provider for Amazon Web Services.
type provider struct{}

func (p *provider) Name() string { return "aws" }

func (p *provider) NewFetcher() pricing.PriceFetcher {
	return awskit.NewFetcher()
}

func (p *provider) InitRegionMapping() {
	awskit.InitRegionMapping()
}

// RegisterHandlers populates the registry with all AWS resource handlers.
func (p *provider) RegisterHandlers(r *handler.Registry) {
	// EC2
	r.Register("aws_instance", &ec2.InstanceHandler{})
	r.Register("aws_ebs_volume", &ec2.EBSHandler{})
	r.Register("aws_eip", &ec2.EIPHandler{})
	r.Register("aws_nat_gateway", &ec2.NATHandler{})

	// RDS
	r.Register("aws_db_instance", &rds.InstanceHandler{})
	r.Register("aws_rds_cluster", &rds.ClusterHandler{})
	r.Register("aws_rds_cluster_instance", &rds.ClusterInstanceHandler{})

	// ELB
	r.Register("aws_lb", &elb.ALBHandler{})
	r.Register("aws_alb", &elb.ALBHandler{})
	r.Register("aws_elb", &elb.ClassicHandler{})

	// ElastiCache
	r.Register("aws_elasticache_cluster", &elasticache.ClusterHandler{})
	r.Register("aws_elasticache_replication_group", &elasticache.ReplicationGroupHandler{})
	r.Register("aws_elasticache_serverless_cache", &elasticache.ServerlessHandler{})

	// EKS
	r.Register("aws_eks_cluster", &eks.ClusterHandler{})
	r.Register("aws_eks_node_group", &eks.NodeGroupHandler{})

	// Serverless
	r.Register("aws_lambda_function", &serverless.LambdaHandler{})
	r.Register("aws_dynamodb_table", &serverless.DynamoDBHandler{})
	r.Register("aws_sqs_queue", &serverless.SQSHandler{})
	r.Register("aws_sns_topic", &serverless.SNSHandler{})

	// Storage & misc
	r.Register("aws_s3_bucket", &storage.S3Handler{})
	r.Register("aws_cloudwatch_log_group", &storage.LogGroupHandler{})
	r.Register("aws_cloudwatch_metric_alarm", &storage.AlarmHandler{})
	r.Register("aws_secretsmanager_secret", &storage.SecretsManagerHandler{})
	r.Register("aws_kms_key", &storage.KMSHandler{})
	r.Register("aws_route53_zone", &storage.Route53Handler{})
}
