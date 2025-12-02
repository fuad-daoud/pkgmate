//go:build pacman || all_backends

package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type aurResponse struct {
	Results []aurPackage `json:"results"`
}

type aurPackage struct {
	Name    string `json:"Name"`
	Version string `json:"Version"`
}

const CACHE_KEY = "aur_versions"

func CheckAURVersions(packageNames []string) (map[string]string, error) {
	if cached, ok := cache.Get(CACHE_KEY); ok {
		return cached.(map[string]string), nil
	}

	if len(packageNames) == 0 {
		return make(map[string]string), nil
	}

	baseURL := "https://aur.archlinux.org/rpc/v5/info"
	params := url.Values{}
	for _, name := range packageNames {
		params.Add("arg[]", name)
	}

	resp, err := http.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to query AUR: %w", err)
	}
	defer resp.Body.Close()

	var aurResp aurResponse
	if err := json.NewDecoder(resp.Body).Decode(&aurResp); err != nil {
		return nil, fmt.Errorf("failed to decode AUR response: %w", err)
	}

	versions := make(map[string]string)
	for _, pkg := range aurResp.Results {
		versions[pkg.Name] = pkg.Version
	}

	cache.Set(CACHE_KEY, versions)
	return versions, nil
}

func ClearAURCache() {
	cache.Clear()
}
