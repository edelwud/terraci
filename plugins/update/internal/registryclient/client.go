package registryclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
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
func (c *Client) ModuleVersions(ctx context.Context, namespace, name, provider string) ([]string, error) {
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
func (c *Client) ProviderVersions(ctx context.Context, namespace, typeName string) ([]string, error) {
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
