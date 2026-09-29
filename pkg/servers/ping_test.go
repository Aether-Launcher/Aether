package servers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"
)

// serveOneStatus speaks just enough SLP: reads handshake + status request,
// replies with a fixed status JSON, then answers the ping packet.
func serveOneStatus(t *testing.T, ln net.Listener, statusJSON string) {
	t.Helper()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		// Handshake frame.
		if _, _, err := readFrame(conn); err != nil {
			return
		}
		// Status request frame.
		if _, _, err := readFrame(conn); err != nil {
			return
		}
		body := &bytes.Buffer{}
		writeString(body, statusJSON)
		if err := writePacket(conn, 0x00, body); err != nil {
			return
		}
		// Ping packet: echo payload back.
		id, payload, err := readFrame(conn)
		if err != nil || id != 0x01 {
			return
		}
		_ = writePacket(conn, 0x01, payload)
	}()
}

func TestPingOnline(t *testing.T) {
	status := map[string]any{
		"version": map[string]any{"name": "1.21.1", "protocol": 767},
		"players": map[string]any{"max": 100, "online": 7},
		"description": map[string]any{
			"text":  "Welcome ",
			"extra": []any{map[string]any{"text": "to Test!"}},
		},
	}
	raw, _ := json.Marshal(status)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	serveOneStatus(t, ln, string(raw))

	res, err := Ping(fmt.Sprintf("127.0.0.1:%d", ln.Addr().(*net.TCPAddr).Port))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Online {
		t.Fatal("expected online")
	}
	if res.MOTD != "Welcome to Test!" {
		t.Fatalf("motd: got %q", res.MOTD)
	}
	if res.PlayersOnline != 7 || res.PlayersMax != 100 {
		t.Fatalf("players: %+v", res)
	}
	if res.Version != "1.21.1" || res.Protocol != 767 {
		t.Fatalf("version: %+v", res)
	}
	if res.LatencyMs < 0 || res.LatencyMs > 5000 {
		t.Fatalf("latency out of range: %d", res.LatencyMs)
	}
}

func TestPingStringMOTD(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	serveOneStatus(t, ln, `{"version":{"name":"1.20.1"},"players":{"max":20,"online":0},"description":"plain motd"}`)

	res, err := Ping(fmt.Sprintf("127.0.0.1:%d", ln.Addr().(*net.TCPAddr).Port))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Online || res.MOTD != "plain motd" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestPingOffline(t *testing.T) {
	// Port that nothing listens on (TEST-NET range unroutable is slow;
	// use localhost high port for fast refusal).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	res, err := Ping(fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("unreachable must not error, got: %v", err)
	}
	if res.Online {
		t.Fatal("expected offline")
	}
}

func TestPingBadInput(t *testing.T) {
	for _, in := range []string{"", "   ", "example.com:notaport", "example.com:0", "example.com:99999"} {
		if _, err := Ping(in); err == nil {
			t.Fatalf("expected error for %q", in)
		}
	}
}

func TestFlattenMOTD(t *testing.T) {
	if got := flattenMOTD(nil); got != "" {
		t.Fatalf("nil: got %q", got)
	}
	if got := flattenMOTD("hi"); got != "hi" {
		t.Fatalf("string: got %q", got)
	}
	v := map[string]any{"extra": []any{"a", map[string]any{"text": "b"}}}
	if got := flattenMOTD(v); got != "ab" {
		t.Fatalf("tree: got %q", got)
	}
}
