package servers

// Supervised game-server processes for the servers:process capability.
// The launcher owns the child process lifecycle: extensions can start, stop,
// query, and send console input, but never receive handles, PIDs for reuse,
// or shell access. Output streams into a ring buffer and a server:log event.

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"Aether/pkg/fs"
	"Aether/pkg/java"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ErrEULARequired is returned by StartServer when eula.txt does not accept
// the Mojang EULA. Callers should surface a confirmation dialog and then
// call SetEulaAccepted — never accept silently.
var ErrEULARequired = errors.New("server EULA not accepted (eula.txt)")

// MaxConcurrentServers caps simultaneously running managed servers.
const MaxConcurrentServers = 2

// DefaultServerMemoryMB applies when the caller passes no memory.
const DefaultServerMemoryMB = 2048

const maxServerMemoryMB = 16384
const minServerMemoryMB = 512
const maxLogLines = 500
const stopGracePeriod = 10 * time.Second

// ServerStatus describes a managed server process.
type ServerStatus struct {
	ID        string `json:"id"`
	Running   bool   `json:"running"`
	PID       int    `json:"pid,omitempty"`
	StartedAt int64  `json:"startedAt,omitempty"`
	Port      int    `json:"port,omitempty"`
	MCVersion string `json:"mcVersion,omitempty"`
}

// StartOptions tunes a server launch. Zero values get sane defaults.
type StartOptions struct {
	MCVersion string
	MemoryMB  int
	JarName   string
	ExtraArgs []string
}

type runningServer struct {
	ctx       context.Context
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	startedAt time.Time
	port      int
	mcVersion string
	logs      []string
	done      chan struct{}
}

var (
	procMu  sync.Mutex
	running = map[string]*runningServer{}
)

// emitEvent broadcasts to the frontend; replaceable in tests.
// The context must be a live Wails context (app/request scope) —
// context.Background() is rejected by the runtime and drops events.
var emitEvent = func(ctx context.Context, event string, data map[string]any) {
	runtime.EventsEmit(ctx, event, data)
}

func emitState(ctx context.Context, id, state string) {
	emitEvent(ctx, "server:state", map[string]any{"id": id, "state": state})
}

func emitLog(ctx context.Context, id, line string) {
	emitEvent(ctx, "server:log", map[string]any{"id": id, "line": line})
}

// jarVersionRe extracts an MC version from server jar names
// (paper-1.21.1-..., purpur-1.20.4-...).
var jarVersionRe = regexp.MustCompile(`(\d{1,3}\.\d{1,3}(?:\.\d{1,3})?)`)

// requiredJavaVersion maps an MC version to a Java major, handling both the
// legacy 1.x scheme and the newer year-based scheme (e.g. "26.1").
func requiredJavaVersion(mc string) int {
	if parts := strings.Split(mc, "."); len(parts) >= 1 {
		if major, err := strconv.Atoi(parts[0]); err == nil && major >= 21 {
			return 21
		}
	}
	return java.RequiredJavaVersion(mc)
}

// resolveJava prefers a managed JRE, then a system Java, then downloads one.
func resolveJava(ctx context.Context, major int) (string, error) {
	if java.IsManagedJavaInstalled(major) {
		return java.GetManagedJavaPath(major), nil
	}
	if sysPath, err := java.FindJava(major); err == nil {
		return sysPath, nil
	}
	if err := java.DownloadJava(ctx, major); err != nil {
		return "", fmt.Errorf("no Java %d available and download failed: %w", major, err)
	}
	return java.GetManagedJavaPath(major), nil
}

// readServerPort parses server-port from server.properties (default 25565).
// Later lines win, matching Java properties semantics.
func readServerPort(dir string) int {
	data, err := os.ReadFile(filepath.Join(dir, "server.properties"))
	if err != nil {
		return 25565
	}
	port := 25565
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok && strings.TrimSpace(k) == "server-port" {
			if p, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && p > 0 && p <= 65535 {
				port = p
			}
		}
	}
	return port
}

func portInUse(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return true
	}
	_ = ln.Close()
	return false
}

