package servers

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupServersDataDir(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Chdir(tmp)
	if err := os.MkdirAll(filepath.Join(tmp, ".aether"), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestCreateListDeleteServer(t *testing.T) {
	setupServersDataDir(t)

	info, err := CreateServer("Test-Server", "My Test")
	if err != nil {
		t.Fatal(err)
	}
	if info.ID != "test-server" {
		t.Fatalf("id: got %q", info.ID)
	}
	// Idempotent: creating again keeps data.
	if _, err := CreateServer("test-server", "Other"); err != nil {
		t.Fatalf("re-create: %v", err)
	}

	list, err := ListServers()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "test-server" {
		t.Fatalf("list: %+v", list)
	}

	// Starter server.properties exists.
	props, err := ReadServerFile("test-server", "server.properties")
	if err != nil {
		t.Fatalf("default properties: %v", err)
	}
	if !strings.Contains(props, "server-port=25565") {
		t.Fatalf("default properties missing port: %q", props)
	}

	if err := DeleteServer("test-server"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteServer("test-server"); err == nil {
		t.Fatal("expected error deleting missing server")
	}
	list, err = ListServers()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %+v", list)
	}
}

func TestCreateServerBadID(t *testing.T) {
	setupServersDataDir(t)
	for _, bad := range []string{"", "../escape", "has space!", "UPPER ok? no!"} {
		if _, err := CreateServer(bad, "x"); err == nil {
			t.Fatalf("expected error for id %q", bad)
		}
	}
}

func TestWriteReadServerFile(t *testing.T) {
	setupServersDataDir(t)

	if _, err := CreateServer("files", "Files"); err != nil {
		t.Fatal(err)
	}
	content := base64.StdEncoding.EncodeToString([]byte("motd=hello\n"))
	if err := WriteServerFile("files", "config/custom.properties", content); err != nil {
		t.Fatal(err)
	}
	got, err := ReadServerFile("files", "config/custom.properties")
	if err != nil {
		t.Fatal(err)
	}
	if got != "motd=hello\n" {
		t.Fatalf("roundtrip: got %q", got)
	}
	// Traversal blocked.
	if err := WriteServerFile("files", "../../evil.txt", content); err == nil {
		t.Fatal("expected traversal rejection")
	}
	if _, err := ReadServerFile("files", "../../evil.txt"); err == nil {
		t.Fatal("expected traversal rejection on read")
	}
	// Oversize rejected.
	big := base64.StdEncoding.EncodeToString(make([]byte, MaxServerFileBytes+1))
	if err := WriteServerFile("files", "big.bin", big); err == nil {
		t.Fatal("expected oversize rejection")
	}
	// Missing file.
	if _, err := ReadServerFile("files", "nope.txt"); err == nil {
		t.Fatal("expected missing-file error")
	}
}

func TestReadInstanceServersMissing(t *testing.T) {
	setupServersDataDir(t)
	entries, err := ReadInstanceServers("does-not-exist")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty, got %+v", entries)
	}
}
