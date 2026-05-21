// Package versionflow composes version command output from binary metadata and
// plugin version providers.
package versionflow

import (
	"fmt"
	"io"
	"maps"
	"sort"

	"github.com/edelwud/terraci/pkg/plugin"
)

// Metadata describes the built binary.
type Metadata struct {
	Version string
	Commit  string
	Date    string
}

// PluginInfo describes one registered plugin for version output.
type PluginInfo struct {
	Name        string
	Description string
}

// Result is the render-ready version command payload.
type Result struct {
	Metadata Metadata
	Info     map[string]string
	Plugins  []PluginInfo
}

type pluginSource interface {
	All() []plugin.Plugin
	VersionProviders() []plugin.VersionProvider
}

// Build collects version information in deterministic output order.
func Build(meta Metadata, source pluginSource) Result {
	result := Result{
		Metadata: meta,
		Info:     make(map[string]string),
	}
	if source == nil {
		return result
	}
	for _, vp := range source.VersionProviders() {
		maps.Copy(result.Info, vp.VersionInfo())
	}
	for _, p := range source.All() {
		result.Plugins = append(result.Plugins, PluginInfo{
			Name:        p.Name(),
			Description: p.Description(),
		})
	}
	return result
}

// Write renders Result in the historical terraci version text format.
func Write(w io.Writer, result Result) error {
	if _, err := fmt.Fprintf(w, "terraci %s\n", result.Metadata.Version); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  commit: %s\n", result.Metadata.Commit); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  built:  %s\n", result.Metadata.Date); err != nil {
		return err
	}

	keys := make([]string, 0, len(result.Info))
	for key := range result.Info {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, err := fmt.Fprintf(w, "  %s: %s\n", key, result.Info[key]); err != nil {
			return err
		}
	}

	if len(result.Plugins) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w, "  plugins:"); err != nil {
		return err
	}
	for _, p := range result.Plugins {
		if _, err := fmt.Fprintf(w, "    - %s: %s\n", p.Name, p.Description); err != nil {
			return err
		}
	}
	return nil
}
