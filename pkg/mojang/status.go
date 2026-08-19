package mojang

import (
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"sync"
	"time"
)

// ServiceStatus describes the reachability of a single service Aether depends on.
type ServiceStatus struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	Reachable bool   `json:"reachable"`
	LatencyMs int64  `json:"latencyMs"`
	Error     string `json:"error,omitempty"`
}

// ConnectivityStatus is the aggregate health of the services needed to
// install and launch Minecraft instances.
type ConnectivityStatus struct {
	Overall   string          `json:"overall"` // "online" | "degraded" | "offline" | "unknown"
	CheckedAt time.Time       `json:"checkedAt"`
	Services  []ServiceStatus `json:"services"`
}

var (
	statusCache    ConnectivityStatus
	statusCacheAt  time.Time
	statusCacheMu  sync.Mutex
	statusCacheTTL = 20 * time.Second
)

// CheckConnectivity probes the endpoints used by the install/launch pipeline
// in parallel and reports which are reachable. Results are cached briefly so
// the UI can poll without hammering the services.
func CheckConnectivity() ConnectivityStatus {
	statusCacheMu.Lock()
	defer statusCacheMu.Unlock()

	if time.Since(statusCacheAt) < statusCacheTTL && len(statusCache.Services) > 0 {
		return statusCache
	}

	endpoints := []struct {
		Name string
		URL  string
	}{
		{"Version Manifest", ManifestURL},
		{"Version Meta", "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"},
		{"Download Server", "https://piston-data.mojang.com"},
		{"Assets CDN", "https://resources.download.minecraft.net"},
		{"Libraries CDN", "https://libraries.minecraft.net"},
		{"Java Runtime", "https://api.adoptium.net/v3/info/available_releases"},
		{"Extension Registry", "https://raw.githubusercontent.com/wayback09/Aether-Extensions/main/index.json"},
	}

	services := make([]ServiceStatus, len(endpoints))
	var wg sync.WaitGroup
	for i, ep := range endpoints {
		wg.Add(1)
		go func(i int, name, url string) {
			defer wg.Done()
			services[i] = probeService(name, url)
		}(i, ep.Name, ep.URL)
	}
	wg.Wait()

	reachable := 0
	for _, s := range services {
		if s.Reachable {
			reachable++
		}
	}

	overall := "offline"
	switch {
	case reachable == len(services):
		overall = "online"
	case reachable > 0:
		overall = "degraded"
	}

	statusCache = ConnectivityStatus{Overall: overall, CheckedAt: time.Now(), Services: services}
	statusCacheAt = time.Now()
	return statusCache
}

// probeService performs a short GET against the given URL and reports whether
// the host responded. A missing resource (404) still counts as reachable — the
// server itself answered.
func probeService(name, rawURL string) ServiceStatus {
	host := rawURL
	if u, err := neturl.Parse(rawURL); err == nil && u.Host != "" {
		host = u.Host
	}

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return ServiceStatus{Name: name, Host: host, Reachable: false, Error: err.Error()}
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return ServiceStatus{Name: name, Host: host, Reachable: false, Error: err.Error(), LatencyMs: time.Since(start).Milliseconds()}
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 256))

	latency := time.Since(start).Milliseconds()
	if resp.StatusCode >= 500 {
		return ServiceStatus{Name: name, Host: host, Reachable: false, Error: fmt.Sprintf("HTTP %d", resp.StatusCode), LatencyMs: latency}
	}
	return ServiceStatus{Name: name, Host: host, Reachable: true, LatencyMs: latency}
}