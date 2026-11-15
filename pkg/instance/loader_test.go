package instance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalLoaderName(t *testing.T) {
	cases := map[string]string{
		"":                     "Vanilla",
		"vanilla":              "Vanilla",
		"VANILLA":              "Vanilla",
		"minecraft":            "Vanilla",
		"fabric":               "Fabric",
		"Fabric":               "Fabric",
		"fabric-loader-0.15.0": "Fabric",
		"forge-47.2.0":         "Forge",
		"Forge":                "Forge",
		"neoforge-21.1.44":     "NeoForge",
		"quilt-loader-0.24.0":  "Quilt",
		"my-custom-loader":     "my-custom-loader",
		"  fabric  ":           "Fabric",
	}
	for in, want := range cases {
		if got := canonicalLoaderName(in); got != want {
			t.Fatalf("canonicalLoaderName(%q): got %q, want %q", in, got, want)
		}
	}
}

func TestGetInstancesCanonicalizesLoader(t *testing.T) {
	instancesDir := setupDataDir(t)
	dir := filepath.Join(instancesDir, "lowercase")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(Instance{
		ID:      "lowercase",
		Name:    "Lowercase Loader",
		Version: "1.21.11",
		Loader:  "fabric",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "instance.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	all := GetInstances()
	if len(all) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(all))
	}
	if all[0].Loader != "Fabric" {
		t.Fatalf("loader: got %q, want %q", all[0].Loader, "Fabric")
	}
}

func TestCreateAndUpdateCanonicalizeLoader(t *testing.T) {
	setupDataDir(t)

	inst, err := Create("Loader Test", "1.21.11", "fabric")
	if err != nil {
		t.Fatal(err)
	}
	if inst.Loader != "Fabric" {
		t.Fatalf("created loader: got %q, want %q", inst.Loader, "Fabric")
	}

	inst.Loader = "neoforge"
	if err := UpdateInstance(inst); err != nil {
		t.Fatal(err)
	}
	all := GetInstances()
	if len(all) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(all))
	}
	if all[0].Loader != "NeoForge" {
		t.Fatalf("updated loader: got %q, want %q", all[0].Loader, "NeoForge")
	}
}
