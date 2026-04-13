package storage

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

const (
	// SecretsManagerSecretCost is the monthly cost per secret.
	SecretsManagerSecretCost = 0.40
)

// SecretsManagerSpec declares aws_secretsmanager_secret cost estimation.
func SecretsManagerSpec() resourcespec.TypedSpec[resourcespec.NoAttrs] {
	return resourcespec.FixedMonthlyNoAttrsSpec(resourcedef.ResourceType(awskit.ResourceSecretsManagerSecret), SecretsManagerSecretCost)
}
