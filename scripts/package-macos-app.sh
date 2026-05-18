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

# Finder/Rosetta can occasionally report a different architecture than the
# binary we bundled. Fall back to any bundled macOS Midir binary.
if [[ ! -x "$BIN" ]]; then
  for candidate in "$RESOURCES_DIR"/Midir-darwin-*; do
    if [[ -x "$candidate" ]]; then
      BIN="$candidate"
      break
    fi
  done
fi

if [[ ! -x "$BIN" ]]; then
  /usr/bin/osascript <<ERR
    display alert "Midir" message "The bundled Midir binary is missing or not executable. Looked in: $RESOURCES_DIR" as critical
ERR
  exit 1
fi

# Launch in a visible Terminal window, like the Windows .exe console.
# Ask which admin account to use, then run through su + sudo in Terminal.
# Closing the Terminal window terminates Midir.
/usr/bin/osascript <<OSA
try
  set dialogResult to display dialog "Enter the macOS admin username to run Midir packet capture:" default answer "" buttons {"Cancel", "Run"} default button "Run"
  set adminUser to text returned of dialogResult
on error number -128
  return
end try

if adminUser is "" then return

set quotedAdminUser to quoted form of adminUser
set quotedResourcesDir to quoted form of "$RESOURCES_DIR"
set quotedBin to quoted form of "$BIN"
set innerCommand to "cd " & quotedResourcesDir & " && sudo " & quotedBin
set midirCommand to "clear; echo 'Midir Damage Meter for macOS'; echo; echo 'Admin user: " & adminUser & "'; echo 'Enter that admin account password when prompted.'; echo 'Close this Terminal window to stop Midir.'; echo; su -l " & quotedAdminUser & " -c " & quoted form of innerCommand & "; echo; echo 'Midir exited. You can close this window.'; read -n 1 -s -r -p 'Press any key to close...'"

tell application "Terminal"
  activate
  do script midirCommand
end tell
OSA
LAUNCHER
chmod 755 "$MACOS_DIR/MidirLauncher"

# Strip local extended attributes before zipping, and disable AppleDouble
# metadata files so release zips do not contain ._ sidecar entries.
xattr -cr "$APP_DIR" 2>/dev/null || true
find "$APP_DIR" -name '.DS_Store' -delete

ZIP_PATH="$BUILD_DIR/${APP_NAME}-macOS-${GOARCH_VALUE}.zip"
rm -f "$ZIP_PATH"
(
  cd "$BUILD_DIR"
  COPYFILE_DISABLE=1 /usr/bin/zip -qry "$(basename "$ZIP_PATH")" "$(basename "$APP_DIR")"
)

echo "Created app bundle: $APP_DIR"
echo "Created zip:        $ZIP_PATH"
echo
echo "Double-click $APP_DIR to run Midir with a macOS admin prompt."
