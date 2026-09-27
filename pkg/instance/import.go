package instance

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"Aether/pkg/fs"
)

// Format identifies the launcher an instance folder came from.
type Format string

const (
	FormatNative     Format = "native"
	FormatMultiMC    Format = "multimc"
	FormatCurseForge Format = "curseforge"
	FormatModrinth   Format = "modrinth"
	FormatUnknown    Format = ""
)

// ImportProgress is called with the number of files copied so far, the total
// file count, and the current file's mapped destination path.
type ImportProgress func(done, total int, file string)

// mmcPack mirrors the Prism/MultiMC mmc-pack.json structure.
type mmcPack struct {
	Components []struct {
		UID     string `json:"uid"`
		Version string `json:"version"`
	} `json:"components"`
}

// curseforgeManifest mirrors the CurseForge app manifest.json structure.
type curseforgeManifest struct {
	Minecraft struct {
		Version    string `json:"version"`
		ModLoaders []struct {
			ID string `json:"id"`
		} `json:"modLoaders"`
	} `json:"minecraft"`
}

// cfInstance mirrors the CurseForge App minecraftinstance.json found in
// locally installed (not exported) instance folders.
type cfInstance struct {
	Name          string `json:"name"`
	GameVersion   string `json:"gameVersion"`
	GameVersionID int    `json:"gameVersionId"`
	IsVanilla     bool   `json:"isVanilla"`
	BaseModLoader *struct {
		Name             string `json:"name"`
		MinecraftVersion string `json:"minecraftVersion"`
	} `json:"baseModLoader"`
}

// normalizeLoaderID maps any launcher-specific loader spelling to Aether's
// canonical display IDs. Unknown or empty values fall back to "Vanilla".
// Canonical case matters: the launcher passes strings.ToLower to the mod
// loader hook, but the UI compares exact case (e.g. loader === 'Fabric').
func normalizeLoaderID(raw string) string {
	l := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(l, "neoforge") || strings.Contains(l, "neoforged"):
		return "NeoForge"
	case strings.Contains(l, "quilt"):
		return "Quilt"
	case strings.Contains(l, "fabric"):
		return "Fabric"
	case strings.Contains(l, "forge"):
		return "Forge"
	default:
		return "Vanilla"
	}
}

// modrinthProfile mirrors the Modrinth App (Theseus) profile.json structure.
type modrinthProfile struct {
	Name         string `json:"name"`
	GameVersion  string `json:"game_version"`
	Loader       string `json:"loader"`
	GameVersion2 string `json:"gameVersion"`
	ModLoader    string `json:"modloader"`
	ModLoader2   string `json:"modLoader"`
	Memory       int    `json:"memory"`
}

// modrinthIndex mirrors the modrinth.index.json structure used by exported packs.
type modrinthIndex struct {
	Name         string            `json:"name"`
	Game         string            `json:"game"`
	Dependencies map[string]string `json:"dependencies"`
}

// DetectFormat identifies which launcher produced the given instance folder.
func DetectFormat(source string) Format {
	if _, err := os.Stat(filepath.Join(source, "instance.json")); err == nil {
		return FormatNative
	}
	if _, err := os.Stat(filepath.Join(source, "mmc-pack.json")); err == nil {
		return FormatMultiMC
	}
	if data, err := os.ReadFile(filepath.Join(source, "manifest.json")); err == nil {
		var m curseforgeManifest
		if json.Unmarshal(data, &m) == nil && m.Minecraft.Version != "" {
			return FormatCurseForge
		}
	}
	// CurseForge App installs keep minecraftinstance.json instead of manifest.json.
	if data, err := os.ReadFile(filepath.Join(source, "minecraftinstance.json")); err == nil {
		var ci cfInstance
		if json.Unmarshal(data, &ci) == nil {
			return FormatCurseForge
		}
	}
	if data, err := os.ReadFile(filepath.Join(source, "profile.json")); err == nil {
		var p modrinthProfile
		if json.Unmarshal(data, &p) == nil && (p.GameVersion != "" || p.GameVersion2 != "" || p.Loader != "") {
			return FormatModrinth
		}
	}
	if data, err := os.ReadFile(filepath.Join(source, "modrinth.index.json")); err == nil {
		var idx modrinthIndex
		if json.Unmarshal(data, &idx) == nil {
			if _, ok := idx.Dependencies["minecraft"]; ok {
				return FormatModrinth
			}
			if strings.EqualFold(idx.Game, "minecraft") {
				return FormatModrinth
			}
		}
	}
	return FormatUnknown
}

