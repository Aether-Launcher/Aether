package servers

// Minimal NBT reader (big-endian, full tag set) for parsing servers.dat.
// Only the structures Vanilla Minecraft writes are exercised, but all tag
// types decode so unknown fields don't break parsing.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	tagEnd       = 0
	tagByte      = 1
	tagShort     = 2
	tagInt       = 3
	tagLong      = 4
	tagFloat     = 5
	tagDouble    = 6
	tagByteArray = 7
	tagString    = 8
	tagList      = 9
	tagCompound  = 10
	tagIntArray  = 11
	tagLongArray = 12
)

// ServerEntry is one server from a servers.dat server list.
type ServerEntry struct {
	Name    string `json:"name"`
	IP      string `json:"ip"`
	Hidden  bool   `json:"hidden,omitempty"`
	HasIcon bool   `json:"hasIcon,omitempty"`
}

type reader struct {
	r *bytes.Reader
}

func (n *reader) u8() (byte, error) {
	b, err := n.r.ReadByte()
	return b, err
}

func (n *reader) u16() (uint16, error) {
	var v uint16
	err := binary.Read(n.r, binary.BigEndian, &v)
	return v, err
}

func (n *reader) i32() (int32, error) {
	var v int32
	err := binary.Read(n.r, binary.BigEndian, &v)
	return v, err
}

func (n *reader) i64() (int64, error) {
	var v int64
	err := binary.Read(n.r, binary.BigEndian, &v)
	return v, err
}

func (n *reader) f32() (float32, error) {
	var v float32
	err := binary.Read(n.r, binary.BigEndian, &v)
	return v, err
}

func (n *reader) f64() (float64, error) {
	var v float64
	err := binary.Read(n.r, binary.BigEndian, &v)
	return v, err
}

