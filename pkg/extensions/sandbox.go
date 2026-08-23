package extensions

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"Aether/pkg/discord"
	"Aether/pkg/fs"
	"Aether/pkg/netutil"
	"github.com/dop251/goja"
)

// Sandbox represents an isolated JavaScript environment for an extension
type Sandbox struct {
	ctx               context.Context
	vm                *goja.Runtime
	manifest          Manifest
	onMessageCallback func(map[string]interface{}) (map[string]interface{}, error)
	callbackMu        sync.Mutex
	eventsMu          sync.Mutex
	eventHandlers     map[string][]goja.Callable
}

// InstanceInfo is a minimal view of an instance passed into the sandbox
type InstanceInfo struct {
	ID      string
	Name    string
	Version string
	Loader  string
}

const maxExtensionHTTPResponse = 10 * 1024 * 1024

// httpGetWithRetry performs the request, retrying transient network failures
// (e.g. temporary DNS resolution failures) a few times with short backoff.
// Mod loader metadata endpoints are usually reachable again within seconds.
func httpGetWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			time.Sleep(time.Duration(attempt-1) * 500 * time.Millisecond)
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !netutil.IsTransientNetworkError(err) {
			break
		}
	}
	return nil, lastErr
}

type modLoaderCallbackKey struct {
	extensionID string
	loaderID    string
}

var (
	lastRegisteredModLoaderCallbacks = map[modLoaderCallbackKey]func(map[string]interface{}) (map[string]interface{}, error){}
	modLoaderCallbackMu              sync.Mutex
)

func clearModLoaderCallbackCache(extensionID string) {
	modLoaderCallbackMu.Lock()
	defer modLoaderCallbackMu.Unlock()
	if extensionID == "" {
		lastRegisteredModLoaderCallbacks = make(map[modLoaderCallbackKey]func(map[string]interface{}) (map[string]interface{}, error))
		return
	}
	for key := range lastRegisteredModLoaderCallbacks {
		if key.extensionID == extensionID {
			delete(lastRegisteredModLoaderCallbacks, key)
		}
	}
}

