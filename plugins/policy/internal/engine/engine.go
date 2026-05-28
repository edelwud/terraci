package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/version"

	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
	policyinput "github.com/edelwud/terraci/plugins/policy/internal/input"
)

func OPAVersion() string {
	return version.Version
}

type Engine struct {
	policyDirs  []string
	policyFiles []string
}

func New(policyDirs []string) *Engine {
	return &Engine{
		policyDirs: append([]string(nil), policyDirs...),
	}
}

func (e *Engine) Evaluate(ctx context.Context, input policyinput.Envelope, namespaces policyengine.Namespaces) (*policyengine.Evaluation, error) {
	policyFiles, err := e.regoFiles()
	if err != nil {
		return nil, err
	}
	if len(policyFiles) == 0 {
		return policyengine.EmptyEvaluation(), nil
	}

	var allDenies []policyengine.Finding
	var allWarns []policyengine.Finding
	for _, namespace := range namespaces {
		denies, err := e.runQuery(ctx, input, policyFiles, fmt.Sprintf("data.%s.deny", namespace), namespace)
		if err != nil {
			return nil, fmt.Errorf("evaluate %s deny rules: %w", namespace, err)
		}
		warns, err := e.runQuery(ctx, input, policyFiles, fmt.Sprintf("data.%s.warn", namespace), namespace)
		if err != nil {
			return nil, fmt.Errorf("evaluate %s warn rules: %w", namespace, err)
		}
		allDenies = append(allDenies, denies...)
		allWarns = append(allWarns, warns...)
	}

	sortFindings(allDenies)
	sortFindings(allWarns)
	return policyengine.NewEvaluation(allDenies, allWarns), nil
}

func (e *Engine) regoFiles() ([]string, error) {
	if e.policyFiles != nil {
		return e.policyFiles, nil
	}

	var files []string
	for _, dir := range e.policyDirs {
		err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".rego") || strings.HasSuffix(path, "_test.rego") {
				return nil
			}
			files = append(files, path)
			return nil
		})
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("collect rego files from %s: %w", dir, err)
		}
	}

	sort.Strings(files)
	e.policyFiles = files
	return files, nil
}

func (e *Engine) runQuery(ctx context.Context, input policyinput.Envelope, policyFiles []string, query string, namespace policyengine.Namespace) ([]policyengine.Finding, error) {
	r := rego.New(
		rego.Query(query),
		rego.Input(input.OPAValue()),
		rego.Load(policyFiles, nil),
	)

	rs, err := r.Eval(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "undefined") {
			return nil, nil
		}
		return nil, err
	}

	var findings []policyengine.Finding
	for _, result := range rs {
		for _, expr := range result.Expressions {
			findings = append(findings, parseExpression(expr.Value, namespace)...)
		}
	}
	return findings, nil
}

func parseExpression(value any, namespace policyengine.Namespace) []policyengine.Finding {
	switch v := value.(type) {
	case []any:
		findings := make([]policyengine.Finding, 0, len(v))
		for _, item := range v {
			if finding, ok := parseFinding(item, namespace); ok {
				findings = append(findings, finding)
			}
		}
		return findings
	case map[string]any, string:
		if finding, ok := parseFinding(v, namespace); ok {
			return []policyengine.Finding{finding}
		}
	}
	return nil
}

func parseFinding(value any, namespace policyengine.Namespace) (policyengine.Finding, bool) {
	switch v := value.(type) {
	case string:
		return policyengine.NewFinding(namespace, v, policyengine.FindingMetadata{}), true
	case map[string]any:
		metadata := make(map[string]any)
		message := ""
		if msg, ok := v["msg"].(string); ok {
			message = msg
		} else if msg, ok := v["message"].(string); ok {
			message = msg
		}
		for k, value := range v {
			if k != "msg" && k != "message" {
				metadata[k] = value
			}
		}
		if message == "" {
			data, err := json.Marshal(v)
			if err != nil {
				return policyengine.Finding{}, false
			}
			message = string(data)
		}
		return policyengine.NewFinding(namespace, message, policyengine.NewFindingMetadata(metadata)), true
	default:
		return policyengine.Finding{}, false
	}
}

func sortFindings(findings []policyengine.Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Namespace == findings[j].Namespace {
			return findings[i].Message < findings[j].Message
		}
		return findings[i].Namespace < findings[j].Namespace
	})
}
