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
		{Type: handler.ResourceType(awskit.ResourceInstance), Definition: resourcespec.MustCompile(ec2.InstanceSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceEBSVolume), Definition: resourcespec.MustCompile(ec2.EBSSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceEIP), Definition: resourcespec.MustCompile(ec2.EIPSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceNATGateway), Definition: resourcespec.MustCompile(ec2.NATSpec(deps))},
	}
}

func rdsResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: handler.ResourceType(awskit.ResourceDBInstance), Definition: resourcespec.MustCompile(rds.InstanceSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceRDSCluster), Definition: resourcespec.MustCompile(rds.ClusterSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceRDSClusterInstance), Definition: resourcespec.MustCompile(rds.ClusterInstanceSpec(deps))},
	}
}

func elbResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: handler.ResourceType(awskit.ResourceLoadBalancer), Definition: resourcespec.MustCompile(elb.ALBSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceApplicationLoadBalancerAlias), Definition: resourcespec.MustCompile(elb.ALBSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceClassicLoadBalancer), Definition: resourcespec.MustCompile(elb.ClassicSpec(deps))},
	}
}

func elasticacheResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: handler.ResourceType(awskit.ResourceElastiCacheCluster), Definition: resourcespec.MustCompile(elasticache.ClusterSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceElastiCacheReplicationGroup), Definition: resourcespec.MustCompile(elasticache.ReplicationGroupSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceElastiCacheServerlessCache), Definition: resourcespec.MustCompile(elasticache.ServerlessSpec(deps))},
	}
}

func eksResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: handler.ResourceType(awskit.ResourceEKSCluster), Definition: resourcespec.MustCompile(eks.ClusterSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceEKSNodeGroup), Definition: resourcespec.MustCompile(eks.NodeGroupSpec(deps))},
	}
}

func serverlessResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: handler.ResourceType(awskit.ResourceLambdaFunction), Definition: resourcespec.MustCompile(serverless.LambdaSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceDynamoDBTable), Definition: resourcespec.MustCompile(serverless.DynamoDBSpec(deps))},
		{Type: handler.ResourceType(awskit.ResourceSQSQueue), Definition: resourcespec.MustCompile(serverless.SQSSpec())},
		{Type: handler.ResourceType(awskit.ResourceSNSTopic), Definition: resourcespec.MustCompile(serverless.SNSSpec())},
	}
}

func storageResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: handler.ResourceType(awskit.ResourceS3Bucket), Definition: resourcespec.MustCompile(storage.S3Spec())},
		{Type: handler.ResourceType(awskit.ResourceCloudWatchLogGroup), Definition: resourcespec.MustCompile(storage.LogGroupSpec())},
		{Type: handler.ResourceType(awskit.ResourceCloudWatchMetricAlarm), Definition: resourcespec.MustCompile(storage.AlarmSpec())},
		{Type: handler.ResourceType(awskit.ResourceSecretsManagerSecret), Definition: resourcespec.MustCompile(storage.SecretsManagerSpec())},
		{Type: handler.ResourceType(awskit.ResourceKMSKey), Definition: resourcespec.MustCompile(storage.KMSSpec())},
		{Type: handler.ResourceType(awskit.ResourceRoute53Zone), Definition: resourcespec.MustCompile(storage.Route53Spec())},
	}
}
