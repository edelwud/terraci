package runner

import (
	"maps"
	"os"
	"sort"
	"strings"
)

func mergeEnv(values ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, env := range values {
		maps.Copy(result, env)
	}
	return result
}

func environMap() map[string]string {
	result := make(map[string]string)
	for _, entry := range os.Environ() {
		if k, v, ok := strings.Cut(entry, "="); ok {
			result[k] = v
		}
	}
	return result
}

func envMapToList(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+env[key])
	}
	return result
}

func rewriteTerraciCommand(command, selfPath string) string {
	if selfPath == "" {
		return command
	}
	if command == "terraci" {
		return selfPath
	}
	const prefix = "terraci "
	if len(command) > len(prefix) && command[:len(prefix)] == prefix {
		return selfPath + command[len("terraci"):]
	}
	return command
}