// ImportInstance imports an instance folder from any supported launcher into
// targetRoot (the Aether instances directory). User content (mods, config,
// saves) is preserved; launcher-managed binaries are dropped so the Install
// flow re-downloads them into Aether's layout. It returns the created
// instance manifest.
func ImportInstance(source, targetRoot string, onProgress ImportProgress) (*Instance, error) {
	format := DetectFormat(source)
	if format == FormatUnknown {
		return nil, fmt.Errorf("this doesn't look like an Aether, Prism/MultiMC, Modrinth, or CurseForge instance folder (expected one of: instance.json, mmc-pack.json, profile.json, modrinth.index.json, manifest.json, minecraftinstance.json) — make sure you selected the instance folder itself, not the launcher root")
	}

	inst := &Instance{Loader: "Vanilla", Memory: "4G", LastPlayed: "Never"}
	baseName := filepath.Base(source)

	switch format {
	case FormatNative:
		data, err := os.ReadFile(filepath.Join(source, "instance.json"))
		if err != nil {
			return nil, fmt.Errorf("invalid instance: missing instance.json: %w", err)
		}
		if err := json.Unmarshal(data, inst); err != nil {
			return nil, fmt.Errorf("invalid instance manifest: %w", err)
		}
		if inst.ID == "" {
			inst.ID = baseName
		}
		if inst.Name == "" {
			inst.Name = baseName
		}
		if inst.Version == "" {
			return nil, fmt.Errorf("instance manifest must include a version")
		}
	case FormatMultiMC:
		version, loader, name, memory, err := parseMultiMC(source)
		if err != nil {
			return nil, err
		}
		inst.Version = version
		inst.Loader = loader
		inst.Name = name
		if name == "" {
			inst.Name = baseName
		}
		if memory != "" {
			inst.Memory = memory
		}
		inst.ID = baseName
	case FormatCurseForge:
		version, loader, name, err := parseCurseForge(source)
		if err != nil {
			return nil, err
		}
		inst.Version = version
		inst.Loader = loader
		inst.Name = name
		if name == "" {
			inst.Name = baseName
		}
		inst.ID = baseName
	case FormatModrinth:
		version, loader, name, memory, err := parseModrinth(source)
		if err != nil {
			return nil, err
		}
		inst.Version = version
		inst.Loader = loader
		inst.Name = name
		if name == "" {
			inst.Name = baseName
		}
		if memory != "" {
			inst.Memory = memory
		}
		inst.ID = baseName
	}

	if inst.Version == "" {
		return nil, fmt.Errorf("could not determine the Minecraft version for this instance")
	}

	inst.ID = uniqueInstanceID(inst.ID, targetRoot)
	target := filepath.Join(targetRoot, inst.ID)
	if err := os.MkdirAll(target, 0755); err != nil {
		return nil, err
	}

	if err := copyInstanceTree(source, target, copyPlanFor(format), onProgress); err != nil {
		_ = os.RemoveAll(target)
		return nil, fmt.Errorf("failed to import instance: %w", err)
	}

	// Installed state is recomputed from bin/<version>.jar on load; foreign
	// imports always re-download binaries via the Install flow.
	inst.Installed = false

	data, err := json.MarshalIndent(inst, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(target, "instance.json"), data, 0644); err != nil {
		return nil, err
	}
	return inst, nil
}

