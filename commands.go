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

type CommandRequest struct {
	ID     int           `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

type CommandResponse struct {
	ID     int           `json:"id"`
	Result []interface{} `json:"result,omitempty"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func parseLocation(location string) (string, error) {
	addr := strings.TrimPrefix(location, "yeelight://")
	if addr == location {
		return "", fmt.Errorf("invalid location format: %s", location)
	}
	return addr, nil
}

func SendCommand(device *DeviceInfo, method string, params []interface{}) (*CommandResponse, error) {
	addr, err := parseLocation(device.Location)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(3 * time.Second))

	cmd := CommandRequest{ID: 1, Method: method, Params: params}
	cmdJSON, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to encode command: %w", err)
	}

	if _, err = conn.Write(append(cmdJSON, '\r', '\n')); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	reader := bufio.NewReader(conn)
	responseBytes, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response CommandResponse
	if err = json.Unmarshal(responseBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if response.Error != nil {
		return &response, fmt.Errorf("device error: [%d] %s", response.Error.Code, response.Error.Message)
	}

	return &response, nil
}

func GetProp(device *DeviceInfo, properties ...string) (map[string]string, error) {
	params := make([]interface{}, len(properties))
	for i, prop := range properties {
		params[i] = prop
	}

	response, err := SendCommand(device, "get_prop", params)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for i, prop := range properties {
		if i < len(response.Result) {
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

func TogglePower(device *DeviceInfo) error {
	response, err := SendCommand(device, "toggle", []interface{}{})
	if err != nil {
		return fmt.Errorf("failed to toggle power: %w", err)
	}

	if len(response.Result) > 0 {
		if result, ok := response.Result[0].(string); ok && result == "ok" {
			return nil
		}
	}

	return fmt.Errorf("unexpected response from device: %+v", response.Result)
}

func encodeRGBColor(r, g, b uint8) string {
	return base64.StdEncoding.EncodeToString([]byte{r, g, b})
}

// ActivateFxMode activates direct mode for manual LED control on Matrix devices.
func ActivateFxMode(device *DeviceInfo) error {
	params := []interface{}{map[string]string{"mode": "direct"}}
	response, err := SendCommand(device, "activate_fx_mode", params)
	if err != nil {
		return fmt.Errorf("failed to activate fx mode: %w", err)
	}

	if len(response.Result) > 0 {
		if result, ok := response.Result[0].(string); ok && result == "ok" {
			return nil
		}
	}

	return fmt.Errorf("unexpected response from device: %+v", response.Result)
}

func SetBrightness(device *DeviceInfo, brightness int) error {
	if brightness < 1 || brightness > 100 {
		return fmt.Errorf("brightness must be between 1 and 100, got %d", brightness)
	}

	response, err := SendCommand(device, "set_bright", []interface{}{brightness, "sudden", 0})
	if err != nil {
		return fmt.Errorf("failed to set brightness: %w", err)
	}

	if len(response.Result) > 0 {
		if result, ok := response.Result[0].(string); ok && result == "ok" {
			return nil
		}
	}

	return fmt.Errorf("unexpected response from device: %+v", response.Result)
}

func SendCommandNoResponse(device *DeviceInfo, method string, params []interface{}) error {
	addr, err := parseLocation(device.Location)
	if err != nil {
		return err
	}

	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

	cmd := CommandRequest{ID: 1, Method: method, Params: params}
	cmdJSON, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to encode command: %w", err)
	}

	if _, err = conn.Write(append(cmdJSON, '\r', '\n')); err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	return nil
}

// UpdateLeds sends base64-encoded RGB data to update all LEDs on the Matrix device.
// ActivateFxMode must be called before using this function.
func UpdateLeds(device *DeviceInfo, rgbData string) error {
	if err := SendCommandNoResponse(device, "update_leds", []interface{}{rgbData}); err != nil {
		return fmt.Errorf("failed to update LEDs: %w", err)
	}
	return nil
}
