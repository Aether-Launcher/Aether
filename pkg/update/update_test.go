package update

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAssetNameForPlatformAppImage(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("AppImage asset selection is Linux-specific")
	}
	t.Setenv("APPIMAGE", "/opt/aether/Aether.AppImage")
	if got := assetNameForPlatform(); got != "Aether-linux-amd64.AppImage" {
		t.Fatalf("expected AppImage asset under $APPIMAGE, got %q", got)
	}

	t.Setenv("APPIMAGE", "")
	if got := assetNameForPlatform(); got != "Aether-linux-amd64.tar.gz" {
		t.Fatalf("expected tar.gz asset without $APPIMAGE, got %q", got)
	}
}

func TestVerifyAppImagePayload(t *testing.T) {
	dir := t.TempDir()

	good := filepath.Join(dir, "good.AppImage")
	if err := os.WriteFile(good, []byte{0x7f, 'E', 'L', 'F', 0x02, 0x01, 0x01, 0x00}, 0755); err != nil {
		t.Fatal(err)
	}
	if err := verifyAppImagePayload(good); err != nil {
		t.Fatalf("ELF payload should pass: %v", err)
	}

	bad := filepath.Join(dir, "bad.AppImage")
	if err := os.WriteFile(bad, []byte{0x1f, 0x8b, 0x08, 0x00}, 0644); err != nil {
		t.Fatal(err)
	}
	if err := verifyAppImagePayload(bad); err == nil {
		t.Fatal("gzip payload must be rejected")
	}

	empty := filepath.Join(dir, "empty.AppImage")
	if err := os.WriteFile(empty, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := verifyAppImagePayload(empty); err == nil {
		t.Fatal("empty payload must be rejected")
	}
}

func TestReplaceFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.bin")
	replacement := filepath.Join(dir, "replacement.bin")
	oldContent := []byte("old-binary")
	newContent := []byte("new-binary")

	if err := os.WriteFile(target, oldContent, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(replacement, newContent, 0755); err != nil {
		t.Fatal(err)
	}

	if err := replaceFile(target, replacement); err != nil {
		t.Fatalf("replaceFile failed: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("target missing after replace: %v", err)
	}
	if string(got) != string(newContent) {
		t.Fatalf("target has %q, want %q", got, newContent)
	}
	if _, err := os.Stat(target + ".old"); err == nil {
		t.Fatal(".old sibling should be cleaned up")
	}
}