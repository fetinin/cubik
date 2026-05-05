# SparseAnimation

## Properties

| Name      | Type                                  |
| --------- | ------------------------------------- |
| `version` | string                                |
| `name`    | string                                |
| `width`   | number                                |
| `height`  | number                                |
| `frames`  | Array&lt;Array&lt;SparsePixel&gt;&gt; |

## Example

```typescript
import type { SparseAnimation } from ''

// TODO: Update the object below with actual values
const example = {
  "version": 1.0,
  "name": Rainbow Wave,
  "width": 20,
  "height": 5,
  "frames": null,
} satisfies SparseAnimation

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as SparseAnimation
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)
