package instance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectFormat(t *testing.T) {
	dir := t.TempDir()

	native := filepath.Join(dir, "native")
	writeTestFile(t, filepath.Join(native, "instance.json"), `{"id":"a","name":"A","version":"1.20.1"}`)
	if got := DetectFormat(native); got != FormatNative {
		t.Fatalf("native: got %q, want %q", got, FormatNative)
	}

	mmc := filepath.Join(dir, "mmc")
	writeTestFile(t, filepath.Join(mmc, "mmc-pack.json"), `{"components":[]}`)
	if got := DetectFormat(mmc); got != FormatMultiMC {
		t.Fatalf("multimc: got %q, want %q", got, FormatMultiMC)
	}

	cf := filepath.Join(dir, "cf")
	writeTestFile(t, filepath.Join(cf, "manifest.json"), `{"minecraft":{"version":"1.20.1"}}`)
	if got := DetectFormat(cf); got != FormatCurseForge {
		t.Fatalf("curseforge: got %q, want %q", got, FormatCurseForge)
	}

	cfBad := filepath.Join(dir, "cfbad")
	writeTestFile(t, filepath.Join(cfBad, "manifest.json"), `{"fileId":123}`)
	if got := DetectFormat(cfBad); got != FormatUnknown {
		t.Fatalf("bad curseforge: got %q, want %q", got, FormatUnknown)
	}

	empty := filepath.Join(dir, "empty")
	os.MkdirAll(empty, 0755)
	if got := DetectFormat(empty); got != FormatUnknown {
		t.Fatalf("empty: got %q, want %q", got, FormatUnknown)
	}
}

func TestImportMultiMC(t *testing.T) {
	src := filepath.Join(t.TempDir(), "My World")
	writeTestFile(t, filepath.Join(src, "mmc-pack.json"), `{"components":[
		{"uid":"net.minecraft","version":"1.21.11"},
		{"uid":"net.fabricmc.fabric-loader","version":"0.16.9"}
	]}`)
	writeTestFile(t, filepath.Join(src, "instance.cfg"), "[General]\nName=My World\nMemory=4096\n")
	writeTestFile(t, filepath.Join(src, "minecraft", "mods", "testmod.jar"), "jar")
	writeTestFile(t, filepath.Join(src, "minecraft", "config", "test.toml"), "toml")
	writeTestFile(t, filepath.Join(src, "minecraft", "minecraft.jar"), "ignored-jar")
	writeTestFile(t, filepath.Join(src, "libraries", "shared.lib"), "ignored-lib")

	targetRoot := t.TempDir()
	inst, err := ImportInstance(src, targetRoot, nil)
	if err != nil {
		t.Fatal(err)
	}
	if inst.ID != "my-world" {
		t.Fatalf("id: got %q, want %q", inst.ID, "my-world")
	}
	if inst.Version != "1.21.11" {
		t.Fatalf("version: got %q, want %q", inst.Version, "1.21.11")
	}
	if inst.Loader != "fabric" {
		t.Fatalf("loader: got %q, want %q", inst.Loader, "fabric")
	}
	if inst.Name != "My World" {
		t.Fatalf("name: got %q, want %q", inst.Name, "My World")
	}
	if inst.Memory != "4096" {
		t.Fatalf("memory: got %q, want %q", inst.Memory, "4096")
	}
	if inst.Installed {
		t.Fatal("foreign imports must be marked not installed")
	}

	target := filepath.Join(targetRoot, inst.ID)
	for _, want := range []string{"mods/testmod.jar", "config/test.toml"} {
		if _, err := os.Stat(filepath.Join(target, want)); err != nil {
			t.Fatalf("expected %s: %v", want, err)
		}
	}
	for _, skip := range []string{"minecraft", "libraries", "minecraft/minecraft.jar"} {
		if _, err := os.Stat(filepath.Join(target, skip)); err == nil {
			t.Fatalf("expected %s to be skipped", skip)
		}
	}
	if _, err := os.Stat(filepath.Join(target, "instance.json")); err != nil {
		t.Fatalf("expected instance.json: %v", err)
	}
}

