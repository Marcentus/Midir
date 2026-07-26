# Midir on macOS

This fork can run Midir on a **secondary Mac** that receives mirrored Mabinogi traffic from the gaming PC.

The Mac does not need to run Mabinogi. It only captures packets from the mirrored switch/router port, parses damage packets, and serves the web UI on port `8030`.

## Quick start: double-click app

For normal use, use the bundled macOS app instead of launching from Terminal.

1. Download/extract the macOS release zip, or build it with `./scripts/package-macos-app.sh`.
2. Move `Midir.app` to `Applications` or `~/Applications`.
3. Double-click `Midir.app`.
4. Enter the macOS **admin username** when prompted.
5. A Terminal window opens. Enter that admin account password when prompted.
6. Midir starts and opens the web UI at:

```text
http://127.0.0.1:8030
```

You can also access it from another device on the same network using the LAN URL printed in the Terminal window.

### Stopping Midir

Close the Terminal window that Midir opened. This stops the Midir process, similar to closing the command window on Windows.

### First launch warning

The app is currently unsigned. If macOS blocks it the first time:

1. Right-click `Midir.app`.
2. Choose **Open**.
3. Confirm you want to open it.

After that, normal double-click launching should work.

## Recommended network setup

Use a dedicated Ethernet connection for the mirrored port if possible:

- Gaming PC: normal connection to the switch/router.
- Switch/router: mirror the gaming PC traffic to a monitor/mirroring port.
- Mac: connect its Ethernet port or USB Ethernet adapter to the monitor/mirroring port.
- Mac's normal internet can remain on Wi-Fi or another Ethernet adapter.

For port mirroring, the correct interface is the Mac adapter connected to the monitor/mirroring port. It may be `en0`, `en5`, etc. depending on the hardware.

## Interface selection

In the web UI settings, select the interface connected to the mirrored port. macOS interface names usually look like:

- `en0`, `en1`, etc. for built-in Ethernet/Wi-Fi
- `en5`, `en6`, etc. for USB Ethernet adapters
- `bridge*`, `utun*`, `llw*`, etc. for virtual/tunnel interfaces

If you are unsure which interface is receiving mirrored traffic, run this from Terminal:

```bash
sudo tcpdump -i en0 -nn -e 'host <gaming-pc-ip>'
```

Replace `en0` and `<gaming-pc-ip>` as needed. If the interface is correct, you should see packets involving the gaming PC.

## ExitLag notes

If ExitLag is running on the gaming PC:

- Keep TCP tunnels set to `1`.
- Turn real-time optimizations off.
- Use IPv4 route analysis.
- In Midir, enable ExitLag mode and use auto-detect on the mirrored Mac interface.
- If another working Midir instance already detected the ExitLag IP/port, you can manually enter the same IP/port on macOS.

Occasional logs like this are usually parser resync warnings and are not always fatal:

```text
ExitLag Parse Error: Unreasonable sequence length: ...
```

If they happen constantly and no DPS data appears, re-run auto-detect and confirm the mirrored interface is seeing the correct TCP stream.

## Mudfish notes

If Mudfish is running on the gaming PC:

- Select your normal Ethernet NIC (the mirrored switch port on a capture Mac, or the game PC's physical NIC for same-PC capture). Do not select Mudfish's virtual adapter.
- In Midir, enable Mudfish mode so outer UDP/TCP tunnel payloads are unwrapped to inner game TCP.
- Mudfish Connection Protocol can stay UDP.
- ExitLag and Mudfish modes are mutually exclusive.

Useful capture check for Mudfish tunnels:

```bash
# Mirrored Ethernet: look for UDP/TCP to Mudfish nodes carrying game traffic.
sudo tcpdump -i en0 -nn -e 'udp or tcp and host <gaming-pc-ip>'
```

## Troubleshooting

If no packets appear:

1. Confirm the Mac is connected to the switch monitor/mirroring port.
2. Confirm the selected interface is the mirrored Ethernet interface, not Wi-Fi or a virtual interface.
3. Confirm the app is running through the opened Terminal/admin prompt.
4. Use `tcpdump` to verify the Mac sees traffic from the gaming PC.
5. Check switch port statistics: the Mac's port should receive mirrored traffic when the gaming PC is active.
6. Check whether the switch mirrors ingress, egress, or both directions for the gaming PC port.

Useful capture checks:

```bash
# Replace with the gaming PC IP.
sudo tcpdump -i en0 -nn -e 'host <gaming-pc-ip>'

# ExitLag/game endpoint example; replace with detected values.
sudo tcpdump -i en0 -nn -e 'host <exitlag-relay-ip> or port <game-port>'
```

## Developer build

Requirements:

- Go 1.24+
- Node.js + npm
- Xcode Command Line Tools (`xcode-select --install`)

Build a native CLI binary:

```bash
./build.sh
```

The binary is written to `build/Midir-darwin-arm64` on Apple Silicon or `build/Midir-darwin-amd64` on Intel Macs.

Build the double-clickable app bundle and release zip:

```bash
./scripts/package-macos-app.sh
```

Outputs:

- `build/Midir.app` — double-clickable app bundle
- `build/Midir-macOS-arm64.zip` or `build/Midir-macOS-amd64.zip` — zip suitable for a GitHub release asset

Runtime files such as `settings.json`, logs, and session data are written relative to the app's launch directory.

Cross-build examples:

```bash
GOOS=darwin GOARCH=arm64 ./build.sh
GOOS=darwin GOARCH=amd64 ./build.sh
```
