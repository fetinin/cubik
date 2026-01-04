# Frontend API Client Generation

This project uses `openapitools/openapi-generator-cli` to automatically generate a TypeScript client from the OpenAPI specification.

## Quick Start

```bash
# Generate the API client (run from front/ directory)
cd front
bun run generate-api

# Or using npm
npm run generate-api
```

## What Gets Generated

The command generates TypeScript code in `front/src/api/generated/`:

- **Models** (`models/`): TypeScript interfaces for all schemas
  - `Device.ts` - Device information
  - `RGBPixel.ts` - RGB color pixel
  - `StartAnimationRequest.ts` - Animation start payload
  - `StartAnimationResponse.ts` - Animation start response
  - `StopAnimationRequest.ts` - Animation stop payload
  - `StopAnimationResponse.ts` - Animation stop response
  - `ModelError.ts` - Error response

- **APIs** (`apis/`): API client classes
  - `DefaultApi.ts` - All API endpoints

- **Runtime** (`runtime.ts`): Fetch-based HTTP client utilities

- **Index** (`index.ts`): Re-exports everything

## Usage Example

```typescript
import { DefaultApi, Configuration } from '$lib/api/generated';

// Create API client instance
const api = new DefaultApi(new Configuration({
  basePath: 'http://localhost:8080'
}));

// Get all devices
const response = await api.getDevices();
console.log(response.devices); // Device[]

// Start animation
await api.startAnimation({
  startAnimationRequest: {
    device_location: 'yeelight://192.168.1.100:55443',
    frames: [
      [
        { r: 255, g: 0, b: 0 },    // Red pixel
        { r: 0, g: 255, b: 0 },    // Green pixel
        { r: 0, g: 0, b: 255 }     // Blue pixel
      ]
    ]
  }
});

// Stop animation
await api.stopAnimation({
  stopAnimationRequest: {
    device_location: 'yeelight://192.168.1.100:55443'
  }
});
```

## When to Regenerate

Run `bun run generate-api` whenever:

1. The OpenAPI spec (`spec.yml`) changes
2. New endpoints are added
3. Request/response schemas are modified
4. After pulling changes that update the spec

## Configuration

The generation command is configured in `front/package.json`:

```json
{
  "scripts": {
    "generate-api": "docker run --rm -v \"${PWD}/..:/workspace\" openapitools/openapi-generator-cli generate -i /workspace/spec.yml -g typescript-fetch -o /workspace/front/src/api/generated --additional-properties=supportsES6=true,npmVersion=10.0.0,typescriptThreePlus=true"
  }
}
```

### Generator Options

- **Generator**: `typescript-fetch` - Uses native Fetch API
- **supportsES6**: `true` - Enables ES6+ features
- **npmVersion**: `10.0.0` - Target npm version
- **typescriptThreePlus**: `true` - Uses TypeScript 3+ features

## Git Ignore

The generated code is automatically ignored by git (see `front/.gitignore`):

```gitignore
# Generated API code
/src/api/generated
```

This is intentional - the generated code is reproducible from `spec.yml` and shouldn't be committed.

## Docker Requirement

The generation requires Docker to be running. The command pulls and runs:

```
openapitools/openapi-generator-cli
```

Make sure Docker is installed and running before executing `bun run generate-api`.

## Replacing Mock API

The current mock API (`front/src/lib/api/mock.ts`) can be replaced with the generated client:

**Before (mock):**
```typescript
import { getDevices } from '$lib/api/mock';
const devices = await getDevices();
```

**After (real API):**
```typescript
import { DefaultApi, Configuration } from '$lib/api/generated';

const api = new DefaultApi(new Configuration({
  basePath: 'http://localhost:8080'
}));

const response = await api.getDevices();
const devices = response.devices;
```

## Troubleshooting

### Docker not found
Ensure Docker Desktop is running.

### Permission denied
On Linux/Mac, you may need to run with `sudo` or add your user to the docker group.

### Generated code has TypeScript errors
1. Make sure `spec.yml` is valid OpenAPI 3.1
2. Regenerate: `bun run generate-api`
3. Check that all required fields are defined in the spec

### Import errors
Make sure the import path uses `$lib/api/generated` (SvelteKit alias):
```typescript
import { DefaultApi } from '$lib/api/generated';
```
