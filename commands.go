package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

// CommandRequest represents a JSON command to send to the device
type CommandRequest struct {
	ID     int           `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

// CommandResponse represents the JSON response from the device
type CommandResponse struct {
	ID     int           `json:"id"`
	Result []interface{} `json:"result,omitempty"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// parseLocation extracts IP and port from the Location header
// Example: "yeelight://192.168.1.239:55443" -> "192.168.1.239:55443"
func parseLocation(location string) (string, error) {
	// Remove "yeelight://" prefix
	addr := strings.TrimPrefix(location, "yeelight://")
	if addr == location {
		return "", fmt.Errorf("invalid location format: %s", location)
	}
	return addr, nil
}

// SendCommand sends a command to the device and returns the response
func SendCommand(device *DeviceInfo, method string, params []interface{}) (*CommandResponse, error) {
	// Parse the location to get IP:port
	addr, err := parseLocation(device.Location)
	if err != nil {
		return nil, err
	}

	// Establish TCP connection
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w (note: device may need 'LAN Control' enabled in Yeelight app)", addr, err)
	}
	defer conn.Close()

	// Set read/write timeouts
	conn.SetDeadline(time.Now().Add(3 * time.Second))

	// Create command
	cmd := CommandRequest{
		ID:     1,
		Method: method,
		Params: params,
	}

	// Encode and send command
	cmdJSON, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to encode command: %w", err)
	}

	// Send command with \r\n terminator as per protocol
	_, err = conn.Write(append(cmdJSON, []byte("\r\n")...))
	if err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	responseBytes, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var response CommandResponse
	err = json.Unmarshal(responseBytes, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for error in response
	if response.Error != nil {
		return &response, fmt.Errorf("device error: [%d] %s", response.Error.Code, response.Error.Message)
	}

	return &response, nil
}

// GetProp retrieves the specified properties from the device
func GetProp(device *DeviceInfo, properties ...string) (map[string]string, error) {
	// Convert properties to []interface{} for the command
	params := make([]interface{}, len(properties))
	for i, prop := range properties {
		params[i] = prop
	}

	// Send command
	response, err := SendCommand(device, "get_prop", params)
	if err != nil {
		return nil, err
	}

	// Map property names to values
	result := make(map[string]string)
	for i, prop := range properties {
		if i < len(response.Result) {
			// Convert result to string
			if val, ok := response.Result[i].(string); ok {
				result[prop] = val
			} else {
				result[prop] = fmt.Sprintf("%v", response.Result[i])
			}
		} else {
			result[prop] = ""
		}
	}

	return result, nil
}

// TogglePower toggles the power state of the device (on to off, or off to on)
// According to the Yeelight protocol, the toggle command takes no parameters
func TogglePower(device *DeviceInfo) error {
	// Send toggle command with empty params array
	response, err := SendCommand(device, "toggle", []interface{}{})
	if err != nil {
		return fmt.Errorf("failed to toggle power: %w", err)
	}

	// Check if response indicates success
	if len(response.Result) > 0 {
		if result, ok := response.Result[0].(string); ok && result == "ok" {
			return nil
		}
	}

	return fmt.Errorf("unexpected response from device: %+v", response.Result)
}

// encodeRGBColor converts a single RGB color to base64-encoded string.
// Each color component (r, g, b) should be 0-255.
// Returns a 4-character base64 string representing the 3-byte RGB value.
func encodeRGBColor(r, g, b uint8) string {
	// Create byte array with RGB values
	rgb := []byte{r, g, b}

	// Base64 encode and return
	return base64.StdEncoding.EncodeToString(rgb)
}

// ActivateFxMode activates a special effect mode on the device.
// For Matrix devices, "direct" mode must be activated before updating LEDs manually.
// This function must be called once before any UpdateLeds() calls.
func ActivateFxMode(device *DeviceInfo) error {
	// Create params with map for JSON object parameter
	params := []interface{}{
		map[string]string{"mode": "direct"},
	}

	// Send command
	response, err := SendCommand(device, "activate_fx_mode", params)
	if err != nil {
		return fmt.Errorf("failed to activate fx mode: %w", err)
	}

	// Check if response indicates success
	if len(response.Result) > 0 {
		if result, ok := response.Result[0].(string); ok && result == "ok" {
			return nil
		}
	}

	return fmt.Errorf("unexpected response from device: %+v", response.Result)
}

// SetBrightness sets the device brightness level.
// brightness must be between 1 and 100 (percentage).
// The change happens immediately with no transition effect.
func SetBrightness(device *DeviceInfo, brightness int) error {
	// Validate brightness range
	if brightness < 1 || brightness > 100 {
		return fmt.Errorf("brightness must be between 1 and 100, got %d", brightness)
	}

	// Send command with "sudden" effect for immediate change
	response, err := SendCommand(device, "set_bright", []interface{}{brightness, "sudden", 0})
	if err != nil {
		return fmt.Errorf("failed to set brightness: %w", err)
	}

	// Check if response indicates success
	if len(response.Result) > 0 {
		if result, ok := response.Result[0].(string); ok && result == "ok" {
			return nil
		}
	}

	return fmt.Errorf("unexpected response from device: %+v", response.Result)
}

// SendCommandNoResponse sends a command to the device without waiting for a response.
// This is useful for Matrix devices in direct mode, which don't send responses for update_leds.
// The command is sent fire-and-forget style for maximum performance.
func SendCommandNoResponse(device *DeviceInfo, method string, params []interface{}) error {
	// Parse the location to get IP:port
	addr, err := parseLocation(device.Location)
	if err != nil {
		return err
	}

	// Establish TCP connection
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w (note: device may need 'LAN Control' enabled in Yeelight app)", addr, err)
	}
	defer conn.Close()

	// Set write timeout
	conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

	// Create command
	cmd := CommandRequest{
		ID:     1,
		Method: method,
		Params: params,
	}

	// Encode and send command
	cmdJSON, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to encode command: %w", err)
	}

	// Send command with \r\n terminator as per protocol
	_, err = conn.Write(append(cmdJSON, []byte("\r\n")...))
	if err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	// Don't wait for response - fire and forget
	return nil
}

// UpdateLeds sends RGB data to update all LEDs on the Matrix device.
// The rgbData parameter must be base64-encoded RGB bytes for all LEDs.
// Each LED requires 3 bytes (R, G, B) which encodes to 4 base64 characters.
// For a 20x5 matrix (100 LEDs): 100 * 4 = 400 characters total.
// ActivateFxMode must be called before using this function.
// Note: This command doesn't wait for a response from the device for performance.
func UpdateLeds(device *DeviceInfo, rgbData string) error {
	// Send command without waiting for response (Matrix devices don't respond to update_leds)
	err := SendCommandNoResponse(device, "update_leds", []interface{}{rgbData})
	if err != nil {
		return fmt.Errorf("failed to update LEDs: %w", err)
	}

	return nil
}
