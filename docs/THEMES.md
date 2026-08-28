# Themes Guide

> **Current status:** `.theme` packages and the CSS overwrite pipeline described below are implemented. A `.theme` file is a zip-format package with an Aether-specific file extension — the same idea as `.aex` extensions, just for appearance instead of behavior. Themes cannot run any code; they are CSS and PNGs, and the launcher sanitizes both before applying them.

## What a Theme Is

A theme is a **CSS overwrite**, plus optionally a small set of **PNG asset overrides**. It is not a plugin — it has no `main.js`, no Goja sandbox, and no permissions. It can restyle the app; it cannot read your files, make network requests, or touch instances/mods.

When you activate a theme, Aether:
1. Reads its sanitized stylesheet and injects it into the page *after* Aether's base styles, so the theme's rules win (this is mainly used to override the `:root` CSS variables Aether's UI is built on — colors, radii, spacing).
2. Reads its `overwrite.json` and swaps in any of the small number of images it's allowed to replace.

Nothing about a theme is evaluated as code. The launcher parses the CSS as text, strips a short list of disallowed constructs, and serves the rest — see [What a Theme Cannot Do](#what-a-theme-cannot-do) below.

## Package Structure

```text
my-theme.theme
├── package.json      (required)
├── theme.css         (required — or whatever "css" in package.json points to)
├── overwrite.json    (optional — asset overrides)
├── icon.png          (optional — shown next to the theme's name in Settings)
└── logo.png          (optional — referenced from overwrite.json)
```

A `.theme` file uses the zip container format internally, exactly like `.aex`. You can build one by zipping the folder above and renaming `my-theme.zip` to `my-theme.theme`. If your zip wraps everything in a single root folder (e.g. `my-theme/package.json`), that's fine — the installer walks the archive looking for `package.json` the same way it looks for `manifest.json` in extensions.

## `package.json`

Every theme requires a `package.json` at its root:

```json
{
  "id": "com.example.midnight",
  "name": "Midnight",
  "version": "1.0.0",
  "author": "Your Name",
  "description": "A cool dark-blue theme.",
  "icon": "icon.png"
}
```

| Field | Required | Notes |
|---|---|---|
| `id` | Yes | Unique identifier, e.g. `com.you.themename`. Letters, numbers, `.`, `_`, `-` only. This becomes the folder name under Aether's data directory, so it must be filesystem-safe. |
| `name` | Yes | Display name shown in Settings → Appearance. |
| `version` | Yes | Free-form version string. |
| `author` | No | Shown in Settings. |
| `description` | No | Shown in Settings. |
| `icon` | No | Path (relative to the package root) to a PNG shown next to the theme in the list. |
| `css` | No | Path to the stylesheet. **Defaults to `theme.css`.** |
| `overwrite` | No | Path to the asset-override map. **Defaults to `overwrite.json`.** |

## The Stylesheet

`theme.css` (or whatever `css` points to) is plain CSS. The most useful thing to override is Aether's design-token variables, defined on `:root` in the base stylesheet:

```css
:root {
  --bg-color: #0a0a12;
  --sidebar-bg: #10101c;
  --panel-bg: #16162a;

  --accent-color: #7c3aed;
  --accent-hover: #6d28d9;

  --border-radius: 10px;
}
```

Because your stylesheet is injected after the base one, redeclaring a variable on `:root` (or overriding any other selector Aether already styles) takes effect immediately, everywhere that variable is used. You are not limited to variables — you can write ordinary rules targeting Aether's classes (`.card`, `.btn-primary`, `.sidebar`, etc.) too. Check `docs/STYLEGUIDE.md` for the class names Aether's components use.

### What a Theme Cannot Do

Before a theme is installed, its CSS is passed through a sanitizer. This isn't a style suggestion — the following are actively stripped out, and the installer tells you what (if anything) got removed:

- **`@import` rules** are removed entirely. A theme can't pull in remote stylesheets.
- **`expression()`** is removed (legacy CSS/JS execution vector).
- **Anything touching the window drag region** (`-webkit-app-region`, `--wails-draggable`) is stripped, so a theme can't make the title bar undraggable or turn ordinary content into a fake drag handle.
- **Rules targeting the window controls** (the close/minimize/maximize buttons) are dropped outright. A theme cannot hide, disable, or hijack the buttons that close the app.
- **`content:` declarations on anything that looks like a logo or title selector** are stripped. This is the one CSS property that can visually replace text, and it's the mechanism that would let a theme silently rename the app — see [Locked Assets](#locked-assets) below for why that's off-limits.
- **`pointer-events: none` on `html`, `body`, `#app`, or `:root`** is stripped, so a theme can't make the entire app un-clickable.
- Stylesheets over **256 KB** are truncated.

Everything else — colors, spacing, radii, fonts, shadows, animations, layout tweaks — is fair game.

## Asset Overrides (`overwrite.json`)

A theme can replace a small, fixed set of PNGs. `overwrite.json` maps an **asset key** to a **filename inside the package**:

```json
{
  "sidebar-logo": "logo.png",
  "titlebar-logo": "logo.png",
  "background": "bg.png"
}
```

### Allowed asset keys

| Key | Replaces |
|---|---|
| `sidebar-logo` | The logo shown at the top of the sidebar. |
| `titlebar-logo` | The logo shown in the custom title bar. |
| `background` | A background image behind the whole app window. |

This is a closed whitelist, not a free-form path map — a theme cannot use `overwrite.json` to write to arbitrary files or reference paths outside its own package. Any key that isn't in the table above is dropped during installation, and you'll see a warning explaining why.

Referenced files must be real PNGs (checked by file signature, not just extension) and are capped at 8 MB each, 20 MB total per theme package.

### Locked Assets

`app-icon`, `tray-icon`, and `launcher-name` are **not** overridable, even though they might look like plausible keys. If your `overwrite.json` lists them, they're rejected with a warning and nothing happens. This is intentional:

- The launcher's window/dock icon is compiled into the binary at build time (`build/appicon.png`, `build/windows/icon.ico`, etc.) — there is no runtime asset for a theme to intercept in the first place.
- The name **"Aether"** shown in the title bar and sidebar is a hardcoded string in the frontend. Themes have no channel to change it — not through `overwrite.json`, and not through CSS (see the `content:` restriction above).

In short: a theme can restyle Aether, but it can't make Aether pretend to be a different, unrelated piece of software.

## Installing and Managing Themes

Themes are installed from **Settings → Appearance**:

1. Click **Install Theme (.theme)** and pick a `.theme` file.
2. The launcher validates `package.json`, sanitizes `theme.css`, filters `overwrite.json`, and moves the theme into Aether's data directory under `themes/<id>`.
3. If anything was rejected or modified for safety, you'll see it listed after install.
4. Click **Apply** on a theme to activate it, or **Disable** to go back to Aether's default look. Only one theme can be active at a time.
5. Click **Remove** to uninstall a theme. If it was active, Aether reverts to the default look automatically.

Switching themes takes effect immediately — no restart required.

## For Theme Authors: Quick Start

1. Make a folder with `package.json` and `theme.css` (and optionally `overwrite.json` + your PNGs).
2. Zip the folder's *contents* (not the folder itself, though either works — the installer handles both).
3. Rename the `.zip` to `.theme`.
4. Install it via Settings → Appearance and iterate — reinstalling with the same `id` replaces the previous version.