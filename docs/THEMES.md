# Themes Guide

> `.theme` packages and CSS overrides are supported. A `.theme` file is a zip archive with an Aether-specific extension, much like `.aex`, but it contains appearance files rather than executable code. Aether checks and sanitizes the theme before applying it.

## What a Theme Is

A theme changes Aether's appearance with CSS and, optionally, replacement PNGs. It has no `main.js`, sandbox, or permissions. A theme can restyle the interface, but it cannot read files, make network requests, or change instances or mods.

When you activate a theme, Aether loads its sanitized stylesheet after the base styles, so theme values can override colors, corner shapes, and spacing. It also reads `overwrite.json` and replaces only the supported images.

Theme CSS is treated as text. Aether removes disallowed rules before serving it. See [What a Theme Cannot Do](#what-a-theme-cannot-do) for the restrictions.

## Package Structure

```text
my-theme.theme
├── package.json      (required)
├── theme.css         (required, unless "css" in package.json points elsewhere)
├── overwrite.json    (optional asset overrides)
├── icon.png          (optional icon shown in Settings)
└── logo.png          (optional asset referenced by overwrite.json)
```

Like `.aex`, a `.theme` file is a zip archive. Zip the package contents and rename `my-theme.zip` to `my-theme.theme`. Aether also accepts an archive with one top-level folder and searches it for `package.json`.

## `package.json`

Each theme needs a `package.json` at its root. For example:

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
  --bg-color: #0f0f12;
  --sidebar-bg: #141418;
  --panel-bg: #18181c;

  --accent-color: #3b52d4;
  --accent: var(--accent-color);
  --accent-hover: #3046c4;

  --border-radius: 8px;
  --card-radius: 7px;
  --button-radius: 11px;
}
```

Because the theme stylesheet loads after Aether's base styles, its `:root` values take precedence. Keep `--accent` linked to `--accent-color`; some components use the shorter token. Use `--border-radius` for general controls, or set `--card-radius` and `--button-radius` separately. You can also style classes such as `.card`, `.btn-primary`, and `.sidebar`. The [Style Guide](STYLEGUIDE.md) describes Aether's visual conventions.

### What a Theme Cannot Do

Before installation, Aether sanitizes the theme CSS and reports any rules it removes. It strips:

- **`@import` rules** are removed entirely. A theme can't pull in remote stylesheets.
- **`expression()`** is removed (legacy CSS/JS execution vector).
- **Anything touching the window drag region** (`-webkit-app-region`, `--wails-draggable`) is stripped, so a theme can't make the title bar undraggable or turn ordinary content into a fake drag handle.
- **Rules targeting the window controls** (the close/minimize/maximize buttons) are dropped outright. A theme cannot hide, disable, or hijack the buttons that close the app.
- **`content:` declarations on logo or title selectors** are stripped. This property could visually replace text and make the app appear to have a different name. See [Locked Assets](#locked-assets).
- **`pointer-events: none` on `html`, `body`, `#app`, or `:root`** is stripped, so a theme can't make the entire app un-clickable.
- Stylesheets over **256 KB** are truncated.

Themes can still change colors, spacing, corner shapes, fonts, shadows, animations, and layout.

## Asset Overrides (`overwrite.json`)

Themes can replace a few supported PNGs. In `overwrite.json`, each **asset key** points to a **filename inside the package**:

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

Only the keys in this table are accepted. A theme cannot use `overwrite.json` to write arbitrary files or point outside its package. Aether removes other keys and shows a warning during installation.

Referenced files must be real PNGs (checked by file signature, not just extension) and are capped at 8 MB each, 20 MB total per theme package.

### Locked Assets

`app-icon`, `tray-icon`, and `launcher-name` cannot be overridden. If they appear in `overwrite.json`, Aether rejects them and shows a warning. The app and dock icons are compiled into the launcher, and the name **Aether** in the title bar and sidebar is part of the frontend. CSS cannot replace it either; the sanitizer removes the relevant `content:` rules.

Themes can restyle Aether, but they cannot make it present itself as different software.

## Installing and Managing Themes

Install and manage themes from **Settings → Appearance**:

1. Click **Install Theme (.theme)** and pick a `.theme` file.
2. The launcher validates `package.json`, sanitizes `theme.css`, filters `overwrite.json`, and moves the theme into Aether's data directory under `themes/<id>`.
3. If anything was rejected or modified for safety, you'll see it listed after install.
4. Click **Apply** on a theme to activate it, or **Disable** to go back to Aether's default look. Only one theme can be active at a time.
5. Click **Remove** to uninstall a theme. If it was active, Aether reverts to the default look automatically.

Changes take effect as soon as you apply a theme; there is no need to restart Aether.

## For Theme Authors: Quick Start

1. Create a folder with `package.json` and `theme.css`. Add `overwrite.json` and PNGs if needed.
2. Zip the folder's contents. You can also zip the folder itself.
3. Rename the archive from `.zip` to `.theme`.
4. Install it from Settings → Appearance. Installing a theme with an existing ID replaces that version.
