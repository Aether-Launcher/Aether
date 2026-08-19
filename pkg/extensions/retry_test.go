package extensions

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// TestHTTPGetWithRetryTransient verifies that transient network failures are
// retried and a success after retries is returned.
func TestHTTPGetWithRetryTransient(t *testing.T) {
	var calls int32
	http.DefaultClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			return nil, fmt.Errorf("Get %q: dial tcp: lookup meta.fabricmc.net: Temporary failure in name resolution", r.URL.String())
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       http.NoBody,
			Header:     make(http.Header),
		}, nil
	})
	defer func() { http.DefaultClient.Transport = http.DefaultTransport }()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := httpGetWithRetry(context.Background(), req)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	resp.Body.Close()
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

// TestHTTPGetWithRetryConnectionErrors verifies transient connection-level
// errors (connection refused) are retried up to the attempt limit.
func TestHTTPGetWithRetryConnectionErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // closed listener -> connection refused

	var calls int32
	base := http.DefaultClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	http.DefaultClient.Transport = roundTripCounter{base: base, calls: &calls}
	defer func() { http.DefaultClient.Transport = base }()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := httpGetWithRetry(context.Background(), req); err == nil {
		t.Fatal("expected error for connection-refused server")
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("expected 3 retry attempts, got %d", got)
	}
}

// TestHTTPGetWithRetryNonTransient verifies permanent failures (NXDOMAIN)
// are not retried.
func TestHTTPGetWithRetryNonTransient(t *testing.T) {
	var calls int32
	http.DefaultClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return nil, fmt.Errorf("Get %q: lookup example.invalid: no such host", r.URL.String())
	})
	defer func() { http.DefaultClient.Transport = http.DefaultTransport }()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.invalid/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := httpGetWithRetry(context.Background(), req); err == nil {
		t.Fatal("expected error")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected no retry for non-transient errors, got %d attempts", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type roundTripCounter struct {
	base  http.RoundTripper
	calls *int32
}

func (c roundTripCounter) RoundTrip(r *http.Request) (*http.Response, error) {
	atomic.AddInt32(c.calls, 1)
	return c.base.RoundTrip(r)
}