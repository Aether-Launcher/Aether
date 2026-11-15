package extensions

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadHttpCacheUsesStaleResponse(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	if err := os.MkdirAll(filepath.Join(tmp, ".aether"), 0755); err != nil {
		t.Fatal(err)
	}

	const target = "https://meta.fabricmc.net/v2/versions/loader/1.21.11"
	const body = `[{"loader":{"version":"0.20.1"}}]`
	writeHttpCache(target, body)

	// Simulate a cache entry written long ago. Offline launches must still be
	// able to reuse it after the normal freshness window has passed.
	old := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(httpCachePath(target), old, old); err != nil {
		t.Fatal(err)
	}

	got, ok := readHttpCache(target)
	if !ok {
		t.Fatal("expected stale cache entry to be readable")
	}
	if got != body {
		t.Fatalf("cache body: got %q, want %q", got, body)
	}
}

func TestReadHttpCacheMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	if err := os.MkdirAll(filepath.Join(tmp, ".aether"), 0755); err != nil {
		t.Fatal(err)
	}

	if _, ok := readHttpCache("https://meta.fabricmc.net/v2/versions/loader/1.21.11"); ok {
		t.Fatal("expected missing cache entry to return false")
	}
}
