package registryclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/edelwud/terraci/plugins/tfupdate/internal/registrymeta"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/sourceaddr"
)

const (
	DefaultBaseURL     = "https://registry.terraform.io/v1"
	DefaultHTTPTimeout = 30 * time.Second
)

// Client implements the Terraform Registry HTTP API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New constructs a registry client with default configuration.
func New() *Client {
	return &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: DefaultHTTPTimeout},
	}
}

// NewWithBase constructs a registry client with a custom base URL.
func NewWithBase(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: DefaultHTTPTimeout},
	}
}

type moduleVersionsResponse struct {
	Modules []moduleVersionSet `json:"modules"`
}

type moduleVersionSet struct {
	Versions []moduleVersion `json:"versions"`
}

type moduleVersion struct {
	Version string `json:"version"`
}

type providerVersionsResponse struct {
	Versions []providerVersion `json:"versions"`
}

type providerVersion struct {
	Version   string             `json:"version"`
	Platforms []providerPlatform `json:"platforms"`
}

type providerPlatform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

type providerPackageResponse struct {
	Filename    string `json:"filename"`
	DownloadURL string `json:"download_url"`
	Shasum      string `json:"shasum"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
}

// ModuleVersions fetches available versions for a registry module.
func (c *Client) ModuleVersions(ctx context.Context, address sourceaddr.ModuleAddress) ([]string, error) {
	url := fmt.Sprintf("%s/modules/%s/%s/%s/versions", c.moduleBaseURL(address.Hostname), address.Namespace, address.Name, address.Provider)

	body, err := c.doGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch module versions for %s/%s/%s: %w", address.Namespace, address.Name, address.Provider, err)
	}

	var resp moduleVersionsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode module versions: %w", err)
	}

	if len(resp.Modules) == 0 {
		return nil, nil
	}

	versions := make([]string, 0, len(resp.Modules[0].Versions))
	for _, v := range resp.Modules[0].Versions {
		versions = append(versions, v.Version)
	}
	return versions, nil
}

type moduleDetailResponse struct {
	Root struct {
		ProviderDependencies []registrymeta.ModuleProviderDep `json:"provider_dependencies"`
	} `json:"root"`
}

// ModuleProviderDeps fetches provider dependencies for a specific module version.
func (c *Client) ModuleProviderDeps(ctx context.Context, address sourceaddr.ModuleAddress, version string) ([]registrymeta.ModuleProviderDep, error) {
	url := fmt.Sprintf("%s/modules/%s/%s/%s/%s", c.moduleBaseURL(address.Hostname), address.Namespace, address.Name, address.Provider, version)

	body, err := c.doGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch module details for %s/%s/%s %s: %w", address.Namespace, address.Name, address.Provider, version, err)
	}

	var resp moduleDetailResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode module details: %w", err)
	}

	return resp.Root.ProviderDependencies, nil
}

// ProviderVersions fetches available versions for a registry provider.
func (c *Client) ProviderVersions(ctx context.Context, address sourceaddr.ProviderAddress) ([]string, error) {
	resp, err := c.providerVersions(ctx, address)
	if err != nil {
		return nil, err
	}

	versions := make([]string, 0, len(resp.Versions))
	for _, v := range resp.Versions {
		versions = append(versions, v.Version)
	}
	return versions, nil
}

// ProviderPlatforms fetches all published platforms for a specific provider version.
func (c *Client) ProviderPlatforms(ctx context.Context, address sourceaddr.ProviderAddress, version string) ([]string, error) {
	resp, err := c.providerVersions(ctx, address)
	if err != nil {
		return nil, err
	}

	for _, item := range resp.Versions {
		if item.Version != version {
			continue
		}

		platforms := make([]string, 0, len(item.Platforms))
		for _, platform := range item.Platforms {
			if platform.OS == "" || platform.Arch == "" {
				continue
			}
			platforms = append(platforms, platform.OS+"_"+platform.Arch)
		}
		return platforms, nil
	}

	return nil, nil
}

// ProviderPackage fetches package metadata for a specific provider platform build.
func (c *Client) ProviderPackage(
	ctx context.Context,
	address sourceaddr.ProviderAddress,
	version, platform string,
) (*registrymeta.ProviderPackage, error) {
	osName, arch, ok := strings.Cut(platform, "_")
	if !ok || osName == "" || arch == "" {
		return nil, fmt.Errorf("invalid provider platform %q", platform)
	}

	url := fmt.Sprintf("%s/providers/%s/%s/%s/download/%s/%s", c.providerBaseURL(address.Hostname), address.Namespace, address.Type, version, osName, arch)
	body, err := c.doGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch provider package for %s/%s %s %s: %w", address.Namespace, address.Type, version, platform, err)
	}

	var resp providerPackageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode provider package: %w", err)
	}

	return &registrymeta.ProviderPackage{
		Platform:    platform,
		Filename:    resp.Filename,
		DownloadURL: resp.DownloadURL,
		Shasum:      resp.Shasum,
	}, nil
}

func (c *Client) providerVersions(ctx context.Context, address sourceaddr.ProviderAddress) (*providerVersionsResponse, error) {
	url := fmt.Sprintf("%s/providers/%s/%s/versions", c.providerBaseURL(address.Hostname), address.Namespace, address.Type)

	body, err := c.doGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch provider versions for %s/%s: %w", address.Namespace, address.Type, err)
	}

	var resp providerVersionsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode provider versions: %w", err)
	}

	return &resp, nil
}

func (c *Client) providerBaseURL(hostname string) string {
	if c.baseURL != DefaultBaseURL {
		return c.baseURL
	}
	if hostname == "" {
		hostname = "registry.terraform.io"
	}
	return "https://" + hostname + "/v1"
}

func (c *Client) moduleBaseURL(hostname string) string {
	if c.baseURL != DefaultBaseURL {
		return c.baseURL
	}
	if hostname == "" {
		hostname = "registry.terraform.io"
	}
	return "https://" + hostname + "/v1"
}

func (c *Client) doGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	const maxBody = 10 << 20
	limited := io.LimitReader(resp.Body, maxBody+1)
	body, err := io.ReadAll(limited)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read response from %s: %w", url, err)
	}
	if len(body) > maxBody {
		return nil, fmt.Errorf("response too large from %s", url)
	}
	return body, nil
}
