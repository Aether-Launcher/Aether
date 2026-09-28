package servers

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func testCtx() context.Context {
	return context.Background()
}

// Fake server process: prints a ready line, echoes stdin, exits on "stop".
func TestHelperServerProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Println("FakeServer ready")
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		fmt.Println("FakeServer got: " + line)
		if line == "stop" {
			fmt.Println("FakeServer stopping")
			os.Exit(0)
		}
	}
	os.Exit(0)
}

func helperCommand(t *testing.T, dir string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperServerProcess", "--")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	cmd.Dir = dir
	hideConsole(cmd)
	return cmd
}

func setupProcDataDir(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Chdir(tmp)
	if err := os.MkdirAll(filepath.Join(tmp, ".aether", "servers"), 0755); err != nil {
		t.Fatal(err)
	}
}

func withFakeJava(t *testing.T) {
	t.Helper()
	oldBuild := newServerCommand
	oldBin := testJavaBin
	testJavaBin = "fake-java-for-tests"
	newServerCommand = func(javaBin string, memoryMB int, extraArgs []string, jarPath, dir string) *exec.Cmd {
		_, _, _ = javaBin, memoryMB, extraArgs
		_ = jarPath
		return helperCommand(t, dir)
	}
	t.Cleanup(func() {
		newServerCommand = oldBuild
		testJavaBin = oldBin
	})
}

func makeServerDir(t *testing.T, id string) string {
	t.Helper()
	if _, err := CreateServer(id, "Test"); err != nil {
		t.Fatal(err)
	}
	dir, err := serverDir(id)
	if err != nil {
		t.Fatal(err)
	}
	// Fake jar so findServerJar resolves.
	if err := os.WriteFile(filepath.Join(dir, "paper-1.21.1.jar"), []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestStartRequiresEula(t *testing.T) {
	setupProcDataDir(t)
	makeServerDir(t, "eula-test")

	_, err := StartServer(testCtx(), "eula-test", StartOptions{MCVersion: "1.21.1"})
	if err == nil {
		t.Fatal("expected ErrEULARequired")
	}
	if !isEulaError(err) {
		t.Fatalf("expected EULA error, got: %v", err)
	}
}

func isEulaError(err error) bool {
	return err != nil && (err == ErrEULARequired ||
		strings.Contains(err.Error(), "EULA"))
}

func TestAcceptEulaThenLifecycle(t *testing.T) {
	setupProcDataDir(t)
	withFakeJava(t)
	makeServerDir(t, "life")

	var mu sync.Mutex
	var states []string
	var logs []string
	oldEmit := emitEvent
	emitEvent = func(ctx context.Context, event string, data map[string]any) {
		_ = ctx
		mu.Lock()
		defer mu.Unlock()
		if event == "server:state" {
			states = append(states, data["state"].(string))
		}
		if event == "server:log" {
			logs = append(logs, data["line"].(string))
		}
	}
	t.Cleanup(func() { emitEvent = oldEmit })

	if err := SetEulaAccepted("life"); err != nil {
		t.Fatal(err)
	}
	st, err := StartServer(testCtx(), "life", StartOptions{MCVersion: "1.21.1", MemoryMB: 1024})
	if err != nil {
		t.Fatal(err)
	}
	if !st.Running || st.PID <= 0 || st.Port != 25565 || st.MCVersion != "1.21.1" {
		t.Fatalf("bad status: %+v", st)
	}

	// Double start rejected.
	if _, err := StartServer(testCtx(), "life", StartOptions{MCVersion: "1.21.1"}); err == nil {
		t.Fatal("expected double-start rejection")
	}

	// Console input flows.
	deadline := time.Now().Add(10 * time.Second)
	for {
		mu.Lock()
		got := len(logs) > 0
		mu.Unlock()
		if got || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err := SendCommand("life", "say hello"); err != nil {
		t.Fatalf("send: %v", err)
	}
	deadline = time.Now().Add(10 * time.Second)
	for {
		mu.Lock()
		found := false
		for _, l := range logs {
			if strings.Contains(l, "FakeServer got: say hello") {
				found = true
				break
			}
		}
		mu.Unlock()
		if found || time.Now().After(deadline) {
			if !found {
				t.Fatalf("echo never arrived; logs: %v", logs)
			}
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if err := StopServer("life"); err != nil {
		t.Fatalf("stop: %v", err)
	}
	st, err = Status("life")
	if err != nil {
		t.Fatal(err)
	}
	if st.Running {
		t.Fatal("should not be running after stop")
	}
	recent, err := RecentLogs("life", 10)
	if err != nil {
		t.Fatal(err)
	}
	_ = recent // logs cleared with registry; history available while running

	mu.Lock()
	defer mu.Unlock()
	sawRunning, sawStopped := false, false
	for _, s := range states {
		if s == "running" {
			sawRunning = true
		}
		if s == "stopped" {
			sawStopped = true
		}
	}
	if !sawRunning || !sawStopped {
		t.Fatalf("expected running+stopped states, got %v", states)
	}
}

func TestStopNotRunning(t *testing.T) {
	setupProcDataDir(t)
	makeServerDir(t, "idle")
	if err := StopServer("idle"); err == nil {
		t.Fatal("expected error stopping idle server")
	}
	if err := SendCommand("idle", "say hi"); err == nil {
		t.Fatal("expected error sending to idle server")
	}
	st, err := Status("nope")
	if err != nil {
		t.Fatalf("unknown server should report not-running, not error: %v", err)
	}
	if st.Running {
		t.Fatal("unknown server must not report running")
	}
}

func TestMaxConcurrentServers(t *testing.T) {
	setupProcDataDir(t)
	withFakeJava(t)
	oldEmit := emitEvent
	emitEvent = func(context.Context, string, map[string]any) {}
	t.Cleanup(func() { emitEvent = oldEmit })

	for _, id := range []string{"s1", "s2"} {
		makeServerDir(t, id)
		if err := SetEulaAccepted(id); err != nil {
			t.Fatal(err)
		}
		if _, err := StartServer(testCtx(), id, StartOptions{MCVersion: "1.21.1"}); err != nil {
			t.Fatalf("start %s: %v", id, err)
		}
		defer StopServer(id)
	}
	makeServerDir(t, "s3")
	if err := SetEulaAccepted("s3"); err != nil {
		t.Fatal(err)
	}
	if _, err := StartServer(testCtx(), "s3", StartOptions{MCVersion: "1.21.1"}); err == nil {
		t.Fatal("expected server-limit rejection")
	}
}

func TestPortInUse(t *testing.T) {
	setupProcDataDir(t)
	withFakeJava(t)
	oldEmit := emitEvent
	emitEvent = func(context.Context, string, map[string]any) {}
	t.Cleanup(func() { emitEvent = oldEmit })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("cannot bind loopback: " + err.Error())
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	makeServerDir(t, "porty")
	if err := SetEulaAccepted("porty"); err != nil {
		t.Fatal(err)
	}
	dir, err := serverDir("porty")
	if err != nil {
		t.Fatal(err)
	}
	props, err := os.ReadFile(filepath.Join(dir, "server.properties"))
	if err != nil {
		t.Fatal(err)
	}
	props = append(props, []byte(fmt.Sprintf("server-port=%d\n", port))...)
	if err := os.WriteFile(filepath.Join(dir, "server.properties"), props, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := StartServer(testCtx(), "porty", StartOptions{MCVersion: "1.21.1"}); err == nil {
		t.Fatal("expected port-in-use rejection")
	} else if !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("wrong error: %v", err)
	}
}
