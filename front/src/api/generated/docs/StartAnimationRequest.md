# StartAnimationRequest

## Properties

| Name             | Type                               |
| ---------------- | ---------------------------------- |
| `deviceLocation` | string                             |
| `frames`         | Array&lt;Array&lt;RGBPixel&gt;&gt; |

## Example

```typescript
import type { StartAnimationRequest } from ''

// TODO: Update the object below with actual values
const example = {
  "deviceLocation": yeelight://192.168.1.100:55443,
  "frames": null,
} satisfies StartAnimationRequest

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as StartAnimationRequest
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)
