# StartAnimationResponse

## Properties

| Name         | Type   |
| ------------ | ------ |
| `message`    | string |
| `frameCount` | number |

## Example

```typescript
import type { StartAnimationResponse } from ''

// TODO: Update the object below with actual values
const example = {
  "message": Animation started successfully,
  "frameCount": 30,
} satisfies StartAnimationResponse

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as StartAnimationResponse
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)
