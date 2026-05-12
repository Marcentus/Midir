# Midir on macOS

This fork can run Midir on a **secondary Mac** that receives mirrored Mabinogi traffic from the gaming PC.

The Mac does not need to run Mabinogi. It only captures packets from the mirrored switch/router port, parses damage packets, and serves the web UI on port `8030`.

## Recommended network setup

Use a dedicated USB/Ethernet adapter for the mirrored port if possible:

- Gaming PC: normal connection to the switch/router.
- Switch/router: mirror the gaming PC's inbound traffic to a monitor port.
- Mac: connect the dedicated Ethernet adapter to the monitor port.
- Mac's normal internet can remain on Wi-Fi or another Ethernet adapter.

## Build on macOS

Requirements:

- Go 1.24+
- Node.js + npm
- Xcode Command Line Tools (`xcode-select --install`)

Build native binary:

```bash
./build.sh
```

The binary is written to `build/Midir-darwin-arm64` on Apple Silicon or `build/Midir-darwin-amd64` on Intel Macs.

Cross-build examples:

```bash
GOOS=darwin GOARCH=arm64 ./build.sh
GOOS=darwin GOARCH=amd64 ./build.sh
```


## Double-click macOS app bundle

You can create a local `.app` bundle that prompts for admin permission and starts Midir without using Terminal:

```bash
./scripts/package-macos-app.sh
```

Outputs:

- `build/Midir.app` — double-clickable app bundle
- `build/Midir-macOS-arm64.zip` or `build/Midir-macOS-amd64.zip` — zip suitable for a GitHub release asset

When launched, the app asks for a macOS admin password because packet capture requires elevated privileges. Runtime files such as `settings.json`, logs, and session data are written under `/Users/Shared/Midir`.

## Run on macOS

Packet capture usually requires elevated permissions on macOS:

```bash
sudo ./build/Midir-darwin-arm64
```

Then open:

```text
http://127.0.0.1:8030
```

You can also access it from another device on the same network using the URL printed in the console.

## Interface selection

In the web UI settings, select the interface connected to the mirrored port. macOS interface names usually look like:

- `en0`, `en1`, etc. for built-in or USB Ethernet/Wi-Fi
- `bridge*`, `utun*`, `llw*`, etc. for virtual/tunnel interfaces

For a mirrored port, the correct choice is normally the Ethernet adapter attached to the monitor port, not Wi-Fi.

You can list interfaces from the CLI too:

```bash
sudo ./build/Midir-darwin-arm64 list
```

## ExitLag notes

If ExitLag is running on the gaming PC:

- Keep TCP tunnels set to `1`.
- Turn real-time optimizations off.
- Use IPv4 route analysis.
- In Midir, enable ExitLag mode and use auto-detect on the mirrored Mac interface.

## Troubleshooting

If no packets appear:

1. Confirm port mirroring is enabled and the Mac is connected to the monitor port.
2. Confirm you selected the mirrored Ethernet interface, not Wi-Fi.
3. Run with `sudo`.
4. Try `list` mode to verify the interface is visible.
5. Check whether the switch mirrors ingress, egress, or both directions for the gaming PC port.