// NewSandbox creates a new Goja isolate restricted by the given manifest.
// emit is an optional function for broadcasting events (e.g. runtime.EventsEmit);
// pass nil to disable event broadcasting (useful in tests).
func NewSandbox(
	ctx context.Context,
	manifest Manifest,
	serverURL string,
	onSidebarPage func(map[string]interface{}),
	onModLoader func(ModLoaderConfig),
	listInstances func() []InstanceInfo,
	installMod func(instanceID, jarName, downloadURL string) (string, error),
	listMods func(instanceID string) ([]string, error),
	deleteMod func(instanceID, jarName string) error,
	toggleMod func(instanceID, jarName string, enable bool) error,
	emit func(ctx context.Context, event string, data ...interface{}),
	confirm func(action map[string]interface{}) bool,
) *Sandbox {
	if emit == nil {
		emit = func(_ context.Context, _ string, _ ...interface{}) {}
	}
	vm := goja.New()
	sb := &Sandbox{
		ctx:           ctx,
		vm:            vm,
		manifest:      manifest,
		eventHandlers: make(map[string][]goja.Callable),
	}

	// Create the secure Aether bridge object
	aetherObj := vm.NewObject()

	// URL Whitelist Helper
	isAllowedURL := func(target string) bool {
		parsed, err := url.Parse(target)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
			return false
		}
		requestedHost := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
		for _, allowed := range manifest.Hosts {
			allowedURL, err := url.Parse(allowed)
			allowedHost := allowed
			if err == nil && allowedURL.Hostname() != "" {
				allowedHost = allowedURL.Hostname()
			}
			allowedHost = strings.ToLower(strings.TrimSpace(strings.TrimSuffix(allowedHost, ".")))
			if allowedHost != "" && (requestedHost == allowedHost || strings.HasSuffix(requestedHost, "."+allowedHost)) {
				return true
			}
		}
		return false
	}

	// Capability: ui:sidebar
	if manifest.HasPermission("ui:sidebar") {
		uiObj := vm.NewObject()
		uiObj.Set("registerSidebarPage", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) > 0 {
				arg := call.Argument(0).Export().(map[string]interface{})

				// Reconstruct the URL using the local server
				relURL := arg["url"].(string)
				fullURL := fmt.Sprintf("%s/%s/%s", serverURL, manifest.ID, relURL)

				payload := map[string]interface{}{
					"extensionId": manifest.ID,
					"id":          arg["id"],
					"label":       arg["label"],
					"url":         fullURL,
				}

				if onSidebarPage != nil {
					onSidebarPage(payload)
				}

				emit(ctx, "extension:sidebar:add", payload)
			}
			return goja.Undefined()
		})
		aetherObj.Set("ui", uiObj)
	}

	// Capability: ui:dialogs
	if manifest.HasPermission("ui:dialogs") {
		uiObj := aetherObj.Get("ui")
		if uiObj == nil {
			uiObj = vm.NewObject()
			aetherObj.Set("ui", uiObj)
		}
		uiObj.(*goja.Object).Set("openDialog", func(call goja.FunctionCall) goja.Value {
			fmt.Printf("[Sandbox:%s] Opened dialog\n", manifest.ID)
			return goja.Undefined()
		})
	}

	// Capability: network:http
	if manifest.HasPermission("network:http") {
		httpObj := vm.NewObject()
		httpObj.Set("get", func(call goja.FunctionCall) goja.Value {
			targetURL := call.Argument(0).String()
			if !isAllowedURL(targetURL) {
				panic(vm.NewGoError(fmt.Errorf("access denied to URL: %s", targetURL)))
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
			if err != nil {
				panic(vm.NewGoError(err))
			}
			resp, err := httpGetWithRetry(ctx, req)
			if err != nil {
				panic(vm.NewGoError(err))
			}
			defer resp.Body.Close()
			if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
				panic(vm.NewGoError(fmt.Errorf("HTTP request failed with status %s", resp.Status)))
			}

			body, err := io.ReadAll(io.LimitReader(resp.Body, maxExtensionHTTPResponse+1))
			if err != nil {
				panic(vm.NewGoError(err))
			}
			if len(body) > maxExtensionHTTPResponse {
				panic(vm.NewGoError(fmt.Errorf("HTTP response exceeds %d bytes", maxExtensionHTTPResponse)))
			}
			return vm.ToValue(string(body))
		})
		aetherObj.Set("http", httpObj)
	}

	// Capability: fs:download
	if manifest.HasPermission("fs:download") {
		fsObj := vm.NewObject()
		fsObj.Set("download", func(call goja.FunctionCall) goja.Value {
			targetURL := call.Argument(0).String()
			destPath := call.Argument(1).String()

			if !isAllowedURL(targetURL) {
				panic(vm.NewGoError(fmt.Errorf("access denied to URL: %s", targetURL)))
			}

			// Force destPath to be within the instances/libraries folder
			safePath, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "libraries"), destPath)
			if err != nil {
				panic(vm.NewGoError(err))
			}

			if err := netutil.DownloadFile(ctx, targetURL, safePath, nil); err != nil {
				panic(vm.NewGoError(err))
			}
			return vm.ToValue(safePath)
		})
		aetherObj.Set("fs", fsObj)
	}

	// Instance and mod capabilities are independently controlled. The legacy
	// instances:patch permission is accepted by HasAnyPermission for migration.
	if manifest.HasAnyPermission("instances:list", "mods:list", "mods:install", "mods:delete", "mods:toggle") {
		instancesObj := vm.NewObject()

		if manifest.HasAnyPermission("instances:list") {
			instancesObj.Set("list", func(call goja.FunctionCall) goja.Value {
				if listInstances == nil {
					return vm.ToValue([]interface{}{})
				}
				all := listInstances()
				var result []map[string]interface{}
				for _, inst := range all {
					result = append(result, map[string]interface{}{
						"id":      inst.ID,
						"name":    inst.Name,
						"version": inst.Version,
						"loader":  inst.Loader,
					})
				}
				return vm.ToValue(result)
			})
		}

		if manifest.HasAnyPermission("mods:install") {
			instancesObj.Set("installMod", func(call goja.FunctionCall) goja.Value {
				instanceID := call.Argument(0).String()
				jarName := call.Argument(1).String()
				downloadURL := call.Argument(2).String()

				if !isAllowedURL(downloadURL) {
					panic(vm.NewGoError(fmt.Errorf("access denied to URL: %s", downloadURL)))
				}

				if installMod == nil {
					panic(vm.NewGoError(fmt.Errorf("installMod not available")))
				}
				if confirm != nil && !confirm(map[string]interface{}{
					"action":        "install mod",
					"extensionId":   manifest.ID,
					"extensionName": manifest.Name,
					"instanceId":    instanceID,
					"jarName":       jarName,
					"url":           downloadURL,
				}) {
					panic(vm.NewGoError(fmt.Errorf("user denied mod installation")))
				}

				path, err := installMod(instanceID, jarName, downloadURL)
				if err != nil {
					panic(vm.NewGoError(err))
				}
				return vm.ToValue(path)
			})
		}

		if manifest.HasAnyPermission("mods:list") {
			instancesObj.Set("listMods", func(call goja.FunctionCall) goja.Value {
				instanceID := call.Argument(0).String()
				if listMods == nil {
					panic(vm.NewGoError(fmt.Errorf("listMods not available")))
				}
				mods, err := listMods(instanceID)
				if err != nil {
					panic(vm.NewGoError(err))
				}
				return vm.ToValue(mods)
			})
		}

		if manifest.HasAnyPermission("mods:delete") {
			instancesObj.Set("deleteMod", func(call goja.FunctionCall) goja.Value {
				instanceID := call.Argument(0).String()
				jarName := call.Argument(1).String()
				if deleteMod == nil {
					panic(vm.NewGoError(fmt.Errorf("deleteMod not available")))
				}
				if confirm != nil && !confirm(map[string]interface{}{
					"action":        "delete mod",
					"extensionId":   manifest.ID,
					"extensionName": manifest.Name,
					"instanceId":    instanceID,
					"jarName":       jarName,
				}) {
					panic(vm.NewGoError(fmt.Errorf("user denied mod deletion")))
				}
				if err := deleteMod(instanceID, jarName); err != nil {
					panic(vm.NewGoError(err))
				}
				return goja.Undefined()
			})
		}

		if manifest.HasAnyPermission("mods:toggle") {
			instancesObj.Set("toggleMod", func(call goja.FunctionCall) goja.Value {
				instanceID := call.Argument(0).String()
				jarName := call.Argument(1).String()
				enable := call.Argument(2).ToBoolean()
				if toggleMod == nil {
					panic(vm.NewGoError(fmt.Errorf("toggleMod not available")))
				}
				if confirm != nil && !confirm(map[string]interface{}{
					"action":        "change mod state",
					"extensionId":   manifest.ID,
					"extensionName": manifest.Name,
					"instanceId":    instanceID,
					"jarName":       jarName,
					"enable":        enable,
				}) {
					panic(vm.NewGoError(fmt.Errorf("user denied mod state change")))
				}
				if err := toggleMod(instanceID, jarName, enable); err != nil {
					panic(vm.NewGoError(err))
				}
				return goja.Undefined()
			})
		}

		aetherObj.Set("instances", instancesObj)
	}

	// Capability: launcher:modloader
	if manifest.HasPermission("launcher:modloader") {
		launcherObj := vm.NewObject()
		launcherObj.Set("registerModLoader", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) > 0 {
				exported, ok := call.Argument(0).Export().(map[string]interface{})
				if !ok {
					panic(vm.NewGoError(fmt.Errorf("registerModLoader expects an object with id, name, description and onLaunch")))
				}

				idStr, _ := exported["id"].(string)
				nameStr, _ := exported["name"].(string)
				descStr, _ := exported["description"].(string)
				if idStr == "" || nameStr == "" {
					panic(vm.NewGoError(fmt.Errorf("registerModLoader: id and name are required strings")))
				}
				config := ModLoaderConfig{
					ID:          idStr,
					Name:        nameStr,
					Description: descStr,
					ExtensionID: manifest.ID,
				}

				// Extract onLaunch from the original (un-exported) goja value so we
				// don't rely on goja.Value.ToObject. If a previously registered
				// mod loader with the same ID exists we keep its callback unless
				// this registration provides a new one.
				callbackKey := modLoaderCallbackKey{extensionID: manifest.ID, loaderID: config.ID}
				modLoaderCallbackMu.Lock()
				prev, hadPrev := lastRegisteredModLoaderCallbacks[callbackKey]
				modLoaderCallbackMu.Unlock()

				if cb, ok := goja.AssertFunction(call.Argument(0).ToObject(vm).Get("onLaunch")); ok {
					config.Callback = func(ctx map[string]interface{}) (map[string]interface{}, error) {
						val, err := cb(goja.Undefined(), vm.ToValue(ctx))
						if err != nil {
							return nil, err
						}
						exported := val.Export()
						res, ok := exported.(map[string]interface{})
						if !ok {
							return nil, fmt.Errorf("onLaunch did not return an object, got %T", exported)
						}
						return res, nil
					}
					modLoaderCallbackMu.Lock()
					lastRegisteredModLoaderCallbacks[callbackKey] = config.Callback
					modLoaderCallbackMu.Unlock()
				} else if hadPrev {
					config.Callback = prev
				} else {
					config.Callback = func(_ map[string]interface{}) (map[string]interface{}, error) {
						return nil, fmt.Errorf("mod loader '%s' does not define an onLaunch callback", config.ID)
					}
				}

				if onModLoader != nil {
					onModLoader(config)
				}
				fmt.Printf("[Sandbox:%s] Registered mod loader: %s\n", manifest.ID, config.ID)
			}
			return goja.Undefined()
		})
		aetherObj.Set("launcher", launcherObj)
	}

	// Capability: discord:presence
	if manifest.HasPermission("discord:presence") {
		discordObj := vm.NewObject()
		discordObj.Set("setActivity", func(call goja.FunctionCall) goja.Value {
			details := ""
			state := ""
			largeImage := "aether-logo"
			smallImage := ""
			largeText := ""
			smallText := ""
			var startPtr *time.Time
			if len(call.Arguments) > 0 {
				if arg, ok := call.Argument(0).Export().(map[string]interface{}); ok {
					if v, ok := arg["details"].(string); ok {
						details = v
					}
					if v, ok := arg["state"].(string); ok {
						state = v
					}
					if v, ok := arg["largeImageKey"].(string); ok && v != "" {
						largeImage = v
					}
					if v, ok := arg["largeText"].(string); ok {
						largeText = v
					}
					if v, ok := arg["smallImageKey"].(string); ok {
						smallImage = v
					}
					if v, ok := arg["smallText"].(string); ok {
						smallText = v
					}
					if v, ok := arg["startTimestamp"]; ok {
						switch ts := v.(type) {
						case float64:
							t := time.UnixMilli(int64(ts))
							startPtr = &t
						case int64:
							t := time.UnixMilli(ts)
							startPtr = &t
						}
					}
				}
			}
			// Call discord in background so JS is never blocked
			go func(d, s, li, lt, si, st string, sp *time.Time) {
				defer func() { _ = recover() }()
				_ = discord.SetActivity(d, s, li, lt, si, st, sp)
			}(details, state, largeImage, largeText, smallImage, smallText, startPtr)
			return goja.Undefined()
		})
		discordObj.Set("clearActivity", func(call goja.FunctionCall) goja.Value {
			go func() {
				defer func() { _ = recover() }()
				_ = discord.Clear()
			}()
			return goja.Undefined()
		})
		aetherObj.Set("discord", discordObj)
	}

	// Capability: events (for discord presence and future)
	if manifest.HasPermission("discord:presence") || manifest.HasPermission("instances:list") {
		eventsObj := vm.NewObject()
		eventsObj.Set("on", func(call goja.FunctionCall) goja.Value {
			event := call.Argument(0).String()
			if fn, ok := goja.AssertFunction(call.Argument(1)); ok {
				sb.eventsMu.Lock()
				sb.eventHandlers[event] = append(sb.eventHandlers[event], fn)
				sb.eventsMu.Unlock()
			}
			return goja.Undefined()
		})
		eventsObj.Set("off", func(call goja.FunctionCall) goja.Value {
			// simple off – clears all handlers for the event
			event := call.Argument(0).String()
			sb.eventsMu.Lock()
			delete(sb.eventHandlers, event)
			sb.eventsMu.Unlock()
			return goja.Undefined()
		})
		aetherObj.Set("events", eventsObj)
	}

	// Capability: skin:export
	if manifest.HasPermission("skin:export") {
		skinsObj := vm.NewObject()
		skinsObj.Set("export", func(call goja.FunctionCall) goja.Value {
			b64Data := call.Argument(0).String()
			filename := call.Argument(1).String()
			if filename == "" {
				filename = "skin.png"
			}

			skinsDir := filepath.Join(fs.GetDataDir(), "skins")
			if err := os.MkdirAll(skinsDir, 0755); err != nil {
				panic(vm.NewGoError(err))
			}

			safePath, err := fs.ContainedPath(skinsDir, filename)
			if err != nil {
				panic(vm.NewGoError(err))
			}
			data, err := base64.StdEncoding.DecodeString(b64Data)
			if err != nil {
				panic(vm.NewGoError(fmt.Errorf("invalid base64 data: %w", err)))
			}

			if err := os.WriteFile(safePath, data, 0644); err != nil {
				panic(vm.NewGoError(err))
			}

			return vm.ToValue(safePath)
		})
		aetherObj.Set("skins", skinsObj)
	}

	// Capability: ui:sidebar - also inject IPC methods
	if manifest.HasPermission("ui:sidebar") {
		// Grab the existing uiObj that was created earlier (or create if somehow nil)
		uiIPC := vm.NewObject()

		// Sandbox not yet created. Store callback via pointer closure to allow
		// execution later.
		var jsMessageHandler goja.Callable

		uiIPC.Set("onMessage", func(call goja.FunctionCall) goja.Value {
			if fn, ok := goja.AssertFunction(call.Argument(0)); ok {
				jsMessageHandler = fn
			}
			return goja.Undefined()
		})

		uiIPC.Set("postMessage", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) > 0 {
				emit(ctx, "extension:message:"+manifest.ID, call.Argument(0).Export())
			}
			return goja.Undefined()
		})

		// Merge into existing uiObj (which already has registerSidebarPage)
		if existingUI := aetherObj.Get("ui"); existingUI != nil {
			existingUI.(*goja.Object).Set("onMessage", uiIPC.Get("onMessage"))
			existingUI.(*goja.Object).Set("postMessage", uiIPC.Get("postMessage"))
		} else {
			aetherObj.Set("ui", uiIPC)
		}

		// Build sandbox early to assign the deferred callback closure.
		_ = jsMessageHandler // Captured in callback below

		// Inject the bridge into the global scope
		vm.Set("Aether", aetherObj)

		sb.onMessageCallback = func(payload map[string]interface{}) (map[string]interface{}, error) {
			if jsMessageHandler == nil {
				return nil, fmt.Errorf("no onMessage handler registered")
			}
			val, err := jsMessageHandler(goja.Undefined(), vm.ToValue(payload))
			if err != nil {
				return nil, err
			}
			if exported := val.Export(); exported != nil {
				if m, ok := exported.(map[string]interface{}); ok {
					return m, nil
				}
			}
			return map[string]interface{}{}, nil
		}
		return sb
	}

	// Inject the bridge into the global scope
	vm.Set("Aether", aetherObj)

	return sb
}

