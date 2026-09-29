package servers

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// Minimal NBT writer for fixtures (big-endian, mirrors the reader).
type writer struct {
	buf bytes.Buffer
}

func (w *writer) u8(v byte)               { w.buf.WriteByte(v) }
func (w *writer) u16(v uint16)            { binary.Write(&w.buf, binary.BigEndian, v) }
func (w *writer) i32(v int32)             { binary.Write(&w.buf, binary.BigEndian, v) }
func (w *writer) str(s string)            { w.u16(uint16(len(s))); w.buf.WriteString(s) }
func (w *writer) named(tag byte, name string) {
	w.u8(tag)
	w.str(name)
}

func fixtureServersDat(t *testing.T) []byte {
	t.Helper()
	// List elements carry no tags/names, just compound bodies.
	w2 := &writer{}
	w2.u8(tagCompound)
	w2.str("")
	w2.named(tagList, "servers")
	w2.u8(tagCompound)
	w2.i32(2)
	// entry 1: full
	w2.named(tagString, "name")
	w2.str("Hypixel")
	w2.named(tagString, "ip")
	w2.str("mc.hypixel.net")
	w2.named(tagByte, "hidden")
	w2.u8(0)
	w2.named(tagByteArray, "icon")
	w2.i32(3)
	w2.buf.Write([]byte{1, 2, 3})
	w2.u8(tagEnd)
	// entry 2: hidden, no icon
	w2.named(tagString, "name")
	w2.str("Local Test")
	w2.named(tagString, "ip")
	w2.str("localhost:25565")
	w2.named(tagByte, "hidden")
	w2.u8(1)
	w2.u8(tagEnd)
	w2.u8(tagEnd)
	return w2.buf.Bytes()
}

func TestParseServersDat(t *testing.T) {
	entries, err := ParseServersDat(fixtureServersDat(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "Hypixel" || entries[0].IP != "mc.hypixel.net" {
		t.Fatalf("entry 0 mismatch: %+v", entries[0])
	}
	if !entries[0].HasIcon {
		t.Fatal("entry 0 should report HasIcon")
	}
	if entries[0].Hidden {
		t.Fatal("entry 0 should not be hidden")
	}
	if entries[1].Name != "Local Test" || !entries[1].Hidden || entries[1].HasIcon {
		t.Fatalf("entry 1 mismatch: %+v", entries[1])
	}
}

func TestParseServersDatEmpty(t *testing.T) {
	if _, err := ParseServersDat(nil); err == nil {
		t.Fatal("expected error for empty input")
	}
	if _, err := ParseServersDat([]byte{tagByte, 0, 0}); err == nil {
		t.Fatal("expected error for non-compound root")
	}
	// Compound without a servers list is valid with zero entries.
	w := &writer{}
	w.u8(tagCompound)
	w.str("")
	w.named(tagString, "foo")
	w.str("bar")
	w.u8(tagEnd)
	entries, err := ParseServersDat(w.buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestParseServersDatTruncated(t *testing.T) {
	full := fixtureServersDat(t)
	for _, cut := range []int{1, 5, 20, len(full) - 3} {
		if _, err := ParseServersDat(full[:cut]); err == nil {
			t.Fatalf("expected error for %d-byte truncation", cut)
		}
	}
}
