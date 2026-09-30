# Playback

## Properties

| Name            | Type                               |
| --------------- | ---------------------------------- |
| `animationId`   | string                             |
| `animationName` | string                             |
| `frames`        | Array&lt;Array&lt;RGBPixel&gt;&gt; |

## Example

```typescript
import type { Playback } from '';

// TODO: Update the object below with actual values
const example = {
	animationId: null,
	animationName: null,
	frames: null
} satisfies Playback;

console.log(example);

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example);
console.log(exampleJSON);

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as Playback;
console.log(exampleParsed);
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)
