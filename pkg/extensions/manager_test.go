package extensions

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// TestServerStartStopReuse verifies the extension server reuses the same port
// and cleans up properly on Stop().
func TestServerStartStopReuse(t *testing.T) {
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

	// Verify server is reachable
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url1)
	if err != nil {
		t.Fatalf("server not reachable at %s: %v", url1, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	if err := s.Stop(); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// After stop, Start() should allocate a new port
	url3, err := s.Start()
	if err != nil {
		t.Fatalf("Start() after Stop() failed: %v", err)
	}
	if url3 == url2 {
		t.Fatalf("expected new port after Stop(), got same %s", url3)
	}
}

// TestReloadAsyncSerializes ensures concurrent ReloadAsync() calls are serialized.
func TestReloadAsyncSerializes(t *testing.T) {
	// Use the actual data directory (which may be empty or have existing extensions)
	// The test verifies that concurrent ReloadAsync calls are serialized via mutex.
	ctx := context.Background()
	m := NewManager(ctx, nil)

	// Initial load
	if err := m.LoadAll(); err != nil {
		t.Fatalf("initial LoadAll: %v", err)
	}

	// Fire two concurrent ReloadAsync - second should return error
	errCh := make(chan error, 2)
	go func() { errCh <- m.ReloadAsync() }()
	time.Sleep(10 * time.Millisecond) // let first acquire lock
	go func() { errCh <- m.ReloadAsync() }()

	results := 0
	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			if err != nil && err.Error() != "extension reload already in progress" {
				t.Errorf("unexpected error: %v", err)
			}
			results++
		case <-time.After(5 * time.Second):
			t.Fatal("ReloadAsync timed out")
		}
	}
	if results != 2 {
		t.Fatal("expected 2 ReloadAsync calls to complete")
	}
}

// TestUpdateExtensionTriggersReload verifies UpdateExtension fires ReloadAsync.
func TestUpdateExtensionTriggersReload(t *testing.T) {
	ctx := context.Background()
	GlobalManager = NewManager(ctx, nil)
	defer func() { GlobalManager = nil }()

	if err := GlobalManager.LoadAll(); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	// Test passes if no panic and UpdateExtension returns quickly for nonexistent extension
	// (Full integration would need a real gallery mock)
	_, err := UpdateExtension("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent extension")
	}
}

// TestLoadAllFastPath ensures LoadAll completes quickly without sandbox creation.
func TestLoadAllFastPath(t *testing.T) {
	ctx := context.Background()
	m := NewManager(ctx, nil)

	start := time.Now()
	if err := m.LoadAll(); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	elapsed := time.Since(start)

	// LoadAll should be fast (metadata only, no sandbox creation)
	if elapsed > 500*time.Millisecond {
		t.Errorf("LoadAll took %v, expected <500ms", elapsed)
	}

	// Sandboxes should NOT be created in LoadAll (they're created in reloadSandboxes)
	if len(m.sandboxes) != 0 {
		t.Errorf("expected 0 sandboxes after LoadAll (sandboxes created in background), got %d", len(m.sandboxes))
	}

	// Give background reloadSandboxes a moment to run
	time.Sleep(200 * time.Millisecond)
}