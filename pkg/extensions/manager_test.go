package extensions

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupExtensionsDir isolates the test from the real user data directory.
// Prefers an .aether folder in the working directory, so the test changes
// directory into a temp dir and creates the expected layout.
func setupExtensionsDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Chdir(tmp)

	extDir := filepath.Join(tmp, ".aether", "extensions")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatalf("failed to create extensions dir: %v", err)
	}
	return extDir
}

func writeDummyExtension(t *testing.T, extDir, id string) {
	t.Helper()
	dir := filepath.Join(extDir, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create extension dir: %v", err)
	}
	manifest := fmt.Sprintf(`{"id": %q, "name": %q, "version": "1.0.0", "author": "test", "main": "main.js"}`, id, id)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.js"), []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write main.js: %v", err)
	}
}

// waitForSandboxes polls until the background sandbox reload finishes or the
// deadline passes.
func waitForSandboxes(t *testing.T, m *Manager, want int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if len(m.sandboxes) >= want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("background sandbox reload did not finish: want %d sandboxes, got %d", want, len(m.sandboxes))
}

// TestServerStartStopReuse verifies the extension server reuses the same URL
// while running, serves files from the extensions dir, and releases cleanly
// on Stop().
func TestServerStartStopReuse(t *testing.T) {
	extDir := setupExtensionsDir(t)
	if err := os.WriteFile(filepath.Join(extDir, "hello.txt"), []byte("world"), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	s := NewServer()

	url1, err := s.Start()
	if err != nil {
		t.Fatalf("first Start() failed: %v", err)
	}

	url2, err := s.Start()
	if err != nil {
		t.Fatalf("second Start() failed: %v", err)
	}
	if url1 != url2 {
		t.Fatalf("expected same URL on repeated Start(), got %s then %s", url1, url2)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url1 + "/hello.txt")
	if err != nil {
		t.Fatalf("server not reachable at %s: %v", url1, err)
	}
	body := make([]byte, 16)
	n, _ := resp.Body.Read(body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	if string(body[:n]) != "world" {
		t.Fatalf("expected served content %q, got %q", "world", string(body[:n]))
	}

	if err := s.Stop(); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// After Stop() the listener is released; a subsequent Start() must work
	// again (the OS may hand back any free port).
	if _, err := s.Start(); err != nil {
		t.Fatalf("Start() after Stop() failed: %v", err)
	}
}

// TestReloadAsyncSerializes verifies ReloadAsync refuses to start a second
// reload while one is in progress, then succeeds once the lock is released.
func TestReloadAsyncSerializes(t *testing.T) {
	extDir := setupExtensionsDir(t)
	writeDummyExtension(t, extDir, "com.test.serial")

	m := NewManager(context.Background(), nil)

	if err := m.LoadAll(); err != nil {
		t.Fatalf("initial LoadAll: %v", err)
	}

	// Hold the reload lock to simulate an in-progress reload.
	m.reloadMu.Lock()
	err := m.ReloadAsync()
	m.reloadMu.Unlock()

	if err == nil || err.Error() != "extension reload already in progress" {
		t.Fatalf("expected 'already in progress' error, got %v", err)
	}

	// Once the lock is free the reload goes through and the background
	// sandbox load completes.
	if err := m.ReloadAsync(); err != nil {
		t.Fatalf("ReloadAsync after unlock: %v", err)
	}
	waitForSandboxes(t, m, 1)
}

// TestLoadAllFastPath ensures LoadAll scans metadata quickly and defers
// sandbox creation to the background pipeline.
func TestLoadAllFastPath(t *testing.T) {
	extDir := setupExtensionsDir(t)
	for _, id := range []string{"com.test.a", "com.test.b", "com.test.c"} {
		writeDummyExtension(t, extDir, id)
	}

	m := NewManager(context.Background(), nil)

	start := time.Now()
	if err := m.LoadAll(); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	elapsed := time.Since(start)

	// Metadata scan must be quick even if the registry lookup stalls; the
	// heavy sandbox work happens asynchronously afterwards.
	if elapsed > 6*time.Second {
		t.Errorf("LoadAll took %v, expected well under the registry timeout", elapsed)
	}

	if len(m.LoadedExtensions) != 3 {
		t.Fatalf("expected 3 loaded extensions, got %d", len(m.LoadedExtensions))
	}

	waitForSandboxes(t, m, 3)

	for id, ext := range m.LoadedExtensions {
		if ext.Reloading {
			t.Errorf("extension %s still marked as reloading after completion", id)
		}
	}
}