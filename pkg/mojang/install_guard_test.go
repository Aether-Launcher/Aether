package mojang

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestClaimInstall(t *testing.T) {
	const id = "test-guard-instance"
	ReleaseInstall(id) // ensure clean state

	if !ClaimInstall(id) {
		t.Fatal("first claim should succeed")
	}
	if ClaimInstall(id) {
		t.Fatal("second claim for same instance must fail while running")
	}
	// Different instance is unaffected.
	if !ClaimInstall(id + "-other") {
		t.Fatal("claim for a different instance should succeed")
	}
	ReleaseInstall(id)
	ReleaseInstall(id + "-other")
	if !ClaimInstall(id) {
		t.Fatal("claim after release should succeed")
	}
	ReleaseInstall(id)
}

func TestClaimInstallConcurrent(t *testing.T) {
	const id = "test-guard-concurrent"
	ReleaseInstall(id)

	var wins int64
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ClaimInstall(id) {
				atomic.AddInt64(&wins, 1)
			}
		}()
	}
	wg.Wait()
	ReleaseInstall(id)

	if wins != 1 {
		t.Fatalf("expected exactly 1 winner out of 50, got %d", wins)
	}
}
