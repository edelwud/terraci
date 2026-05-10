package input

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
)

type Envelope struct {
	TerraCi TerraCiInput   `json:"terraci"`
	Plan    map[string]any `json:"plan"`
}

type TerraCiInput struct {
	Module ModuleInput `json:"module"`
	Policy PolicyInput `json:"policy"`
	Plan   PlanInput   `json:"plan"`
}

type ModuleInput struct {
	Path       string            `json:"path"`
	Components map[string]string `json:"components,omitempty"`
}

type PolicyInput struct {
	Namespaces []string `json:"namespaces"`
}

type PlanInput struct {
	Path string `json:"path"`
}

type Request struct {
	PlanJSONPath    string
	PlanDisplayPath string
	ModulePath      string
	Components      map[string]string
	Namespaces      []string
}

func Build(req Request) (*Envelope, error) {
	data, err := os.ReadFile(req.PlanJSONPath)
	if err != nil {
		return nil, fmt.Errorf("read plan JSON: %w", err)
	}

	var plan map[string]any
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("parse plan JSON: %w", err)
	}

	return &Envelope{
		TerraCi: TerraCiInput{
			Module: ModuleInput{
				Path:       req.ModulePath,
				Components: cloneMap(req.Components),
			},
			Policy: PolicyInput{
				Namespaces: append([]string(nil), req.Namespaces...),
			},
			Plan: PlanInput{Path: req.PlanDisplayPath},
		},
		Plan: plan,
	}, nil
}

func cloneMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	maps.Copy(out, in)
	return out
}
