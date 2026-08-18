package mojang

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetVersionManifest(t *testing.T) {
	var server *httptest.Server

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/manifest" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, "{\"latest\":{\"release\":\"1.20.4\"},\"versions\":[{\"id\":\"1.20.4\",\"url\":\"%s/version/1.20.4\"}]}", server.URL)
	}))
	defer server.Close()
	versionManifestURL = server.URL + "/manifest"
	t.Cleanup(func() { versionManifestURL = ManifestURL })
	manifest, err := GetVersionManifest()
	if err != nil {
		t.Fatalf("Failed to get manifest: %v", err)
	}
	if manifest.Latest.Release != "1.20.4" {
		t.Errorf("Expected release 1.20.4, got %q", manifest.Latest.Release)
	}
	if len(manifest.Versions) != 1 {
		t.Errorf("Expected one version, got %d", len(manifest.Versions))
	}
}

func TestGetVersionInfo(t *testing.T) {
	var server *httptest.Server

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest":
			fmt.Fprintf(w, "{\"versions\":[{\"id\":\"1.20.4\",\"url\":\"%s/version/1.20.4\"}]}", server.URL)
		case "/version/1.20.4":
			fmt.Fprint(w, "{\"id\":\"1.20.4\",\"downloads\":{\"client\":{\"url\":\"https://example.test/client.jar\"}},\"libraries\":[{\"name\":\"org.example:test:1.0\"}]}")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	versionManifestURL = server.URL + "/manifest"
	t.Cleanup(func() { versionManifestURL = ManifestURL })
	info, err := GetVersionInfo("1.20.4")
	if err != nil {
		t.Fatalf("Failed to get version info: %v", err)
	}
	if info.Downloads.Client.URL == "" {
		t.Error("Expected client download URL")
	}
	if len(info.Libraries) == 0 {
		t.Error("Expected libraries to be parsed")
	}
}
