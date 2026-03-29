package updateengine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewRegistryClient(t *testing.T) {
	c := NewRegistryClient()
	if c.baseURL != defaultBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, defaultBaseURL)
	}
	if c.httpClient == nil {
		t.Error("httpClient is nil")
	}
}

func TestNewRegistryClientWithBase(t *testing.T) {
	c := NewRegistryClientWithBase("http://custom.example.com/v1")
	if c.baseURL != "http://custom.example.com/v1" {
		t.Errorf("baseURL = %q", c.baseURL)
	}
}

func TestDoGet_BadStatusCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewRegistryClientWithBase(srv.URL)
	_, err := client.doGet(context.Background(), srv.URL+"/test")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("error = %q, want to contain 'HTTP 404'", err.Error())
	}
}

func TestDoGet_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewRegistryClientWithBase(srv.URL)
	_, err := client.doGet(ctx, srv.URL+"/test")
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestModuleVersions_EmptyModules(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(moduleVersionsResponse{})
	}))
	defer srv.Close()

	client := NewRegistryClientWithBase(srv.URL + "/v1")
	versions, err := client.ModuleVersions(context.Background(), "ns", "name", "provider")
	if err != nil {
		t.Fatalf("ModuleVersions() error = %v", err)
	}
	if versions != nil {
		t.Errorf("versions = %v, want nil", versions)
	}
}

func TestProviderVersions_DecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	client := NewRegistryClientWithBase(srv.URL + "/v1")
	_, err := client.ProviderVersions(context.Background(), "hashicorp", "aws")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("error = %q, want to contain 'decode'", err.Error())
	}
}

func TestDoGet_ResponseTooLarge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Write more than 10MB
		buf := make([]byte, 4096)
		for i := range buf {
			buf[i] = 'A'
		}
		for range 3000 {
			w.Write(buf)
		}
	}))
	defer srv.Close()

	client := NewRegistryClientWithBase(srv.URL)
	_, err := client.doGet(context.Background(), srv.URL+"/test")
	if err == nil {
		t.Fatal("expected error for response too large")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("error = %q, want to contain 'too large'", err.Error())
	}
}

func TestProviderVersions_FetchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewRegistryClientWithBase(srv.URL + "/v1")
	_, err := client.ProviderVersions(context.Background(), "hashicorp", "aws")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestModuleVersions_FetchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewRegistryClientWithBase(srv.URL + "/v1")
	_, err := client.ModuleVersions(context.Background(), "ns", "name", "provider")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestModuleVersions_DecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	client := NewRegistryClientWithBase(srv.URL + "/v1")
	_, err := client.ModuleVersions(context.Background(), "ns", "name", "provider")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("error = %q, want to contain 'decode'", err.Error())
	}
}

func TestHTTPRegistryClient_ProviderVersions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/providers/hashicorp/aws/versions" {
			http.NotFound(w, r)
			return
		}
		resp := providerVersionsResponse{
			Versions: []struct {
				Version string `json:"version"`
			}{
				{Version: "5.0.0"},
				{Version: "5.1.0"},
				{Version: "5.2.0"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewRegistryClientWithBase(srv.URL + "/v1")
	versions, err := client.ProviderVersions(context.Background(), "hashicorp", "aws")
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 3 {
		t.Errorf("got %d versions, want 3", len(versions))
	}
}

func TestHTTPRegistryClient_ModuleVersions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/modules/hashicorp/consul/aws/versions" {
			http.NotFound(w, r)
			return
		}
		resp := moduleVersionsResponse{
			Modules: []struct {
				Versions []struct {
					Version string `json:"version"`
				} `json:"versions"`
			}{
				{
					Versions: []struct {
						Version string `json:"version"`
					}{
						{Version: "0.1.0"},
						{Version: "0.2.0"},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewRegistryClientWithBase(srv.URL + "/v1")
	versions, err := client.ModuleVersions(context.Background(), "hashicorp", "consul", "aws")
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 {
		t.Errorf("got %d versions, want 2", len(versions))
	}
}

func TestParseModuleSource(t *testing.T) {
	ns, name, provider, err := ParseModuleSource("terraform-aws-modules/vpc/aws")
	if err != nil {
		t.Fatal(err)
	}
	if ns != "terraform-aws-modules" || name != "vpc" || provider != "aws" {
		t.Errorf("got (%s, %s, %s), want (terraform-aws-modules, vpc, aws)", ns, name, provider)
	}

	_, _, _, err = ParseModuleSource("invalid")
	if err == nil {
		t.Error("expected error for invalid source")
	}
}

func TestParseProviderSource(t *testing.T) {
	ns, typeName, err := ParseProviderSource("hashicorp/aws")
	if err != nil {
		t.Fatal(err)
	}
	if ns != "hashicorp" || typeName != "aws" {
		t.Errorf("got (%s, %s), want (hashicorp, aws)", ns, typeName)
	}

	_, _, err = ParseProviderSource("invalid/format/extra")
	if err == nil {
		t.Error("expected error for invalid source")
	}
}

func TestIsRegistrySource(t *testing.T) {
	tests := []struct {
		source string
		want   bool
	}{
		{"hashicorp/consul/aws", true},
		{"terraform-aws-modules/vpc/aws", true},
		{"./modules/local", false},
		{"git::https://example.com/module.git", false},
		{"hashicorp/aws", false},
		{"s3::https://bucket/module.zip", false},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := IsRegistrySource(tt.source); got != tt.want {
				t.Errorf("IsRegistrySource(%q) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}