// parseMultiMC reads mmc-pack.json (version + loader components) and
// instance.cfg (display name, memory).
func parseMultiMC(source string) (version, loader, name, memory string, err error) {
	data, err := os.ReadFile(filepath.Join(source, "mmc-pack.json"))
	if err != nil {
		return "", "", "", "", err
	}
	var pack mmcPack
	if err := json.Unmarshal(data, &pack); err != nil {
		return "", "", "", "", fmt.Errorf("invalid mmc-pack.json: %w", err)
	}
	loader = "Vanilla"
	for _, c := range pack.Components {
		switch c.UID {
		case "net.minecraft":
			version = c.Version
		case "net.fabricmc.fabric-loader":
			loader = "Fabric"
		case "net.minecraftforge":
			loader = "Forge"
		case "net.neoforged":
			loader = "NeoForge"
		case "net.quiltmc.quilt-loader":
			loader = "Quilt"
		}
	}

	if data, err := os.ReadFile(filepath.Join(source, "instance.cfg")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Name=") {
				name = strings.TrimSpace(strings.TrimPrefix(line, "Name="))
			}
			if strings.HasPrefix(line, "Memory=") {
				memory = strings.TrimSpace(strings.TrimPrefix(line, "Memory="))
			}
			if strings.HasPrefix(line, "MaxMemAlloc=") {
				memory = strings.TrimSpace(strings.TrimPrefix(line, "MaxMemAlloc="))
			}
		}
	}
	return version, loader, name, memory, nil
}

// parseCurseForge reads the CurseForge app manifest.json (exported packs) or
// minecraftinstance.json (local installs) and returns the Minecraft version,
// the canonical loader ID, and the profile name (may be empty).
func parseCurseForge(source string) (version, loader, name string, err error) {
	loader = "Vanilla"
	if data, readErr := os.ReadFile(filepath.Join(source, "manifest.json")); readErr == nil {
		var m curseforgeManifest
		if json.Unmarshal(data, &m) == nil && m.Minecraft.Version != "" {
			version = m.Minecraft.Version
			if len(m.Minecraft.ModLoaders) > 0 {
				loader = normalizeLoaderID(m.Minecraft.ModLoaders[0].ID)
			}
			return version, loader, "", nil
		}
	}

	data, readErr := os.ReadFile(filepath.Join(source, "minecraftinstance.json"))
	if readErr != nil {
		return "", "", "", fmt.Errorf("no manifest.json or minecraftinstance.json found")
	}
	var ci cfInstance
	if err := json.Unmarshal(data, &ci); err != nil {
		return "", "", "", fmt.Errorf("invalid minecraftinstance.json: %w", err)
	}
	name = ci.Name
	version = ci.GameVersion
	if ci.BaseModLoader != nil {
		if ci.BaseModLoader.MinecraftVersion != "" {
			version = ci.BaseModLoader.MinecraftVersion
		}
		if ci.BaseModLoader.Name != "" {
			loader = normalizeLoaderID(ci.BaseModLoader.Name)
		} else if !ci.IsVanilla {
			loader = "Vanilla"
		}
	}
	if version == "" {
		return "", "", "", fmt.Errorf("could not determine Minecraft version from minecraftinstance.json (gameVersionId %d has no version string)", ci.GameVersionID)
	}
	return version, loader, name, nil
}

