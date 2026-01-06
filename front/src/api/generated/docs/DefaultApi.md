# DefaultApi

All URIs are relative to _http://localhost:9080_

| Method                                                        | HTTP request                            | Description                        |
| ------------------------------------------------------------- | --------------------------------------- | ---------------------------------- |
| [**deleteAnimation**](DefaultApi.md#deleteanimation)          | **DELETE** /api/animation/{id}          | Delete a saved animation           |
| [**getAnimation**](DefaultApi.md#getanimation)                | **GET** /api/animation/{id}             | Get a specific saved animation     |
| [**getDevices**](DefaultApi.md#getdevices)                    | **GET** /api/devices                    | Discover Yeelight CubeLite devices |
| [**listAnimations**](DefaultApi.md#listanimations)            | **GET** /api/animation/list/{device_id} | List saved animations for a device |
| [**saveAnimation**](DefaultApi.md#saveanimationoperation)     | **POST** /api/animation/save            | Save animation to database         |
| [**startAnimation**](DefaultApi.md#startanimationoperation)   | **POST** /api/animation/start           | Start animation playback on device |
| [**stopAnimation**](DefaultApi.md#stopanimationoperation)     | **POST** /api/animation/stop            | Stop animation playback on device  |
| [**updateAnimation**](DefaultApi.md#updateanimationoperation) | **PUT** /api/animation/{id}             | Update an existing saved animation |

## deleteAnimation

> DeleteAnimationResponse deleteAnimation(id)

Delete a saved animation

Permanently removes a saved animation from the database

### Example

```ts
import {
  Configuration,
  DefaultApi,
} from '';
import type { DeleteAnimationRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new DefaultApi();

  const body = {
    // string | Animation UUID
    id: 550e8400-e29b-41d4-a716-446655440000,
  } satisfies DeleteAnimationRequest;

  try {
    const data = await api.deleteAnimation(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters

| Name   | Type     | Description    | Notes                     |
| ------ | -------- | -------------- | ------------------------- |
| **id** | `string` | Animation UUID | [Defaults to `undefined`] |

### Return type

[**DeleteAnimationResponse**](DeleteAnimationResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`

### HTTP response details

| Status code | Description                    | Response headers |
| ----------- | ------------------------------ | ---------------- |
| **200**     | Animation deleted successfully | -                |
| **404**     | Animation not found            | -                |
| **500**     | Internal server error          | -                |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

## getAnimation

> GetAnimationResponse getAnimation(id)

Get a specific saved animation

Retrieves a saved animation by its ID

### Example

```ts
import {
  Configuration,
  DefaultApi,
} from '';
import type { GetAnimationRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new DefaultApi();

  const body = {
    // string | Animation UUID
    id: 550e8400-e29b-41d4-a716-446655440000,
  } satisfies GetAnimationRequest;

  try {
    const data = await api.getAnimation(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters

| Name   | Type     | Description    | Notes                     |
| ------ | -------- | -------------- | ------------------------- |
| **id** | `string` | Animation UUID | [Defaults to `undefined`] |

### Return type

[**GetAnimationResponse**](GetAnimationResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`

### HTTP response details

| Status code | Description                      | Response headers |
| ----------- | -------------------------------- | ---------------- |
| **200**     | Animation retrieved successfully | -                |
| **404**     | Animation not found              | -                |
| **500**     | Internal server error            | -                |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

## getDevices

> GetDevices200Response getDevices()

Discover Yeelight CubeLite devices

Performs live SSDP discovery and returns currently available devices on the local network

### Example

```ts
import { Configuration, DefaultApi } from '';
import type { GetDevicesRequest } from '';

async function example() {
	console.log('🚀 Testing  SDK...');
	const api = new DefaultApi();

	try {
		const data = await api.getDevices();
		console.log(data);
	} catch (error) {
		console.error(error);
	}
}

// Run the test
example().catch(console.error);
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**GetDevices200Response**](GetDevices200Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`

### HTTP response details

| Status code | Description                | Response headers |
| ----------- | -------------------------- | ---------------- |
| **200**     | List of discovered devices | -                |
| **500**     | Internal server error      | -                |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

## listAnimations

> ListAnimationsResponse listAnimations(deviceId)

List saved animations for a device

Returns all saved animations for the specified device, ordered by most recently updated

### Example

```ts
import { Configuration, DefaultApi } from '';
import type { ListAnimationsRequest } from '';

async function example() {
	console.log('🚀 Testing  SDK...');
	const api = new DefaultApi();

	const body = {
		// string | Unique device identifier
		deviceId: 0x000000000abc1234
	} satisfies ListAnimationsRequest;

	try {
		const data = await api.listAnimations(body);
		console.log(data);
	} catch (error) {
		console.error(error);
	}
}

// Run the test
example().catch(console.error);
```

### Parameters

| Name         | Type     | Description              | Notes                     |
| ------------ | -------- | ------------------------ | ------------------------- |
| **deviceId** | `string` | Unique device identifier | [Defaults to `undefined`] |

### Return type

[**ListAnimationsResponse**](ListAnimationsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`

### HTTP response details

| Status code | Description              | Response headers |
| ----------- | ------------------------ | ---------------- |
| **200**     | List of saved animations | -                |
| **500**     | Internal server error    | -                |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

## saveAnimation

> SaveAnimationResponse saveAnimation(saveAnimationRequest)

Save animation to database

Saves the current animation frames to the database with a name. Stored per device.

### Example

```ts
import {
  Configuration,
  DefaultApi,
} from '';
import type { SaveAnimationOperationRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new DefaultApi();

  const body = {
    // SaveAnimationRequest
    saveAnimationRequest: ...,
  } satisfies SaveAnimationOperationRequest;

  try {
    const data = await api.saveAnimation(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters

| Name                     | Type                                            | Description | Notes |
| ------------------------ | ----------------------------------------------- | ----------- | ----- |
| **saveAnimationRequest** | [SaveAnimationRequest](SaveAnimationRequest.md) |             |       |

### Return type

[**SaveAnimationResponse**](SaveAnimationResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`

### HTTP response details

| Status code | Description                      | Response headers |
| ----------- | -------------------------------- | ---------------- |
| **200**     | Animation saved successfully     | -                |
| **400**     | Bad request - invalid input data | -                |
| **500**     | Internal server error            | -                |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

## startAnimation

> StartAnimationResponse startAnimation(startAnimationRequest)

Start animation playback on device

Starts playing an animation loop on the specified device. Only one animation can run per device at a time.

### Example

```ts
import {
  Configuration,
  DefaultApi,
} from '';
import type { StartAnimationOperationRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new DefaultApi();

  const body = {
    // StartAnimationRequest
    startAnimationRequest: ...,
  } satisfies StartAnimationOperationRequest;

  try {
    const data = await api.startAnimation(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters

| Name                      | Type                                              | Description | Notes |
| ------------------------- | ------------------------------------------------- | ----------- | ----- |
| **startAnimationRequest** | [StartAnimationRequest](StartAnimationRequest.md) |             |       |

### Return type

[**StartAnimationResponse**](StartAnimationResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`

### HTTP response details

| Status code | Description                                         | Response headers |
| ----------- | --------------------------------------------------- | ---------------- |
| **200**     | Animation started successfully                      | -                |
| **400**     | Bad request - invalid frame data or device location | -                |
| **500**     | Internal server error                               | -                |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

## stopAnimation

> StopAnimationResponse stopAnimation(stopAnimationRequest)

Stop animation playback on device

Stops the currently running animation on the specified device. No-op if no animation is running.

### Example

```ts
import {
  Configuration,
  DefaultApi,
} from '';
import type { StopAnimationOperationRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new DefaultApi();

  const body = {
    // StopAnimationRequest
    stopAnimationRequest: ...,
  } satisfies StopAnimationOperationRequest;

  try {
    const data = await api.stopAnimation(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters

| Name                     | Type                                            | Description | Notes |
| ------------------------ | ----------------------------------------------- | ----------- | ----- |
| **stopAnimationRequest** | [StopAnimationRequest](StopAnimationRequest.md) |             |       |

### Return type

[**StopAnimationResponse**](StopAnimationResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`

### HTTP response details

| Status code | Description                                  | Response headers |
| ----------- | -------------------------------------------- | ---------------- |
| **200**     | Animation stopped successfully               | -                |
| **400**     | Bad request - invalid device location format | -                |
| **500**     | Internal server error                        | -                |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

## updateAnimation

> UpdateAnimationResponse updateAnimation(id, updateAnimationRequest)

Update an existing saved animation

Overwrites an existing animation\&#39;s name and frames

### Example

```ts
import {
  Configuration,
  DefaultApi,
} from '';
import type { UpdateAnimationOperationRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new DefaultApi();

  const body = {
    // string | Animation UUID
    id: 550e8400-e29b-41d4-a716-446655440000,
    // UpdateAnimationRequest
    updateAnimationRequest: ...,
  } satisfies UpdateAnimationOperationRequest;

  try {
    const data = await api.updateAnimation(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters

| Name                       | Type                                                | Description    | Notes                     |
| -------------------------- | --------------------------------------------------- | -------------- | ------------------------- |
| **id**                     | `string`                                            | Animation UUID | [Defaults to `undefined`] |
| **updateAnimationRequest** | [UpdateAnimationRequest](UpdateAnimationRequest.md) |                |                           |

### Return type

[**UpdateAnimationResponse**](UpdateAnimationResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`

### HTTP response details

| Status code | Description                      | Response headers |
| ----------- | -------------------------------- | ---------------- |
| **200**     | Animation updated successfully   | -                |
| **400**     | Bad request - invalid input data | -                |
| **404**     | Animation not found              | -                |
| **500**     | Internal server error            | -                |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)
