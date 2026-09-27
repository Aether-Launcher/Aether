package logger

import (
	"fmt"
	"testing"
	"time"
)

func TestStdoutCapture(t *testing.T) {
	fmt.Printf("[TestTarget] Hello from stdout capture!\n")
	time.Sleep(50 * time.Millisecond)

	entries := GetEntries()
	found := false
	for _, e := range entries {
		if e.Target == "TestTarget" && e.Message == "Hello from stdout capture!" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("Expected log entry with Target TestTarget, got %+v", entries)
	}
}
