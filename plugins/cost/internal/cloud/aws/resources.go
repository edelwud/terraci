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

var providerRuntime = awskit.NewRuntime(awskit.Manifest)

// deps is a single shared RuntimeDeps instance for all AWS handlers.
var deps = awskit.NewRuntimeDeps(providerRuntime)

var definition = cloud.Definition{
	ConfigKey: awskit.ProviderID,
	Manifest:  providerRuntime.Manifest,
	FetcherFactory: func() pricing.PriceFetcher {
		return awskit.NewFetcher()
	},
	Resources: []cloud.ResourceRegistration{
		// EC2
		{Type: handler.ResourceType(awskit.ResourceInstance), Handler: &ec2.InstanceHandler{RuntimeDeps: deps}},
		{Type: handler.ResourceType(awskit.ResourceEBSVolume), Handler: ec2.NewEBSHandler(deps)},
		{Type: handler.ResourceType(awskit.ResourceEIP), Handler: ec2.NewEIPHandler(deps)},
		{Type: handler.ResourceType(awskit.ResourceNATGateway), Handler: &ec2.NATHandler{RuntimeDeps: deps}},
		// RDS
		{Type: handler.ResourceType(awskit.ResourceDBInstance), Handler: &rds.InstanceHandler{RuntimeDeps: deps}},
		{Type: handler.ResourceType(awskit.ResourceRDSCluster), Handler: &rds.ClusterHandler{RuntimeDeps: deps}},
		{Type: handler.ResourceType(awskit.ResourceRDSClusterInstance), Handler: &rds.ClusterInstanceHandler{RuntimeDeps: deps}},
		// ELB
		{Type: handler.ResourceType(awskit.ResourceLoadBalancer), Handler: &elb.ALBHandler{RuntimeDeps: deps}},
		{Type: handler.ResourceType(awskit.ResourceApplicationLoadBalancerAlias), Handler: &elb.ALBHandler{RuntimeDeps: deps}},
		{Type: handler.ResourceType(awskit.ResourceClassicLoadBalancer), Handler: &elb.ClassicHandler{RuntimeDeps: deps}},
		// ElastiCache
		{Type: handler.ResourceType(awskit.ResourceElastiCacheCluster), Handler: &elasticache.ClusterHandler{RuntimeDeps: deps}},
		{Type: handler.ResourceType(awskit.ResourceElastiCacheReplicationGroup), Handler: &elasticache.ReplicationGroupHandler{RuntimeDeps: deps}},
		{Type: handler.ResourceType(awskit.ResourceElastiCacheServerlessCache), Handler: &elasticache.ServerlessHandler{RuntimeDeps: deps}},
		// EKS
		{Type: handler.ResourceType(awskit.ResourceEKSCluster), Handler: &eks.ClusterHandler{RuntimeDeps: deps}},
		{Type: handler.ResourceType(awskit.ResourceEKSNodeGroup), Handler: &eks.NodeGroupHandler{RuntimeDeps: deps}},
		// Serverless
		{Type: handler.ResourceType(awskit.ResourceLambdaFunction), Handler: &serverless.LambdaHandler{RuntimeDeps: deps}},
		{Type: handler.ResourceType(awskit.ResourceDynamoDBTable), Handler: &serverless.DynamoDBHandler{RuntimeDeps: deps}},
		{Type: handler.ResourceType(awskit.ResourceSQSQueue), Handler: &serverless.SQSHandler{}},
		{Type: handler.ResourceType(awskit.ResourceSNSTopic), Handler: &serverless.SNSHandler{}},
		// Storage & misc
		{Type: handler.ResourceType(awskit.ResourceS3Bucket), Handler: &storage.S3Handler{}},
		{Type: handler.ResourceType(awskit.ResourceCloudWatchLogGroup), Handler: &storage.LogGroupHandler{}},
		{Type: handler.ResourceType(awskit.ResourceCloudWatchMetricAlarm), Handler: &storage.AlarmHandler{}},
		{Type: handler.ResourceType(awskit.ResourceSecretsManagerSecret), Handler: &storage.SecretsManagerHandler{}},
		{Type: handler.ResourceType(awskit.ResourceKMSKey), Handler: &storage.KMSHandler{}},
		{Type: handler.ResourceType(awskit.ResourceRoute53Zone), Handler: &storage.Route53Handler{}},
	},
}
