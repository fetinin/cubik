# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go application for discovering and controlling Yeelight CubeLite (Matrix) devices over the local network. The project implements the Yeelight protocol for SSDP device discovery and TCP-based command control, with specific support for the Matrix LED display functionality.

## Build and Run Commands

```bash
# Build the application
go build

# Run the application
go run .

# Run with pre-approval for go build
go build
./cubik
```

## Code Architecture

### Core Components

1. **Device Discovery (discovery.go)**
   - `DiscoverDevices()`: UDP multicast SSDP discovery on `239.255.255.250:1982`
   - `parseDeviceInfo()`: Parses SSDP HTTP-like response headers into `DeviceInfo` struct
   - Only processes devices with model "CubeLite"
   - Implements deduplication based on device Location
   - 3-second timeout for collecting responses

2. **Command Protocol (commands.go)**
   - `SendCommand()`: Establishes TCP connection, sends JSON-RPC command, waits for response
   - `SendCommandNoResponse()`: Fire-and-forget command sending (used for high-frequency Matrix updates)
   - All messages formatted as JSON with `\r\n` terminator
   - Commands use structure: `{"id": <int>, "method": <string>, "params": [<array>]}`

3. **Matrix LED Control (commands.go)**
   - `ActivateFxMode()`: Enables "direct" mode required for manual LED control
   - `UpdateLeds()`: Sends base64-encoded RGB data to update all LEDs
   - `encodeRGBColor()`: Converts single RGB color (0-255 each) to 4-char base64 string
   - `createSolidColor()`: Generates uniform color pattern for all LEDs
   - `createCheckerboard()`: Creates alternating red/blue pattern
   - `createGradient()`: Generates rainbow gradient across LEDs
   - `Framebuffer` and `Digit Rendering System`: Provides high-level API for drawing digits and text on the matrix display.

4. **Demo Program (main.go)**
   - Discovers CubeLite devices on network
   - Tests power toggle functionality
   - Demonstrates Matrix LED control with animated patterns
   - Runs infinite loop cycling through test patterns

### Data Structures

**DeviceInfo**: Contains all device metadata from SSDP response
- Location (format: `yeelight://IP:PORT`)
- ID, Model, FwVer, Support capabilities
- Current state: Power, Bright, ColorMode, CT, RGB, Hue, Sat, Name

**CommandRequest/Response**: JSON-RPC protocol structures

### Important Protocol Details

1. **Matrix LED Encoding**:
   - Each LED requires 3 bytes (R, G, B) → base64 encodes to 4 characters
   - For 20x5 matrix (100 LEDs): 100 LEDs × 4 chars = 400 characters total
   - Must call `ActivateFxMode({"mode": "direct"})` before any `UpdateLeds()` calls
   - LEDs addressed in row-major order

2. **Connection Flow**:
   - Parse location string to extract `IP:PORT` from `yeelight://` prefix
   - Establish TCP connection with 3-second timeout
   - Send commands as JSON + `\r\n`
   - For Matrix updates, use `SendCommandNoResponse()` for performance

3. **Device Requirements**:
   - "LAN Control" must be enabled in Yeelight app for TCP control to work
   - Default port: 55443
   - Must be on same network for discovery

## Reference Documentation

See `docs/yeelight-protocol-guide.md` for comprehensive protocol documentation including:
- Full command reference (set_power, toggle, set_bright, set_rgb, etc.)
- Music mode for high-frequency updates (bypasses rate limiting)
- Multi-module Matrix layouts and image display
- Color encoding details and conversion utilities
- Troubleshooting common issues

Additional protocol documentation in:
- `docs/yeelight-protocol.md`
- `docs/yeelight_protocol_analysis.md`

## Development Notes

- The project uses Go 1.25.5
- No external dependencies beyond standard library
- Matrix devices may send multiple SSDP responses (normal for UDP reliability)
- Some commented-out code in main.go shows previous attempts at property querying
- Current demo runs infinite loop - user must Ctrl+C to exit
- Yeelight cube device is limited to 60 RPS. Make sure not to exceed it