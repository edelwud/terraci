package resolve

import (
	"errors"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

func extractPathTemplate(expr hcl.Expression, ctx *hcl.EvalContext) ([]string, error) {
	val, _ := expr.Value(ctx) //nolint:errcheck
	if val.IsKnown() && val.Type() == cty.String {
		return []string{val.AsString()}, nil
	}

	rng := expr.Range()
	if rng.Filename != "" {
		content, err := os.ReadFile(rng.Filename)
		if err == nil {
			start, end := rng.Start.Byte, rng.End.Byte
			if end <= len(content) {
				return []string{string(content[start:end])}, nil
			}
		}
	}

	return nil, errors.New("could not extract path template")
}
