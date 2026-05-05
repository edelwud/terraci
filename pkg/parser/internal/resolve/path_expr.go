package resolve

import "github.com/hashicorp/hcl/v2"

// Backend config attributes that hold the workspace path expression we
// resolve for terraform_remote_state references. Centralized so foreach
// path lookup and tests cannot drift on a typo.
const (
	configKeyKey    = "key"
	configKeyPrefix = "prefix"
	configKeyValue  = "value"
)

func findPathExpression(ref *Ref) hcl.Expression {
	if expr, ok := ref.Config[configKeyKey]; ok {
		return expr
	}
	if expr, ok := ref.Config[configKeyPrefix]; ok {
		return expr
	}
	return nil
}
