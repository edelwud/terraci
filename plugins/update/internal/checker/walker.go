package checker

import (
	"context"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
)

type parsedModuleHandler func(ctx context.Context, mod *discovery.Module, parsed *parser.ParsedModule) error

type parseErrorHandler func(mod *discovery.Module, err error) error

// walkModules parses modules one by one and dispatches callbacks for parsed/error cases.
func walkModules(
	ctx context.Context,
	moduleParser *parser.Parser,
	modules []*discovery.Module,
	onParsed parsedModuleHandler,
	onParseError parseErrorHandler,
) error {
	for _, mod := range modules {
		if err := ctx.Err(); err != nil {
			return err
		}

		parsed, err := moduleParser.ParseModule(ctx, mod.Path)
		if err != nil {
			log.WithField("module", mod.RelativePath).WithError(err).Warn("failed to parse module")
			if onParseError == nil {
				continue
			}
			if handleErr := onParseError(mod, err); handleErr != nil {
				return handleErr
			}
			continue
		}

		if onParsed == nil {
			continue
		}
		if handleErr := onParsed(ctx, mod, parsed); handleErr != nil {
			return handleErr
		}
	}

	return nil
}
