# PlayAnimationResponse

## Properties

| Name       | Type                    |
| ---------- | ----------------------- |
| `message`  | string                  |
| `playback` | [Playback](Playback.md) |

## Example

```typescript
import type { PlayAnimationResponse } from ''

// TODO: Update the object below with actual values
const example = {
  "message": Animation started successfully,
  "playback": null,
} satisfies PlayAnimationResponse

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as PlayAnimationResponse
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)
