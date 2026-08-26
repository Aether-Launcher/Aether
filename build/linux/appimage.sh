#!/usr/bin/env bash
# Local helper to package a built Aether binary into an AppImage.
# Release CI uses .github/workflows/build.yml as the source of truth;
# this script is for local `wails build` on Linux when you want an AppImage.
# Usage: bash build/linux/appimage.sh build/bin/Aether
set -euo pipefail

BIN="${1:-build/bin/Aether}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

if [ ! -f "$BIN" ]; then
    echo "[AppImage] Binary not found: $BIN" >&2
    echo "[AppImage] Run 'wails build' first." >&2
    exit 1
fi

APPDIR="build/linux/AppDir"
ICON="build/appicon.png"
DESKTOP="build/linux/aether.desktop"

rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/bin" "$APPDIR/usr/share/icons/hicolor/256x256/apps" \
         "$APPDIR/usr/share/applications"

cp "$BIN" "$APPDIR/usr/bin/Aether"
chmod +x "$APPDIR/usr/bin/Aether"
cp "$ICON" "$APPDIR/aether.png"
cp "$ICON" "$APPDIR/usr/share/icons/hicolor/256x256/apps/aether.png"
ln -sf aether.png "$APPDIR/.DirIcon"
cp "$DESKTOP" "$APPDIR/aether.desktop"
cp "$DESKTOP" "$APPDIR/usr/share/applications/aether.desktop"

cat > "$APPDIR/AppRun" <<'APPRUN_EOF'
#!/bin/bash
SELF="$(readlink -f "$0")"
HERE="$(dirname "$SELF")"
export LD_LIBRARY_PATH="$HERE/usr/lib:$HERE/usr/lib/x86_64-linux-gnu${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
exec "$HERE/usr/bin/Aether" "$@"
APPRUN_EOF
chmod +x "$APPDIR/AppRun"

OUTPUT="build/bin/Aether-linux-amd64.AppImage"
mkdir -p build/bin

# Prefer appimagetool from PATH; otherwise download to /tmp (APPIMAGE_EXTRACT_AND_RUN for FUSE-less hosts)
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
        curl -fsSL -o "$TOOL" \
            "https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage"
    fi
    chmod +x "$TOOL"
    APPIMAGE_EXTRACT_AND_RUN=1 "$TOOL" --no-appstream "$APPDIR" "$OUTPUT"
}

if run_appimagetool; then
    echo "[AppImage] Built: $OUTPUT"
    echo "[AppImage] Run with: ./$OUTPUT  (requires host libwebkit2gtk-4.0-37 or libwebkit2gtk-4.1-0)"
else
    echo "[AppImage] WARNING: packaging skipped. AppDir ready at $APPDIR" >&2
fi
