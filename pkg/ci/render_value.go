package ci

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
)

// RenderValueKind identifies one semantic value carried by rendered reports.
type RenderValueKind string

const (
	RenderValueKindText            RenderValueKind = "text"
	RenderValueKindCode            RenderValueKind = "code"
	RenderValueKindStatus          RenderValueKind = "status"
	RenderValueKindLabel           RenderValueKind = "label"
	RenderValueKindMoney           RenderValueKind = "money"
	RenderValueKindMoneyDelta      RenderValueKind = "money_delta"
	RenderValueKindModulePath      RenderValueKind = "module_path"
	RenderValueKindResourceAddress RenderValueKind = "resource_address"
	RenderValueKindInline          RenderValueKind = "inline"
)

// RenderTone describes presentation intent for labels.
type RenderTone string

const (
	RenderToneNeutral RenderTone = "neutral"
	RenderToneInfo    RenderTone = "info"
	RenderToneSuccess RenderTone = "success"
	RenderToneWarning RenderTone = "warning"
	RenderToneFailure RenderTone = "failure"
)

// Valid reports whether the tone is supported by renderers.
func (t RenderTone) Valid() bool {
	switch t {
	case RenderToneNeutral, RenderToneInfo, RenderToneSuccess, RenderToneWarning, RenderToneFailure:
		return true
	default:
		return false
	}
}

// RenderMoneyUnit identifies a display unit suffix for money values.
type RenderMoneyUnit string

const (
	RenderMoneyUnitNone  RenderMoneyUnit = ""
	RenderMoneyUnitMonth RenderMoneyUnit = "mo"
)

// Valid reports whether the money unit is supported by renderers.
func (u RenderMoneyUnit) Valid() bool {
	switch u {
	case RenderMoneyUnitNone, RenderMoneyUnitMonth:
		return true
	default:
		return false
	}
}

// RenderMoneyOptions configures money rendering.
type RenderMoneyOptions struct {
	Unit RenderMoneyUnit
}

// RenderValue is a typed semantic value rendered by the shared renderers.
//
//nolint:recvcheck // MarshalJSON works on values; UnmarshalJSON must mutate the receiver.
type RenderValue struct {
	kind   RenderValueKind
	text   string
	status ReportStatus
	tone   RenderTone
	amount float64
	unit   RenderMoneyUnit
	parts  []RenderValue
}

// RenderText builds a plain text render value.
func RenderText(text string) RenderValue {
	return RenderValue{kind: RenderValueKindText, text: text}
}

// RenderCode builds a monospace code render value.
func RenderCode(text string) RenderValue {
	return RenderValue{kind: RenderValueKindCode, text: text}
}

// RenderStatus builds a report status render value.
func RenderStatus(status ReportStatus) RenderValue {
	return RenderValue{kind: RenderValueKindStatus, status: status}
}

// RenderLabel builds a tone-aware label render value.
func RenderLabel(text string, tone RenderTone) RenderValue {
	return RenderValue{kind: RenderValueKindLabel, text: text, tone: tone}
}

// RenderMoney builds a money render value.
func RenderMoney(amount float64, opts RenderMoneyOptions) RenderValue {
	return RenderValue{kind: RenderValueKindMoney, amount: amount, unit: opts.Unit}
}

// RenderMoneyDelta builds a signed money delta render value.
func RenderMoneyDelta(amount float64, opts RenderMoneyOptions) RenderValue {
	return RenderValue{kind: RenderValueKindMoneyDelta, amount: amount, unit: opts.Unit}
}

// RenderModulePath builds a Terraform module path render value.
func RenderModulePath(path string) RenderValue {
	return RenderValue{kind: RenderValueKindModulePath, text: path}
}

// RenderResourceAddress builds a Terraform resource address render value.
func RenderResourceAddress(address string) RenderValue {
	return RenderValue{kind: RenderValueKindResourceAddress, text: address}
}

