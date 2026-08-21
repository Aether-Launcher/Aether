package extensions

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSandboxCapabilities(t *testing.T) {
	// 1. Create a manifest WITH sidebar permission
	manifestAllowed := Manifest{
		ID:          "com.test.allowed",
		Permissions: []string{"ui:sidebar"},
	}

	sandbox1 := NewSandbox(
		context.Background(),
		manifestAllowed,
		"http://localhost",
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil, // emit: nil disables Wails event broadcasting in tests
		nil, // confirm: nil allows operations without UI in tests
	)

	// This should succeed without panic
	script1 := `
		Aether.ui.registerSidebarPage({ id: "test", label: "Test Page", url: "ui/index.html" });
	`
	err := sandbox1.Execute(script1)
	if err != nil {
		t.Fatalf("Expected script to run successfully, got: %v", err)
	}

	// 2. Create a manifest WITHOUT any permissions
	manifestDenied := Manifest{
		ID:          "com.test.denied",
		Permissions: []string{},
	}

	sandbox2 := NewSandbox(
		context.Background(),
		manifestDenied,
		"http://localhost",
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil, // emit: nil disables Wails event broadcasting in tests
		nil, // confirm: nil allows operations without UI in tests
	)

	// This should throw a JS error because Aether.ui is undefined
	script2 := `
		Aether.ui.registerSidebarPage({ id: "test", label: "Test Page", url: "ui/index.html" });
	`
	err = sandbox2.Execute(script2)
	if err == nil {
		t.Fatalf("Expected script to fail due to missing capabilities, but it succeeded")
	}
}

func TestSandboxURLAllowList(t *testing.T) {
	manifest := Manifest{
		ID:          "com.test.network",
		Permissions: []string{"network:http"},
		Hosts:       []string{"example.com"},
	}
	sandbox := NewSandbox(context.Background(), manifest, "http://localhost", nil, nil, nil, nil, nil, nil, nil, nil, nil)

	if err := sandbox.Execute(`Aether.http.get("https://example.com.evil.test/data");`); err == nil {
		t.Fatal("expected a deceptive hostname to be rejected")
	}
}

func TestSandboxGranularModPermissionAndConfirmation(t *testing.T) {
	install := func(string, string, string) (string, error) { return "mod.jar", nil }
	confirmCalled := false
	confirm := func(action map[string]interface{}) bool {
		confirmCalled = action["action"] == "install mod"
		return false
	}
	manifest := Manifest{
		ID:          "com.test.mods",
		Name:        "Mod Test",
		Permissions: []string{"mods:install"},
		Hosts:       []string{"example.com"},
	}
	sandbox := NewSandbox(context.Background(), manifest, "http://localhost", nil, nil, nil, install, nil, nil, nil, nil, confirm)
	if err := sandbox.Execute(`Aether.instances.installMod("instance", "mod.jar", "https://example.com/mod.jar");`); err == nil {
		t.Fatal("expected denied confirmation to stop mod installation")
	}
	if !confirmCalled {
		t.Fatal("expected mod installation to request confirmation")
	}

	listOnly := Manifest{ID: "com.test.list", Permissions: []string{"instances:list"}}
	listSandbox := NewSandbox(context.Background(), listOnly, "http://localhost", nil, nil, nil, install, nil, nil, nil, nil, nil)
	if err := listSandbox.Execute(`Aether.instances.installMod("instance", "mod.jar", "https://example.com/mod.jar");`); err == nil {
		t.Fatal("expected list-only extension to lack installMod")
	}
}

// TestSandboxModLoaderCallbackNonNil guards against the regression where
// registerModLoader produced a config with a nil Callback (due to a
// goja.Value.ToObject path), which later panicked at launch.
func TestSandboxModLoaderCallbackNonNil(t *testing.T) {
	var got ModLoaderConfig
	onModLoader := func(c ModLoaderConfig) { got = c }

	manifest := Manifest{
		ID:          "com.test.fabric",
		Permissions: []string{"launcher:modloader"},
	}
	sandbox := NewSandbox(context.Background(), manifest, "http://localhost",
		nil, onModLoader, nil, nil, nil, nil, nil, nil, nil)

	script := `
		Aether.launcher.registerModLoader({
			id: "fabric",
			name: "Fabric",
			description: "test loader",
			onLaunch: function(ctx) {
				ctx.mainClass = "net.fabricmc.loader.impl.launch.knot.KnotClient";
				ctx.gameArgs = ["--modded"];
				return ctx;
			}
		});
	`
	if err := sandbox.Execute(script); err != nil {
		t.Fatalf("registerModLoader execution failed: %v", err)
	}
	if got.ID != "fabric" {
		t.Fatalf("expected loader id 'fabric', got %q", got.ID)
	}
	if got.Callback == nil {
		t.Fatal("Callback must not be nil after a JS onLaunch function was provided")
	}

	out, err := got.Callback(map[string]interface{}{
		"mainClass": "net.minecraft.client.main.Main",
		"classpath": []string{"a.jar", "b.jar"},
	})
	if err != nil {
		t.Fatalf("callback invocation error: %v", err)
	}
	if out["mainClass"] != "net.fabricmc.loader.impl.launch.knot.KnotClient" {
		t.Fatalf("callback did not patch mainClass, got %v", out["mainClass"])
	}
}

