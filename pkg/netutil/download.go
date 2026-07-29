package netutil

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Shared client. No global timeout so large downloads can take as long as needed.
// Timeout is enforced by the ResponseHeaderTimeout and Context.
var defaultClient = &http.Client{
	Transport: &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       30 * time.Second,
	},
}

// ProgressCallback is called periodically with the downloaded bytes and total bytes.
type ProgressCallback func(downloaded, total int64)

// DownloadFile downloads a file from url to dest, with support for resuming (Range requests)
// and exponential backoff retries. If expectedSha1 is non-empty, the downloaded file's
// SHA1 is verified and the file is deleted and an error returned on mismatch.
func DownloadFile(ctx context.Context, url string, dest string, onProgress ProgressCallback, expectedSha1 ...string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	// If the file already exists, verify its checksum before skipping.
	if _, err := os.Stat(dest); err == nil {
		if len(expectedSha1) > 0 && expectedSha1[0] != "" {
			if ok, _ := verifySha1(dest, expectedSha1[0]); !ok {
				// Stale or corrupt — remove and re-download
				_ = os.Remove(dest)
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	tempDest := dest + ".tmp"
	maxRetries := 5
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		err := func() error {
			var startBytes int64 = 0

			// Check if temp file exists to resume
			if info, err := os.Stat(tempDest); err == nil {
				startBytes = info.Size()
			}

			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return err
			}

			if startBytes > 0 {
				req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startBytes))
			}

			resp, err := defaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var out *os.File

			// If server supports range requests, it returns 206 Partial Content
			if resp.StatusCode == http.StatusPartialContent {
				out, err = os.OpenFile(tempDest, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return err
				}
			} else if resp.StatusCode == http.StatusOK {
				// Server didn't respect Range or we didn't send it, start from scratch
				startBytes = 0
				out, err = os.Create(tempDest)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("unexpected status %d", resp.StatusCode)
			}
			defer out.Close()

			var totalBytes int64 = resp.ContentLength
			if totalBytes > 0 {
				totalBytes += startBytes // True total size
			}

			// Copy with progress reporting
			buf := make([]byte, 32*1024)
			var written int64 = startBytes

			for {
				nr, readErr := resp.Body.Read(buf)
				if nr > 0 {
					nw, writeErr := out.Write(buf[:nr])
					if nw > 0 {
						written += int64(nw)
						if onProgress != nil {
							onProgress(written, totalBytes)
						}
					}
					if writeErr != nil {
						return writeErr
					}
				}
				if readErr == io.EOF {
					break
				}
				if readErr != nil {
					return readErr
				}
			}

			return nil
		}()

		if err == nil {
			if err := os.Rename(tempDest, dest); err != nil {
				return fmt.Errorf("failed to rename temp file: %w", err)
			}

			// Verify checksum after successful download
			if len(expectedSha1) > 0 && expectedSha1[0] != "" {
				if ok, gotHash := verifySha1(dest, expectedSha1[0]); !ok {
					_ = os.Remove(dest)
					return fmt.Errorf("checksum mismatch for %s: expected %s got %s", filepath.Base(dest), expectedSha1[0], gotHash)
				}
			}

			return nil
		}

		lastErr = err

		// Exponential backoff: 1s, 2s, 4s, 8s
		select {
		case <-ctx.Done():
			return ctx.Err() // Abort immediately if cancelled
		case <-time.After(time.Duration(1<<i) * time.Second):
			// Retry after delay
		}
	}

	return fmt.Errorf("failed to download after %d retries, last error: %w", maxRetries, lastErr)
}

// verifySha1 computes the SHA1 of the file at path and compares it to expected.
// Returns (true, actualHash) on match, (false, actualHash) on mismatch or error.
func verifySha1(path string, expected string) (bool, string) {
	f, err := os.Open(path)
	if err != nil {
		return false, ""
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, ""
	}
	actual := hex.EncodeToString(h.Sum(nil))
	return actual == expected, actual
}
