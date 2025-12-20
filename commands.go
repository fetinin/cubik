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

// Matrix LED Control Functions

// encodeRGBColor converts a single RGB color to base64-encoded string.
// Each color component (r, g, b) should be 0-255.
// Returns a 4-character base64 string representing the 3-byte RGB value.
func encodeRGBColor(r, g, b uint8) string {
	// Create byte array with RGB values
	rgb := []byte{r, g, b}

	// Base64 encode and return
	return base64.StdEncoding.EncodeToString(rgb)
}

// createSolidColor generates base64-encoded RGB data for a solid color pattern.
// All LEDs will be set to the same color specified by (r, g, b).
// numLEDs specifies the total number of LEDs (100 for a 20x5 matrix).
// Returns a string of length numLEDs * 4 characters.
func createSolidColor(r, g, b uint8, numLEDs int) string {
	// Encode single pixel
	encoded := encodeRGBColor(r, g, b)

	// Use strings.Builder for efficient concatenation
	var builder strings.Builder
	builder.Grow(numLEDs * 4) // Pre-allocate capacity

	// Repeat for all LEDs
	for i := 0; i < numLEDs; i++ {
		builder.WriteString(encoded)
	}

	return builder.String()
}

// createCheckerboard creates a checkerboard pattern alternating red and blue.
// The pattern alternates based on LED position in a row-major order.
// For a 20x5 matrix (100 LEDs), this creates an alternating pattern across all LEDs.
// Returns base64-encoded RGB data for numLEDs.
func createCheckerboard(numLEDs int) string {
	// Pre-encode colors
	red := encodeRGBColor(255, 0, 0)
	blue := encodeRGBColor(0, 0, 255)

	var builder strings.Builder
	builder.Grow(numLEDs * 4)

	for i := 0; i < numLEDs; i++ {
		if i%2 == 0 {
			builder.WriteString(red)
		} else {
			builder.WriteString(blue)
		}
	}

	return builder.String()
}

// createGradient creates a rainbow gradient across all LEDs.
// The gradient transitions through red → yellow → green → cyan → blue → magenta → red.
// Each LED gets a color based on its position, creating a smooth rainbow effect.
// For a 20x5 matrix, the gradient flows across all 100 LEDs in row-major order.
func createGradient(numLEDs int) string {
	var builder strings.Builder
	builder.Grow(numLEDs * 4)

	for i := 0; i < numLEDs; i++ {
		// Calculate position in gradient (0.0 to 1.0)
		pos := float64(i) / float64(numLEDs)

		var r, g, b uint8

		// Rainbow gradient using piecewise linear interpolation
		if pos < 0.167 { // Red to Yellow
			r = 255
			g = uint8(pos / 0.167 * 255)
			b = 0
		} else if pos < 0.333 { // Yellow to Green
			r = uint8((0.333 - pos) / 0.166 * 255)
			g = 255
			b = 0
		} else if pos < 0.5 { // Green to Cyan
			r = 0
			g = 255
			b = uint8((pos - 0.333) / 0.167 * 255)
		} else if pos < 0.667 { // Cyan to Blue
			r = 0
			g = uint8((0.667 - pos) / 0.167 * 255)
			b = 255
		} else if pos < 0.833 { // Blue to Magenta
			r = uint8((pos - 0.667) / 0.166 * 255)
			g = 0
			b = 255
		} else { // Magenta to Red
			r = 255
			g = 0
			b = uint8((1.0 - pos) / 0.167 * 255)
		}

		builder.WriteString(encodeRGBColor(r, g, b))
	}

	return builder.String()
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

// Matrix Display - Digit Rendering System

// Color represents an RGB color value
type Color struct {
	R, G, B uint8
}

// Framebuffer represents the LED matrix state before encoding
type Framebuffer struct {
	Width  int     // 20 for CubeLite
	Height int     // 5 for CubeLite
	Pixels []Color // Length = Width * Height (100)
}

// Alignment specifies horizontal positioning for multi-digit numbers
type Alignment int

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
)

