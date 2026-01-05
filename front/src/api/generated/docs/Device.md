# Device

## Properties

| Name       | Type   |
| ---------- | ------ |
| `id`       | string |
| `name`     | string |
| `location` | string |

## Example

```typescript
import type { Device } from ''

// TODO: Update the object below with actual values
const example = {
  "id": 0x000000000abc1234,
  "name": Living Room Cube,
  "location": yeelight://192.168.1.100:55443,
} satisfies Device

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as Device
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)
