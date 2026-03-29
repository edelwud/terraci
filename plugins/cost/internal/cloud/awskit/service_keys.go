package awskit

import "github.com/edelwud/terraci/plugins/cost/internal/pricing"

// ServiceKey is a typed key into the AWS provider service catalog.
type ServiceKey string

const (
	ServiceKeyEC2            ServiceKey = "ec2"
	ServiceKeyRDS            ServiceKey = "rds"
	ServiceKeyS3             ServiceKey = "s3"
	ServiceKeyEBS            ServiceKey = "ebs"
	ServiceKeyELB            ServiceKey = "elb"
	ServiceKeyELBv2          ServiceKey = "elbv2"
	ServiceKeyLambda         ServiceKey = "lambda"
	ServiceKeyDynamoDB       ServiceKey = "dynamodb"
	ServiceKeyCloudWatch     ServiceKey = "cloudwatch"
	ServiceKeySNS            ServiceKey = "sns"
	ServiceKeySQS            ServiceKey = "sqs"
	ServiceKeyElastiCache    ServiceKey = "elasticache"
	ServiceKeyEKS            ServiceKey = "eks"
	ServiceKeyECS            ServiceKey = "ecs"
	ServiceKeySecretsManager ServiceKey = "secretsmanager"
	ServiceKeyKMS            ServiceKey = "kms"
	ServiceKeyRoute53        ServiceKey = "route53"
	ServiceKeyCloudFront     ServiceKey = "cloudfront"
	ServiceKeyNAT            ServiceKey = "nat"
	ServiceKeyVPC            ServiceKey = "vpc"
)

// Service resolves a typed catalog key into a provider service id.
func Service(key ServiceKey) (pricing.ServiceID, bool) {
	return DefaultRuntime.Manifest.Service(string(key))
}

// MustService resolves a typed catalog key or panics if the service is not registered.
func MustService(key ServiceKey) pricing.ServiceID {
	return DefaultRuntime.MustService(key)
}
