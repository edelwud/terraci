package registryclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/edelwud/terraci/plugins/tfupdate/internal/sourceaddr"
)

var (
	testProviderAddress = sourceaddr.ProviderAddress{Hostname: "registry.terraform.io", Namespace: "hashicorp", Type: "aws"}
	testModuleAddress   = sourceaddr.ModuleAddress{Hostname: "registry.terraform.io", Namespace: "hashicorp", Name: "consul", Provider: "aws"}
)

func TestNew(t *testing.T) {
	c := New()
	if c.baseURL != DefaultBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, DefaultBaseURL)
	}
	if c.httpClient == nil {
		t.Error("httpClient is nil")
	}
}

func TestNewWithBase(t *testing.T) {
	c := NewWithBase("http://custom.example.com/v1")
	if c.baseURL != "http://custom.example.com/v1" {
		t.Errorf("baseURL = %q", c.baseURL)
	}
}

func TestClient_ProviderBaseURL_UsesRequestedHostname(t *testing.T) {
	client := New()

	got := client.providerBaseURL("registry.opentofu.org")

	if got != "https://registry.opentofu.org/v1" {
		t.Fatalf("providerBaseURL() = %q", got)
	}
}

func TestDoGet_BadStatusCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewWithBase(srv.URL)
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

	client := NewWithBase(srv.URL)
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

	client := NewWithBase(srv.URL + "/v1")
	versions, err := client.ModuleVersions(context.Background(), sourceaddr.ModuleAddress{Hostname: "registry.terraform.io", Namespace: "ns", Name: "name", Provider: "provider"})
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

	client := NewWithBase(srv.URL + "/v1")
	_, err := client.ProviderVersions(context.Background(), testProviderAddress)
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
		buf := make([]byte, 4096)
		for i := range buf {
			buf[i] = 'A'
		}
		for range 3000 {
			w.Write(buf)
		}
	}))
	defer srv.Close()

	client := NewWithBase(srv.URL)
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

	client := NewWithBase(srv.URL + "/v1")
	_, err := client.ProviderVersions(context.Background(), testProviderAddress)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestModuleVersions_FetchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewWithBase(srv.URL + "/v1")
	_, err := client.ModuleVersions(context.Background(), sourceaddr.ModuleAddress{Hostname: "registry.terraform.io", Namespace: "ns", Name: "name", Provider: "provider"})
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

	client := NewWithBase(srv.URL + "/v1")
	_, err := client.ModuleVersions(context.Background(), sourceaddr.ModuleAddress{Hostname: "registry.terraform.io", Namespace: "ns", Name: "name", Provider: "provider"})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("error = %q, want to contain 'decode'", err.Error())
	}
}

func TestClient_ProviderVersions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/providers/hashicorp/aws/versions" {
			http.NotFound(w, r)
			return
		}
		resp := providerVersionsResponse{
			Versions: []providerVersion{
				{Version: "5.0.0"},
				{Version: "5.1.0"},
				{Version: "5.2.0"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewWithBase(srv.URL + "/v1")
	versions, err := client.ProviderVersions(context.Background(), testProviderAddress)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 3 {
		t.Errorf("got %d versions, want 3", len(versions))
	}
}

func TestClient_ProviderPlatforms(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/providers/hashicorp/aws/versions" {
			http.NotFound(w, r)
			return
		}
		resp := providerVersionsResponse{
			Versions: []providerVersion{
				{
					Version: "5.2.0",
					Platforms: []providerPlatform{
						{OS: "linux", Arch: "amd64"},
						{OS: "darwin", Arch: "arm64"},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewWithBase(srv.URL + "/v1")
	platforms, err := client.ProviderPlatforms(context.Background(), testProviderAddress, "5.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(platforms) != 2 {
		t.Fatalf("got %d platforms, want 2", len(platforms))
	}
	if platforms[0] != "linux_amd64" {
		t.Fatalf("platform[0] = %s, want linux_amd64", platforms[0])
	}
}

func TestClient_ProviderPackage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/providers/hashicorp/aws/5.2.0/download/linux/amd64" {
			http.NotFound(w, r)
			return
		}
		resp := providerPackageResponse{
			Filename:    "terraform-provider-aws_5.2.0_linux_amd64.zip",
			DownloadURL: "https://example.test/aws.zip",
			Shasum:      "abc123",
			OS:          "linux",
			Arch:        "amd64",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewWithBase(srv.URL + "/v1")
	pkg, err := client.ProviderPackage(context.Background(), testProviderAddress, "5.2.0", "linux_amd64")
	if err != nil {
		t.Fatal(err)
	}
	if pkg == nil {
		t.Fatal("ProviderPackage() returned nil package")
	}
	if pkg.Platform != "linux_amd64" {
		t.Fatalf("pkg.Platform = %q, want linux_amd64", pkg.Platform)
	}
	if pkg.DownloadURL != "https://example.test/aws.zip" {
		t.Fatalf("pkg.DownloadURL = %q", pkg.DownloadURL)
	}
}

func TestClient_ModuleVersions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/modules/hashicorp/consul/aws/versions" {
			http.NotFound(w, r)
			return
		}
		resp := moduleVersionsResponse{
			Modules: []moduleVersionSet{
				{
					Versions: []moduleVersion{
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

	client := NewWithBase(srv.URL + "/v1")
	versions, err := client.ModuleVersions(context.Background(), testModuleAddress)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 {
		t.Errorf("got %d versions, want 2", len(versions))
	}
}
