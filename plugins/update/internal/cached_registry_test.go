package updateengine

import (
	"context"
	"errors"
	"testing"
)

type countingRegistryClient struct {
	moduleVersions   []string
	providerVersions []string
	moduleErr        error
	providerErr      error
	moduleCalls      int
	providerCalls    int
}

func (c *countingRegistryClient) ModuleVersions(_ context.Context, _, _, _ string) ([]string, error) {
	c.moduleCalls++
	return c.moduleVersions, c.moduleErr
}

func (c *countingRegistryClient) ProviderVersions(_ context.Context, _, _ string) ([]string, error) {
	c.providerCalls++
	return c.providerVersions, c.providerErr
}

func TestCachedRegistryClient_ModuleVersions(t *testing.T) {
	base := &countingRegistryClient{moduleVersions: []string{"1.0.0", "1.1.0"}}
	client := NewCachedRegistryClient(base)

	first, err := client.ModuleVersions(context.Background(), "hashicorp", "consul", "aws")
	if err != nil {
		t.Fatalf("ModuleVersions() first error = %v", err)
	}
	second, err := client.ModuleVersions(context.Background(), "hashicorp", "consul", "aws")
	if err != nil {
		t.Fatalf("ModuleVersions() second error = %v", err)
	}

	if base.moduleCalls != 1 {
		t.Fatalf("moduleCalls = %d, want 1", base.moduleCalls)
	}
	first[0] = "mutated"
	if second[0] != "1.0.0" {
		t.Fatalf("cached versions were mutated through caller slice: got %q", second[0])
	}
}

func TestCachedRegistryClient_ProviderVersionsCachesErrors(t *testing.T) {
	base := &countingRegistryClient{providerErr: errors.New("boom")}
	client := NewCachedRegistryClient(base)

	_, err := client.ProviderVersions(context.Background(), "hashicorp", "aws")
	if err == nil {
		t.Fatal("expected provider error on first call")
	}
	_, err = client.ProviderVersions(context.Background(), "hashicorp", "aws")
	if err == nil {
		t.Fatal("expected provider error on second call")
	}

	if base.providerCalls != 1 {
		t.Fatalf("providerCalls = %d, want 1", base.providerCalls)
	}
}
