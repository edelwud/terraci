// Package aws implements the AWS cloud provider for cost estimation.
// Registers itself via init() into the cloud provider registry.
package aws

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
)

func init() {
	cloud.Register(&provider{})
}

// provider implements cloud.Provider for Amazon Web Services.
type provider struct{}

func (p *provider) Definition() cloud.Definition { return definition }

var _ cloud.Provider = (*provider)(nil)
