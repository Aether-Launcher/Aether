// sandbox_http.go - HTTP helpers for the extension sandbox: retry and cache.

package extensions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Aether-Launcher/Aether/pkg/fs"
	"github.com/Aether-Launcher/Aether/pkg/netutil"
)
const maxExtensionHTTPResponse = 10 * 1024 * 1024

// httpGetWithRetry performs the request, retrying transient network failures
// (e.g. temporary DNS resolution failures) a few times with short backoff.
// Mod loader metadata endpoints are usually reachable again within seconds.
func httpGetWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			time.Sleep(time.Duration(attempt-1) * 500 * time.Millisecond)
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !netutil.IsTransientNetworkError(err) {
			break
		}
	}
	return nil, lastErr
}

const httpCacheTTL = 24 * time.Hour

func httpCachePath(target string) string {
	h := sha256.Sum256([]byte(target))
	name := hex.EncodeToString(h[:]) + ".cache"
	return filepath.Join(fs.GetDataDir(), "libraries", ".cache", "http", name)
}

func writeHttpCache(target, body string) {
	path := httpCachePath(target)
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	_ = os.WriteFile(path, []byte(body), 0644)
}

func readHttpCache(target string) (string, bool) {
	path := httpCachePath(target)
	info, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	if time.Since(info.ModTime()) > httpCacheTTL {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(data), true
}
