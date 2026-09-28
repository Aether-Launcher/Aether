package servers

// Server list ping (modern handshake, MC 1.7+). Speaks just enough of the
// status protocol to fetch MOTD/players/version: handshake + status request,
// optional ping packet for latency. No legacy (pre-1.7) support.

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// PingResult is the outcome of pinging one Minecraft server.
// Unreachable servers yield Online=false and a nil error.
type PingResult struct {
	Online        bool   `json:"online"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	MOTD          string `json:"motd,omitempty"`
	PlayersOnline int    `json:"playersOnline,omitempty"`
	PlayersMax    int    `json:"playersMax,omitempty"`
	Version       string `json:"version,omitempty"`
	Protocol      int    `json:"protocol,omitempty"`
	LatencyMs     int64  `json:"latencyMs,omitempty"`
}

func writeVarInt(buf *bytes.Buffer, v int) {
	for {
		b := byte(v & 0x7F)
		v >>= 7
		if v != 0 {
			b |= 0x80
		}
		buf.WriteByte(b)
		if v == 0 {
			return
		}
	}
}

func writeString(buf *bytes.Buffer, s string) {
	writeVarInt(buf, len(s))
	buf.WriteString(s)
}

func writePacket(conn net.Conn, id int, payload *bytes.Buffer) error {
	body := &bytes.Buffer{}
	writeVarInt(body, id)
	if payload != nil {
		body.Write(payload.Bytes())
	}
	frame := &bytes.Buffer{}
	writeVarInt(frame, body.Len())
	frame.Write(body.Bytes())
	_, err := conn.Write(frame.Bytes())
	return err
}

func readVarInt(r io.Reader) (int, error) {
	v := 0
	for i := 0; i < 5; i++ {
		var b [1]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return 0, err
		}
		v |= int(b[0]&0x7F) << (7 * i)
		if b[0]&0x80 == 0 {
			return v, nil
		}
	}
	return 0, fmt.Errorf("slp: varint too long")
}

func readString(r io.Reader) (string, error) {
	n, err := readVarInt(r)
	if err != nil {
		return "", err
	}
	if n < 0 || n > 4*1024*1024 {
		return "", fmt.Errorf("slp: absurd string length %d", n)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func readFrame(r io.Reader) (id int, payload *bytes.Buffer, err error) {
	length, err := readVarInt(r)
	if err != nil {
		return 0, nil, err
	}
	if length <= 0 || length > 4*1024*1024 {
		return 0, nil, fmt.Errorf("slp: absurd frame length %d", length)
	}
	raw := make([]byte, length)
	if _, err := io.ReadFull(r, raw); err != nil {
		return 0, nil, err
	}
	br := bytes.NewReader(raw)
	id, err = readVarInt(br)
	if err != nil {
		return 0, nil, err
	}
	rest, err := io.ReadAll(br)
	if err != nil {
		return 0, nil, err
	}
	return id, bytes.NewBuffer(rest), nil
}

// flattenMOTD extracts plain text from a MOTD that may be a string,
// {"text":...}, or {"extra":[...]} chat component tree.
func flattenMOTD(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case map[string]any:
		var sb strings.Builder
		if text, ok := t["text"].(string); ok {
			sb.WriteString(text)
		}
		if extra, ok := t["extra"].([]any); ok {
			for _, e := range extra {
				sb.WriteString(flattenMOTD(e))
			}
		}
		return sb.String()
	case []any:
		var sb strings.Builder
		for _, e := range t {
			sb.WriteString(flattenMOTD(e))
		}
		return sb.String()
	default:
		return ""
	}
}

func splitHostPort(hostport string) (string, int, error) {
	hostport = strings.TrimSpace(hostport)
	if hostport == "" {
		return "", 0, fmt.Errorf("slp: empty host")
	}
	host, port := hostport, 25565
	if i := strings.LastIndex(hostport, ":"); i >= 0 {
		host = hostport[:i]
		var p int
		if _, err := fmt.Sscanf(hostport[i+1:], "%d", &p); err != nil || p <= 0 || p > 65535 {
			return "", 0, fmt.Errorf("slp: invalid port in %q", hostport)
		}
		port = p
	}
	host = strings.Trim(host, "[]")
	if host == "" {
		return "", 0, fmt.Errorf("slp: empty host")
	}
	return host, port, nil
}

// Ping queries a Minecraft server's status. Unreachable/refusing servers
// yield Online=false with a nil error; only invalid input errors.
func Ping(hostport string) (PingResult, error) {
	host, port, err := splitHostPort(hostport)
	if err != nil {
		return PingResult{}, err
	}
	res := PingResult{Online: false, Host: host, Port: port}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
	if err != nil {
		return res, nil
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(8 * time.Second))

	// Handshake (next state: status).
	hs := &bytes.Buffer{}
	writeVarInt(hs, 47) // protocol version; ignored for status
	writeString(hs, host)
	var portBytes [2]byte
	binary.BigEndian.PutUint16(portBytes[:], uint16(port))
	hs.Write(portBytes[:])
	writeVarInt(hs, 1)
	if err := writePacket(conn, 0x00, hs); err != nil {
		return res, nil
	}
	// Status request.
	if err := writePacket(conn, 0x00, nil); err != nil {
		return res, nil
	}
	id, payload, err := readFrame(conn)
	if err != nil || id != 0x00 {
		return res, nil
	}
	text, err := readString(bytes.NewReader(payload.Bytes()))
	if err != nil {
		return res, nil
	}
	var status struct {
		Version struct {
			Name     string `json:"name"`
			Protocol int    `json:"protocol"`
		} `json:"version"`
		Players struct {
			Max    int `json:"max"`
			Online int `json:"online"`
		} `json:"players"`
		Description any `json:"description"`
	}
	if err := json.Unmarshal([]byte(text), &status); err != nil {
		return res, nil
	}
	res.Online = true
	res.MOTD = flattenMOTD(status.Description)
	res.PlayersOnline = status.Players.Online
	res.PlayersMax = status.Players.Max
	res.Version = status.Version.Name
	res.Protocol = status.Version.Protocol

	// Ping packet for latency (best-effort).
	start := time.Now()
	pingPayload := &bytes.Buffer{}
	var ts [8]byte
	binary.BigEndian.PutUint64(ts[:], uint64(start.UnixNano()/1e6))
	pingPayload.Write(ts[:])
	if err := writePacket(conn, 0x01, pingPayload); err == nil {
		if pid, _, err := readFrame(conn); err == nil && pid == 0x01 {
			res.LatencyMs = time.Since(start).Milliseconds()
		}
	}
	return res, nil
}
