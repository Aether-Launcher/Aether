package theme

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// AllowedAssetKeys is the whitelist of asset "slots" a theme's overwrite.json
// is permitted to replace. Any key in a theme's overwrite.json that is not in
// this list is silently dropped during installation (see installer.go).
//
// This is intentionally a closed set rather than free-form paths: it keeps
// themes from writing/overwriting arbitrary files elsewhere in the launcher
// and keeps the surface area of "what can a theme visually touch" auditable.
var AllowedAssetKeys = map[string]bool{
	"sidebar-logo":  true,
	"titlebar-logo": true,
	"background":    true,
}

// LockedAssetKeys can NEVER be set by a theme, even though they look like
// asset keys. They exist purely so InstallFromArchive can explain *why* a key
// was rejected (as opposed to it just not existing). The launcher's app icon,
// tray icon, and displayed name are not runtime-swappable assets at all — the
// icon is baked into the binary at build time and the "Aether" name is a
// hardcoded string in the frontend that themes have no channel to reach.
var LockedAssetKeys = map[string]bool{
	"app-icon":      true,
	"tray-icon":     true,
	"launcher-name": true,
}

// MaxThemeCSSBytes caps how large a single theme.css may be after sanitization.
const MaxThemeCSSBytes = 256 * 1024

// Legacy asset key aliases for backward compatibility when launcher renames slots
var LegacyAssetKeys = map[string]string{
	"sidebar-logo": "sidebar-logo",
	"titlebar-logo": "titlebar-logo",
}

// Data-aether attribute constants for stable theme targeting
const (
	DataAetherRoot        = "aether-root"
	DataAetherSidebar     = "sidebar"
	DataAetherCard        = "card"
	DataAetherWindowCtrl  = "window-controls"
	DataAetherLogo        = "logo"
	DataAetherTitle       = "title"
	DataAetherBackground  = "background"
	DataAetherSidebarLogo = "sidebar-logo"
	DataAetherTitlebar    = "titlebar-logo"
)

// blockedSelectorSubstrings identifies UI parts a theme must never be able to
// hide or disable entirely: the native window controls. Letting a theme make
// the close/minimize/maximize buttons unclickable or invisible would leave a
// user with no way to control the window.
// We now target stable data-aether attributes instead of fragile class names.
var blockedSelectorSubstrings = []string{
	"win-btn",
	"close-btn",
	"[data-aether=\"window-controls\"]",
}

var (
	// Patterns for detecting prohibited constructs
	urlRe         = regexp.MustCompile(`(?i)url\s*\(`)
	importRuleRe  = regexp.MustCompile(`(?i)@import[^;]*;`)
	appRegionDecl = regexp.MustCompile(`(?i)(-webkit-app-region|--wails-draggable)\s*:[^;]*;?`)
	expressionCall = regexp.MustCompile(`(?i)expression\s*\([^)]*\)`)
	contentDecl   = regexp.MustCompile(`(?i)content\s*:[^;]*;?`)
	pointerEvents = regexp.MustCompile(`(?i)pointer-events\s*:\s*none\s*;?`)
	positionFixed = regexp.MustCompile(`(?i)position\s*:\s*(fixed|absolute)\s*;`)
)

// Selectors that protect brand elements (logo, title) from content injection
var selectorsProtectingBrand = []string{
	"logo",
	"title",
	"brand",
}

// lockedSelectorTargets are selectors that must always stay interactive.
var lockedSelectorTargets = map[string]bool{
	"html":         true,
	"body":         true,
	"#app":         true,
	":root":        true,
	"#aether-root": true,
}

// Data-aether attribute prefix for stable selectors
const DataAetherPrefix = "data-aether"