func TestModLoaderCallbackCacheIsScopedAndCleared(t *testing.T) {
	clearModLoaderCallbackCache("")

	register := func(id string) ModLoaderConfig {
		var got ModLoaderConfig
		manifest := Manifest{ID: id, Permissions: []string{"launcher:modloader"}}
		sandbox := NewSandbox(context.Background(), manifest, "http://localhost",
			nil, func(config ModLoaderConfig) { got = config }, nil, nil, nil, nil, nil, nil, nil)
		if err := sandbox.Execute(`Aether.launcher.registerModLoader({id: "fabric", name: "Fabric", description: "test loader"});`); err != nil {
			t.Fatalf("registerModLoader execution failed for %s: %v", id, err)
		}
		return got
	}

	first := register("com.test.first")
	if first.Callback == nil {
		t.Fatal("first loader callback must not be nil")
	}
	second := register("com.test.second")
	if second.Callback == nil {
		t.Fatal("second loader callback must not be nil")
	}
	if _, err := second.Callback(map[string]interface{}{}); err == nil {
		t.Fatal("second extension reused the first extension callback")
	}

	clearModLoaderCallbackCache("com.test.first")
	third := register("com.test.first")
	if third.Callback == nil {
		t.Fatal("third loader callback must not be nil")
	}
	if _, err := third.Callback(map[string]interface{}{}); err == nil {
		t.Fatal("clearing an extension cache did not remove its callback")
	}
	clearModLoaderCallbackCache("")
}

// TestSandboxInvokeMessageSerializesConcurrentCalls guards against concurrent
// goja runtime access: HandleIPCMessage runs on Wails binding goroutines while
// extension callbacks may block (downloads, confirmation dialogs), so message
// invocations must be serialized per sandbox.
func TestSandboxInvokeMessageSerializesConcurrentCalls(t *testing.T) {
	manifest := Manifest{
		ID:          "com.test.serial",
		Permissions: []string{"ui:sidebar", "mods:install"},
		Hosts:       []string{"example.com"},
	}

	var concurrent int32
	release := make(chan struct{})
	confirm := func(action map[string]interface{}) bool {
		cur := atomic.AddInt32(&concurrent, 1)
		defer atomic.AddInt32(&concurrent, -1)
		if cur > 1 {
			t.Error("sandbox onMessage handler ran concurrently")
		}
		<-release
		return true
	}
	install := func(string, string, string) (string, error) { return "mod.jar", nil }

	sandbox := NewSandbox(context.Background(), manifest, "http://localhost",
		nil, nil, nil, install, nil, nil, nil, nil, confirm)
	script := `
		Aether.ui.onMessage(function(msg) {
			if (msg.type === "install") {
				Aether.instances.installMod(msg.instanceId, msg.jarName, msg.downloadUrl);
				return { type: "install_result", requestId: msg.requestId, success: true };
			}
			return {};
		});
	`
	if err := sandbox.Execute(script); err != nil {
		t.Fatalf("failed to register onMessage handler: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := sandbox.InvokeMessage(map[string]interface{}{
				"type": "install", "instanceId": "inst", "jarName": "mod.jar",
				"downloadUrl": "https://example.com/mod.jar",
			}); err != nil {
				t.Errorf("InvokeMessage error: %v", err)
			}
		}()
	}

	// Give every goroutine a chance to reach the confirmation barrier, then
	// release them. With serialization only one handler runs at a time.
	time.Sleep(100 * time.Millisecond)
	close(release)
	wg.Wait()
}

// TestSandboxInvokeMessageRecoversPanics ensures a panicking extension handler
// surfaces as an error (never an unrecovered panic) and the sandbox remains
// usable afterwards.
func TestSandboxInvokeMessageRecoversPanics(t *testing.T) {
	manifest := Manifest{
		ID:          "com.test.panic",
		Permissions: []string{"ui:sidebar", "network:http"},
		Hosts:       []string{"example.com"},
	}
	sandbox := NewSandbox(context.Background(), manifest, "http://localhost",
		nil, nil, nil, nil, nil, nil, nil, nil, nil)

	script := `
		Aether.ui.onMessage(function(msg) {
			if (msg.type === "echo") {
				return { type: "echo_result", requestId: msg.requestId, value: msg.value };
			}
			if (msg.type === "boom") {
				Aether.http.get("https://blocked.example.test/"); // not allowlisted
			}
			return {};
		});
	`
	if err := sandbox.Execute(script); err != nil {
		t.Fatalf("failed to register onMessage handler: %v", err)
	}

	res, err := sandbox.InvokeMessage(map[string]interface{}{"type": "echo", "requestId": float64(1), "value": "hi"})
	if err != nil {
		t.Fatalf("unexpected error on echo: %v", err)
	}
	if res["value"] != "hi" {
		t.Fatalf("expected echoed value, got %v", res["value"])
	}

	if _, err := sandbox.InvokeMessage(map[string]interface{}{"type": "boom", "requestId": float64(2)}); err == nil {
		t.Fatal("expected panicking handler to surface as an error")
	}

	res, err = sandbox.InvokeMessage(map[string]interface{}{"type": "echo", "requestId": float64(3), "value": "again"})
	if err != nil {
		t.Fatalf("sandbox unusable after recovered panic: %v", err)
	}
	if res["value"] != "again" {
		t.Fatalf("expected echo to work after panic, got %v", res["value"])
	}
}