// parseModrinth reads profile.json or modrinth.index.json to extract
// Minecraft version, mod loader, instance name, and optional memory.
func parseModrinth(source string) (version, loader, name, memory string, err error) {
	loader = "Vanilla"

	// Try profile.json first (local Modrinth App instances)
	if data, readErr := os.ReadFile(filepath.Join(source, "profile.json")); readErr == nil {
		var p modrinthProfile
		if json.Unmarshal(data, &p) == nil {
			name = p.Name
			version = p.GameVersion
			if version == "" {
				version = p.GameVersion2
			}
		raw := strings.TrimSpace(p.Loader)
		if raw == "" {
			raw = strings.TrimSpace(p.ModLoader)
		}
		if raw == "" {
			raw = strings.TrimSpace(p.ModLoader2)
		}
		loader = normalizeLoaderID(raw)
			if p.Memory > 0 {
				memory = fmt.Sprintf("%dM", p.Memory)
			}
			if version != "" {
				return version, loader, name, memory, nil
			}
		}
	}

	// Fall back to modrinth.index.json (exported modpacks)
	if data, readErr := os.ReadFile(filepath.Join(source, "modrinth.index.json")); readErr == nil {
		var idx modrinthIndex
		if json.Unmarshal(data, &idx) == nil {
			if name == "" {
				name = idx.Name
			}
			if v, ok := idx.Dependencies["minecraft"]; ok {
				version = v
			}
		for dep := range idx.Dependencies {
			if dep == "minecraft" {
				continue
			}
			if got := normalizeLoaderID(dep); got != "Vanilla" {
				loader = got
			}
		}
		}
	}

	if version == "" {
		return "", "", "", "", fmt.Errorf("could not determine Minecraft version from Modrinth instance")
	}
	return version, loader, name, memory, nil
}

// uniqueInstanceID returns a folder ID that does not collide with any
// existing instance, appending -2, -3, ... when needed.
func uniqueInstanceID(base, targetRoot string) string {
	id := slug(base)
	if _, err := os.Stat(filepath.Join(targetRoot, id)); os.IsNotExist(err) {
		return id
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", id, i)
		if _, err := os.Stat(filepath.Join(targetRoot, candidate)); os.IsNotExist(err) {
			return candidate
		}
	}
}

