package instance

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// craftLevelDat builds a minimal gzipped level.dat: root{"Data":
// {"LevelName", "LastPlayed", "GameType"}}. raw=true skips gzip.
func craftLevelDat(t *testing.T, name string, lastPlayed int64, gameType int32, raw bool) []byte {
	t.Helper()
	var buf bytes.Buffer
	putU16 := func(v uint16) { _ = binary.Write(&buf, binary.BigEndian, v) }
	putI32 := func(v int32) { _ = binary.Write(&buf, binary.BigEndian, v) }
	putI64 := func(v int64) { _ = binary.Write(&buf, binary.BigEndian, v) }
	writeStr := func(s string) { putU16(uint16(len(s))); buf.WriteString(s) }

	buf.WriteByte(10) // TAG_Compound root
	writeStr("")
	buf.WriteByte(10) // TAG_Compound "Data"
	writeStr("Data")
	buf.WriteByte(8) // TAG_String "LevelName"
	writeStr("LevelName")
	writeStr(name)
	buf.WriteByte(4) // TAG_Long "LastPlayed"
	writeStr("LastPlayed")
	putI64(lastPlayed)
	buf.WriteByte(3) // TAG_Int "GameType"
	writeStr("GameType")
	putI32(gameType)
	buf.WriteByte(0) // TAG_End (Data)
	buf.WriteByte(0) // TAG_End (root)

	if raw {
		return buf.Bytes()
	}
	var out bytes.Buffer
	zw := gzip.NewWriter(&out)
	if _, err := zw.Write(buf.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestListWorldsReadsLevelDat(t *testing.T) {
	dir := t.TempDir()
	saves := filepath.Join(dir, "saves", "My World")
	if err := os.MkdirAll(saves, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(saves, "level.dat"), craftLevelDat(t, "My World", 12345, 1, false), 0644); err != nil {
		t.Fatal(err)
	}
	// Raw (uncompressed) level.dat + a corrupt world + a stray file.
	rawDir := filepath.Join(dir, "saves", "Old World")
	if err := os.MkdirAll(rawDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rawDir, "level.dat"), craftLevelDat(t, "Old World", 99999, 0, true), 0644); err != nil {
		t.Fatal(err)
	}
	broken := filepath.Join(dir, "saves", "Broken")
	if err := os.MkdirAll(broken, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(broken, "level.dat"), []byte("not nbt"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "saves", "stray.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	worlds, err := listWorldsIn(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(worlds) != 3 {
		t.Fatalf("expected 3 worlds, got %d: %+v", len(worlds), worlds)
	}
	// Sorted by LastPlayed desc: Old World (99999) first.
	if worlds[0].ID != "Old World" || worlds[0].Name != "Old World" || worlds[0].LastPlayed != 99999 || worlds[0].GameMode != 0 {
		t.Fatalf("unexpected first world: %+v", worlds[0])
	}
	if worlds[1].ID != "My World" || worlds[1].Name != "My World" || worlds[1].LastPlayed != 12345 || worlds[1].GameMode != 1 {
		t.Fatalf("unexpected second world: %+v", worlds[1])
	}
	// Corrupt world still listed by folder name.
	if worlds[2].ID != "Broken" || worlds[2].Name != "Broken" {
		t.Fatalf("unexpected third world: %+v", worlds[2])
	}
}

func TestListWorldsNoSaves(t *testing.T) {
	worlds, err := listWorldsIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(worlds) != 0 {
		t.Fatalf("expected no worlds, got %+v", worlds)
	}
}

func TestSupportsQuickPlay(t *testing.T) {
	cases := map[string]bool{
		"1.20": true, "1.20.4": true, "1.21": true, "1.21.1": true, "2.0": true,
		"1.19.4": false, "1.16.5": false, "1.8.9": false,
		"23w14a": false, "": false, "abc": false, "1": false,
	}
	for v, want := range cases {
		if got := SupportsQuickPlay(v); got != want {
			t.Errorf("SupportsQuickPlay(%q) = %v, want %v", v, got, want)
		}
	}
}

func TestValidateLaunchHost(t *testing.T) {
	if _, err := validateLaunchHost(""); err == nil {
		t.Error("empty host must fail")
	}
	if _, err := validateLaunchHost("--evil"); err == nil {
		t.Error("flag-like host must fail")
	}
	if _, err := validateLaunchHost("my server.com"); err == nil {
		t.Error("host with whitespace must fail")
	}
	if h, err := validateLaunchHost("play.example.com"); err != nil || h != "play.example.com" {
		t.Errorf("valid host rejected: %v", h)
	}
}

func TestValidateWorldFolder(t *testing.T) {
	dir := t.TempDir()
	worldDir := filepath.Join(dir, "saves", "Good World")
	if err := os.MkdirAll(worldDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worldDir, "level.dat"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", ".", "..", "../x", "a/b", "-x", "Nope"} {
		if _, err := validateWorldFolder(dir, bad); err == nil {
			t.Errorf("world %q must fail validation", bad)
		}
	}
	if _, err := validateWorldFolder(dir, "Good World"); err != nil {
		t.Errorf("valid world rejected: %v", err)
	}
}