// DigitBitmap represents a 5x5 pixel font for a single digit
// [row][column] where row=0 is top, column=0 is left
type DigitBitmap [5][5]bool

// NewFramebuffer creates a new framebuffer with all pixels set to black (off)
func NewFramebuffer(width, height int) *Framebuffer {
	fb := &Framebuffer{
		Width:  width,
		Height: height,
		Pixels: make([]Color, width*height),
	}
	// Pixels are already initialized to zero values (black)
	return fb
}

// Clear sets all pixels to the specified color
func (fb *Framebuffer) Clear(color Color) {
	for i := range fb.Pixels {
		fb.Pixels[i] = color
	}
}

// SetPixel sets a single pixel at (x, y) to the specified color
// Returns error if coordinates are out of bounds
func (fb *Framebuffer) SetPixel(x, y int, color Color) error {
	if x < 0 || x >= fb.Width || y < 0 || y >= fb.Height {
		return fmt.Errorf("pixel coordinates (%d, %d) out of bounds (width=%d, height=%d)", x, y, fb.Width, fb.Height)
	}
	index := y*fb.Width + x
	fb.Pixels[index] = color
	return nil
}

// GetPixel returns the color at (x, y)
func (fb *Framebuffer) GetPixel(x, y int) (Color, error) {
	if x < 0 || x >= fb.Width || y < 0 || y >= fb.Height {
		return Color{}, fmt.Errorf("pixel coordinates (%d, %d) out of bounds (width=%d, height=%d)", x, y, fb.Width, fb.Height)
	}
	index := y*fb.Width + x
	return fb.Pixels[index], nil
}

// Encode converts the framebuffer to base64-encoded RGB string for UpdateLeds
// Returns 400-character string for 20x5 matrix (100 LEDs * 4 chars each)
// Note: X-axis is reversed because hardware addresses LEDs right-to-left
func (fb *Framebuffer) Encode() string {
	var builder strings.Builder
	builder.Grow(len(fb.Pixels) * 4) // Pre-allocate: each LED = 4 base64 chars

	// Iterate through pixels in row-major order, but reverse X within each row
	// This compensates for the hardware addressing LEDs from right-to-left
	for y := 0; y < fb.Height; y++ {
		for x := fb.Width - 1; x >= 0; x-- {
			index := y*fb.Width + x
			pixel := fb.Pixels[index]
			encoded := encodeRGBColor(pixel.R, pixel.G, pixel.B)
			builder.WriteString(encoded)
		}
	}

	return builder.String()
}

