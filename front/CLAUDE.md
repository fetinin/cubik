# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a SvelteKit single-page application for drawing and animating Yeelight CubeLite (Matrix) devices. The frontend provides the user interface for creating visual content and forming payloads that describe how images should be displayed on the cube's LED matrix. All actual device manipulation is performed by the backend - the frontend is responsible only for displaying information to the user and forming the display commands.

This application is designed to run locally without authorization.

## Build and Development Commands

**This project uses Bun as the package manager and runtime.**

```bash
# Install dependencies
bun install

# Generate TypeScript API client from OpenAPI spec
bun run generate-api

# Start development server
bun run dev

# Start development server and open in browser
bun run dev -- --open

# Build for production
bun run build

# Preview production build
bun run preview

# Type checking
bun run check

# Type checking in watch mode
bun run check:watch

# Format code
bun run format

# Lint code
bun run lint

# Run all tests
bun test

# Run tests in watch mode
bun run test:unit
```

## Code Quality and Verification

**CRITICAL: After making any code changes, you MUST:**

1. **Run formatters and linters** to ensure code quality:

   ```bash
   bun run format  # Format code
   bun run lint    # Lint code
   ```

2. **Verify frontend changes in the browser**:
   - Start the dev server: `bun run dev -- --open`
   - Manually test the changed functionality in the browser
   - Ensure no console errors or visual regressions
   - Verify the user-facing behavior matches expectations

These steps are mandatory and should never be skipped, even for small changes.

## Testing

This project uses Vitest with two test configurations:

- **Client tests**: Run in Playwright browser using Chromium (files matching `src/**/*.svelte.{test,spec}.{js,ts}`)
- **Server tests**: Run in Node environment (files matching `src/**/*.{test,spec}.{js,ts}` excluding `.svelte.{test,spec}`)

Tests require assertions (`expect.requireAssertions: true`).

## API Client Generation

The project uses automatic TypeScript client generation from the OpenAPI specification:

**Generate the client:**

```bash
bun run generate-api
```

This runs `openapitools/openapi-generator-cli` via Docker to generate:

- TypeScript types and models in `src/api/generated/models/`
- API client in `src/api/generated/apis/DefaultApi.ts`
- Runtime utilities in `src/api/generated/runtime.ts`

**IMPORTANT:** Run this command whenever the backend's `spec.yml` changes to keep types in sync.

**Using the generated client:**

```typescript
import { DefaultApi, Configuration } from '$lib/api/generated';

// Create API client
const api = new DefaultApi(
	new Configuration({
		basePath: 'http://localhost:9080'
	})
);

// Get devices
const response = await api.getDevices();
console.log(response.devices); // Device[]

// Start animation
await api.startAnimation({
	startAnimationRequest: {
		device_location: 'yeelight://192.168.1.100:55443',
		frames: [
			[
				{ r: 255, g: 0, b: 0 },
				{ r: 0, g: 255, b: 0 }
			]
		]
	}
});
```

The generated code lives in `src/api/generated/` and is gitignored. The mock API in `src/lib/api/client.ts` can be replaced with calls to the generated client.

## Code Architecture

### Tech Stack

- **Framework**: SvelteKit with Svelte 5
- **Styling**: Tailwind CSS v4
- **Testing**: Vitest with Playwright browser provider
- **Language**: TypeScript
- **Runtime**: Bun
- **API Client**: Auto-generated from OpenAPI spec via openapi-generator-cli

### Core Architecture

The application follows a single-page app pattern with centralized state management:

1. **State Management (`src/lib/state/editor.ts`)**
   - `EditorState`: Centralized state using Svelte stores (writable/derived)
   - Manages devices, matrix size, pixel buffer, frames, and apply status
   - All state mutations happen through pure functions that return new arrays/objects
   - `PackedRGB`: Colors stored as `0xRRGGBB` integers for efficiency
   - Frame management with unique IDs generated via timestamp + random

2. **API Layer (`src/lib/api/client.ts`)**
   - Currently mocked - will be replaced with real HTTP calls to Go backend
   - `getDevices()`: Fetch available Yeelight devices
   - `getMatrixSize()`: Get matrix dimensions (currently hardcoded to 20×5)
   - `applyAnimation()`: Send animation payload to backend for device control
   - `AnimationPayload` format: Fixed 1 FPS, frames as arrays of packed RGB integers in row-major order

3. **Main Editor (`src/routes/+page.svelte`)**
   - Single-page application orchestrating all components
   - Uses Svelte 5 runes (`$state`, `$derived`) implicitly through stores
   - Device selection triggers matrix size fetch and pixel buffer initialization
   - Frame workflow: paint pixels → save as frame → build animation → apply to device

4. **Components (`src/lib/components/`)**
   - `DeviceBar`: Device selection dropdown
   - `MatrixGrid`: Interactive LED matrix for pixel painting
   - `ColorPickerRGB`: RGB color selector for paint tool
   - `FramesPanel`: Frame list with save/load/delete/reorder operations
   - `AnimationPreview`: Plays animation sequence at 1 FPS (device limitation)

### Data Flow

1. **Device Selection**: User selects device → fetch matrix size → initialize empty pixel buffer
2. **Drawing**: User paints pixels on matrix grid → updates pixel buffer in editor state
3. **Frame Creation**: User saves current pixel buffer as named frame → added to frames list
4. **Animation Building**: User creates multiple frames → builds `AnimationPayload` with frame sequence
5. **Device Application**: User clicks "Apply animation" → sends payload to backend via API

### Important Details

- **Color Encoding**: RGB values (0-255 each) packed into single integer `0xRRGGBB` for memory efficiency
- **Pixel Layout**: Row-major order (index = y × width + x)
- **Frame Rate**: Fixed at 1 FPS due to Yeelight device limitation (60 RPS max)
- **State Immutability**: All pixel/frame updates create new arrays to trigger Svelte reactivity
- **Backend Integration**: API layer is currently mocked; replace `src/lib/api/client.ts` functions with real HTTP calls to Go backend

## Related Projects

The Go backend for this project is located at `/Users/inv-denisf/dev/personal/Cubik/` which handles:

- SSDP device discovery on local network
- TCP command protocol for Yeelight devices
- Matrix LED encoding and transmission
- See backend `CLAUDE.md` for protocol details
