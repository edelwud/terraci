package input

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"

	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

type Envelope struct {
	terraci TerraCiContext
	plan    PlanDocument
}

// PlanDocument owns the decoded Terraform plan JSON used as OPA input.
type PlanDocument struct {
	value map[string]any
}

// TerraCiContext is the TerraCi-specific OPA input namespace.
type TerraCiContext struct {
	module ModuleContext
	policy PolicyContext
	plan   PlanContext
}

type ModuleContext struct {
	path       string
	components map[string]string
}

type PolicyContext struct {
	namespaces policyengine.Namespaces
}

type PlanContext struct {
	path string
}

type Request struct {
	PlanJSONPath    string
	PlanDisplayPath string
	ModulePath      string
	Components      map[string]string
	Namespaces      policyengine.Namespaces
}

func Build(req Request) (Envelope, error) {
	data, err := os.ReadFile(req.PlanJSONPath)
	if err != nil {
		return Envelope{}, fmt.Errorf("read plan JSON: %w", err)
	}

	plan, err := NewPlanDocument(data)
	if err != nil {
		return Envelope{}, err
	}

	return NewEnvelope(NewTerraCiContext(
		NewModuleContext(req.ModulePath, req.Components),
		NewPolicyContext(req.Namespaces),
		NewPlanContext(req.PlanDisplayPath),
	), plan), nil
}

func NewPlanDocument(data []byte) (PlanDocument, error) {
	var plan map[string]any
	if err := json.Unmarshal(data, &plan); err != nil {
		return PlanDocument{}, fmt.Errorf("parse plan JSON: %w", err)
	}
	return PlanDocument{value: cloneAnyMap(plan)}, nil
}

func NewEnvelope(terraci TerraCiContext, plan PlanDocument) Envelope {
	return Envelope{
		terraci: terraci,
		plan:    plan,
	}
}

func NewTerraCiContext(module ModuleContext, policy PolicyContext, plan PlanContext) TerraCiContext {
	return TerraCiContext{
		module: module,
		policy: policy,
		plan:   plan,
	}
}

func NewModuleContext(path string, components map[string]string) ModuleContext {
	return ModuleContext{
		path:       path,
		components: cloneStringMap(components),
	}
}

func NewPolicyContext(namespaces policyengine.Namespaces) PolicyContext {
	return PolicyContext{namespaces: namespaces.Clone()}
}

func NewPlanContext(path string) PlanContext {
	return PlanContext{path: path}
}

func (e Envelope) OPAValue() map[string]any {
	return map[string]any{
		"terraci": e.terraci.OPAValue(),
		"plan":    e.plan.OPAValue(),
	}
}

func (e Envelope) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.OPAValue())
}

func (p PlanDocument) OPAValue() map[string]any {
	return cloneAnyMap(p.value)
}

func (c TerraCiContext) OPAValue() map[string]any {
	return map[string]any{
		"module": c.module.OPAValue(),
		"policy": c.policy.OPAValue(),
		"plan":   c.plan.OPAValue(),
	}
}

func (c ModuleContext) OPAValue() map[string]any {
	value := map[string]any{
		"path": c.path,
	}
	if len(c.components) > 0 {
		components := make(map[string]any, len(c.components))
		for key, value := range c.components {
			components[key] = value
		}
		value["components"] = components
	}
	return value
}

func (c PolicyContext) OPAValue() map[string]any {
	return map[string]any{
		"namespaces": c.namespaces.Strings(),
	}
}

func (c PlanContext) OPAValue() map[string]any {
	return map[string]any{"path": c.path}
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	maps.Copy(out, in)
	return out
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneAnyValue(value)
	}
	return out
}

func cloneAnyValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneAnyMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = cloneAnyValue(item)
		}
		return out
	default:
		return typed
	}
}
