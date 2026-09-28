package netutil

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func sha1Of(b []byte) string {
	h := sha1.Sum(b)
	return hex.EncodeToString(h[:])
}

// Concurrent downloads of the same URL to the same dest must all succeed
// with correct bytes (regression: shared "<dest>.tmp" caused rename
// collisions and checksum mismatches on Windows).
func TestDownloadFileConcurrentSameDest(t *testing.T) {
	payload := []byte("concurrent-download-payload-0123456789")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "lib.jar")
	want := sha1Of(payload)

	var wg sync.WaitGroup
	errs := make([]error, 10)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			errs[i] = DownloadFile(ctx, srv.URL, dest, nil, want)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatal("downloaded content mismatch")
	}
}

// A transient checksum mismatch must be retried, not abort the download.
func TestDownloadFileChecksumMismatchRetries(t *testing.T) {
	good := []byte("correct-content")
	bad := []byte("corrupted-content!!")
	var hits int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt64(&hits, 1)
		body := bad
		if n > 2 {
			body = good
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "lib.jar")
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if err := DownloadFile(ctx, srv.URL, dest, nil, sha1Of(good)); err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(good) {
		t.Fatal("expected good content after retry")
	}
	if atomic.LoadInt64(&hits) < 3 {
		t.Fatalf("expected at least 3 attempts, got %d", hits)
	}
}
