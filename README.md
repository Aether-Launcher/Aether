# Aether Launcher

<p align="center">
  <img src="frontend/public/logo.png" alt="Aether Logo" width="120" />
</p>

<p align="center">
  <em>A minimal, extensible, lightning-fast Minecraft launcher.</em>
</p>

<p align="center">
  <a href="https://discord.gg/hyPWTs9FfM">
    <img src="https://img.shields.io/badge/discord-Join%20our%20Discord-5865F2?logo=discord&logoColor=white&style=for-the-badge" alt="Discord">
  </a>
  <a href="https://github.com/wayback09/Aether/releases">
    <img src="https://img.shields.io/github/v/release/wayback09/Aether?style=for-the-badge&label=Latest%20Release&color=2ea44f" alt="Latest Release">
  </a>
  <a href="https://github.com/wayback09/Aether/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/wayback09/Aether?style=for-the-badge&color=1f6feb" alt="License">
  </a>
</p>

## Overview

Aether is designed with a single core principle: **This launcher exists to launch Minecraft, and nothing more.**

Every feature that is not essential to launching the game—like downloading mods, managing servers, or viewing logs—is implemented as an extension. By keeping the core launcher intentionally minimal, Aether remains fast, predictable, and free from bloat. Think of it as the VS Code of Minecraft launchers. You install only the features you actually want.

## Features

- **Blazing Fast**: Cold starts in under 2 seconds with an idle memory footprint below 100MB.
- **Minimalist UI**: Clean, native-feeling, elegant design that gets out of your way, complete with smooth custom toast notifications.
- **Snapshot Support**: Effortlessly toggle between stable releases and latest snapshots when creating instances.
- **Extensible Architecture**: Everything from Modrinth integration to server browsers is an extension.
- **Capability-Based Extensions**: Extension backend code runs in a restricted Goja runtime and can only use the launcher APIs granted by its manifest permissions.

## Documentation

All project documentation is located in the `docs/` directory. If you are looking to contribute, build an extension, or just understand how Aether works, start here:

- **[Project Philosophy & Design](docs/DESIGN.md)** - The core principles guiding Aether's development.
- **[Architecture](docs/ARCHITECTURE.md)** - Overview of the Go backend, Extension Manager, and launcher pipeline.
- **[API & Interoperability](docs/API.md)** - Details on the extension API and permission model.
- **[Extensions Guide](docs/EXTENSIONS.md)** - How to build, package, and publish extensions for Aether.
  - *Looking for the official extension registry? Visit the [Aether-Extensions](https://github.com/wayback09/Aether-Extensions) repository.*
- **[Themes Guide](docs/THEMES.md)** - How to build and package `.theme` CSS/asset overwrites for Aether.
- **[Security & Sandboxing](docs/SECURITY.md)** - Threat models, capability isolation, and review guidelines.
- **[UI Specifications](docs/UI.md)** - Layout rules, components, and empty states.
- **[Styleguide](docs/STYLEGUIDE.md)** - Visual language, typography, colors, and animations.

## Developer Tooling

The following tools are available for building Aether extensions. They are maintained outside this core launcher repository and are available for terminal installation:

- [Aether SDK](https://github.com/wayback09/Aether-SDK) (`@aethermc/sdk`) - TypeScript type definitions and helper utilities for extension development, published to [npm](https://www.npmjs.com/package/@aethermc/sdk). Install with `npm install --save-dev @aethermc/sdk`.
- [Aether CLI](https://github.com/wayback09/Aether-Cli) (`aether-cli`) - Scaffold, validate, and package extensions into `.aex` format. Install with `go install github.com/wayback09/aether-cli@latest`.

## Getting Started

Download the latest release for your platform from the [Releases](../../releases) page.

| Platform | File |
|---|---|
| Windows (x64) | `Aether-windows-amd64-installer.exe` |
| macOS (Apple Silicon) | `Aether-macos-arm64.dmg` |
| Linux (x64) | `Aether-linux-amd64.AppImage` (or `.tar.gz`) |

<details>
<summary>Linux troubleshooting</summary>

**Always run the AppImage itself**, not the extracted binary:

```bash
chmod +x Aether-linux-amd64.AppImage
./Aether-linux-amd64.AppImage
# do NOT do: ./Aether-linux-amd64.AppImage --appimage-extract && ./squashfs-root/usr/bin/Aether
# (that bypasses AppRun and will always fail with libwebkit missing)
```

If you see `libwebkit2gtk-4.0.so.37: cannot open shared object file` or `Unable to spawn ... WebKitNetworkProcess (No such file or directory)`:

```bash
# Ubuntu 22.04 / Debian 12 / Mint 21
sudo apt update && sudo apt install libwebkit2gtk-4.0-37 libgtk-3-0 libayatana-appindicator3-1
# Ubuntu 24.04+ / Mint 22+ (4.0 → 4.1)
sudo apt install libwebkit2gtk-4.1-0 libgtk-3-0 libayatana-appindicator3-1
# Arch Linux — also needs Debian-path symlink for the Ubuntu-built AppImage:
sudo pacman -S webkit2gtk gtk3 libappindicator-gtk3
sudo mkdir -p /usr/lib/x86_64-linux-gnu/webkit2gtk-4.0
sudo ln -sf /usr/lib/webkit2gtk-4.0/WebKitNetworkProcess /usr/lib/x86_64-linux-gnu/webkit2gtk-4.0/WebKitNetworkProcess 2>/dev/null || true
sudo ln -sf /usr/lib/webkit2gtk-4.0/WebKitWebProcess /usr/lib/x86_64-linux-gnu/webkit2gtk-4.0/WebKitWebProcess 2>/dev/null || true
# Fedora
sudo dnf install webkit2gtk4.0 gtk3 libappindicator-gtk3
```

The AppImage does not bundle WebKit itself — it uses your host's WebKit, like Chrome/Electron do.
</details>

## Building from Source

If you'd like to build Aether from source, ensure you have the following installed:
- [Go 1.25+](https://go.dev/doc/install)
- [Node.js 20+](https://nodejs.org/)
- [Wails CLI v2](https://wails.io/docs/gettingstarted/installation)
- **Linux only:** `sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev libayatana-appindicator3-dev` to build, and `sudo apt install libwebkit2gtk-4.0-37 libgtk-3-0 libayatana-appindicator3-1` to run the AppImage/binary. On newer distros where `libwebkit2gtk-4.0` is unavailable, use `libwebkit2gtk-4.1-0` / `libwebkit2gtk-4.1-dev` instead.

```bash
# Clone the repository
git clone https://github.com/wayback09/Aether.git
cd Aether
# Run in development mode (live reload)
wails dev

# Or build a production binary
wails build
```

## License

The Aether launcher and first-party tooling are licensed under the GNU GPL v3.0-only. See [LICENSE](LICENSE).
