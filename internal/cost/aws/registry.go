// Package aws provides AWS resource cost estimation handlers
package aws

import (
	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

// ResourceHandler extracts pricing information from terraform resource attributes
type ResourceHandler interface {
	// ServiceCode returns the AWS service code for pricing API
	ServiceCode() pricing.ServiceCode
	// BuildLookup creates a PriceLookup from terraform resource attributes
	BuildLookup(region string, attrs map[string]interface{}) (*pricing.PriceLookup, error)
	// CalculateCost calculates monthly cost from price and resource attributes
	CalculateCost(price *pricing.Price, attrs map[string]interface{}) (hourly, monthly float64)
}

// Registry maps terraform resource types to handlers
type Registry struct {
	handlers map[string]ResourceHandler
}

// NewRegistry creates a new resource registry with all supported handlers
func NewRegistry() *Registry {
	r := &Registry{
		handlers: make(map[string]ResourceHandler),
	}
	r.registerAll()
	return r
}

// registerAll registers all supported resource handlers
func (r *Registry) registerAll() {
	// EC2
	r.Register("aws_instance", &EC2InstanceHandler{})
	r.Register("aws_ebs_volume", &EBSVolumeHandler{})
	r.Register("aws_eip", &EIPHandler{})
	r.Register("aws_nat_gateway", &NATGatewayHandler{})

	// RDS
	r.Register("aws_db_instance", &RDSInstanceHandler{})
	r.Register("aws_rds_cluster", &RDSClusterHandler{})
	r.Register("aws_rds_cluster_instance", &RDSClusterInstanceHandler{})

	// ELB
	r.Register("aws_lb", &LBHandler{})
	r.Register("aws_alb", &LBHandler{}) // alias
	r.Register("aws_elb", &ClassicLBHandler{})

	// ElastiCache
	r.Register("aws_elasticache_cluster", &ElastiCacheClusterHandler{})
	r.Register("aws_elasticache_replication_group", &ElastiCacheReplicationGroupHandler{})

	// EKS
	r.Register("aws_eks_cluster", &EKSClusterHandler{})
	r.Register("aws_eks_node_group", &EKSNodeGroupHandler{})

	// Lambda
	r.Register("aws_lambda_function", &LambdaHandler{})

	// DynamoDB
	r.Register("aws_dynamodb_table", &DynamoDBTableHandler{})

	// S3 - storage only, no request pricing
	r.Register("aws_s3_bucket", &S3BucketHandler{})

	// CloudWatch
	r.Register("aws_cloudwatch_log_group", &CloudWatchLogGroupHandler{})
	r.Register("aws_cloudwatch_metric_alarm", &CloudWatchAlarmHandler{})

	// Secrets Manager
	r.Register("aws_secretsmanager_secret", &SecretsManagerHandler{})

	// KMS
	r.Register("aws_kms_key", &KMSKeyHandler{})

	// Route53
	r.Register("aws_route53_zone", &Route53ZoneHandler{})

	// SQS
	r.Register("aws_sqs_queue", &SQSQueueHandler{})

	// SNS
	r.Register("aws_sns_topic", &SNSTopicHandler{})
}

// Register adds a handler for a resource type
func (r *Registry) Register(resourceType string, handler ResourceHandler) {
	r.handlers[resourceType] = handler
}

// GetHandler returns a handler for a resource type
func (r *Registry) GetHandler(resourceType string) (ResourceHandler, bool) {
	h, ok := r.handlers[resourceType]
	return h, ok
}

// IsSupported checks if a resource type is supported
func (r *Registry) IsSupported(resourceType string) bool {
	_, ok := r.handlers[resourceType]
	return ok
}

// SupportedTypes returns all supported resource types
func (r *Registry) SupportedTypes() []string {
	types := make([]string, 0, len(r.handlers))
	for t := range r.handlers {
		types = append(types, t)
	}
	return types
}

// RequiredServices returns services needed for given resource types
func (r *Registry) RequiredServices(resourceTypes []string) map[pricing.ServiceCode]bool {
	services := make(map[pricing.ServiceCode]bool)
	for _, rt := range resourceTypes {
		if h, ok := r.handlers[rt]; ok {
			services[h.ServiceCode()] = true
		}
	}
	return services
}

// LogUnsupported logs unsupported resource types at debug level
func LogUnsupported(resourceType, address string) {
	log.WithField("type", resourceType).
		WithField("address", address).
		Debug("resource type not supported for cost estimation")
}
