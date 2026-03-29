package awskit

import (
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

const (
	// ProviderID is the AWS provider id used by the cost engine.
	ProviderID = "aws"
)

var (
	Manifest = pricing.ProviderManifest{
		ID:          ProviderID,
		DisplayName: "Amazon Web Services",
		PriceSource: "aws-bulk-api",
		Services: pricing.ServiceCatalog{
			"ec2":            {Provider: ProviderID, Name: "AmazonEC2"},
			"rds":            {Provider: ProviderID, Name: "AmazonRDS"},
			"s3":             {Provider: ProviderID, Name: "AmazonS3"},
			"ebs":            {Provider: ProviderID, Name: "AmazonEC2"}, // EBS is part of EC2 pricing.
			"elb":            {Provider: ProviderID, Name: "AWSELB"},
			"elbv2":          {Provider: ProviderID, Name: "AmazonEC2"}, // ALB/NLB pricing in EC2.
			"lambda":         {Provider: ProviderID, Name: "AWSLambda"},
			"dynamodb":       {Provider: ProviderID, Name: "AmazonDynamoDB"},
			"cloudwatch":     {Provider: ProviderID, Name: "AmazonCloudWatch"},
			"sns":            {Provider: ProviderID, Name: "AmazonSNS"},
			"sqs":            {Provider: ProviderID, Name: "AWSQueueService"},
			"elasticache":    {Provider: ProviderID, Name: "AmazonElastiCache"},
			"eks":            {Provider: ProviderID, Name: "AmazonEKS"},
			"ecs":            {Provider: ProviderID, Name: "AmazonECS"},
			"secretsmanager": {Provider: ProviderID, Name: "AWSSecretsManager"},
			"kms":            {Provider: ProviderID, Name: "awskms"},
			"route53":        {Provider: ProviderID, Name: "AmazonRoute53"},
			"cloudfront":     {Provider: ProviderID, Name: "AmazonCloudFront"},
			"nat":            {Provider: ProviderID, Name: "AmazonEC2"}, // NAT Gateway in EC2.
			"vpc":            {Provider: ProviderID, Name: "AmazonVPC"},
		},
		Regions: pricing.RegionResolver{
			LocationNames:      awsRegionMapping,
			UsagePrefixes:      awsRegionUsagePrefix,
			DefaultUsagePrefix: DefaultUsagePrefix,
		},
	}
)
