// sandbox_profile.go - account / skin / cape APIs for the skin-selector extension.
//
// These APIs are injected separately from NewSandbox so existing call sites
// and tests keep working. Manager calls InjectProfileAPIs immediately after
// NewSandbox and before Execute, so main.js sees the APIs at load time.
//
// Security: tokens never cross the bridge. All Mojang calls run in Go via
// pkg/auth using the active account. Upload / equip / hide require user
// confirmation through the standard extension confirmation flow.

package extensions

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Aether-Launcher/Aether/pkg/auth"
	"github.com/dop251/goja"
)

// ProfileConfirm mirrors Manager.requestConfirmation.
type ProfileConfirm func(action map[string]interface{}) bool

// InjectProfileAPIs adds Aether.account / Aether.skins.manage / Aether.capes
// according to manifest permissions. Safe to call when no new permissions
// are granted (no-op). Must be called before sandbox.Execute.
func (s *Sandbox) InjectProfileAPIs(confirm ProfileConfirm) {
	if s == nil || s.vm == nil {
		return
	}
	av := s.vm.Get("Aether")
	if av == nil || goja.IsUndefined(av) || goja.IsNull(av) {
		return
	}
	aetherObj, ok := av.(*goja.Object)
	if !ok {
		if exported := av.Export(); exported != nil {
			_ = exported
		}
		return
	}
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	// Capability: account:read — safe subset only, never tokens.
	if s.manifest.HasPermission("account:read") {
		accountObj := s.vm.NewObject()
		_ = accountObj.Set("getActive", func(call goja.FunctionCall) goja.Value {
			return s.vm.ToValue(auth.ActiveAccountPublic())
		})
		_ = aetherObj.Set("account", accountObj)
	}

	// Resolve or create the skins object (NewSandbox may already have set
	// Aether.skins with export when skin:export was granted).
	skinsObj := resolveOrCreateChild(s.vm, aetherObj, "skins",
		s.manifest.HasPermission("skin:export") || s.manifest.HasPermission("skin:manage"))

	if s.manifest.HasPermission("skin:manage") && skinsObj != nil {
		_ = skinsObj.Set("listMine", func(call goja.FunctionCall) goja.Value {
			c, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			prof, err := auth.GetProfileTextures(c)
			if err != nil {
				panic(s.vm.NewGoError(err))
			}
			out := make([]map[string]interface{}, 0, len(prof.Skins))
			for _, sk := range prof.Skins {
				out = append(out, map[string]interface{}{
					"id": sk.ID, "state": sk.State, "url": sk.URL, "variant": sk.Variant,
				})
			}
			return s.vm.ToValue(out)
		})
		_ = skinsObj.Set("upload", func(call goja.FunctionCall) goja.Value {
			b64Data := call.Argument(0).String()
			variant := "classic"
			if len(call.Arguments) > 1 {
				if v := call.Argument(1).String(); v != "" && v != "undefined" {
					variant = v
				}
			}
			if confirm != nil && !confirm(map[string]interface{}{
				"action": "upload skin", "extensionId": s.manifest.ID,
				"extensionName": s.manifest.Name, "variant": variant,
			}) {
				panic(s.vm.NewGoError(fmt.Errorf("user denied skin upload")))
			}
			raw, err := base64.StdEncoding.DecodeString(b64Data)
			if err != nil {
				// Also accept data URLs from file inputs.
				if idx := strings.Index(b64Data, "base64,"); idx >= 0 {
					raw, err = base64.StdEncoding.DecodeString(b64Data[idx+len("base64,"):])
				}
				if err != nil {
					panic(s.vm.NewGoError(fmt.Errorf("invalid base64 skin data: %w", err)))
				}
			}
			c, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()
			skin, err := auth.UploadSkin(c, raw, variant)
			if err != nil {
				panic(s.vm.NewGoError(err))
			}
			return s.vm.ToValue(map[string]interface{}{
				"id": skin.ID, "state": skin.State, "url": skin.URL, "variant": skin.Variant,
			})
		})
		_ = skinsObj.Set("applyUrl", func(call goja.FunctionCall) goja.Value {
			skinURL := strings.TrimSpace(call.Argument(0).String())
			variant := "classic"
			if len(call.Arguments) > 1 {
				if v := call.Argument(1).String(); v != "" && v != "undefined" {
					variant = v
				}
			}
			if skinURL == "" || !(strings.HasPrefix(skinURL, "https://")) {
				panic(s.vm.NewGoError(fmt.Errorf("applyUrl requires an https skin URL")))
			}
			if confirm != nil && !confirm(map[string]interface{}{
				"action": "apply gallery skin", "extensionId": s.manifest.ID,
				"extensionName": s.manifest.Name, "url": skinURL, "variant": variant,
			}) {
				panic(s.vm.NewGoError(fmt.Errorf("user denied skin apply")))
			}
			png, err := downloadSkinPNG(ctx, skinURL)
			if err != nil {
				panic(s.vm.NewGoError(err))
			}
			c, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()
			skin, err := auth.UploadSkin(c, png, variant)
			if err != nil {
				panic(s.vm.NewGoError(err))
			}
			return s.vm.ToValue(map[string]interface{}{
				"id": skin.ID, "state": skin.State, "url": skin.URL, "variant": skin.Variant,
			})
		})
	}

	// Capability: cape:manage
	if s.manifest.HasPermission("cape:manage") {
		capesObj := s.vm.NewObject()
		_ = capesObj.Set("listMine", func(call goja.FunctionCall) goja.Value {
			c, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			prof, err := auth.GetProfileTextures(c)
			if err != nil {
				panic(s.vm.NewGoError(err))
			}
			out := make([]map[string]interface{}, 0, len(prof.Capes))
			for _, cp := range prof.Capes {
				out = append(out, map[string]interface{}{
					"id": cp.ID, "state": cp.State, "url": cp.URL, "alias": cp.Alias,
				})
			}
			return s.vm.ToValue(out)
		})
		_ = capesObj.Set("equip", func(call goja.FunctionCall) goja.Value {
			capeID := strings.TrimSpace(call.Argument(0).String())
			if capeID == "" {
				panic(s.vm.NewGoError(fmt.Errorf("cape ID is required")))
			}
			if confirm != nil && !confirm(map[string]interface{}{
				"action": "equip cape", "extensionId": s.manifest.ID,
				"extensionName": s.manifest.Name, "capeId": capeID,
			}) {
				panic(s.vm.NewGoError(fmt.Errorf("user denied cape equip")))
			}
			c, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := auth.EquipCape(c, capeID); err != nil {
				panic(s.vm.NewGoError(err))
			}
			return goja.Undefined()
		})
		_ = capesObj.Set("hide", func(call goja.FunctionCall) goja.Value {
			if confirm != nil && !confirm(map[string]interface{}{
				"action": "hide cape", "extensionId": s.manifest.ID,
				"extensionName": s.manifest.Name,
			}) {
				panic(s.vm.NewGoError(fmt.Errorf("user denied cape hide")))
			}
			c, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := auth.HideCape(c); err != nil {
				panic(s.vm.NewGoError(err))
			}
			return goja.Undefined()
		})
		_ = aetherObj.Set("capes", capesObj)
	}
}

func resolveOrCreateChild(vm *goja.Runtime, parent *goja.Object, key string, want bool) *goja.Object {
	if !want {
		return nil
	}
	if existing := parent.Get(key); existing != nil && !goja.IsUndefined(existing) && !goja.IsNull(existing) {
		if obj, ok := existing.(*goja.Object); ok {
			return obj
		}
	}
	obj := vm.NewObject()
	_ = parent.Set(key, obj)
	return obj
}

func downloadSkinPNG(ctx context.Context, skinURL string) ([]byte, error) {
	const maxSkinBytes = 5 * 1024 * 1024
	c, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(c, http.MethodGet, skinURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Aether-Launcher")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("skin download failed with status %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSkinBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxSkinBytes {
		return nil, fmt.Errorf("skin exceeds 5 MiB limit")
	}
	return data, nil
}
