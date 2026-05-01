package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSchema_ValidJSON(t *testing.T) {
	dir := fixtureDir(t, "basic")

	// schema command writes to stdout via fmt.Print, or to file with -o
	outFile := filepath.Join(t.TempDir(), "schema.json")
	err := runTerraCi(t, dir, "schema", "-o", outFile)
	if err != nil {
		t.Fatalf("schema failed: %v", err)
	}

	data, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatalf("failed to read schema file: %v", readErr)
	}

	var schema map[string]any
	if jsonErr := json.Unmarshal(data, &schema); jsonErr != nil {
		t.Fatalf("invalid JSON schema: %v", jsonErr)
	}

	// Should have properties
	props, ok := schema["properties"].(map[string]any)
	if !ok || props == nil {
		t.Fatal("schema missing properties")
	}

	// Should have extensions section
	extensions, ok := props["extensions"].(map[string]any)
	if !ok || extensions == nil {
		t.Fatal("schema missing extensions property")
	}

	// Should have structure section
	if _, ok := props["structure"]; !ok {
		t.Error("schema missing structure property")
	}
}

func TestSchema_ToStdout(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "schema")
	if err != nil {
		t.Fatalf("schema to stdout failed: %v", err)
	}

	var schema map[string]any
	if jsonErr := json.Unmarshal([]byte(output), &schema); jsonErr != nil {
		t.Fatalf("invalid JSON from stdout: %v", jsonErr)
	}
}
