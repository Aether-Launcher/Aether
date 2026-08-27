# Changelog

All notable changes to this project will be documented in this file.

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
