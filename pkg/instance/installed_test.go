package instance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// setupDataDir points fs.GetDataDir() at a temp .aether folder.
func setupDataDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Chdir(tmp)
	if err := os.MkdirAll(filepath.Join(tmp, ".aether", "instances"), 0755); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(tmp, ".aether", "instances")
}

func writeManifest(t *testing.T, dir, id, version string) {
	t.Helper()
	inst := Instance{ID: id, Name: id, Version: version, Loader: "Vanilla"}
	data, err := json.Marshal(inst)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "instance.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestInstalledRequiresVersionJson(t *testing.T) {
	instancesDir := setupDataDir(t)

	// Jar present but no version.json (e.g. freshly imported with reused
	// shared files, install never completed) -> NOT installed.
	dir := filepath.Join(instancesDir, "partial")
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	writeManifest(t, dir, "partial", "1.20.1")
	if err := os.WriteFile(filepath.Join(dir, "bin", "1.20.1.jar"), []byte("jar"), 0644); err != nil {
		t.Fatal(err)
	}

	all := GetInstances()
	if len(all) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(all))
	}
	if all[0].Installed {
		t.Fatal("instance with jar but no version.json must not be marked installed")
	}

	// version.json written (Install completed) -> installed.
	if err := os.WriteFile(filepath.Join(dir, "version.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	all = GetInstances()
	if len(all) != 1 || !all[0].Installed {
		t.Fatal("instance with jar + version.json must be marked installed")
	}
}

func TestInstalledMissingJar(t *testing.T) {
	instancesDir := setupDataDir(t)

	dir := filepath.Join(instancesDir, "nojar")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	writeManifest(t, dir, "nojar", "1.20.1")
	if err := os.WriteFile(filepath.Join(dir, "version.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	all := GetInstances()
	if len(all) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(all))
	}
	if all[0].Installed {
		t.Fatal("instance with version.json but no client jar must not be marked installed")
	}
}
