package instance

import "strings"

// canonicalLoaderName returns the display name Aether uses for a mod loader.
// It accepts the case-insensitive IDs used by launchers and loader extensions
// (for example "fabric", "Fabric", and "fabric-loader-0.15.0"). Unknown
// custom loader IDs are preserved so extensions can register their own
// loaders; an empty value is treated as Vanilla.
func canonicalLoaderName(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "Vanilla"
	}

	lower := strings.ToLower(name)
	switch {
	case lower == "vanilla" || lower == "minecraft" || lower == "none":
		return "Vanilla"
	case strings.Contains(lower, "neoforge") || strings.Contains(lower, "neoforged"):
		return "NeoForge"
	case strings.Contains(lower, "quilt"):
		return "Quilt"
	case strings.Contains(lower, "fabric"):
		return "Fabric"
	case strings.Contains(lower, "forge"):
		return "Forge"
	default:
		return name
	}
}
