# ImportAnimationRequest

## Properties

| Name        | Type                                  |
| ----------- | ------------------------------------- |
| `deviceId`  | string                                |
| `animation` | [SparseAnimation](SparseAnimation.md) |

## Example

```typescript
import type { ImportAnimationRequest } from '';

// TODO: Update the object below with actual values
const example = {
	deviceId: 0x000000000abc1234,
	animation: null
} satisfies ImportAnimationRequest;

console.log(example);

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example);
console.log(exampleJSON);

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as ImportAnimationRequest;
console.log(exampleParsed);
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)
