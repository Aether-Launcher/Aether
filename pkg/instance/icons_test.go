package instance

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupIconDataDir points fs.GetDataDir() at a temp .aether folder and
// returns the instances dir.
func setupIconDataDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Chdir(tmp)
	dir := filepath.Join(tmp, ".aether", "instances", "test-inst")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func pngBytes() []byte {
	// 1x1 transparent PNG.
	raw, _ := base64.StdEncoding.DecodeString(
		"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==")
	return raw
}

func TestSetAndGetIconFromFile(t *testing.T) {
	setupIconDataDir(t)

	src := filepath.Join(t.TempDir(), "picked.png")
	if err := os.WriteFile(src, pngBytes(), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SetIconFromFile("test-inst", src); err != nil {
		t.Fatalf("SetIconFromFile: %v", err)
	}
	got, err := GetInstanceIcon("test-inst")
	if err != nil {
		t.Fatalf("GetInstanceIcon: %v", err)
	}
	if !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Fatalf("expected png data URL, got %.40s", got)
	}
	if err := RemoveInstanceIcon("test-inst"); err != nil {
		t.Fatalf("RemoveInstanceIcon: %v", err)
	}
	got, err = GetInstanceIcon("test-inst")
	if err != nil || got != "" {
		t.Fatalf("expected empty icon after remove, got %q err %v", got, err)
	}
	// Removing again is not an error.
	if err := RemoveInstanceIcon("test-inst"); err != nil {
		t.Fatalf("second remove: %v", err)
	}
}

func TestSetIconRejectsNonImage(t *testing.T) {
	setupIconDataDir(t)

	src := filepath.Join(t.TempDir(), "evil.exe")
	if err := os.WriteFile(src, []byte("MZ not really an image at all...."), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SetIconFromFile("test-inst", src); err == nil {
		t.Fatal("expected rejection of non-image file")
	}
}

func TestSetIconRejectsOversize(t *testing.T) {
	setupIconDataDir(t)

	src := filepath.Join(t.TempDir(), "big.png")
	big := make([]byte, MaxIconBytes+1)
	big[0], big[1], big[2], big[3] = 0x89, 0x50, 0x4E, 0x47
	if err := os.WriteFile(src, big, 0644); err != nil {
		t.Fatal(err)
	}
	if err := SetIconFromFile("test-inst", src); err == nil {
		t.Fatal("expected rejection of oversize file")
	}
}

func TestSetIconFromURL(t *testing.T) {
	setupIconDataDir(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes())
	}))
	defer srv.Close()

	// httptest host is not allowlisted.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := SetIconFromURL(ctx, "test-inst", srv.URL+"/icon.png"); err == nil {
		t.Fatal("expected rejection of non-allowlisted host")
	}

	// Allowlisted host check is pure: cdn.modrinth.com passes validation
	// (network fetch itself is not exercised here).
	if !isAllowedIconURL("https://cdn.modrinth.com/icon.png") {
		t.Fatal("cdn.modrinth.com should be allowlisted")
	}
	if isAllowedIconURL("http://cdn.modrinth.com/icon.png") {
		t.Fatal("plain http must be rejected")
	}
	if isAllowedIconURL("https://evil.example/icon.png") {
		t.Fatal("unknown host must be rejected")
	}
	if isAllowedIconURL("https://cdn.modrinth.com.evil.example/icon.png") {
		t.Fatal("suffix-spoofed host must be rejected")
	}
}

func TestGetInstanceIconsMap(t *testing.T) {
	setupIconDataDir(t)

	src := filepath.Join(t.TempDir(), "picked.png")
	if err := os.WriteFile(src, pngBytes(), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SetIconFromFile("test-inst", src); err != nil {
		t.Fatal(err)
	}
	// GetInstanceIcons walks GetInstances, so craft an instance.json.
	manifest := `{"id":"test-inst","name":"Test","version":"1.20.1","loader":"Vanilla"}`
	instDir := filepath.Join(t.TempDir(), ".aether", "instances", "test-inst")
	_ = instDir
	// NOTE: setupIconDataDir chdir'd into its own temp dir; resolve it via CWD.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(cwd, ".aether", "instances", "test-inst", "instance.json"),
		[]byte(manifest), 0644,
	); err != nil {
		t.Fatal(err)
	}
	m := GetInstanceIcons()
	url, ok := m["test-inst"]
	if !ok {
		t.Fatal("expected test-inst in icons map")
	}
	if !strings.HasPrefix(url, "data:image/png;base64,") {
		t.Fatalf("expected png data URL, got %.40s", url)
	}
}
