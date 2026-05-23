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

cat > "$RESOURCES_DIR/install-bpf-permissions.sh" <<'BPFINSTALL'
#!/usr/bin/env bash
set -euo pipefail

GROUP="access_bpf"
USER_NAME="${SUDO_USER:-$(stat -f %Su /dev/console)}"
LAUNCH_DAEMON="/Library/LaunchDaemons/com.midir.chmodbpf.plist"
HELPER="/Library/Application Support/Midir/chmodbpf.sh"

if ! /usr/bin/dscl . -read "/Groups/${GROUP}" >/dev/null 2>&1; then
  /usr/sbin/dseditgroup -q -o create "$GROUP"
fi

/usr/sbin/dseditgroup -q -o edit -a "$USER_NAME" -t user "$GROUP"

/bin/mkdir -p "/Library/Application Support/Midir"
/bin/cat > "$HELPER" <<HELPER_SCRIPT
#!/usr/bin/env bash
set -euo pipefail

GROUP="access_bpf"
USER_NAME="$USER_NAME"
for dev in /dev/bpf*; do
  [ -e "\$dev" ] || continue
  /usr/sbin/chown root:"\$GROUP" "\$dev" || true
  /bin/chmod 660 "\$dev" || true
  /bin/chmod +a "\$USER_NAME allow read,write" "\$dev" 2>/dev/null || true
done
HELPER_SCRIPT
/usr/sbin/chown root:wheel "$HELPER"
/bin/chmod 755 "$HELPER"

/bin/cat > "$LAUNCH_DAEMON" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.midir.chmodbpf</string>
  <key>ProgramArguments</key>
  <array>
    <string>$HELPER</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>StartInterval</key>
  <integer>60</integer>
</dict>
</plist>
PLIST
/usr/sbin/chown root:wheel "$LAUNCH_DAEMON"
/bin/chmod 644 "$LAUNCH_DAEMON"

/bin/launchctl bootout system "$LAUNCH_DAEMON" >/dev/null 2>&1 || true
/bin/launchctl bootstrap system "$LAUNCH_DAEMON" >/dev/null 2>&1 || true
/bin/launchctl kickstart -k system/com.midir.chmodbpf >/dev/null 2>&1 || "$HELPER"
"$HELPER"

echo "Installed Midir packet capture permissions for ${USER_NAME}."
echo "If capture still fails, log out and back in once so macOS refreshes group membership."
BPFINSTALL
chmod 755 "$RESOURCES_DIR/install-bpf-permissions.sh"

cat > "$RESOURCES_DIR/uninstall-macos.sh" <<'UNINSTALL'
#!/usr/bin/env bash
set -euo pipefail

GROUP="access_bpf"
LAUNCH_DAEMON="/Library/LaunchDaemons/com.midir.chmodbpf.plist"
HELPER_DIR="/Library/Application Support/Midir"
USER_DATA_DIR="${HOME}/Library/Application Support/Midir"

cat <<INFO
Midir clean uninstall helper

This removes the system-level packet capture helper installed by Midir:
  ${LAUNCH_DAEMON}
  ${HELPER_DIR}/chmodbpf.sh

It can also remove this user's Midir data:
  ${USER_DATA_DIR}

The Midir.app bundle itself is not removed by this script.
INFO

read -r -p "Remove system packet capture helper? [y/N] " remove_helper
if [[ "$remove_helper" =~ ^[Yy]$ ]]; then
  /usr/bin/osascript <<OSA
try
  do shell script "/bin/launchctl bootout system '$LAUNCH_DAEMON' >/dev/null 2>&1 || true; /bin/rm -f '$LAUNCH_DAEMON'; /bin/rm -f '$HELPER_DIR/chmodbpf.sh'; /bin/rmdir '$HELPER_DIR' >/dev/null 2>&1 || true; /usr/sbin/dseditgroup -q -o delete '$GROUP' >/dev/null 2>&1 || true" with administrator privileges
  display dialog "Midir packet capture helper removed." buttons {"OK"} default button "OK"
on error errMsg
  display alert "Midir uninstall" message ("Could not remove packet capture helper:\n\n" & errMsg) as critical
end try
OSA
fi

read -r -p "Remove this user's Midir settings, database, and logs? [y/N] " remove_data
if [[ "$remove_data" =~ ^[Yy]$ ]]; then
  /bin/rm -rf "$USER_DATA_DIR"
  echo "Removed: $USER_DATA_DIR"
fi

echo "Done. You can delete Midir.app manually if you have not already."
UNINSTALL
chmod 755 "$RESOURCES_DIR/uninstall-macos.sh"

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

RUNTIME_DIR="${MIDIR_RUNTIME_DIR:-$HOME/Library/Application Support/Midir}"
mkdir -p "$RUNTIME_DIR"
LOG_DIR="$RUNTIME_DIR/logs"
mkdir -p "$LOG_DIR"

can_capture_packets() {
  for dev in /dev/bpf*; do
    [[ -r "$dev" && -w "$dev" ]] && return 0
  done
  return 1
}

install_capture_permissions() {
  /usr/bin/osascript <<OSA
try
  display dialog "Midir needs one-time packet capture permissions. This installs a small LaunchDaemon that grants the current user access to /dev/bpf*, similar to Wireshark." buttons {"Cancel", "Install"} default button "Install" with icon caution
on error number -128
  return "cancelled"
end try

try
  do shell script quoted form of "$RESOURCES_DIR/install-bpf-permissions.sh" with administrator privileges
  display dialog "Packet capture permissions were installed. If Start Capture still fails, log out and back in once, then reopen Midir." buttons {"Open Midir"} default button "Open Midir"
  return "installed"
on error errMsg
  display alert "Midir" message ("Could not install packet capture permissions:\n\n" & errMsg) as critical
  return "failed"
end try
OSA
}

if ! can_capture_packets; then
  result="$(install_capture_permissions)"
  if [[ "$result" != *installed* ]]; then
    exit 1
  fi
  if ! can_capture_packets; then
    /usr/bin/osascript <<'ERR'
display alert "Midir" message "Packet capture permissions were installed, but macOS has not applied them to this login session yet. Log out and back in once, then reopen Midir." as critical
ERR
    exit 1
  fi
fi

# Launch in a visible Terminal window, like the Windows .exe console.
# Closing the Terminal window terminates Midir.
/usr/bin/osascript <<OSA
set quotedRuntimeDir to quoted form of "$RUNTIME_DIR"
set quotedBin to quoted form of "$BIN"
set quotedLogDir to quoted form of "$LOG_DIR"
set quotedLogFile to quoted form of "$LOG_DIR/midir.log"
set innerCommand to "mkdir -p " & quotedRuntimeDir & " " & quotedLogDir & " && cd " & quotedRuntimeDir & " && " & quotedBin & " 2>&1 | tee -a " & quotedLogFile
set midirCommand to "clear; echo 'Midir Damage Meter for macOS'; echo; echo 'Runtime dir: $RUNTIME_DIR'; echo 'Log dir: $LOG_DIR'; echo 'Close this Terminal window to stop Midir.'; echo; " & innerCommand & "; echo; echo 'Midir exited. You can close this window.'; read -n 1 -s -r -p 'Press any key to close...'"

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
echo "Double-click $APP_DIR to run Midir. First launch may ask for admin approval to install packet capture permissions."
