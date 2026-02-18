# Claude Usage Meter

A lightweight desktop widget for monitoring your [Claude Code](https://claude.ai/code) usage in real time. Shows token consumption, estimated cost, and progress toward your subscription limit — living in your system tray.

## Platform support

| Platform | Distribution | Notes |
|----------|-------------|-------|
| Windows 11 | Portable `.exe` | Primary development platform |
| macOS 12+ | `.app` bundle | Requires Gatekeeper workaround (unsigned) |
| Linux | Binary | Requires WebKitGTK 4.1 |

## Download

Pre-built binaries are available on the [GitHub Releases](../../releases) page:

- **Windows**: `claude_usage_meter.exe`
- **macOS**: `Claude_Usage_Meter_macOS.zip`
- **Linux**: `claude_usage_meter_linux`

## Installation

### Windows

1. Download `claude_usage_meter.exe` from Releases
2. Run it — a tray icon appears in the system tray
3. Double-click the tray icon to show/hide the widget

No installation required. Place the `.exe` wherever you like (e.g. `%LOCALAPPDATA%\ClaudeUsageMeter\`).

### macOS

1. Download `Claude_Usage_Meter_macOS.zip` from Releases
2. Unzip and drag **Claude Usage Meter.app** to `/Applications`
3. Because the app is unsigned, macOS Gatekeeper will block it on first launch. Run this once in Terminal:

```bash
xattr -cr "/Applications/Claude Usage Meter.app"
```

4. Launch the app normally from Finder or Spotlight

### Linux

1. Download `claude_usage_meter_linux` from Releases
2. Install the WebKitGTK dependency:

```bash
sudo apt install libwebkit2gtk-4.1-0   # Debian/Ubuntu
sudo dnf install webkit2gtk4.1         # Fedora
```

3. Make the binary executable and run it:

```bash
chmod +x claude_usage_meter_linux
./claude_usage_meter_linux
```

## Building from source

### Prerequisites

- [Go 1.21+](https://go.dev/dl/)
- Wails v3 CLI:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.72
```

- **Linux only**: WebKitGTK dev headers and GCC:

```bash
sudo apt install libwebkit2gtk-4.1-dev gcc
```

### Build commands

**Windows** (produces a GUI `.exe` with no console window):
```bash
go build -tags production -ldflags="-s -w -H windowsgui" -o claude_usage_meter.exe .
```

**macOS** (produces a `.app` bundle via Wails):
```bash
wails3 build
```

**Linux**:
```bash
go build -tags production -ldflags="-s -w" -o claude_usage_meter_linux .
```

## Configuration

The app reads Claude Code's local JSONL logs from `~/.claude/projects/**/*.jsonl` to estimate usage, and optionally calls the Anthropic API for real-time data.

To override the default budget tier, create `~/.claude/usage-meter-config.json`:

```json
{
  "plan": "max_5x"
}
```

Supported values for `plan`: `pro`, `max_5x`, `max_20x`.

## License

MIT
