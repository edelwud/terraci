package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/version"

	"github.com/edelwud/terraci/plugins/policy/internal/domain"
)

func OPAVersion() string {
	return version.Version
}

type Engine struct {
	policyDirs  []string
	namespaces  []string
	policyFiles []string
}

func New(policyDirs, namespaces []string) *Engine {
	return &Engine{
		policyDirs: append([]string(nil), policyDirs...),
		namespaces: append([]string(nil), namespaces...),
	}
}

func (e *Engine) Evaluate(ctx context.Context, input any) (*domain.Evaluation, error) {
	policyFiles, err := e.regoFiles()
	if err != nil {
		return nil, err
	}
	if len(policyFiles) == 0 {
		return &domain.Evaluation{}, nil
	}

	eval := &domain.Evaluation{}
	for _, namespace := range e.namespaces {
		denies, err := e.runQuery(ctx, input, policyFiles, fmt.Sprintf("data.%s.deny", namespace), namespace)
		if err != nil {
			return nil, fmt.Errorf("evaluate %s deny rules: %w", namespace, err)
		}
		warns, err := e.runQuery(ctx, input, policyFiles, fmt.Sprintf("data.%s.warn", namespace), namespace)
		if err != nil {
			return nil, fmt.Errorf("evaluate %s warn rules: %w", namespace, err)
		}
		eval.Denies = append(eval.Denies, denies...)
		eval.Warns = append(eval.Warns, warns...)
	}

	return eval, nil
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

	e.policyFiles = files
	return files, nil
}

func (e *Engine) runQuery(ctx context.Context, input any, policyFiles []string, query, namespace string) ([]domain.Finding, error) {
	r := rego.New(
		rego.Query(query),
		rego.Input(input),
		rego.Load(policyFiles, nil),
	)

	rs, err := r.Eval(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "undefined") {
			return nil, nil
		}
		return nil, err
	}

	var findings []domain.Finding
	for _, result := range rs {
		for _, expr := range result.Expressions {
			findings = append(findings, parseExpression(expr.Value, namespace)...)
		}
	}
	return findings, nil
}

func parseExpression(value any, namespace string) []domain.Finding {
	switch v := value.(type) {
	case []any:
		findings := make([]domain.Finding, 0, len(v))
		for _, item := range v {
			if finding, ok := parseFinding(item, namespace); ok {
				findings = append(findings, finding)
			}
		}
		return findings
	case map[string]any, string:
		if finding, ok := parseFinding(v, namespace); ok {
			return []domain.Finding{finding}
		}
	}
	return nil
}

func parseFinding(value any, namespace string) (domain.Finding, bool) {
	switch v := value.(type) {
	case string:
		return domain.Finding{Message: v, Namespace: namespace}, true
	case map[string]any:
		finding := domain.Finding{Namespace: namespace, Metadata: make(map[string]any)}
		if msg, ok := v["msg"].(string); ok {
			finding.Message = msg
		} else if msg, ok := v["message"].(string); ok {
			finding.Message = msg
		}
		for k, value := range v {
			if k != "msg" && k != "message" {
				finding.Metadata[k] = value
			}
		}
		if finding.Message == "" {
			data, err := json.Marshal(v)
			if err != nil {
				return domain.Finding{}, false
			}
			finding.Message = string(data)
		}
		return finding, true
	default:
		return domain.Finding{}, false
	}
}
