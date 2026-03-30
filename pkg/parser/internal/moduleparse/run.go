package moduleparse

import (
	"context"

	"github.com/edelwud/terraci/pkg/parser/internal/model"
)

func Run(ctx context.Context, modulePath string, segments []string) (*model.ParsedModule, error) {
	return newRunner(modulePath, segments).Run(ctx)
}
