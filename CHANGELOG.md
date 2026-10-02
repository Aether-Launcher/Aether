# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Theme System**: Added `.theme` packages — a zip-based CSS/asset overwrite format (`package.json`, `theme.css`, optional `overwrite.json`) installable and switchable from Settings → Appearance. Themes are sanitized on install: `@import`, window-control rules, drag-region tampering, and `content:` injection on brand elements are stripped, and PNG overrides are limited to a fixed whitelist (`sidebar-logo`, `titlebar-logo`, `background`) that never includes the app icon or the launcher's name. See `docs/THEMES.md`.
- **Global State Management**: Implemented `gameStore.ts` to preserve launch status and game console logs when navigating between pages.
- **Checksum Verification**: Added SHA1 checksum validation to `netutil.DownloadFile` to prevent partial or corrupted downloads from crashing instance launches.
- **Log4j XML Parsing**: The game console now parses raw Log4j XML output into clean, readable text (#14).

### Changed
- **macOS Titlebars**: Updated Wails configuration to use native macOS titlebar and traffic-light controls.
- **Memory Display**: Normalised memory allocation labels (e.g. `4096`, `4G`) into a consistent `X GB` / `X MB` format across the UI (#6).
- **Console Scroll**: The log panel is now horizontally scrollable for long crash messages (#14).

### Fixed
- **Dropdown Mutual Exclusion**: Opening a dropdown will now automatically close any other open dropdowns (#3).
- **macOS Text Selection**: Explicitly enforced `-webkit-user-select: none` to prevent unintended text highlighting on UI elements (#5).
- **Layout Shifts**: Fixed the "Installing" button shifting around when status text changes length (#9).
- **macOS Clipping**: Fixed the sidebar and instance settings cards getting clipped or cutting off absolute-positioned dropdowns during scroll (#7, #11).
- **Extension Tooltips**: Removed the stray native browser tooltip from the Extension iframe view (#13).
- **Modrinth Extension (Registry)**: Modrinth extension now correctly filters compatible instances based on mod loaders and auto-selects the appropriate mod version (#12).

## [v1.0.0-beta.19] - 2026-09-29

### Added
- **Servers API**: Supervised local server processes (start/stop/send/status, EULA gate, logs), `servers:list/ping/manage`, bulk `listWithStatus` with concurrent pings and status cache.
- **Instances API**: `ListWorlds`, `saves:list` and `instances:launch` bridges for quick-launch to server/world, instance icon art on cards with gradient fallback, custom icons from disk, modpack auto-icons, Open Folder button.
- **My-Servers extension**: Official 1.0.0 server and world quick-launch extension.

### Fixed
- Reuse shared vanilla artifacts (jar, libraries, assets) so installs skip re-downloads.
- Guard concurrent install pipelines per instance; unique tmp files; retry checksum mismatches and Windows file-lock renames.
- Support CurseForge App `minecraftinstance.json`, tolerate non-UTF8 metadata, generic game-folder import for Theseus forks; normalize loader IDs to canonical case.
- Surface Play errors via toast; guide to Install when `version.json` is missing; dedupe identical toasts; trim narration comments.

## [v1.0.0-beta.18] - 2026-09-27

No user-facing changes since v1.0.0-beta.17 (release pipeline tag).

## [v1.0.0-beta.17] - 2026-09-27

### Added
- **Import**: Modrinth App instance import support.
- **Dashboard**: 2-column home with recent instances, screenshots, services health, and news feed widgets; compact widget layout with Settings toggles.

### Fixed
- Linux build with `-tags webkit2_41` and `libwebkit2gtk-4.1-dev` for modern distros.
- CI: remove duplicate `GlobalSettings` in `models.ts` to pass `svelte-check`.

## [v1.0.0-beta.16] - 2026-09-16

### Fixed
- Updates: resolve GitHub 404 on release check, canonical repo URLs, fix double `v` in version badge.
- Extensions: canonical gallery URL and prerelease semver comparison support.

## [v1.0.0-beta.15] - 2026-09-16

### Changed
- UI: full-height sidebar with integrated brand header; remove duplicate titlebar header, compact installed extensions list.

## [v1.0.0-beta.14] - 2026-09-10

### Added
- Update modal, dev logs terminal; fix extension install reactivity.
- Sanitize instance display names with symbols into valid IDs (e.g. `Fresh & Smooth` -> `fresh-smooth`).

## [v1.0.0-beta.13] - 2026-09-01

### Added
- Screenshot Manager extension v1.0.0 with sandbox permissions and IPC handlers; screenshots served via local HTTP for native multi-threaded rendering.
- Extension version compatibility gate (block install when launcher is too old).
- Modrinth v1.4.0: Resource Packs and Shaders tabs; modpack browse/install with pagination.
- CurseForge v1.2.0: Mods, Modpacks, Resource Packs, Shaders, pagination, popular content auto-load.

### Fixed
- CurseForge edge.forgecdn.net DNS fallback with User-Agent; Modrinth auto-load top content and background modpack install pipeline.

## [v1.0.0-beta.12] - 2026-08-31

### Changed
- AccountManager card UI refresh; library downloader concurrency/protocol tuning.

### Fixed
- Account dropdown now fixed-position so it escapes sidebar overflow.
- UI Svelte warnings, manifest singleflight, download hardening.

## [v1.0.0-beta.11] - 2026-08-30

### Fixed
- Stop infinite install loop; fix concurrent library download races.

## [v1.0.0-beta.10] - 2026-08-28

No user-facing changes since v1.0.0 (release pipeline tag).

## [v1.0.0] - 2026-08-28

### Added
- **Themes base**: theme packages merged from `feature/themes` (see Unreleased for full description).

### Fixed
- Theme CSS sanitization syntax/compiler errors and manifest fixes.

## [v1.0.0-beta.9] - 2026-08-27

### Fixed
- Linux AppImage: AppRun/`.DirIcon` handling, Arch vs Debian webkit paths, AppRun diagnostics.
- CI: add `svelte-check` to pipeline.

## [v1.0.0-beta.8] - 2026-08-25

### Added
- Help & Support section with bug-report Discord link.
- Modrinth 1.2.2: clean version labels, tooltips, truncation; mod-manager 1.0.1 puzzle-piece icon.
- Extension meta caching for offline resilience (24h TTL, serve on transient DNS fail).

### Fixed
- Microsoft auth resilient to transient DNS (retry + offline hint).
- Discord RPC 1.0.1 edge cases; sandbox IPC serialization + panic recovery.
- Linux AppImage packaging (AppDir + appimagetool); Wails hook path fix with graceful degrade.

## [v1.0.0-beta.7] - 2026-08-23

### Added
- Discord Rich Presence extension (official, auto-start); Quilt mod loader extension source.

### Fixed
- Discord RPC shows Playing for vanilla via Go fallback in `StateChangeHook`.
- Prevent `context canceled` on Quilt 26.2 launch; `vm.Interrupt` execute timeout; clear stale Reloading state.
- Forge/NeoForge 1.0.1 Maven JSON optional with Prism upstream fallback; retry transient mod-loader network errors.
- Async extension reload/update system; hermetic manager tests via temp `.aether` dir; registry force-refresh on update checks.

## [v1.0.0-beta.6] - 2026-08-20

### Added
- Instance import from Prism/MultiMC and CurseForge launchers.

### Fixed
- AppImage self-update; surface Minecraft service connectivity before installs.

## [v1.0.3] - 2026-08-19

### Fixed
- Place app icon at AppDir root so `appimagetool` finds it.

## [v1.0.2] - 2026-08-19

### Added
- In-app auto-updater and Linux AppImage builds; native window titlebar on KDE Wayland.

### Fixed
- Bounded, retrying Linux apt setup so CI cannot hang.

## [v1.0.1] - 2026-08-18

### Added
- Extension update and reload support from the UI; gallery download validation.

## [v1.0.0-beta.5] - 2026-08-18

### Fixed
- Cross-platform packaging and modded launches; macOS `.dmg` extension; Intel macOS runner.

## [v1.0.0-beta.4] - 2026-07-30

### Added
- `CHANGELOG.md` started; multi-account management with switcher/logout; official Microsoft PKCE auth chain; `.aex` file association via NSIS.

### Changed
- Parallel `GetInstances`; robust HTTP downloader with resume + backoff; Java status detects system installs.

### Fixed
- Issues #3-#21 batch: dropdown exclusion, macOS chrome/select/clipping, memory labels, install-button shift, console scroll/XML, toasts, version-load network errors, range-request temp corruption, classpath mutation, Fabric deps, loader hints, gallery redirect, invalid dates, JVM args, CI PR workflow, Wails `go:embed` frontend build.

## [v1.0.0-beta.3] - 2026-07-24

### Added
- SDK/CLI references (`@aethermc/sdk`, `aether-cli`); release download table in README.

### Fixed
- Install button label `.zip` -> `.aex`; localhost `<port>` redirect URI for Azure.

## [v1.0.0-beta.2] - 2026-07-17

Release pipeline tag (see v0.1.0 for changes).

## [v0.1.0] - 2026-07-17

### Added
- Snapshot toggle for instance creation; custom Toast notifications; Forge/NeoForge icons.

### Changed
- Microsoft auth migrated to Device Code Flow; LWJGL 3 classpath fix; logo + website docs.

### Fixed
- Zip download bug; linter/a11y sweep; docs aligned with Goja sandbox.

## [v1.0.0-beta.1] - 2026-07-15

### Added
- Forge and NeoForge mod-loader extensions; Fabric icon and gallery grid layout.

### Fixed
- Placeholder icons replaced; Modrinth `script.js` separator corruption; gallery cache buster.

## [v1.0.0-alpha.2] - 2026-07-13

### Fixed
- Gallery fetch cache buster; real icons in gallery cards.

## [v1.0.0-alpha.1] - 2026-07-13

### Changed
- CI: macOS-14 arm64 runners (macos-13 retired).