func TestImportMultiMCForge(t *testing.T) {
	src := filepath.Join(t.TempDir(), "forgepack")
	writeTestFile(t, filepath.Join(src, "mmc-pack.json"), `{"components":[
		{"uid":"net.minecraft","version":"1.20.1"},
		{"uid":"net.minecraftforge","version":"47.2.0"}
	]}`)
	inst, err := ImportInstance(src, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if inst.Loader != "forge" {
		t.Fatalf("loader: got %q, want %q", inst.Loader, "forge")
	}
}

func TestImportCurseForge(t *testing.T) {
	src := filepath.Join(t.TempDir(), "curseforge pack")
	writeTestFile(t, filepath.Join(src, "manifest.json"), `{"minecraft":{"version":"1.20.1","modLoaders":[{"id":"forge-47.2.0"}]}}`)
	writeTestFile(t, filepath.Join(src, "mods", "mod.jar"), "jar")
	writeTestFile(t, filepath.Join(src, "config", "x.toml"), "toml")
	writeTestFile(t, filepath.Join(src, "logs", "old.log"), "ignored")

	targetRoot := t.TempDir()
	inst, err := ImportInstance(src, targetRoot, nil)
	if err != nil {
		t.Fatal(err)
	}
	if inst.ID != "curseforge-pack" {
		t.Fatalf("id: got %q, want %q", inst.ID, "curseforge-pack")
	}
	if inst.Version != "1.20.1" {
		t.Fatalf("version: got %q, want %q", inst.Version, "1.20.1")
	}
	if inst.Loader != "forge" {
		t.Fatalf("loader: got %q, want %q", inst.Loader, "forge")
	}

	target := filepath.Join(targetRoot, inst.ID)
	if _, err := os.Stat(filepath.Join(target, "mods", "mod.jar")); err != nil {
		t.Fatalf("expected mods/mod.jar: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "config", "x.toml")); err != nil {
		t.Fatalf("expected config/x.toml: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "logs")); err == nil {
		t.Fatal("expected logs to be skipped")
	}
}

func TestImportCurseForgeOverrides(t *testing.T) {
	src := filepath.Join(t.TempDir(), "pack")
	writeTestFile(t, filepath.Join(src, "manifest.json"), `{"minecraft":{"version":"1.21.1","modLoaders":[{"id":"neoforge-21.1.44"}]}}`)
	writeTestFile(t, filepath.Join(src, "overrides", "config", "client.toml"), "toml")

	targetRoot := t.TempDir()
	inst, err := ImportInstance(src, targetRoot, nil)
	if err != nil {
		t.Fatal(err)
	}
	if inst.Loader != "neoforge" {
		t.Fatalf("loader: got %q, want %q", inst.Loader, "neoforge")
	}
	target := filepath.Join(targetRoot, inst.ID)
	if _, err := os.Stat(filepath.Join(target, "config", "client.toml")); err != nil {
		t.Fatalf("expected overrides to be merged to config/client.toml: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "overrides")); err == nil {
		t.Fatal("expected overrides/ to be merged away")
	}
}

func TestImportNativeKeepsBinaries(t *testing.T) {
	src := filepath.Join(t.TempDir(), "native")
	writeTestFile(t, filepath.Join(src, "instance.json"), `{"id":"keep","name":"Keep","version":"1.20.1","loader":"Vanilla"}`)
	writeTestFile(t, filepath.Join(src, "bin", "1.20.1.jar"), "client")

	targetRoot := t.TempDir()
	inst, err := ImportInstance(src, targetRoot, nil)
	if err != nil {
		t.Fatal(err)
	}
	if inst.ID != "keep" {
		t.Fatalf("id: got %q, want %q", inst.ID, "keep")
	}
	target := filepath.Join(targetRoot, inst.ID)
	if _, err := os.Stat(filepath.Join(target, "bin", "1.20.1.jar")); err != nil {
		t.Fatalf("native import must keep binaries: %v", err)
	}
}

func TestUniqueInstanceID(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "test"), 0755)
	os.MkdirAll(filepath.Join(root, "test-2"), 0755)
	if got := uniqueInstanceID("Test", root); got != "test-3" {
		t.Fatalf("got %q, want %q", got, "test-3")
	}
	if got := uniqueInstanceID("fresh", root); got != "fresh" {
		t.Fatalf("got %q, want %q", got, "fresh")
	}
}

func TestImportUnsupported(t *testing.T) {
	src := filepath.Join(t.TempDir(), "random")
	writeTestFile(t, filepath.Join(src, "notes.txt"), "hi")
	if _, err := ImportInstance(src, t.TempDir(), nil); err == nil {
		t.Fatal("expected an error for an unsupported folder")
	}
}

func TestImportProgressReports(t *testing.T) {
	src := filepath.Join(t.TempDir(), "pack")
	writeTestFile(t, filepath.Join(src, "manifest.json"), `{"minecraft":{"version":"1.20.1"}}`)
	writeTestFile(t, filepath.Join(src, "mods", "a.jar"), "a")
	writeTestFile(t, filepath.Join(src, "mods", "b.jar"), "b")

	var last int
	var total int
	_, err := ImportInstance(src, t.TempDir(), func(done, t int, file string) {
		last = done
		total = t
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 { // manifest.json + a.jar + b.jar
		t.Fatalf("total: got %d, want 3", total)
	}
	if last != 3 {
		t.Fatalf("last done: got %d, want 3", last)
	}
}

func TestImportWritesValidManifest(t *testing.T) {
	src := filepath.Join(t.TempDir(), "pack")
	writeTestFile(t, filepath.Join(src, "mmc-pack.json"), `{"components":[{"uid":"net.minecraft","version":"1.20.1"}]}`)

	targetRoot := t.TempDir()
	inst, err := ImportInstance(src, targetRoot, nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(targetRoot, inst.ID, "instance.json"))
	if err != nil {
		t.Fatal(err)
	}
	var parsed Instance
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.ID != inst.ID || parsed.Version != "1.20.1" {
		t.Fatalf("manifest mismatch: %+v", parsed)
	}
}