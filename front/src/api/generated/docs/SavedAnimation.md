# SavedAnimation

## Properties

| Name        | Type                               |
| ----------- | ---------------------------------- |
| `id`        | string                             |
| `deviceId`  | string                             |
| `name`      | string                             |
| `frames`    | Array&lt;Array&lt;RGBPixel&gt;&gt; |
| `createdAt` | Date                               |
| `updatedAt` | Date                               |

## Example

```typescript
import type { SavedAnimation } from ''

// TODO: Update the object below with actual values
const example = {
  "id": 550e8400-e29b-41d4-a716-446655440000,
  "deviceId": yeelight://192.168.1.100:55443,
  "name": Rainbow Wave,
  "frames": null,
  "createdAt": 2026-01-05T14:30:00Z,
  "updatedAt": 2026-01-05T15:45:00Z,
} satisfies SavedAnimation

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as SavedAnimation
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)