// normalizeCSS prepares CSS for sanitization by:
// - Stripping /* ... */ comments (including nested, non-nested per CSS spec)
// - Decoding \XXXX unicode escapes
// - Collapsing whitespace (newlines, tabs, multiple spaces)
func normalizeCSS(css string) string {
	var sb strings.Builder
	sb.Grow(len(css))

	inComment := false
	inString := false
	var quoteChar byte
	escaped := false

	for i := 0; i < len(css); i++ {
		c := css[i]

		if inComment {
			if i+1 < len(css) && css[i] == '*' && css[i+1] == '/' {
				inComment = false
				i++ // skip '/'
			}
			continue
		}

		if inString {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quoteChar {
				inString = false
			}
			continue
		}

		if i+1 < len(css) && c == '/' && css[i+1] == '*' {
			inComment = true
			i++
			continue
		}

		if c == '"' || c == '\'' {
			inString = true
			quoteChar = c
			continue
		}

		if c == '\\' && !inString && i+1 < len(css) {
			// Potential unicode escape \XXXX or \XXXXXX
			// We'll handle full decoding later
		}

		sb.WriteByte(c)
	}

	// Decode unicode escapes \XXXX or \XXXXXX
	result := sb.String()
	var out strings.Builder
	out.Grow(len(result))
	runes := []rune(result)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\\' && i+1 < len(runes) {
			next := runes[i+1]
			if next == '0' || next == '1' || next == '2' || next == '3' ||
				next == '4' || next == '5' || next == '6' || next == '7' ||
				next == '8' || next == '9' || next == 'a' || next == 'b' ||
				next == 'c' || next == 'd' || next == 'e' || next == 'f' ||
				next == 'A' || next == 'B' || next == 'C' || next == 'D' ||
				next == 'E' || next == 'F' {
				// Parse hex digits (1-6)
				j := i + 1
				hexStr := ""
				for j < len(runes) && len(hexStr) < 6 {
					c := runes[j]
					if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
						hexStr += string(c)
						j++
					} else {
						break
					}
				}
				if len(hexStr) > 0 {
					val, err := strconv.ParseInt(hexStr, 16, 32)
					if err == nil && val <= 0x10FFFF {
						out.WriteRune(rune(val))
						i = j - 1
						continue
					}
				}
			}
		}
		out.WriteRune(runes[i])
	}

	// Collapse whitespace: replace runs of whitespace with single space
	result = out.String()
	var final strings.Builder
	final.Grow(len(result))
	inSpace := false
	for _, r := range result {
		if unicode.IsSpace(r) {
			if !inSpace {
				final.WriteByte(' ')
				inSpace = true
			}
		} else {
			final.WriteRune(r)
			inSpace = false
		}
	}

	return strings.TrimSpace(final.String())
}

// isOverlayBlocked checks if a rule body contains overlay/phishing patterns
func isOverlayBlocked(body string) bool {
	// position: fixed|absolute + inset:0 + high z-index
	if positionFixed.MatchString(body) {
		if strings.Contains(body, "inset:") || strings.Contains(body, "top:") || strings.Contains(body, "left:") {
			// Check for high z-index
			if strings.Contains(body, "z-index:") {
				// crude check - look for large z-index values
				// More robust: parse z-index value
				return true
			}
		}
		return false
	}
	return false
}

// SanitizeCSS strips the small set of things a theme is not allowed to do and
// returns the cleaned CSS plus a human-readable list of what (if anything)
// was removed, so the installer can surface it to the user.
//
// Pipeline:
// 1. Normalize (strip comments, decode escapes, collapse whitespace)
// 2. Truncate to 256KB
// 3. Run global prohibitions (@import, url(), position:fixed overlay)
// 4. Run per-rule brace-aware pass with #aether-root scoping
func SanitizeCSS(css string) (string, []string) {
	var warnings []string

	// 1. Normalize first (before any regex matching)
	css = normalizeCSS(css)

	// 2. Truncate to 256KB (post-normalization, before sanitization)
	if len(css) > MaxThemeCSSBytes {
		css = css[:MaxThemeCSSBytes]
		warnings = append(warnings, "theme.css exceeded the size limit and was truncated")
	}

	// Global prohibitions (before rule parsing)
	if importRuleRe.MatchString(css) {
		css = importRuleRe.ReplaceAllString(css, "")
		warnings = append(warnings, "@import rules are not allowed and were removed")
	}

	// Block url() entirely - use overwrite.json for images
	if urlRe.MatchString(css) {
		warnings = append(warnings, "url() is not allowed in theme.css; use overwrite.json for images")
		// Remove url() declarations entirely
		css = regexp.MustCompile(`(?i)url\s*\([^)]*\)`).ReplaceAllString(css, "/* url() removed */")
	}

	if expressionCall.MatchString(css) {
		css = expressionCall.ReplaceAllString(css, "")
		warnings = append(warnings, "expression() is not allowed and was removed")
	}

	if appRegionDecl.MatchString(css) {
		css = appRegionDecl.ReplaceAllString(css, "")
		warnings = append(warnings, "declarations touching the window drag region were removed")
	}

	cleaned, ruleWarnings := sanitizeRules(css)
	warnings = append(warnings, ruleWarnings...)

	return cleaned, warnings
}

