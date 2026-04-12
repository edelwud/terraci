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
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
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
		{Type: resourcedef.ResourceType(awskit.ResourceInstance), Definition: resourcespec.MustCompileTyped(ec2.InstanceSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceEBSVolume), Definition: resourcespec.MustCompileTyped(ec2.EBSSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceEIP), Definition: resourcespec.MustCompileTyped(ec2.EIPSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceNATGateway), Definition: resourcespec.MustCompileTyped(ec2.NATSpec(deps))},
	}
}

func rdsResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: resourcedef.ResourceType(awskit.ResourceDBInstance), Definition: resourcespec.MustCompileTyped(rds.InstanceSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceRDSCluster), Definition: resourcespec.MustCompileTyped(rds.ClusterSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceRDSClusterInstance), Definition: resourcespec.MustCompileTyped(rds.ClusterInstanceSpec(deps))},
	}
}

func elbResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: resourcedef.ResourceType(awskit.ResourceLoadBalancer), Definition: resourcespec.MustCompileTyped(elb.ALBSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceApplicationLoadBalancerAlias), Definition: resourcespec.MustCompileTyped(elb.ALBSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceClassicLoadBalancer), Definition: resourcespec.MustCompileTyped(elb.ClassicSpec(deps))},
	}
}

func elasticacheResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: resourcedef.ResourceType(awskit.ResourceElastiCacheCluster), Definition: resourcespec.MustCompileTyped(elasticache.ClusterSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceElastiCacheReplicationGroup), Definition: resourcespec.MustCompileTyped(elasticache.ReplicationGroupSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceElastiCacheServerlessCache), Definition: resourcespec.MustCompileTyped(elasticache.ServerlessSpec(deps))},
	}
}

func eksResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: resourcedef.ResourceType(awskit.ResourceEKSCluster), Definition: resourcespec.MustCompileTyped(eks.ClusterSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceEKSNodeGroup), Definition: resourcespec.MustCompileTyped(eks.NodeGroupSpec(deps))},
	}
}

func serverlessResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: resourcedef.ResourceType(awskit.ResourceLambdaFunction), Definition: resourcespec.MustCompileTyped(serverless.LambdaSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceDynamoDBTable), Definition: resourcespec.MustCompileTyped(serverless.DynamoDBSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceSQSQueue), Definition: resourcespec.MustCompileTyped(serverless.SQSSpec())},
		{Type: resourcedef.ResourceType(awskit.ResourceSNSTopic), Definition: resourcespec.MustCompileTyped(serverless.SNSSpec())},
	}
}

func storageResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: resourcedef.ResourceType(awskit.ResourceS3Bucket), Definition: resourcespec.MustCompileTyped(storage.S3Spec())},
		{Type: resourcedef.ResourceType(awskit.ResourceCloudWatchLogGroup), Definition: resourcespec.MustCompileTyped(storage.LogGroupSpec())},
		{Type: resourcedef.ResourceType(awskit.ResourceCloudWatchMetricAlarm), Definition: resourcespec.MustCompileTyped(storage.AlarmSpec())},
		{Type: resourcedef.ResourceType(awskit.ResourceSecretsManagerSecret), Definition: resourcespec.MustCompileTyped(storage.SecretsManagerSpec())},
		{Type: resourcedef.ResourceType(awskit.ResourceKMSKey), Definition: resourcespec.MustCompileTyped(storage.KMSSpec())},
		{Type: resourcedef.ResourceType(awskit.ResourceRoute53Zone), Definition: resourcespec.MustCompileTyped(storage.Route53Spec())},
	}
}
