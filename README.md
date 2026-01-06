# Cubik

A web application for discovering and controlling Yeelight CubeLite (Matrix) LED devices over your local network. Create custom animations, draw patterns, and control your CubeLite's 20×5 LED matrix display through an intuitive web interface.

## Features

- **Device Discovery**: Automatic SSDP discovery of Yeelight CubeLite devices on your local network
- **Visual Editor**: Interactive matrix grid for drawing and creating LED patterns
- **Animation Creator**: Build multi-frame animations with frame management and preview
- **Single Binary Deployment**: Complete frontend and backend packaged in one executable

## Quick Start with Docker

The fastest way to get started is using Docker:

```bash
# Build the image
docker build -t cubik:latest .

# Run the container (ephemeral - data lost on restart)
docker run -p 9080:9080 cubik:latest

# Access the application
open http://localhost:9080
```

### Running with Persistent Database Storage

To preserve your animations and settings across container restarts, mount a volume for the database:

```bash
# Create a directory for persistent data
mkdir -p data

# Run with persistent storage
docker run -p 9080:9080 \
  -v $(pwd)/data:/data \
  -e SERVER_DB_PATH=/data/cubik.db \
  cubik:latest
```

**What this does:**
- `-v $(pwd)/data:/data` mounts your local `data/` directory into the container
- `-e SERVER_DB_PATH=/data/cubik.db` tells the app to store the database in the mounted volume
- Your animations and settings will persist in `data/cubik.db` even when the container is stopped or removed

## Prerequisites

### For Local Development

- **Go**: 1.25.5 or later
- **Bun**: Latest version (package manager for frontend)
- **Docker**: Optional, for containerized deployment

### For Yeelight Device Control

- Yeelight CubeLite device on the same local network
- "LAN Control" enabled in the Yeelight mobile app (Settings → Device → LAN Control)

## Local Development Setup

### Backend

```bash
# Install Go dependencies
go mod download

# Generate API code from OpenAPI spec
go generate ./...

# Build the application
go build

# Run in server mode
./cubik 
```

The server will start on `http://localhost:9080`

### Frontend

```bash
cd front

# Install dependencies
bun install

# Generate TypeScript API client from spec
bun run generate-api

# Start development server
bun run dev

# Build for production
bun run build
```

## Project Structure

```
cubik/
├── main.go              # Application entry point
├── server.go            # HTTP server with embedded frontend
├── handler.go           # API endpoint handlers
├── discovery.go         # SSDP device discovery
├── commands.go          # Yeelight protocol commands
├── animations.go        # Animation helpers
├── drawing.go           # Matrix framebuffer rendering
├── storage.go           # Animation persistence
├── database.go          # SQLite database setup
├── migrations/          # Database migration files
├── spec.yml             # OpenAPI 3.1 API specification
├── generate.go          # go:generate directive for ogen
├── api/                 # Auto-generated API code (gitignored)
├── front/               # SvelteKit frontend
│   ├── src/
│   │   ├── routes/      # SPA pages
│   │   ├── lib/
│   │   │   ├── components/   # Svelte components
│   │   │   ├── state/        # State management
│   │   │   └── api/          # API client (generated)
│   └── build/           # Production build output
├── docs/                # Protocol documentation
├── Dockerfile           # Multi-stage Docker build
└── .dockerignore        # Docker build exclusions
```

## API Development Workflow

This project uses API-first development with OpenAPI 3.1:

1. **Edit the spec**: Modify `spec.yml` to add/change endpoints
2. **Regenerate backend**: Run `go generate ./...` to update Go server code
3. **Regenerate frontend**: Run `cd front && bun run generate-api` for TypeScript client
4. **Implement handlers**: Add handler logic in `handler.go`
5. **Build**: Run `go build` to compile

The API code is automatically generated and should never be edited manually.

### Environment Variables

The application supports the following environment variables for configuration:

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | HTTP server port | `9080` |
| `SERVER_DB_PATH` | SQLite database file path | `cubik.db` |

**Example usage:**

```bash
# Run with custom port
SERVER_PORT=3000 ./cubik

# Run with custom database path
SERVER_DB_PATH=/data/cubik.db ./cubik

# Run with both
SERVER_PORT=3000 SERVER_DB_PATH=/data/cubik.db ./cubik
```

**Docker usage:**

```bash
# Run with environment variables
docker run -p 3000:3000 -e SERVER_PORT=3000 cubik:latest

# Run with volume-mounted database
docker run -p 9080:9080 -v $(pwd)/data:/data -e SERVER_DB_PATH=/data/cubik.db cubik:latest
```

## Yeelight Protocol Details

Cubik implements the Yeelight LAN protocol:

- **Discovery**: UDP multicast SSDP on `239.255.255.250:1982`
- **Control**: TCP JSON-RPC on port `55443`
- **Rate Limit**: Maximum 60 requests per second per device
- **Color Encoding**: RGB values (0-255) encoded to base64 for transmission

For detailed protocol documentation, see [protocol guide](docs/yeelight-protocol-guide.md).

## Troubleshooting

### Devices Not Discovered

1. Ensure "LAN Control" is enabled in the Yeelight app
2. Check that devices are on the same network
3. Verify no firewall blocking UDP port 1982

### Docker Networking Issues

When running in Docker, device discovery requires host network access:

```bash
docker run --network host cubik:latest
```

## Development Notes

- Frontend uses Svelte 5 with runes for reactivity
- Backend uses ogen for OpenAPI code generation
- Database migrations use golang-migrate
- Matrix layout is row-major: `index = y × 20 + x`
- Animation frame rate is limited to 1 FPS due to device constraints

## License

This project is for personal use and experimentation with Yeelight devices.

## Contributing

This is a personal project. Feel free to fork and modify for your own use.

## References

- [Yeelight Developer Documentation](https://www.yeelight.com/download/Yeelight_Inter-Operation_Spec.pdf)
- [SSDP Protocol](https://en.wikipedia.org/wiki/Simple_Service_Discovery_Protocol)
- [OpenAPI 3.1 Specification](https://spec.openapis.org/oas/v3.1.0)
- [ogen-go Code Generator](https://github.com/ogen-go/ogen)
