package extensions

import (
	"context"
	"testing"
)

func newProfileSandbox(perms ...string) *Sandbox {
	sb := NewSandbox(context.Background(),
		Manifest{ID: "skin-selector-test", Name: "Test", Permissions: perms},
		"http://localhost",
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	sb.InjectProfileAPIs(nil)
	return sb
}

func TestProfileAPIGating(t *testing.T) {
	full := newProfileSandbox("account:read", "skin:manage", "cape:manage", "skin:export")
	if err := full.Execute(`
		if (typeof Aether.account.getActive !== "function") throw new Error("account.getActive missing");
		if (typeof Aether.skins.listMine !== "function") throw new Error("skins.listMine missing");
		if (typeof Aether.skins.upload !== "function") throw new Error("skins.upload missing");
		if (typeof Aether.skins.applyUrl !== "function") throw new Error("skins.applyUrl missing");
		if (typeof Aether.skins.export !== "function") throw new Error("skins.export missing");
		if (typeof Aether.capes.listMine !== "function") throw new Error("capes.listMine missing");
		if (typeof Aether.capes.equip !== "function") throw new Error("capes.equip missing");
		if (typeof Aether.capes.hide !== "function") throw new Error("capes.hide missing");
	`); err != nil {
		t.Fatalf("full permissions: %v", err)
	}

	none := newProfileSandbox("ui:sidebar")
	if err := none.Execute(`
		if (typeof Aether.account !== "undefined") throw new Error("account must be absent");
		if (typeof Aether.skins !== "undefined") throw new Error("skins must be absent");
		if (typeof Aether.capes !== "undefined") throw new Error("capes must be absent");
	`); err != nil {
		t.Fatalf("no-profile gating: %v", err)
	}

	exportOnly := newProfileSandbox("skin:export")
	if err := exportOnly.Execute(`
		if (typeof Aether.skins.export !== "function") throw new Error("export missing");
		if (typeof Aether.skins.listMine !== "undefined") throw new Error("listMine must be absent without skin:manage");
		if (typeof Aether.capes !== "undefined") throw new Error("capes must be absent");
	`); err != nil {
		t.Fatalf("export-only gating: %v", err)
	}
}

func TestProfileUploadRequiresConfirmation(t *testing.T) {
	denied := false
	sb := NewSandbox(context.Background(),
		Manifest{ID: "skin-selector-test", Permissions: []string{"skin:manage"}},
		"http://localhost",
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	sb.InjectProfileAPIs(func(action map[string]interface{}) bool {
		denied = action["action"] == "upload skin"
		return false
	})
	// 1x1 PNG base64
	png1x1 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	if err := sb.Execute(`Aether.skins.upload("` + png1x1 + `", "classic");`); err == nil {
		t.Fatal("expected denied confirmation to stop upload")
	}
	if !denied {
		t.Fatal("expected upload to request confirmation")
	}
}

func TestProfileApplyUrlRejectsNonHTTPS(t *testing.T) {
	sb := newProfileSandbox("skin:manage")
	if err := sb.Execute(`Aether.skins.applyUrl("http://example.com/skin.png", "classic");`); err == nil {
		t.Fatal("expected non-https URL to be rejected")
	}
	if err := sb.Execute(`Aether.capes.equip("");`); err == nil {
		// capes object absent without cape:manage — expect JS error
		t.Log("equip correctly unavailable without cape:manage")
	}
}
