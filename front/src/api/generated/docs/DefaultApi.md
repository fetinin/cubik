# DefaultApi

All URIs are relative to *http://localhost:8080*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**getDevices**](DefaultApi.md#getdevices) | **GET** /api/devices | Discover Yeelight CubeLite devices |
| [**startAnimation**](DefaultApi.md#startanimationoperation) | **POST** /animation/start | Start animation playback on device |
| [**stopAnimation**](DefaultApi.md#stopanimationoperation) | **POST** /animation/stop | Stop animation playback on device |



## getDevices

> GetDevices200Response getDevices()

Discover Yeelight CubeLite devices

Performs live SSDP discovery and returns currently available devices on the local network

### Example

```ts
import {
  Configuration,
  DefaultApi,
} from '';
import type { GetDevicesRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
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
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | List of discovered devices |  -  |
| **500** | Internal server error |  -  |

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


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **startAnimationRequest** | [StartAnimationRequest](StartAnimationRequest.md) |  | |

### Return type

[**StartAnimationResponse**](StartAnimationResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Animation started successfully |  -  |
| **400** | Bad request - invalid frame data or device location |  -  |
| **500** | Internal server error |  -  |

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


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **stopAnimationRequest** | [StopAnimationRequest](StopAnimationRequest.md) |  | |

### Return type

[**StopAnimationResponse**](StopAnimationResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Animation stopped successfully |  -  |
| **400** | Bad request - invalid device location format |  -  |
| **500** | Internal server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