// sanitizeRules walks top-level CSS rules (including once inside @media
// blocks) using brace-depth tracking, drops rules that target protected
// window-control selectors, and strips `content` from rules whose selector
// looks like it renders the brand name/logo, and strips app-lockout
// `pointer-events: none` from locked selector rules.
// Also prefixes selectors with #aether-root for scoping.
func sanitizeRules(css string) (string, []string) {
	var out strings.Builder
	var warnings []string
	dropped := 0
	strippedContent := 0
	strippedPointerEvents := 0
	strippedOverlay := 0

	i := 0
	n := len(css)
	for i < n {
		// Find the next rule boundary: selector text up to '{'
		braceIdx := strings.IndexByte(css[i:], '{')
		if braceIdx == -1 {
			out.WriteString(css[i:])
			break
		}
		braceIdx += i
		selector := css[i:braceIdx]

		// Find the matching closing brace for this rule (depth-aware, so
		// @media { ... } blocks with nested rules aren't cut short).
		depth := 1
		j := braceIdx + 1
		for j < n && depth > 0 {
			switch css[j] {
			case '{':
				depth++
			case '}':
				depth--
			}
			j++
		}
		body := css[braceIdx+1 : j-1]

		normalizedSelector := strings.ToLower(selector)

		if selectorIsBlocked(normalizedSelector) {
			dropped++
			i = j
			continue
		}

		if selectorProtectsBrand(normalizedSelector) && contentDecl.MatchString(body) {
			body = contentDecl.ReplaceAllString(body, "")
			strippedContent++
		}

		if selectorIsLockedTarget(normalizedSelector) && pointerEvents.MatchString(body) {
			body = pointerEvents.ReplaceAllString(body, "")
			strippedPointerEvents++
		}

		// Check for overlay patterns
		if isOverlayBlocked(body) {
			body = positionFixed.ReplaceAllString(body, "")
			strippedOverlay++
		}

		// Prefix selector with #aether-root for scoping
		scopedSelector := scopeSelector(selector)

		// Recurse into @media / @supports blocks so nested rules get the
		// same treatment.
		trimmedSelector := strings.TrimSpace(normalizedSelector)
		if strings.HasPrefix(trimmedSelector, "@media") ||
			strings.HasPrefix(trimmedSelector, "@supports") {
			var ruleWarnings []string
			body, ruleWarnings = sanitizeRules(body)
			warnings = append(warnings, ruleWarnings...)
			out.WriteString(selector)
			out.WriteByte('{')
			out.WriteString(body)
			out.WriteByte('}')
		} else {
			out.WriteString(scopedSelector)
			out.WriteByte('{')
			out.WriteString(body)
			out.WriteByte('}')
		}

		i = j
	}

	if dropped > 0 {
		warnings = append(warnings, "rules targeting window controls are not allowed and were removed")
	}
	if strippedContent > 0 {
		warnings = append(warnings, "content declarations on brand elements are not allowed and were removed")
	}
	if strippedPointerEvents > 0 {
		warnings = append(warnings, "pointer-events: none on the app root is not allowed and was removed")
	}
	if strippedOverlay > 0 {
		warnings = append(warnings, "position: fixed/absolute overlays are not allowed and were removed")
	}

	return out.String(), warnings
}

// scopeSelector prefixes all simple selectors with #aether-root
// Handles comma-separated selectors, pseudo-classes, pseudo-elements
func scopeSelector(selector string) string {
	parts := strings.Split(selector, ",")
	var scoped []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// Skip @media, @supports, @keyframes, @font-face, @layer, etc.
		trimmed := strings.TrimSpace(part)
		if strings.HasPrefix(trimmed, "@") {
			continue
		}
		if strings.HasPrefix(part, ":root") || strings.HasPrefix(part, "html") || strings.HasPrefix(part, "body") || strings.HasPrefix(part, "#aether-root") {
			scoped = append(scoped, part)
		} else {
			scoped = append(scoped, "#aether-root "+part)
		}
	}
	return strings.Join(scoped, ", ")
}

// selectorIsBlocked checks if a selector targets protected window controls
func selectorIsBlocked(normalizedSelector string) bool {
	for _, s := range blockedSelectorSubstrings {
		if strings.Contains(normalizedSelector, s) {
			return true
		}
	}
	return false
}

func selectorProtectsBrand(normalizedSelector string) bool {
	for _, s := range selectorsProtectingBrand {
		if strings.Contains(normalizedSelector, s) {
			return true
		}
	}
	return false
}

func selectorIsLockedTarget(normalizedSelector string) bool {
	parts := strings.Split(normalizedSelector, ",")
	for _, p := range parts {
		if lockedSelectorTargets[strings.TrimSpace(p)] {
			return true
		}
	}
	return false
}
