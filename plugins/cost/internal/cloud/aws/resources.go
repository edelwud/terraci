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
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
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
	Resources: awsResources(),
}

func awsResources() []cloud.ResourceRegistration {
	resources := make([]cloud.ResourceRegistration, 0, 24)
	resources = append(resources, ec2Resources()...)
	resources = append(resources, rdsResources()...)
	resources = append(resources, elbResources()...)
	resources = append(resources, elasticacheResources()...)
	resources = append(resources, eksResources()...)
	resources = append(resources, serverlessResources()...)
	resources = append(resources, storageResources()...)
	return resources
}

func ec2Resources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: handler.ResourceType(awskit.ResourceInstance), Handler: resourcespec.MustHandler(ec2.InstanceSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceEBSVolume), Handler: resourcespec.MustHandler(ec2.EBSSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceEIP), Handler: resourcespec.MustHandler(ec2.EIPSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceNATGateway), Handler: resourcespec.MustHandler(ec2.NATSpec(deps))},
	}
}

func rdsResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: handler.ResourceType(awskit.ResourceDBInstance), Handler: resourcespec.MustHandler(rds.InstanceSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceRDSCluster), Handler: resourcespec.MustHandler(rds.ClusterSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceRDSClusterInstance), Handler: resourcespec.MustHandler(rds.ClusterInstanceSpec(deps))},
	}
}

func elbResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: handler.ResourceType(awskit.ResourceLoadBalancer), Handler: resourcespec.MustHandler(elb.ALBSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceApplicationLoadBalancerAlias), Handler: resourcespec.MustHandler(elb.ALBSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceClassicLoadBalancer), Handler: resourcespec.MustHandler(elb.ClassicSpec(deps))},
	}
}

func elasticacheResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: handler.ResourceType(awskit.ResourceElastiCacheCluster), Handler: resourcespec.MustHandler(elasticache.ClusterSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceElastiCacheReplicationGroup), Handler: resourcespec.MustHandler(elasticache.ReplicationGroupSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceElastiCacheServerlessCache), Handler: resourcespec.MustHandler(elasticache.ServerlessSpec(deps))},
	}
}

func eksResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: handler.ResourceType(awskit.ResourceEKSCluster), Handler: resourcespec.MustHandler(eks.ClusterSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceEKSNodeGroup), Handler: resourcespec.MustHandler(eks.NodeGroupSpec(deps))},
	}
}

func serverlessResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: handler.ResourceType(awskit.ResourceLambdaFunction), Handler: resourcespec.MustHandler(serverless.LambdaSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceDynamoDBTable), Handler: resourcespec.MustHandler(serverless.DynamoDBSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceSQSQueue), Handler: resourcespec.MustHandler(serverless.SQSSpec())},
		{Type: handler.ResourceType(awskit.ResourceSNSTopic), Handler: resourcespec.MustHandler(serverless.SNSSpec())},
	}
}

func storageResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: handler.ResourceType(awskit.ResourceS3Bucket), Handler: resourcespec.MustHandler(storage.S3Spec())},
		{Type: handler.ResourceType(awskit.ResourceCloudWatchLogGroup), Handler: resourcespec.MustHandler(storage.LogGroupSpec())},
		{Type: handler.ResourceType(awskit.ResourceCloudWatchMetricAlarm), Handler: resourcespec.MustHandler(storage.AlarmSpec())},
		{Type: handler.ResourceType(awskit.ResourceSecretsManagerSecret), Handler: resourcespec.MustHandler(storage.SecretsManagerSpec())},
		{Type: handler.ResourceType(awskit.ResourceKMSKey), Handler: resourcespec.MustHandler(storage.KMSSpec())},
		{Type: handler.ResourceType(awskit.ResourceRoute53Zone), Handler: resourcespec.MustHandler(storage.Route53Spec())},
	}
}