// Execute runs a JS script inside the sandbox
func (s *Sandbox) Execute(script string) error {
	_, err := s.vm.RunString(script)
	if err != nil {
		return fmt.Errorf("sandbox execution error: %w", err)
	}
	return nil
}

// InvokeMessage runs the registered onMessage handler for an IPC message.
// goja runtimes are not thread-safe and extension callbacks may block for a
// long time (mod downloads, confirmation dialogs), so invocations are
// serialized per sandbox and panics are recovered so a broken extension can
// never leave an iframe waiting on a response forever.
func (s *Sandbox) InvokeMessage(payload map[string]interface{}) (result map[string]interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = fmt.Errorf("sandbox %s panicked handling IPC message: %v", s.manifest.ID, r)
		}
	}()

	s.callbackMu.Lock()
	defer s.callbackMu.Unlock()

	if s.onMessageCallback == nil {
		return nil, fmt.Errorf("extension %s has no onMessage handler registered", s.manifest.ID)
	}
	return s.onMessageCallback(payload)
}

// EmitEvent dispatches a JS event to all handlers registered via Aether.events.on.
// It is safe to call from any goroutine.
func (s *Sandbox) EmitEvent(event string, payload map[string]interface{}) {
	s.eventsMu.Lock()
	handlers := append([]goja.Callable(nil), s.eventHandlers[event]...)
	s.eventsMu.Unlock()
	if len(handlers) == 0 {
		return
	}
	s.callbackMu.Lock()
	defer s.callbackMu.Unlock()
	for _, h := range handlers {
		func() {
			defer func() { _ = recover() }()
			_, _ = h(goja.Undefined(), s.vm.ToValue(payload))
		}()
	}
}
