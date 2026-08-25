#!/usr/bin/env bash
# Packages the Aether Linux binary into an AppImage.
# Usage: bash build/linux/appimage.sh <binary_path>   (run from repo root)
set -euo pipefail

BIN="${1:-build/bin/Aether}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

if [ ! -f "$BIN" ]; then
    echo "[AppImage] Binary not found: $BIN" >&2
    exit 1
fi

APPDIR="build/linux/AppDir"
ICON="build/appicon.png"
DESKTOP="build/linux/aether.desktop"

rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/bin" "$APPDIR/usr/share/icons/hicolor/256x256/apps" \
         "$APPDIR/usr/share/applications"

# Binary
cp "$BIN" "$APPDIR/usr/bin/Aether"
chmod +x "$APPDIR/usr/bin/Aether"

# Icon + .DirIcon (required by appimaged / file managers)
if [ ! -f "$ICON" ]; then
    echo "[AppImage] Warning: $ICON not found; generating 256x256 placeholder." >&2
    ICON="$APPDIR/aether.png"
    if command -v convert >/dev/null 2>&1; then
        convert -size 256x256 xc:'#0d0d0d' "$ICON"
    elif command -v python3 >/dev/null 2>&1; then
        python3 -c "
import zlib, struct
def chunk(t, d):
    c = struct.pack('>I', len(d)) + t + d
    return c + struct.pack('>I', zlib.crc32(t + d) & 0xffffffff)
w = h = 256
raw = b''.join(b'\x00' + b'\x0d\x0d\x0d' * w for _ in range(h))
png = b'\x89PNG\r\n\x1a\n'
png += chunk(b'IHDR', struct.pack('>IIBBBBB', w, h, 8, 2, 0, 0, 0))
png += chunk(b'IDAT', zlib.compress(raw))
png += chunk(b'IEND', b'')
open('$ICON', 'wb').write(png)"
    else
        echo "[AppImage] Error: no tool available to create icon" >&2
        exit 1
    fi
fi
cp "$ICON" "$APPDIR/aether.png"
cp "$ICON" "$APPDIR/usr/share/icons/hicolor/256x256/apps/aether.png"
ln -sf aether.png "$APPDIR/.DirIcon"

# Desktop entries (top-level for appimagetool, applications/ for extracted installs)
if [ ! -f "$DESKTOP" ]; then
    echo "[AppImage] Error: $DESKTOP not found" >&2
    exit 1
fi
cp "$DESKTOP" "$APPDIR/aether.desktop"
cp "$DESKTOP" "$APPDIR/usr/share/applications/aether.desktop"

OUTPUT="build/bin/Aether-linux-amd64.AppImage"
mkdir -p build/bin

TOOL="/tmp/appimagetool-x86_64.AppImage"
run_appimagetool() {
    if command -v appimagetool >/dev/null 2>&1; then
        appimagetool --no-appstream "$APPDIR" "$OUTPUT"
        return $?
    fi
    if ! command -v curl >/dev/null 2>&1; then
        echo "[AppImage] curl unavailable; skipping packaging." >&2
        return 1
    fi
    if [ ! -f "$TOOL" ]; then
        echo "[AppImage] Downloading appimagetool..."
        if ! curl -fsSL -o "$TOOL" \
            "https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage"; then
            echo "[AppImage] Could not download appimagetool; skipping packaging." >&2
            return 1
        fi
    fi
    chmod +x "$TOOL"
    # APPIMAGE_EXTRACT_AND_RUN=1 works on distros without FUSE2
    APPIMAGE_EXTRACT_AND_RUN=1 "$TOOL" --no-appstream "$APPDIR" "$OUTPUT"
}

if run_appimagetool; then
    echo "[AppImage] Built: $OUTPUT"
else
    echo "[AppImage] WARNING: binary built, but AppImage packaging was skipped." >&2
    echo "[AppImage] A ready-to-package AppDir is at: $APPDIR" >&2
fi