// findServerJar picks the server jar: explicit name first, then paper-*,
// purpur-*, server.jar. Source/distribution jars are skipped.
func findServerJar(dir, explicit string) (string, error) {
	if explicit != "" {
		p, err := fs.ContainedPath(dir, explicit)
		if err != nil {
			return "", err
		}
		if st, err := os.Stat(p); err != nil || st.IsDir() {
			return "", fmt.Errorf("server jar %q not found", explicit)
		}
		if !strings.HasSuffix(strings.ToLower(p), ".jar") {
			return "", fmt.Errorf("server jar %q must be a .jar file", explicit)
		}
		return p, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var fallback string
	score := func(name string) int {
		lower := strings.ToLower(name)
		if !strings.HasSuffix(lower, ".jar") {
			return -1
		}
		if strings.Contains(lower, "sources") || strings.Contains(lower, "javadoc") {
			return -1
		}
		switch {
		case strings.HasPrefix(lower, "paper-"):
			return 3
		case strings.HasPrefix(lower, "purpur-"):
			return 3
		case lower == "server.jar":
			return 2
		default:
			return 0
		}
	}
	best := -1
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if s := score(e.Name()); s > best {
			best = s
			fallback = e.Name()
		}
	}
	if best < 0 {
		return "", fmt.Errorf("no server jar found in %s (expected paper-*.jar, purpur-*.jar, or server.jar)", dir)
	}
	return filepath.Join(dir, fallback), nil
}

// mcVersionFromJar extracts an MC version from a server jar filename.
func mcVersionFromJar(jarPath string) string {
	if m := jarVersionRe.FindStringSubmatch(filepath.Base(jarPath)); m != nil {
		return m[1]
	}
	return ""
}

// eulaAccepted reports whether eula.txt contains eula=true.
func eulaAccepted(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "eula.txt"))
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok &&
			strings.EqualFold(strings.TrimSpace(k), "eula") {
			return strings.EqualFold(strings.TrimSpace(v), "true")
		}
	}
	return false
}

// EulaAccepted reports whether the server's eula.txt accepts the EULA.
func EulaAccepted(id string) (bool, error) {
	dir, err := serverDir(id)
	if err != nil {
		return false, err
	}
	return eulaAccepted(dir), nil
}

// SetEulaAccepted writes eula.txt with eula=true. Call only after the user
// explicitly confirmed acceptance through the launcher UI.
func SetEulaAccepted(id string) error {
	dir, err := serverDir(id)
	if err != nil {
		return err
	}
	content := "# By changing the setting below to TRUE you are indicating your agreement to the EULA (https://aka.ms/MinecraftEULA).\n" +
		"# Accepted via Aether on " + time.Now().UTC().Format(time.RFC3339) + "\n" +
		"eula=true\n"
	return os.WriteFile(filepath.Join(dir, "eula.txt"), []byte(content), 0644)
}

// newServerCommand builds the java invocation. Overridden in tests.
var newServerCommand = func(javaBin string, memoryMB int, extraArgs []string, jarPath, dir string) *exec.Cmd {
	args := []string{
		fmt.Sprintf("-Xmx%dM", memoryMB),
		fmt.Sprintf("-Xms%dM", memoryMB),
	}
	args = append(args, extraArgs...)
	args = append(args, "-jar", jarPath, "--nogui")
	cmd := exec.Command(javaBin, args...)
	cmd.Dir = dir
	hideConsole(cmd)
	return cmd
}

func clampMemory(mb int) int {
	if mb <= 0 {
		return DefaultServerMemoryMB
	}
	if mb < minServerMemoryMB {
		return minServerMemoryMB
	}
	if mb > maxServerMemoryMB {
		return maxServerMemoryMB
	}
	return mb
}

