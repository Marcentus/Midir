#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_NAME="${APP_NAME:-Midir}"
GOOS_VALUE="darwin"
GOARCH_VALUE="${GOARCH:-$(go env GOARCH)}"
BUILD_DIR="$ROOT_DIR/build"
BINARY_NAME="Midir-${GOOS_VALUE}-${GOARCH_VALUE}"
APP_DIR="$BUILD_DIR/${APP_NAME}.app"
CONTENTS_DIR="$APP_DIR/Contents"
MACOS_DIR="$CONTENTS_DIR/MacOS"
RESOURCES_DIR="$CONTENTS_DIR/Resources"
SUPPORT_DIR="/Users/Shared/Midir"

cd "$ROOT_DIR"
GOOS="$GOOS_VALUE" GOARCH="$GOARCH_VALUE" OUTPUT_NAME="$BINARY_NAME" ./build.sh

rm -rf "$APP_DIR"
mkdir -p "$MACOS_DIR" "$RESOURCES_DIR"
cp "$BUILD_DIR/$BINARY_NAME" "$RESOURCES_DIR/$BINARY_NAME"
chmod 755 "$RESOURCES_DIR/$BINARY_NAME"

cat > "$CONTENTS_DIR/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key>
  <string>en</string>
  <key>CFBundleExecutable</key>
  <string>MidirLauncher</string>
  <key>CFBundleIdentifier</key>
  <string>com.midir.damage-meter</string>
  <key>CFBundleInfoDictionaryVersion</key>
  <string>6.0</string>
  <key>CFBundleName</key>
  <string>${APP_NAME}</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>0.1.0</string>
  <key>CFBundleVersion</key>
  <string>1</string>
  <key>LSMinimumSystemVersion</key>
  <string>11.0</string>
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
PLIST

cat > "$MACOS_DIR/MidirLauncher" <<'LAUNCHER'
#!/usr/bin/env bash
set -euo pipefail

APP_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RESOURCES_DIR="$APP_DIR/Resources"
BIN=""
case "$(uname -m)" in
  arm64) BIN="$RESOURCES_DIR/Midir-darwin-arm64" ;;
  x86_64) BIN="$RESOURCES_DIR/Midir-darwin-amd64" ;;
  *) BIN="$RESOURCES_DIR/Midir-darwin-arm64" ;;
esac

if [[ ! -x "$BIN" ]]; then
  osascript -e 'display alert "Midir" message "The bundled Midir binary is missing or not executable." as critical'
  exit 1
fi

SUPPORT_DIR="/Users/Shared/Midir"
mkdir -p "$SUPPORT_DIR"
chmod 755 "$SUPPORT_DIR" 2>/dev/null || true

# Midir needs packet-capture privileges. AppleScript provides the normal macOS
# admin-password prompt. Keep data files in /Users/Shared/Midir instead of
# writing logs/settings inside the app bundle.
/usr/bin/osascript <<OSA
try
  do shell script "cd '$SUPPORT_DIR' && '$BIN'" with administrator privileges
on error errMsg number errNum
  if errNum is not -128 then
    display alert "Midir failed to start" message errMsg as critical
  end if
end try
OSA
LAUNCHER
chmod 755 "$MACOS_DIR/MidirLauncher"

# Keep a copy of the CLI binary in /Users/Shared for manual fallback/testing.
mkdir -p "$SUPPORT_DIR"
cp "$BUILD_DIR/$BINARY_NAME" "$SUPPORT_DIR/$BINARY_NAME"
chmod 755 "$SUPPORT_DIR" "$SUPPORT_DIR/$BINARY_NAME"

ZIP_PATH="$BUILD_DIR/${APP_NAME}-macOS-${GOARCH_VALUE}.zip"
rm -f "$ZIP_PATH"
/usr/bin/ditto -c -k --keepParent "$APP_DIR" "$ZIP_PATH"

echo "Created app bundle: $APP_DIR"
echo "Created zip:        $ZIP_PATH"
echo
echo "Double-click $APP_DIR to run Midir with a macOS admin prompt."
