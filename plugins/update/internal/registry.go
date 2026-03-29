package updateengine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	defaultBaseURL     = "https://registry.terraform.io/v1"
	defaultHTTPTimeout = 30 * time.Second
)

// RegistryClient queries the Terraform Registry for version information.
type RegistryClient interface {
	ModuleVersions(ctx context.Context, namespace, name, provider string) ([]string, error)
	ProviderVersions(ctx context.Context, namespace, typeName string) ([]string, error)
}

// HTTPRegistryClient implements RegistryClient using the public Terraform Registry API.
type HTTPRegistryClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewRegistryClient creates a new HTTP-based registry client.
func NewRegistryClient() *HTTPRegistryClient {
	return &HTTPRegistryClient{
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
	}
}

// NewRegistryClientWithBase creates a registry client with a custom base URL (for testing).
func NewRegistryClientWithBase(baseURL string) *HTTPRegistryClient {
	return &HTTPRegistryClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
	}
}

type moduleVersionsResponse struct {
	Modules []struct {
		Versions []struct {
			Version string `json:"version"`
		} `json:"versions"`
	} `json:"modules"`
}

type providerVersionsResponse struct {
	Versions []struct {
		Version string `json:"version"`
	} `json:"versions"`
}

// ModuleVersions fetches available versions for a registry module.
func (c *HTTPRegistryClient) ModuleVersions(ctx context.Context, namespace, name, provider string) ([]string, error) {
	url := fmt.Sprintf("%s/modules/%s/%s/%s/versions", c.baseURL, namespace, name, provider)

	body, err := c.doGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch module versions for %s/%s/%s: %w", namespace, name, provider, err)
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

// ProviderVersions fetches available versions for a registry provider.
func (c *HTTPRegistryClient) ProviderVersions(ctx context.Context, namespace, typeName string) ([]string, error) {
	url := fmt.Sprintf("%s/providers/%s/%s/versions", c.baseURL, namespace, typeName)

	body, err := c.doGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch provider versions for %s/%s: %w", namespace, typeName, err)
	}

	var resp providerVersionsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode provider versions: %w", err)
	}

	versions := make([]string, 0, len(resp.Versions))
	for _, v := range resp.Versions {
		versions = append(versions, v.Version)
	}
	return versions, nil
}

func (c *HTTPRegistryClient) doGet(ctx context.Context, url string) ([]byte, error) {
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

	const maxBody = 10 << 20 // 10MB
	body := make([]byte, 0, 4096)
	buf := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			body = append(body, buf[:n]...)
			if len(body) > maxBody {
				return nil, fmt.Errorf("response too large from %s", url)
			}
		}
		if readErr != nil {
			break
		}
	}
	return body, nil
}

// ParseModuleSource parses a registry module source like "hashicorp/consul/aws"
// into (namespace, name, provider).
func ParseModuleSource(source string) (namespace, name, provider string, err error) {
	parts := strings.Split(source, "/")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid registry module source %q: expected namespace/name/provider", source)
	}
	return parts[0], parts[1], parts[2], nil
}

// ParseProviderSource parses a provider source like "hashicorp/aws"
// into (namespace, typeName).
func ParseProviderSource(source string) (namespace, typeName string, err error) {
	parts := strings.Split(source, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid provider source %q: expected namespace/type", source)
	}
	return parts[0], parts[1], nil
}

// IsRegistrySource returns true if the source looks like a Terraform registry reference.
func IsRegistrySource(source string) bool {
	// Local paths and URL-based sources are not registry references.
	if strings.HasPrefix(source, "./") || strings.HasPrefix(source, "../") {
		return false
	}
	if strings.Contains(source, "://") || strings.Contains(source, "::") {
		return false
	}
	parts := strings.Split(source, "/")
	return len(parts) == 3
}