// StartServer launches a supervised server process. Returns ErrEULARequired
// when eula.txt is missing or not accepted.
func StartServer(ctx context.Context, id string, opts StartOptions) (ServerStatus, error) {
	dir, err := serverDir(id)
	if err != nil {
		return ServerStatus{}, err
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return ServerStatus{}, fmt.Errorf("server %q does not exist", id)
	}

	procMu.Lock()
	if _, ok := running[id]; ok {
		procMu.Unlock()
		return ServerStatus{}, fmt.Errorf("server %q is already running", id)
	}
	count := len(running)
	procMu.Unlock()
	if count >= MaxConcurrentServers {
		return ServerStatus{}, fmt.Errorf("server limit reached (%d concurrent max)", MaxConcurrentServers)
	}

	if !eulaAccepted(dir) {
		return ServerStatus{}, ErrEULARequired
	}

	jarPath, err := findServerJar(dir, opts.JarName)
	if err != nil {
		return ServerStatus{}, err
	}
	mcVersion := strings.TrimSpace(opts.MCVersion)
	if mcVersion == "" {
		mcVersion = mcVersionFromJar(jarPath)
	}
	if mcVersion == "" {
		return ServerStatus{}, fmt.Errorf("could not determine Minecraft version (pass mcVersion or use a versioned jar name)")
	}

	javaBin := ""
	if testJavaBin != "" {
		javaBin = testJavaBin
	} else {
		javaBin, err = resolveJava(ctx, requiredJavaVersion(mcVersion))
		if err != nil {
			return ServerStatus{}, err
		}
	}

	port := readServerPort(dir)
	if portInUse(port) {
		return ServerStatus{}, fmt.Errorf("port %d is already in use", port)
	}

	memoryMB := clampMemory(opts.MemoryMB)
	var extra []string
	for _, a := range opts.ExtraArgs {
		a = strings.TrimSpace(a)
		if a == "" || len(a) > 500 {
			continue
		}
		extra = append(extra, a)
		if len(extra) >= 32 {
			break
		}
	}

	cmd := newServerCommand(javaBin, memoryMB, extra, jarPath, dir)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return ServerStatus{}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return ServerStatus{}, err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return ServerStatus{}, err
	}
	if err := cmd.Start(); err != nil {
		return ServerStatus{}, fmt.Errorf("failed to start server: %w", err)
	}

	rs := &runningServer{
		ctx:       ctx,
		cmd:       cmd,
		stdin:     stdin,
		startedAt: time.Now(),
		port:      port,
		mcVersion: mcVersion,
		done:      make(chan struct{}),
	}
	procMu.Lock()
	// Re-check under lock: two starters may have raced past the first check.
	if _, ok := running[id]; ok {
		procMu.Unlock()
		_ = cmd.Process.Kill()
		return ServerStatus{}, fmt.Errorf("server %q is already running", id)
	}
	running[id] = rs
	procMu.Unlock()

	feed := func(r io.Reader) {
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			line := sc.Text()
			procMu.Lock()
			rs.logs = append(rs.logs, line)
			if len(rs.logs) > maxLogLines {
				rs.logs = rs.logs[len(rs.logs)-maxLogLines:]
			}
			procMu.Unlock()
			emitLog(rs.ctx, id, line)
		}
	}
	go feed(stdout)
	go feed(stderr)
	go func() {
		_ = cmd.Wait()
		procMu.Lock()
		delete(running, id)
		close(rs.done)
		procMu.Unlock()
		emitState(rs.ctx, id, "stopped")
	}()

	emitState(ctx, id, "running")
	return Status(id)
}

// testJavaBin, when non-empty, replaces java resolution (tests only).
var testJavaBin = ""

// StopServer sends "stop" for a graceful shutdown, then kills after grace.
func StopServer(id string) error {
	procMu.Lock()
	rs, ok := running[id]
	procMu.Unlock()
	if !ok {
		return fmt.Errorf("server %q is not running", id)
	}
	_, _ = io.WriteString(rs.stdin, "stop\n")
	select {
	case <-rs.done:
		return nil
	case <-time.After(stopGracePeriod):
		_ = rs.cmd.Process.Kill()
		<-rs.done
		return nil
	}
}

// SendCommand writes a console line to a running server's stdin.
func SendCommand(id, command string) error {
	procMu.Lock()
	rs, ok := running[id]
	procMu.Unlock()
	if !ok {
		return fmt.Errorf("server %q is not running", id)
	}
	command = strings.TrimRight(command, "\r\n") + "\n"
	if len(command) > 4096 {
		return fmt.Errorf("command too long")
	}
	_, err := io.WriteString(rs.stdin, command)
	return err
}

// Status reports a server's process state.
func Status(id string) (ServerStatus, error) {
	if _, err := serverDir(id); err != nil {
		return ServerStatus{}, err
	}
	procMu.Lock()
	rs, ok := running[id]
	procMu.Unlock()
	st := ServerStatus{ID: id}
	if !ok {
		return st, nil
	}
	st.Running = true
	if rs.cmd.Process != nil {
		st.PID = rs.cmd.Process.Pid
	}
	st.StartedAt = rs.startedAt.UnixMilli()
	st.Port = rs.port
	st.MCVersion = rs.mcVersion
	return st, nil
}

// RecentLogs returns up to n buffered log lines (newest last).
func RecentLogs(id string, n int) ([]string, error) {
	procMu.Lock()
	rs, ok := running[id]
	procMu.Unlock()
	if !ok {
		return []string{}, nil
	}
	procMu.Lock()
	defer procMu.Unlock()
	if n <= 0 || n > len(rs.logs) {
		n = len(rs.logs)
	}
	out := make([]string, n)
	copy(out, rs.logs[len(rs.logs)-n:])
	return out, nil
}
