package logger

import (
	"fmt"
	"testing"
	"time"
)

func TestStdoutCapture(t *testing.T) {
	fmt.Printf("[TestTarget] Hello from stdout capture!\n")

	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		for _, e := range GetEntries() {
			if e.Target == "TestTarget" && e.Message == "Hello from stdout capture!" {
				return
			}
		}

		select {
		case <-deadline:
			t.Fatal("Timed out waiting for stdout capture")
		case <-ticker.C:
		}
	}
}