func (n *reader) str() (string, error) {
	length, err := n.u16()
	if err != nil {
		return "", err
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(n.r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func (n *reader) skipPayload(tag byte) error {
	switch tag {
	case tagEnd:
		return nil
	case tagByte:
		_, err := n.u8()
		return err
	case tagShort:
		_, err := n.u16()
		return err
	case tagInt:
		_, err := n.i32()
		return err
	case tagLong:
		_, err := n.i64()
		return err
	case tagFloat:
		_, err := n.f32()
		return err
	case tagDouble:
		_, err := n.f64()
		return err
	case tagByteArray:
		length, err := n.i32()
		if err != nil {
			return err
		}
		if length < 0 || length > 64*1024*1024 {
			return fmt.Errorf("nbt: absurd byte array length %d", length)
		}
		_, err = n.r.Seek(int64(length), io.SeekCurrent)
		return err
	case tagString:
		_, err := n.str()
		return err
	case tagList:
		elem, err := n.u8()
		if err != nil {
			return err
		}
		length, err := n.i32()
		if err != nil {
			return err
		}
		if length < 0 || length > 1024*1024 {
			return fmt.Errorf("nbt: absurd list length %d", length)
		}
		for i := int32(0); i < length; i++ {
			if err := n.skipPayload(elem); err != nil {
				return err
			}
		}
		return nil
	case tagCompound:
		for {
			t, err := n.u8()
			if err != nil {
				return err
			}
			if t == tagEnd {
				return nil
			}
			if _, err := n.str(); err != nil {
				return err
			}
			if err := n.skipPayload(t); err != nil {
				return err
			}
		}
	case tagIntArray:
		length, err := n.i32()
		if err != nil {
			return err
		}
		if length < 0 || length > 16*1024*1024 {
			return fmt.Errorf("nbt: absurd int array length %d", length)
		}
		_, err = n.r.Seek(int64(length)*4, io.SeekCurrent)
		return err
	case tagLongArray:
		length, err := n.i32()
		if err != nil {
			return err
		}
		if length < 0 || length > 16*1024*1024 {
			return fmt.Errorf("nbt: absurd long array length %d", length)
		}
		_, err = n.r.Seek(int64(length)*8, io.SeekCurrent)
		return err
	default:
		return fmt.Errorf("nbt: unknown tag %d", tag)
	}
}

// readCompound parses a TAG_Compound payload into a generic map. Values are
// Go primitives, []any for lists, and map[string]any for nested compounds.
func (n *reader) readValue(tag byte) (any, error) {
	switch tag {
	case tagByte:
		return n.u8()
	case tagShort:
		return n.u16()
	case tagInt:
		return n.i32()
	case tagLong:
		return n.i64()
	case tagFloat:
		return n.f32()
	case tagDouble:
		return n.f64()
	case tagString:
		return n.str()
	case tagByteArray: {
		length, err := n.i32()
		if err != nil {
			return nil, err
		}
		if length < 0 || length > 64*1024*1024 {
			return nil, fmt.Errorf("nbt: absurd byte array length %d", length)
		}
		buf := make([]byte, length)
		if _, err := io.ReadFull(n.r, buf); err != nil {
			return nil, err
		}
		return buf, nil
	}
	case tagList: {
		elem, err := n.u8()
		if err != nil {
			return nil, err
		}
		length, err := n.i32()
		if err != nil {
			return nil, err
		}
		if length < 0 || length > 1024*1024 {
			return nil, fmt.Errorf("nbt: absurd list length %d", length)
		}
		out := make([]any, 0, length)
		for i := int32(0); i < length; i++ {
			if elem == tagCompound {
				m, err := n.readCompound()
				if err != nil {
					return nil, err
				}
				out = append(out, m)
				continue
			}
			v, err := n.readValue(elem)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	}
	case tagCompound:
		return n.readCompound()
	case tagIntArray: {
		length, err := n.i32()
		if err != nil {
			return nil, err
		}
		out := make([]int32, 0, length)
		for i := int32(0); i < length; i++ {
			v, err := n.i32()
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	}
	case tagLongArray: {
		length, err := n.i32()
		if err != nil {
			return nil, err
		}
		out := make([]int64, 0, length)
		for i := int32(0); i < length; i++ {
			v, err := n.i64()
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	}
	default:
		return nil, fmt.Errorf("nbt: unknown tag %d", tag)
	}
}

func (n *reader) readCompound() (map[string]any, error) {
	out := map[string]any{}
	for {
		t, err := n.u8()
		if err != nil {
			return nil, err
		}
		if t == tagEnd {
			return out, nil
		}
		name, err := n.str()
		if err != nil {
			return nil, err
		}
		v, err := n.readValue(t)
		if err != nil {
			return nil, err
		}
		out[name] = v
	}
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// ParseServersDat parses raw servers.dat bytes into server entries.
func ParseServersDat(data []byte) ([]ServerEntry, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("nbt: empty input")
	}
	if len(data) > 4*1024*1024 {
		return nil, fmt.Errorf("nbt: servers.dat exceeds 4 MiB")
	}
	n := &reader{r: bytes.NewReader(data)}
	rootType, err := n.u8()
	if err != nil {
		return nil, fmt.Errorf("nbt: %w", err)
	}
	if rootType != tagCompound {
		return nil, fmt.Errorf("nbt: root must be a compound, got tag %d", rootType)
	}
	if _, err := n.str(); err != nil { // root name (usually empty)
		return nil, fmt.Errorf("nbt: %w", err)
	}
	root, err := n.readCompound()
	if err != nil {
		return nil, fmt.Errorf("nbt: %w", err)
	}
	rawList, ok := root["servers"].([]any)
	if !ok {
		return nil, nil // no server list — not an error
	}
	var out []ServerEntry
	for _, item := range rawList {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := asString(m["name"])
		ip := asString(m["ip"])
		if name == "" && ip == "" {
			continue
		}
		entry := ServerEntry{Name: name, IP: ip}
		if b, ok := m["hidden"].(byte); ok && b != 0 {
			entry.Hidden = true
		}
		if _, ok := m["icon"].([]byte); ok {
			entry.HasIcon = true
		}
		out = append(out, entry)
	}
	return out, nil
}