// slug converts a folder name into a safe instance ID.
func slug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	if s == "" {
		return "imported-instance"
	}
	// Keep only allow-list characters; other runs become a single dash.
	s = regexpMustCompile(`[^a-z0-9._-]+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-._")
	s = regexpMustCompile(`-+`).ReplaceAllString(s, "-")
	if s == "" {
		return "imported-instance"
	}
	if s[0] == '.' || s[0] == '_' || s[0] == '-' {
		s = "a" + s
	}
	if len(s) > 65 {
		s = s[:65]
		s = strings.TrimRight(s, "-._")
	}
	return s
}

// regexpMustCompile is a tiny helper to avoid importing regexp at top for a single use.
// Defined here to keep import list minimal for tests that don't need regexp.
func regexpMustCompile(expr string) *regexp.Regexp {
	r, _ := regexp.Compile(expr)
	return r
}

// copyPlanFor returns the copy rules for a launcher format.
func copyPlanFor(format Format) copyPlan {
	switch format {
	case FormatNative:
		// Native instances keep everything, including bin/ and libraries/.
		return copyPlan{remap: func(rel string) (string, bool) { return rel, true }}
	case FormatMultiMC:
		skipped := map[string]bool{
			"libraries": true, "patches": true, "cache": true,
			"logs": true, "crash-reports": true, "bin": true,
		}
		remapped := map[string]string{
			"mods":               "mods",
			"config":             "config",
			"resourcepacks":      "resourcepacks",
			"saves":              "saves",
			"options.txt":        "options.txt",
			"servers.dat":        "servers.dat",
			"usernamecache.json": "usernamecache.json",
		}
		return copyPlan{remap: func(rel string) (string, bool) {
			root := rel
			if i := strings.IndexByte(rel, '/'); i > 0 {
				root = rel[:i]
			}
			if skipped[root] {
				return "", false
			}
			if root == "minecraft" {
				rest := strings.TrimPrefix(rel, "minecraft/")
				for prefix, mapped := range remapped {
					if rest == prefix {
						return mapped, true
					}
					if strings.HasPrefix(rest, prefix+"/") {
						return mapped + "/" + strings.TrimPrefix(rest, prefix+"/"), true
					}
				}
				return "", false // jar, natives, logs — re-downloaded
			}
			return rel, true
		}}
	case FormatCurseForge:
		skipped := map[string]bool{"logs": true, "crash-reports": true}
		return copyPlan{remap: func(rel string) (string, bool) {
			root := rel
			if i := strings.IndexByte(rel, '/'); i > 0 {
				root = rel[:i]
			}
			if skipped[root] {
				return "", false
			}
			// Extracted modpack zips keep their overrides/ separate; merge
			// them into the instance root like the CurseForge app does.
			if rel == "overrides" {
				return "", false
			}
			if strings.HasPrefix(rel, "overrides/") {
				return strings.TrimPrefix(rel, "overrides/"), true
			}
			return rel, true
		}}
	case FormatModrinth:
		skipped := map[string]bool{
			"logs": true, "crash-reports": true, "bin": true,
			"libraries": true, "natives": true, ".fabric": true,
			".quilt": true, "cache": true,
		}
		skipFiles := map[string]bool{
			"profile.json": true, "modrinth.index.json": true,
		}
		return copyPlan{remap: func(rel string) (string, bool) {
			root := rel
			if i := strings.IndexByte(rel, '/'); i > 0 {
				root = rel[:i]
			}
			if skipped[root] {
				return "", false
			}
			if skipFiles[rel] {
				return "", false
			}
			// Strip overrides/ prefix (exported packs)
			if rel == "overrides" {
				return "", false
			}
			if strings.HasPrefix(rel, "overrides/") {
				return strings.TrimPrefix(rel, "overrides/"), true
			}
			// Strip .minecraft/ prefix (some Modrinth layouts)
			if strings.HasPrefix(rel, ".minecraft/") {
				return strings.TrimPrefix(rel, ".minecraft/"), true
			}
			if rel == ".minecraft" {
				return "", false
			}
			return rel, true
		}}
	}
	return copyPlan{remap: func(rel string) (string, bool) { return rel, true }}
}

// copyPlan decides how source paths map into the target instance.
type copyPlan struct {
	remap func(rel string) (string, bool)
}

// copyInstanceTree streams the source tree into target, applying the plan's
// skip/remap rules. Symlinks pointing inside the source are followed; links
// to external locations (e.g. shared libraries) are skipped with a warning.
func copyInstanceTree(source, target string, plan copyPlan, onProgress ImportProgress) error {
	total := 0
	_ = filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			total++
		}
		return nil
	})

	done := 0
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == source {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		mapped, keep := plan.remap(rel)
		if !keep {
			return nil
		}
		dest, err := fs.ContainedPath(target, mapped)
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return os.MkdirAll(dest, 0755)
		}

		if entry.Type()&os.ModeSymlink != 0 {
			return copySymlink(source, path, dest)
		}
		if err := copyFile(path, dest); err != nil {
			return err
		}
		done++
		if onProgress != nil {
			onProgress(done, total, mapped)
		}
		return nil
	})
}

// copySymlink follows a symlink when it resolves to a regular file inside the
// source tree; external links are skipped.
func copySymlink(source, path, dest string) error {
	resolved := path
	if link, err := os.Readlink(path); err == nil {
		if filepath.IsAbs(link) {
			resolved = link
		} else {
			resolved = filepath.Join(filepath.Dir(path), link)
		}
	}
	eval, err := filepath.EvalSymlinks(resolved)
	if err != nil {
		fmt.Printf("[Import] skipping broken symlink %s\n", path)
		return nil
	}
	rel, err := filepath.Rel(source, eval)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		fmt.Printf("[Import] skipping external symlink %s\n", path)
		return nil
	}
	info, err := os.Stat(eval)
	if err != nil || info.IsDir() {
		fmt.Printf("[Import] skipping directory/broken symlink %s\n", path)
		return nil
	}
	return copyFile(eval, dest)
}

// copyFile streams src into dest without loading it into memory.
func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
