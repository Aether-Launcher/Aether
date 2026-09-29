package servers

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// craftServersDat builds a minimal servers.dat: root{"servers":
// [{name, ip}, ...]}.
func craftServersDat(entries [][2]string) []byte {
	var buf bytes.Buffer
	putU16 := func(v uint16) { _ = binary.Write(&buf, binary.BigEndian, v) }
	putI32 := func(v int32) { _ = binary.Write(&buf, binary.BigEndian, v) }
	writeStr := func(s string) { putU16(uint16(len(s))); buf.WriteString(s) }
	buf.WriteByte(10) // root compound
	writeStr("")
	buf.WriteByte(9) // TAG_List "servers"
	writeStr("servers")
	buf.WriteByte(10) // element type: compound
	putI32(int32(len(entries)))
	for _, e := range entries {
		buf.WriteByte(8) // TAG_String "name"
		writeStr("name")
		writeStr(e[0])
		buf.WriteByte(8) // TAG_String "ip"
		writeStr("ip")
		writeStr(e[1])
		buf.WriteByte(0) // TAG_End
	}
	buf.WriteByte(0) // TAG_End (root)
	return buf.Bytes()
}

func TestListServersWithStatusOffline(t *testing.T) {
	dir := t.TempDir()
	dat := filepath.Join(dir, "servers.dat")
	// Closed ports on loopback fail fast (refused, no DNS, no timeout wait).
	content := craftServersDat([][2]string{{"Alpha", "127.0.0.1:1"}, {"Beta", "127.0.0.1:2"}})
	if err := os.WriteFile(dat, content, 0644); err != nil {
		t.Fatal(err)
	}
	rows, err := listServersWithStatusIn(dat, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %+v", rows)
	}
	// Order must match servers.dat regardless of ping completion order.
	if rows[0].Name != "Alpha" || rows[0].IP != "127.0.0.1:1" {
		t.Errorf("unexpected first row: %+v", rows[0])
	}
	if rows[1].Name != "Beta" || rows[1].IP != "127.0.0.1:2" {
		t.Errorf("unexpected second row: %+v", rows[1])
	}
	for i, r := range rows {
		if r.Online {
			t.Errorf("row %d: loopback closed port must be offline", i)
		}
		if r.Host != "127.0.0.1" {
			t.Errorf("row %d: host not parsed: %+v", i, r)
		}
	}
	if rows[0].Port != 1 || rows[1].Port != 2 {
		t.Errorf("ports not parsed: %+v", rows)
	}
}

func TestListServersWithStatusMissing(t *testing.T) {
	rows, err := listServersWithStatusIn(filepath.Join(t.TempDir(), "servers.dat"), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected no rows, got %+v", rows)
	}
}

func TestListServersWithStatusCache(t *testing.T) {
	InvalidateStatusCache()
	dir := t.TempDir()
	dat := filepath.Join(dir, "servers.dat")
	if err := os.WriteFile(dat, craftServersDat([][2]string{{"One", "127.0.0.1:1"}}), 0644); err != nil {
		t.Fatal(err)
	}
	first, err := listServersWithStatusIn(dat, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || first[0].Name != "One" {
		t.Fatalf("unexpected first fetch: %+v", first)
	}
	// Rewrite with different content (different size forces invalidation even
	// if the mtime granularity collides).
	if err := os.WriteFile(dat, craftServersDat([][2]string{{"TwoLonger", "127.0.0.1:3"}}), 0644); err != nil {
		t.Fatal(err)
	}
	second, err := listServersWithStatusIn(dat, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 || second[0].Name != "TwoLonger" {
		t.Fatalf("cache did not invalidate on rewrite: %+v", second)
	}
	// Expired TTL must re-fetch (exercises the expiry branch without sleeping).
	oldTTL := statusCacheTTL
	statusCacheTTL = -time.Second
	defer func() { statusCacheTTL = oldTTL }()
	third, err := listServersWithStatusIn(dat, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(third) != 1 || third[0].Name != "TwoLonger" {
		t.Fatalf("expired cache did not re-fetch: %+v", third)
	}
	InvalidateStatusCache()
}
