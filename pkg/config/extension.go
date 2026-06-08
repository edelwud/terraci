package config

import (
	"sort"

	"go.yaml.in/yaml/v4"
)

type extensionNodeMap map[string]yaml.Node

// ExtensionDocument is a read-only extension config section captured from
// Config.
type ExtensionDocument struct {
	key  ExtensionKey
	node yaml.Node
}

// ExtensionDocumentSet is a deterministic read-only set of extension config
// documents.
type ExtensionDocumentSet struct {
	documents []ExtensionDocument
}

// Extension returns the named extension config document.
func (c *Config) Extension(key ExtensionKey) (ExtensionDocument, bool) {
	if c == nil || c.extensions == nil || key.String() == "" {
		return ExtensionDocument{}, false
	}
	node, ok := c.extensions[key.String()]
	if !ok {
		return ExtensionDocument{}, false
	}
	return ExtensionDocument{key: key, node: cloneYAMLNode(node)}, true
}

// ExtensionDocuments returns all extension config documents in deterministic
// key order.
func (c *Config) ExtensionDocuments() ExtensionDocumentSet {
	if c == nil || len(c.extensions) == 0 {
		return ExtensionDocumentSet{}
	}
	return newExtensionDocumentSet(c.extensions)
}

// Key returns the validated extension key.
func (d ExtensionDocument) Key() ExtensionKey {
	return d.key
}

// Decode decodes the extension config into target.
func (d ExtensionDocument) Decode(target any) error {
	return d.node.Decode(target)
}

// Clone returns a defensive copy of d.
func (d ExtensionDocument) Clone() ExtensionDocument {
	d.node = cloneYAMLNode(d.node)
	return d
}

// Documents returns defensive document copies.
func (s ExtensionDocumentSet) Documents() []ExtensionDocument {
	if len(s.documents) == 0 {
		return nil
	}
	out := make([]ExtensionDocument, len(s.documents))
	for i := range s.documents {
		out[i] = s.documents[i].Clone()
	}
	return out
}

// Len returns the number of extension documents.
func (s ExtensionDocumentSet) Len() int {
	return len(s.documents)
}

// IsEmpty reports whether the set has no extension documents.
func (s ExtensionDocumentSet) IsEmpty() bool {
	return len(s.documents) == 0
}

// Keys returns extension keys in deterministic order.
func (s ExtensionDocumentSet) Keys() []ExtensionKey {
	if len(s.documents) == 0 {
		return nil
	}
	out := make([]ExtensionKey, len(s.documents))
	for i := range s.documents {
		out[i] = s.documents[i].Key()
	}
	return out
}

func newExtensionDocumentSet(nodes extensionNodeMap) ExtensionDocumentSet {
	keys := make([]string, 0, len(nodes))
	for key := range nodes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	documents := make([]ExtensionDocument, 0, len(keys))
	for _, key := range keys {
		extensionKey, err := NewExtensionKey(key)
		if err != nil {
			continue
		}
		documents = append(documents, ExtensionDocument{
			key:  extensionKey,
			node: cloneYAMLNode(nodes[key]),
		})
	}
	return ExtensionDocumentSet{documents: documents}
}
