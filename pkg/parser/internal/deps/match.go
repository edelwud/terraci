package deps

import (
	"path/filepath"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
)

func MatchPathToModule(index *discovery.ModuleIndex, statePath string, from *discovery.Module) *discovery.Module {
	normalized := NormalizeStatePath(statePath)
	parts := strings.Split(normalized, "/")

	strategies := []func() *discovery.Module{
		func() *discovery.Module { return index.ByID(normalized) },
		func() *discovery.Module {
			return index.ByID(strings.ReplaceAll(normalized, "/", string(filepath.Separator)))
		},
		func() *discovery.Module { return tryTrailingMatch(index, parts, 5) },
		func() *discovery.Module { return tryTrailingMatch(index, parts, 4) },
		func() *discovery.Module { return tryContextMatch(index, parts, from) },
	}

	for _, strategy := range strategies {
		if module := strategy(); module != nil {
			return module
		}
	}

	return nil
}

func NormalizeStatePath(path string) string {
	path = strings.TrimSuffix(path, "/terraform.tfstate")
	path = strings.TrimSuffix(path, ".tfstate")
	path = strings.TrimPrefix(path, "env:/")
	return path
}

func ContainsDynamicPattern(path string) bool {
	return strings.Contains(path, "${lookup(") ||
		strings.Contains(path, "${each.") ||
		strings.Contains(path, "${var.") ||
		strings.Contains(path, "\"}")
}

func BackendIndexKey(backendType, bucket, stateKey, modulePath string) string {
	if bucket == "" {
		return ""
	}
	if stateKey != "" {
		stateKey = NormalizeStatePath(stateKey)
	} else {
		stateKey = modulePath
	}
	return backendType + ":" + bucket + ":" + stateKey
}

func MatchByBackend(
	backendIndex map[string]*discovery.Module,
	backendType, bucket, statePath string,
) *discovery.Module {
	if len(backendIndex) == 0 || backendType == "" || bucket == "" {
		return nil
	}

	key := backendType + ":" + bucket + ":" + NormalizeStatePath(statePath)
	return backendIndex[key]
}

func tryTrailingMatch(index *discovery.ModuleIndex, parts []string, n int) *discovery.Module {
	if len(parts) < n {
		return nil
	}
	return index.ByID(strings.Join(parts[len(parts)-n:], "/"))
}

func tryContextMatch(index *discovery.ModuleIndex, parts []string, from *discovery.Module) *discovery.Module {
	prefix := from.ContextPrefix()

	if len(parts) == 1 {
		if module := index.ByID(prefix + "/" + parts[0]); module != nil {
			return module
		}
		if from.IsSubmodule() {
			if module := index.ByID(prefix + "/" + from.LeafValue() + "/" + parts[0]); module != nil {
				return module
			}
		}
	}

	if len(parts) == 2 {
		return index.ByID(prefix + "/" + parts[0] + "/" + parts[1])
	}

	return nil
}
