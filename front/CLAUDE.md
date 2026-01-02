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

## Testing

This project uses Vitest with two test configurations:

- **Client tests**: Run in Playwright browser using Chromium (files matching `src/**/*.svelte.{test,spec}.{js,ts}`)
- **Server tests**: Run in Node environment (files matching `src/**/*.{test,spec}.{js,ts}` excluding `.svelte.{test,spec}`)

Tests require assertions (`expect.requireAssertions: true`).

## Code Architecture

### Tech Stack

- **Framework**: SvelteKit with Svelte 5
- **Styling**: Tailwind CSS v4
- **Testing**: Vitest with Playwright browser provider
- **Language**: TypeScript
- **Runtime**: Bun

### Core Architecture

The application follows a single-page app pattern with centralized state management:

1. **State Management (`src/lib/state/editor.ts`)**
   - `EditorState`: Centralized state using Svelte stores (writable/derived)
   - Manages devices, matrix size, pixel buffer, frames, and apply status
   - All state mutations happen through pure functions that return new arrays/objects
   - `PackedRGB`: Colors stored as `0xRRGGBB` integers for efficiency
   - Frame management with unique IDs generated via timestamp + random

2. **API Layer (`src/lib/api/mock.ts`)**
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
- **Backend Integration**: API layer is currently mocked; replace `src/lib/api/mock.ts` functions with real HTTP calls to Go backend

## Related Projects

The Go backend for this project is located at `/Users/inv-denisf/dev/personal/Cubik/` which handles:
- SSDP device discovery on local network
- TCP command protocol for Yeelight devices
- Matrix LED encoding and transmission
- See backend `CLAUDE.md` for protocol details