// RenderInline builds a composite inline value from typed fragments.
func RenderInline(parts ...RenderValue) RenderValue {
	if len(parts) == 1 {
		return parts[0].Clone()
	}
	return RenderValue{kind: RenderValueKindInline, parts: cloneRenderValues(parts)}
}

// Kind returns the semantic value type.
func (v RenderValue) Kind() RenderValueKind {
	return v.kind
}

// Text returns the value text for text-like values.
func (v RenderValue) Text() string {
	return v.text
}

// Status returns the report status for status values.
func (v RenderValue) Status() ReportStatus {
	return v.status
}

// Tone returns the label tone.
func (v RenderValue) Tone() RenderTone {
	return v.tone
}

// Amount returns the money amount.
func (v RenderValue) Amount() float64 {
	return v.amount
}

// Unit returns the money unit.
func (v RenderValue) Unit() RenderMoneyUnit {
	return v.unit
}

// Parts returns a defensive copy of inline fragments.
func (v RenderValue) Parts() []RenderValue {
	return cloneRenderValues(v.parts)
}

// Clone returns a defensive copy of the render value.
func (v RenderValue) Clone() RenderValue {
	cloned := v
	cloned.parts = cloneRenderValues(v.parts)
	return cloned
}

// MarshalJSON preserves the render value wire shape.
func (v RenderValue) MarshalJSON() ([]byte, error) {
	raw := renderValueJSON{Kind: v.kind}
	switch v.kind {
	case RenderValueKindText, RenderValueKindCode, RenderValueKindModulePath, RenderValueKindResourceAddress:
		raw.Text = v.text
	case RenderValueKindStatus:
		raw.Status = v.status
	case RenderValueKindLabel:
		raw.Text = v.text
		raw.Tone = v.tone
	case RenderValueKindMoney, RenderValueKindMoneyDelta:
		amount := v.amount
		raw.Amount = &amount
		raw.Unit = v.unit
	case RenderValueKindInline:
		raw.Parts = cloneRenderValues(v.parts)
	default:
		raw.Text = v.text
	}
	return json.Marshal(raw)
}

// UnmarshalJSON decodes the render value wire shape.
func (v *RenderValue) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, `"`) {
		return errors.New(legacyRenderPayloadError)
	}
	var raw renderValueJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	v.kind = raw.Kind
	v.text = raw.Text
	v.status = raw.Status
	v.tone = raw.Tone
	if raw.Amount != nil {
		v.amount = *raw.Amount
	} else {
		v.amount = 0
	}
	v.unit = raw.Unit
	v.parts = cloneRenderValues(raw.Parts)
	return nil
}

// Validate verifies one render value.
func (v RenderValue) Validate() error {
	switch v.kind {
	case RenderValueKindText, RenderValueKindCode, RenderValueKindModulePath, RenderValueKindResourceAddress:
		if v.text == "" {
			return fmt.Errorf("%s value requires text", v.kind)
		}
	case RenderValueKindStatus:
		if !v.status.Valid() {
			return fmt.Errorf("status value %q is invalid", v.status)
		}
	case RenderValueKindLabel:
		if v.text == "" {
			return errors.New("label value requires text")
		}
		if !v.tone.Valid() {
			return fmt.Errorf("label tone %q is invalid", v.tone)
		}
	case RenderValueKindMoney, RenderValueKindMoneyDelta:
		if math.IsNaN(v.amount) || math.IsInf(v.amount, 0) {
			return fmt.Errorf("%s value requires finite amount", v.kind)
		}
		if !v.unit.Valid() {
			return fmt.Errorf("money unit %q is invalid", v.unit)
		}
	case RenderValueKindInline:
		if len(v.parts) == 0 {
			return errors.New("inline value requires at least one part")
		}
		for i := range v.parts {
			if err := v.parts[i].Validate(); err != nil {
				return fmt.Errorf("inline part %d: %w", i, err)
			}
		}
	default:
		return fmt.Errorf("unsupported render value kind %q", v.kind)
	}
	return nil
}
