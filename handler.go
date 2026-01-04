package main

import (
	"context"
	"cubik/api"
	"log"
)

// APIHandler implements the ogen-generated Handler interface
type APIHandler struct{}

// Ensure APIHandler implements the interface at compile time
var _ api.Handler = (*APIHandler)(nil)

// GetDevices implements GET /api/devices
// Performs live discovery and transforms DeviceInfo to API response model
func (h *APIHandler) GetDevices(ctx context.Context) (api.GetDevicesRes, error) {
	// Call existing discovery function
	devices, err := DiscoverDevices()
	if err != nil {
		// Log error for debugging
		log.Printf("Discovery error: %v", err)

		// Return 500 error response
		return &api.Error{
			Error: err.Error(),
		}, nil
	}

	// Transform []*DeviceInfo to []api.Device
	apiDevices := make([]api.Device, 0, len(devices))
	for _, device := range devices {
		apiDevices = append(apiDevices, api.Device{
			ID:       device.ID,
			Name:     device.Name,
			Location: device.Location,
		})
	}

	// Return 200 success response
	return &api.GetDevicesOK{
		Devices: apiDevices,
	}, nil
}
