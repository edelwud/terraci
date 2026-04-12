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
		{Type: resourcedef.ResourceType(awskit.ResourceInstance), Definition: resourcespec.MustCompile(ec2.InstanceSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceEBSVolume), Definition: resourcespec.MustCompile(ec2.EBSSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceEIP), Definition: resourcespec.MustCompile(ec2.EIPSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceNATGateway), Definition: resourcespec.MustCompile(ec2.NATSpec(deps))},
	}
}

func rdsResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: resourcedef.ResourceType(awskit.ResourceDBInstance), Definition: resourcespec.MustCompile(rds.InstanceSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceRDSCluster), Definition: resourcespec.MustCompile(rds.ClusterSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceRDSClusterInstance), Definition: resourcespec.MustCompile(rds.ClusterInstanceSpec(deps))},
	}
}

func elbResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: resourcedef.ResourceType(awskit.ResourceLoadBalancer), Definition: resourcespec.MustCompile(elb.ALBSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceApplicationLoadBalancerAlias), Definition: resourcespec.MustCompile(elb.ALBSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceClassicLoadBalancer), Definition: resourcespec.MustCompile(elb.ClassicSpec(deps))},
	}
}

func elasticacheResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: resourcedef.ResourceType(awskit.ResourceElastiCacheCluster), Definition: resourcespec.MustCompile(elasticache.ClusterSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceElastiCacheReplicationGroup), Definition: resourcespec.MustCompile(elasticache.ReplicationGroupSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceElastiCacheServerlessCache), Definition: resourcespec.MustCompile(elasticache.ServerlessSpec(deps))},
	}
}

func eksResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: resourcedef.ResourceType(awskit.ResourceEKSCluster), Definition: resourcespec.MustCompile(eks.ClusterSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceEKSNodeGroup), Definition: resourcespec.MustCompile(eks.NodeGroupSpec(deps))},
	}
}

func serverlessResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: resourcedef.ResourceType(awskit.ResourceLambdaFunction), Definition: resourcespec.MustCompile(serverless.LambdaSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceDynamoDBTable), Definition: resourcespec.MustCompile(serverless.DynamoDBSpec(deps))},
		{Type: resourcedef.ResourceType(awskit.ResourceSQSQueue), Definition: resourcespec.MustCompile(serverless.SQSSpec())},
		{Type: resourcedef.ResourceType(awskit.ResourceSNSTopic), Definition: resourcespec.MustCompile(serverless.SNSSpec())},
	}
}

func storageResources() []cloud.ResourceRegistration {
	return []cloud.ResourceRegistration{
		{Type: resourcedef.ResourceType(awskit.ResourceS3Bucket), Definition: resourcespec.MustCompile(storage.S3Spec())},
		{Type: resourcedef.ResourceType(awskit.ResourceCloudWatchLogGroup), Definition: resourcespec.MustCompile(storage.LogGroupSpec())},
		{Type: resourcedef.ResourceType(awskit.ResourceCloudWatchMetricAlarm), Definition: resourcespec.MustCompile(storage.AlarmSpec())},
		{Type: resourcedef.ResourceType(awskit.ResourceSecretsManagerSecret), Definition: resourcespec.MustCompile(storage.SecretsManagerSpec())},
		{Type: resourcedef.ResourceType(awskit.ResourceKMSKey), Definition: resourcespec.MustCompile(storage.KMSSpec())},
		{Type: resourcedef.ResourceType(awskit.ResourceRoute53Zone), Definition: resourcespec.MustCompile(storage.Route53Spec())},
	}
}
