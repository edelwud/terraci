package resourcespec

import (
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
)

// NewHandler compiles a typed resource spec into a runtime handler.
func NewHandler(spec ResourceSpec) (handler.ResourceHandler, error) {
	def, err := Compile(spec)
	if err != nil {
		return nil, err
	}
	return resourcedef.NewLegacyHandler(def)
}

// MustHandler compiles a resource spec and panics on invalid configuration.
func MustHandler(spec ResourceSpec) handler.ResourceHandler {
	return resourcedef.MustLegacyHandler(MustCompile(spec))
}