// digitFont maps digit characters '0'-'9' to their 5x5 bitmap representations
// Each bitmap is organized as [row][column] where row=0 is top, column=0 is left
// true = pixel on (digit color), false = pixel off (background color)
var digitFont = map[rune]DigitBitmap{
	'0': {
		{true, true, true, true, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
	},
	'1': {
		{false, false, true, false, false},
		{false, true, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, true, true, true, false},
	},
	'2': {
		{true, true, true, true, true},
		{false, false, false, false, true},
		{true, true, true, true, true},
		{true, false, false, false, false},
		{true, true, true, true, true},
	},
	'3': {
		{true, true, true, true, true},
		{false, false, false, false, true},
		{true, true, true, true, true},
		{false, false, false, false, true},
		{true, true, true, true, true},
	},
	'4': {
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
		{false, false, false, false, true},
		{false, false, false, false, true},
	},
	'5': {
		{true, true, true, true, true},
		{true, false, false, false, false},
		{true, true, true, true, true},
		{false, false, false, false, true},
		{true, true, true, true, true},
	},
	'6': {
		{true, true, true, true, true},
		{true, false, false, false, false},
		{true, true, true, true, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
	},
	'7': {
		{true, true, true, true, true},
		{false, false, false, false, true},
		{false, false, false, true, false},
		{false, false, true, false, false},
		{false, true, false, false, false},
	},
	'8': {
		{true, true, true, true, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
	},
	'9': {
		{true, true, true, true, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
		{false, false, false, false, true},
		{true, true, true, true, true},
	},
}

// DrawDigit renders a single digit at the specified position
// Parameters:
//   - fb: target framebuffer
//   - digit: character '0'-'9'
//   - x, y: top-left corner position in framebuffer
//   - color: color for the digit pixels (lit pixels)
//   - background: color for non-digit pixels (typically black/off)
//
// Returns error if digit is invalid or position is out of bounds
func DrawDigit(fb *Framebuffer, digit rune, x, y int, color, background Color) error {
	// Look up digit bitmap
	bitmap, exists := digitFont[digit]
	if !exists {
		return fmt.Errorf("invalid digit character: '%c' (only '0'-'9' supported)", digit)
	}

	// Check bounds (digit is 5x5)
	if x+5 > fb.Width || y+5 > fb.Height {
		return fmt.Errorf("digit at position (%d, %d) would exceed framebuffer bounds (width=%d, height=%d)", x, y, fb.Width, fb.Height)
	}
	if x < 0 || y < 0 {
		return fmt.Errorf("digit position (%d, %d) cannot be negative", x, y)
	}

	// Iterate through 5x5 bitmap and set pixels
	for row := 0; row < 5; row++ {
		for col := 0; col < 5; col++ {
			pixelColor := background
			if bitmap[row][col] {
				pixelColor = color
			}
			fb.SetPixel(x+col, y+row, pixelColor)
		}
	}

	return nil
}

// DrawNumber renders a multi-digit number with alignment
// Parameters:
//   - fb: target framebuffer
//   - number: integer to display (e.g., 42, 123)
//   - y: vertical position (typically 0 for centered on 5-row display)
//   - spacing: pixels between digits (0 or 1 recommended)
//   - alignment: how to position the number horizontally (AlignLeft, AlignCenter, AlignRight)
//   - color: color for digit pixels
//   - background: color for background (black = off)
//
// Returns error if number doesn't fit or invalid parameters
func DrawNumber(fb *Framebuffer, number int, y, spacing int, alignment Alignment, color, background Color) error {
	// Convert number to string
	str := fmt.Sprintf("%d", number)
	return DrawString(fb, str, y, spacing, alignment, color, background)
}

// DrawString renders a string of digits with alignment
// Similar to DrawNumber but accepts string input for flexibility
// Parameters:
//   - fb: target framebuffer
//   - str: string of digits to display (e.g., "42", "123")
//   - y: vertical position (typically 0 for 5-row display)
//   - spacing: pixels between digits (0 or 1 recommended)
//   - alignment: how to position horizontally (AlignLeft, AlignCenter, AlignRight)
//   - color: color for digit pixels
//   - background: color for background (black = off)
//
// Returns error if string contains non-digit characters, doesn't fit, or invalid parameters
func DrawString(fb *Framebuffer, str string, y, spacing int, alignment Alignment, color, background Color) error {
	if len(str) == 0 {
		return fmt.Errorf("cannot draw empty string")
	}

	// Calculate total width needed
	// Each digit = 5 pixels, spacing between = spacing pixels
	// Total = (numDigits * 5) + ((numDigits - 1) * spacing)
	totalWidth := len(str)*5 + (len(str)-1)*spacing

	// Check if number fits in framebuffer
	if totalWidth > fb.Width {
		return fmt.Errorf("string '%s' too wide: needs %d pixels, framebuffer width is %d", str, totalWidth, fb.Width)
	}

	// Calculate starting X position based on alignment
	var startX int
	switch alignment {
	case AlignLeft:
		startX = 0
	case AlignCenter:
		startX = (fb.Width - totalWidth) / 2
	case AlignRight:
		startX = fb.Width - totalWidth
	default:
		return fmt.Errorf("invalid alignment value: %d", alignment)
	}

	// Draw each digit sequentially
	currentX := startX
	for _, digit := range str {
		err := DrawDigit(fb, digit, currentX, y, color, background)
		if err != nil {
			return fmt.Errorf("failed to draw digit '%c': %w", digit, err)
		}
		currentX += 5 + spacing // Move to next digit position
	}

	return nil
}

