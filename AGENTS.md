# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go application for discovering and controlling Yeelight CubeLite (Matrix) devices over the local network. The project implements the Yeelight protocol for SSDP device discovery and TCP-based command control, with specific support for the Matrix LED display functionality.

## Build and Run Commands

```bash
# Build the application
go build

go run .
```

## API Specification and Code Generation

The project uses an **API spec-first development approach** with OpenAPI 3.1 and automatic code generation.

### OpenAPI Specification

The API is defined in `spec.yml` (OpenAPI 3.1 format):
- **Endpoint**: `GET /api/devices`
- **Server**: `http://localhost:8080`
- **Response**: JSON array of devices with `id` and `name` fields
- **Error handling**: 500 responses with error messages

### Code Generation with ogen-go

The project uses [ogen-go/ogen](https://github.com/ogen-go/ogen) for automatic Go code generation from the OpenAPI spec:

```bash
# Generate API code from spec.yml
go generate ./...

# This creates the api/ directory with ~17 auto-generated files
# The api/ directory is gitignored as it's reproducible from spec.yml
```

**Generated code includes:**
- Server implementation (`api.NewServer`)
- Request/response types (`api.Device`, `api.GetDevicesOK`, `api.Error`)
- Handler interface (`api.Handler`)
- JSON serialization/deserialization
- HTTP routing and middleware
- **Automatic validation** - ogen validates all inputs based on OpenAPI spec constraints before calling handlers

### Validation Behavior

**IMPORTANT**: ogen automatically validates all request data based on the OpenAPI spec. Do NOT implement manual validation in handlers for:
- Pattern matching (e.g., `pattern: '^yeelight://[0-9.]+:[0-9]+$'`)
- Min/max length constraints (e.g., `minLength: 1`, `maxLength: 100`)
- Required fields (e.g., `required: [device_id, name, frames]`)
- Array constraints (e.g., `minItems: 1`)
- Number ranges (e.g., `minimum: 0`, `maximum: 255`)
- Type validation (e.g., string, integer, UUID format)

If validation fails, ogen returns a 400 Bad Request with error details **before** the handler is called. Handlers only receive valid data that meets all spec constraints.

### API Implementation Files

**Manual implementation (checked into git):**
- `spec.yml` - OpenAPI 3.1 specification (source of truth)
- `generate.go` - go:generate directive for code generation
- `handler.go` - Implements `api.Handler` interface, calls `DiscoverDevices()`
- `server.go` - HTTP server setup with CORS middleware on port 8080

**Auto-generated (gitignored):**
- `api/*.go` - Generated server code (~17 files)

### Running Modes

The application supports two modes via command-line flag:

**Demo Mode (default):**
```bash
./cubik
```
- Discovers devices and runs animated LED patterns
- Original CLI demonstration behavior

**Server Mode:**
```bash
./cubik --server
```
- Starts HTTP API server on port 8080
- Endpoint: `GET http://localhost:8080/api/devices`
- CORS enabled for frontend integration
- Live device discovery on each API request (3-second timeout)

### Modifying the API

1. Update `spec.yml` with new endpoints/schemas
2. Run `go generate ./...` to regenerate backend code
3. Run `cd front && bun run generate-api` to regenerate frontend client
4. Implement new handler methods in `handler.go`
5. Run `go build` to compile

**Example workflow:**
```bash
# Edit spec.yml to add new endpoint
vim spec.yml

# Regenerate backend API code
go generate ./...

# Regenerate frontend API client
cd front && bun run generate-api

# Implement handler method
vim handler.go

# Build and test
go build
./cubik --server
curl http://localhost:8080/api/devices
```

### Frontend API Client Generation

The project includes automatic TypeScript client generation for the frontend:

**Generate the client:**
```bash
cd front
bun run generate-api
```

This generates TypeScript types and API client code in `front/src/api/generated/` using the `openapitools/openapi-generator-cli` Docker image.

**Using the generated client:**
```typescript
import { DefaultApi, Configuration } from '$lib/api/generated';

const api = new DefaultApi(new Configuration({
  basePath: 'http://localhost:8080'
}));

// Get devices
const response = await api.getDevices();
console.log(response.devices);

// Start animation
await api.startAnimation({
  startAnimationRequest: {
    device_location: 'yeelight://192.168.1.100:55443',
    frames: [
      [{ r: 255, g: 0, b: 0 }, { r: 0, g: 255, b: 0 }]
    ]
  }
});
```

The generated code is gitignored and should be regenerated after any changes to `spec.yml`.

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
   - `encodeRGBColor()`: Converts single RGB color (0-255 each) to 4-char base64 string

3. **Matrix Patterns and Animations (animations.go)**
   - `createSolidColor()`: Generates uniform color pattern for all LEDs
   - `createCheckerboard()`: Creates alternating red/blue pattern
   - `createGradient()`: Generates rainbow gradient across LEDs

4. **Matrix Display and Rendering (drawing.go)**
   - `Framebuffer` and `Digit Rendering System`: Provides high-level API for drawing digits and text on the matrix display.
   - `DrawDigit`, `DrawNumber`, `DrawString`: Rendering functions for the matrix.

5. **HTTP API Server (server.go, handler.go)**
   - `StartServer()`: Initializes HTTP server on port 8080 with CORS middleware
   - `APIHandler`: Implements ogen-generated `api.Handler` interface
   - `GetDevices()`: REST endpoint that calls `DiscoverDevices()` and transforms results to JSON
   - CORS middleware allows frontend access from any origin (development mode)

6. **Demo Program (main.go)**
   - Command-line flag parsing for server vs demo mode
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
- External dependencies:
  - `github.com/ogen-go/ogen` - OpenAPI code generation
  - Standard library for core functionality (no deps for device discovery/control)
- Matrix devices may send multiple SSDP responses (normal for UDP reliability)
- Some commented-out code in main.go shows previous attempts at property querying
- Demo mode runs infinite loop - user must Ctrl+C to exit
- Yeelight cube device is limited to 60 RPS. Make sure not to exceed it
- API code in `api/` directory is auto-generated - never edit manually, regenerate from `spec.yml`