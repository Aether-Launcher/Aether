package extensions

import (
	"context"
	"testing"
)

func TestSidebarPinConsentIsStoredPerVersion(t *testing.T) {
	setupExtensionsDir(t)

	var manager *Manager
	prompts := 0
	manager = NewManager(context.Background(), func(_ context.Context, event string, args ...interface{}) {
		if event != "extension:confirmation" {
			return
		}
		prompts++
		request, ok := args[0].(map[string]interface{})
		if !ok || request["permission"] != "ui:sidebar" {
			t.Errorf("unexpected confirmation request: %#v", args)
			return
		}
		approved := prompts == 1
		if err := manager.ResolveConfirmation(request["requestId"].(string), approved); err != nil {
			t.Errorf("resolve confirmation: %v", err)
		}
	})

	manifest := Manifest{
		ID:           "com.test.sidebar",
		Name:         "Sidebar Test",
		Version:      "1.0.0",
		PinToSidebar: true,
		Permissions:  []string{"ui:sidebar"},
	}
	if err := manager.RequestSidebarPinConsent(manifest); err != nil {
		t.Fatalf("request sidebar consent: %v", err)
	}
	if !manager.hasSidebarPinConsent(manifest) {
		t.Fatal("expected the approved version to be pinned")
	}

	// The decision survives a launcher restart and is not asked on every load.
	reloaded := NewManager(context.Background(), nil)
	if err := reloaded.loadSidebarConsents(); err != nil {
		t.Fatalf("load sidebar consent state: %v", err)
	}
	if !reloaded.hasSidebarPinConsent(manifest) {
		t.Fatal("expected the approved consent to persist")
	}
	if err := reloaded.RequestSidebarPinConsent(manifest); err != nil {
		t.Fatalf("repeat consent request: %v", err)
	}
	if prompts != 1 {
		t.Fatalf("expected one prompt for version 1.0.0, got %d", prompts)
	}

	// A new version asks again; declining it removes the previous version's grant.
	manifest.Version = "1.1.0"
	if err := manager.RequestSidebarPinConsent(manifest); err != nil {
		t.Fatalf("request updated-version consent: %v", err)
	}
	if manager.hasSidebarPinConsent(manifest) {
		t.Fatal("a declined update must not retain sidebar pin approval")
	}
	if err := manager.RequestSidebarPinConsent(manifest); err != nil {
		t.Fatalf("repeat declined-version request: %v", err)
	}
	if prompts != 2 {
		t.Fatalf("expected a new prompt for version 1.1.0 only, got %d", prompts)
	}
}

func TestSidebarPinConsentRequiresSidebarRequest(t *testing.T) {
	setupExtensionsDir(t)
	prompts := 0
	manager := NewManager(context.Background(), func(context.Context, string, ...interface{}) {
		prompts++
	})

	for _, manifest := range []Manifest{
		{ID: "com.test.no-pin", Version: "1.0.0", Permissions: []string{"ui:sidebar"}},
		{ID: "com.test.no-permission", Version: "1.0.0", PinToSidebar: true},
	} {
		if err := manager.RequestSidebarPinConsent(manifest); err != nil {
			t.Fatalf("request for %s: %v", manifest.ID, err)
		}
	}
	if prompts != 0 {
		t.Fatalf("expected no prompts for unrequested pins, got %d", prompts)
	}
}
