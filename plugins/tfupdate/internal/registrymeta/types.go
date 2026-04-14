package registrymeta

// ProviderPackage is a provider package DTO decoupled from transport-layer types.
type ProviderPackage struct {
	Platform    string
	Filename    string
	DownloadURL string
	Shasum      string
}

// ModuleProviderDep describes a provider dependency declared by a registry module version.
type ModuleProviderDep struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Source    string `json:"source"`
	Version   string `json:"version"`
}
