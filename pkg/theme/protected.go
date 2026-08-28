package theme

import (
	"regexp"
	"strings"
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

// blockedSelectorSubstrings identifies UI parts a theme must never be able to
// hide or disable entirely: the native window controls. Letting a theme make
// the close/minimize/maximize buttons unclickable or invisible would leave a
// user with no way to control the window.
var blockedSelectorSubstrings = []string{
	"win-btn",
	"close-btn",
}

// selectorsProtectingBrand marks selectors that render the launcher's name or
// logo. Themes may still restyle color, size, spacing, etc. on these, but
// they cannot inject replacement text via `content`, which is the one CSS
// property capable of visually renaming the app.
var selectorsProtectingBrand = []string{
	"logo",
	"title",
}

var (
	importRuleRe   = regexp.MustCompile(`(?i)@import[^;]*;`)
	appRegionDecl  = regexp.MustCompile(`(?i)(-webkit-app-region|--wails-draggable)\s*:[^;]*;?`)
	expressionCall = regexp.MustCompile(`(?i)expression\s*\([^)]*\)`)
	contentDecl    = regexp.MustCompile(`(?i)content\s*:[^;]*;?`)
	pointerEvents  = regexp.MustCompile(`(?i)pointer-events\s*:\s*none\s*;?`)
)

// lockedSelectorTargets are selectors that must always stay interactive.
var lockedSelectorTargets = map[string]bool{
	"html":  true,
	"body":  true,
	"#app":  true,
	":root": true,
}

// SanitizeCSS strips the small set of things a theme is not allowed to do and
// returns the cleaned CSS plus a human-readable list of what (if anything)
// was removed, so the installer can surface it to the user.
//
// This is a lightweight, brace-aware pass rather than a full CSS parser —
// it's deliberately conservative: when in doubt about a rule's boundaries it
// leaves the CSS alone rather than risk mangling legitimate styles.
func SanitizeCSS(css string) (string, []string) {
	var warnings []string

	if len(css) > MaxThemeCSSBytes {
		css = css[:MaxThemeCSSBytes]
		warnings = append(warnings, "theme.css exceeded the size limit and was truncated")
	}

	if importRuleRe.MatchString(css) {
		css = importRuleRe.ReplaceAllString(css, "")
		warnings = append(warnings, "@import rules are not allowed and were removed")
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
// `pointer-events: none` from html/body/#app/:root rules.
func sanitizeRules(css string) (string, []string) {
	var out strings.Builder
	var warnings []string
	dropped := 0
	strippedContent := 0
	strippedPointerEvents := 0

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

		// Recurse into @media / @supports blocks so nested rules get the
		// same treatment.
		if strings.HasPrefix(strings.TrimSpace(normalizedSelector), "@media") ||
			strings.HasPrefix(strings.TrimSpace(normalizedSelector), "@supports") {
			body, ruleWarnings := sanitizeRules(body)
			warnings = append(warnings, ruleWarnings...)
			out.WriteString(selector)
			out.WriteByte('{')
			out.WriteString(body)
			out.WriteByte('}')
		} else {
			out.WriteString(selector)
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

	return out.String(), warnings
}

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
