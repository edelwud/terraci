package ci

import (
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
)

const managedLabelsPrefix = "<!-- terraci-managed-labels:v1:"
const managedLabelsSuffix = " -->"

// EmbedManagedLabels records the labels currently owned by TerraCI in the
// review comment body. The encoded payload is intentionally hidden in an HTML
// comment so providers can remove only labels TerraCI created earlier.
func EmbedManagedLabels(body string, labels []string) string {
	clean := StripManagedLabelsMetadata(body)
	metadata := EncodeManagedLabelsMetadata(labels)
	if strings.Contains(clean, CommentMarker) {
		return strings.Replace(clean, CommentMarker, CommentMarker+"\n"+metadata, 1)
	}
	return CommentMarker + "\n" + metadata + "\n\n" + clean
}

// StripManagedLabelsMetadata removes all TerraCI managed-label metadata comments.
func StripManagedLabelsMetadata(body string) string {
	for {
		start := strings.Index(body, managedLabelsPrefix)
		if start < 0 {
			return body
		}
		end := strings.Index(body[start:], managedLabelsSuffix)
		if end < 0 {
			return body[:start]
		}
		body = body[:start] + body[start+end+len(managedLabelsSuffix):]
	}
}

// EncodeManagedLabelsMetadata serializes labels into a hidden comment payload.
func EncodeManagedLabelsMetadata(labels []string) string {
	normalized := NormalizeManagedLabels(labels)
	payload, err := json.Marshal(normalized)
	if err != nil {
		payload = []byte("[]")
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return managedLabelsPrefix + encoded + managedLabelsSuffix
}

// ExtractManagedLabels returns the TerraCI-owned labels recorded in a comment body.
func ExtractManagedLabels(body string) []string {
	start := strings.Index(body, managedLabelsPrefix)
	if start < 0 {
		return nil
	}
	payloadStart := start + len(managedLabelsPrefix)
	end := strings.Index(body[payloadStart:], managedLabelsSuffix)
	if end < 0 {
		return nil
	}
	raw := body[payloadStart : payloadStart+end]
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil
	}
	var labels []string
	if err := json.Unmarshal(decoded, &labels); err != nil {
		return nil
	}
	return NormalizeManagedLabels(labels)
}

// NormalizeManagedLabels trims labels, removes empty entries, deduplicates them,
// and returns deterministic case-preserving order.
func NormalizeManagedLabels(labels []string) []string {
	if len(labels) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(labels))
	out := make([]string, 0, len(labels))
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		out = append(out, label)
	}
	sort.Strings(out)
	return out
}

// DiffManagedLabels computes provider label operations for TerraCI-owned labels.
func DiffManagedLabels(previous, current []string) (add, remove []string) {
	previous = NormalizeManagedLabels(previous)
	current = NormalizeManagedLabels(current)

	currentSet := make(map[string]struct{}, len(current))
	for _, label := range current {
		currentSet[label] = struct{}{}
	}
	previousSet := make(map[string]struct{}, len(previous))
	for _, label := range previous {
		previousSet[label] = struct{}{}
		if _, ok := currentSet[label]; !ok {
			remove = append(remove, label)
		}
	}
	for _, label := range current {
		if _, ok := previousSet[label]; !ok {
			add = append(add, label)
		}
	}
	return add, remove
}
