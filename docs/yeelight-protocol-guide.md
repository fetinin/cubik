# Yeelight Protocol and Matrix Device Guide

A comprehensive technical reference for the Yeelight protocol and controlling Yeelight Matrix/Canvas devices.

## Table of Contents

1. [Introduction](#introduction)
2. [Yeelight Protocol Fundamentals](#yeelight-protocol-fundamentals)
3. [Matrix Device Specific Implementation](#matrix-device-specific-implementation)
4. [Code Examples](#code-examples)
5. [Troubleshooting and Best Practices](#troubleshooting-and-best-practices)
6. [Reference Tables](#reference-tables)
7. [Appendix](#appendix)

---

## 1. Introduction

### Overview

The Yeelight protocol is a JSON-RPC based communication protocol used to control Yeelight smart lighting devices over a local network. This document provides a complete technical reference with a focus on the **Yeelight Matrix/Canvas** devices and how to set custom image patterns.

### Use Cases

- Control smart bulbs (color, brightness, temperature)
- Create custom lighting effects and animations
- Display images and patterns on Matrix/Canvas devices
- Build home automation integrations
- Develop interactive lighting applications

### Supported Devices

- Yeelight Color Bulb
- Yeelight White Bulb
- Yeelight LED Strip
- Yeelight Lightstrip Plus
- Yeelight Ceiling Light
- **Yeelight Matrix/Canvas** (primary focus)

---

## 2. Yeelight Protocol Fundamentals

### 2.1 Protocol Overview

The Yeelight protocol uses **JSON-RPC over TCP** for command-response communication.

**Key Characteristics:**
- **Protocol**: JSON-RPC (JavaScript Object Notation Remote Procedure Call)
- **Transport**: TCP (Transmission Control Protocol)
- **Default Port**: `55443`
- **Message Termination**: CRLF (`\r\n`)
- **Encoding**: UTF-8
- **Rate Limit**: 60 commands per minute (standard mode)

### 2.2 Connection Establishment

#### TCP Socket Connection

```python
import socket

# Create TCP socket
sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sock.settimeout(5)  # 5 second timeout

# Connect to device
sock.connect(("192.168.0.34", 55443))
```

#### Device Discovery via SSDP

Yeelight devices can be discovered on the local network using **SSDP (Simple Service Discovery Protocol)**.

**Discovery Process:**

1. Send multicast UDP packet to `239.255.255.250:1982`
2. Wait for device responses
3. Parse device capabilities from response headers

**Discovery Packet:**
```
M-SEARCH * HTTP/1.1
HOST: 239.255.255.250:1982
MAN: "ssdp:discover"
ST: wifi_bulb
```

**Example Discovery Response:**
```
HTTP/1.1 200 OK
Location: yeelight://192.168.0.19:55443
id: 0x0000000002dfb19a
model: color
fw_ver: 45
support: get_prop set_power toggle set_bright set_rgb set_hsv set_ct_abx start_cf stop_cf set_scene cron_add cron_get cron_del set_adjust set_music set_name
power: on
bright: 50
color_mode: 2
ct: 4000
rgb: 16711680
hue: 100
sat: 35
name: my_bulb
```

**Python Discovery Example:**
```python
import socket

def discover_bulbs(timeout=2):
    """Discover Yeelight devices on local network."""
    msg = "\r\n".join([
        "M-SEARCH * HTTP/1.1",
        "HOST: 239.255.255.250:1982",
        'MAN: "ssdp:discover"',
        "ST: wifi_bulb",
    ]) + "\r\n"

    # Create UDP socket
    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM, socket.IPPROTO_UDP)
    s.setsockopt(socket.IPPROTO_IP, socket.IP_MULTICAST_TTL, 2)
    s.settimeout(timeout)

    # Send discovery packet
    s.sendto(msg.encode(), ("239.255.255.250", 1982))

    # Collect responses
    bulbs = []
    try:
        while True:
            data, addr = s.recvfrom(65507)
            bulbs.append(data.decode())
    except socket.timeout:
        pass

    return bulbs
```

### 2.3 Message Format

#### Request Structure

All commands sent to the device follow this JSON structure:

```json
{
    "id": <command_id>,
    "method": "<method_name>",
    "params": [<param1>, <param2>, ...]
}
```

**Fields:**
- `id` (integer): Incrementing command identifier for tracking responses
- `method` (string): Command name (e.g., "set_rgb", "get_prop")
- `params` (array): List of parameters for the command

**Example Request:**
```json
{
    "id": 1,
    "method": "set_power",
    "params": ["on", "smooth", 500]
}
```

Messages must be terminated with `\r\n`:
```python
message = json.dumps(command) + "\r\n"
```

#### Response Structure

**Success Response:**
```json
{
    "id": <command_id>,
    "result": ["ok"]
}
```

Or with returned values:
```json
{
    "id": 2,
    "result": ["on", "50", "3500"]
}
```

**Error Response:**
```json
{
    "id": 1,
    "error": {
        "code": -1,
        "message": "general error"
    }
}
```

**Common Error Codes:**
- `-1`: General error
- `-2`: Invalid parameter
- `-3`: Operation not supported
- `-4`: Device in wrong state
- `-5`: Connection/network error

#### Notifications

Devices send **unsolicited notifications** when properties change:

```json
{
    "method": "props",
    "params": {
        "power": "on",
        "bright": "50",
        "ct": "3500"
    }
}
```

**Important:** When reading responses, you must handle both command responses AND notifications on the same connection.

### 2.4 Standard Commands Reference

#### Power Control

**set_power** - Turn device on or off
```python
# Params: [power, effect, duration, mode]
# power: "on" or "off"
# effect: "sudden" or "smooth"
# duration: transition time in milliseconds (min 30)
# mode: 0=normal, 1=CT mode, 2=RGB mode, 3=HSV mode, 5=color flow

bulb.send_command("set_power", ["on", "smooth", 500])
```

**toggle** - Toggle power state
```python
bulb.send_command("toggle", [])
```

#### Brightness Control

**set_bright** - Set brightness (1-100)
```python
# Params: [brightness, effect, duration]
bulb.send_command("set_bright", [50, "smooth", 300])
```

#### Color Control

**set_rgb** - Set RGB color
```python
# Params: [rgb_value, effect, duration]
# rgb_value = red * 65536 + green * 256 + blue
# Example: Red (255,0,0) = 255*65536 = 16711680

bulb.send_command("set_rgb", [16711680, "smooth", 300])
```

**set_hsv** - Set HSV color
```python
# Params: [hue, saturation, effect, duration]
# hue: 0-359
# saturation: 0-100

bulb.send_command("set_hsv", [120, 100, "smooth", 300])
```

**set_ct_abx** - Set color temperature
```python
# Params: [ct_value, effect, duration]
# ct_value: 1700-6500 (Kelvin)

bulb.send_command("set_ct_abx", [4700, "smooth", 300])
```

#### Property Queries

**get_prop** - Get device properties
```python
# Params: list of property names
response = bulb.send_command("get_prop", ["power", "bright", "rgb"])
# Returns: {"id": 1, "result": ["on", "50", "16711680"]}
```

#### Scene Control

**set_scene** - Set predefined scene
```python
# Params: [class, val1, val2, ...]
# class: "color", "hsv", "ct", "cf", "auto_delay_off"

# RGB scene
bulb.send_command("set_scene", ["color", 16711680, 50])

# HSV scene
bulb.send_command("set_scene", ["hsv", 300, 70, 100])

# CT scene
bulb.send_command("set_scene", ["ct", 5400, 100])
```

#### Color Flow

**start_cf** - Start color flow
```python
# Params: [count, action, flow_expression]
# count: 0=infinite, >0=number of times
# action: 0=recover, 1=stay, 2=turn off
# flow_expression: "duration,mode,value,brightness,..."
#   mode: 1=color, 2=CT, 7=sleep

flow = "1000,1,16711680,100,1000,1,65280,100"  # Red->Green
bulb.send_command("start_cf", [0, 0, flow])
```

**stop_cf** - Stop color flow
```python
bulb.send_command("stop_cf", [])
```

#### Music Mode

**set_music** - Enter/exit music mode
```python
# Enter music mode (bulb connects back to host:port)
bulb.send_command("set_music", [1, "192.168.0.100", 54321])

# Exit music mode
bulb.send_command("set_music", [0])
```

#### Other Commands

**set_default** - Save current state as default
```python
bulb.send_command("set_default", [])
```

**set_name** - Set device name
```python
bulb.send_command("set_name", ["Living Room Light"])
```

**set_adjust** - Adjust brightness/color/temperature
```python
# Params: [action, prop]
# action: "increase", "decrease", "circle"
# prop: "bright", "ct", "color"

bulb.send_command("set_adjust", ["increase", "bright"])
```

**cron_add** - Add cron job
```python
# Params: [type, value]
# type: 0=power off timer (value in minutes)
bulb.send_command("cron_add", [0, 10])
```

**cron_get** - Get cron job
```python
# Params: [type]
response = bulb.send_command("cron_get", [0])
```

**cron_del** - Delete cron job
```python
# Params: [type]
bulb.send_command("cron_del", [0])
```

### 2.5 Device Properties

**Queryable Properties:**

| Property | Type | Range | Description |
|----------|------|-------|-------------|
| `power` | string | "on", "off" | Power state |
| `bright` | integer | 1-100 | Brightness percentage |
| `ct` | integer | 1700-6500 | Color temperature (Kelvin) |
| `rgb` | integer | 0-16777215 | RGB color value |
| `hue` | integer | 0-359 | Hue |
| `sat` | integer | 0-100 | Saturation |
| `color_mode` | integer | 1, 2, 3 | 1=RGB, 2=CT, 3=HSV |
| `flowing` | integer | 0, 1 | Color flow status |
| `delayoff` | integer | 0-60 | Delay off timer (minutes) |
| `flow_params` | string | - | Current flow parameters |
| `music_on` | integer | 0, 1 | Music mode status |
| `name` | string | - | Device name |
| `bg_power` | string | "on", "off" | Background light power |
| `bg_flowing` | integer | 0, 1 | Background flow status |
| `bg_flow_params` | string | - | Background flow params |
| `bg_ct` | integer | 1700-6500 | Background CT |
| `bg_lmode` | integer | 1, 2, 3 | Background light mode |
| `bg_bright` | integer | 1-100 | Background brightness |
| `bg_rgb` | integer | 0-16777215 | Background RGB |
| `bg_hue` | integer | 0-359 | Background hue |
| `bg_sat` | integer | 0-100 | Background saturation |
| `nl_br` | integer | 1-100 | Night light brightness |
| `active_mode` | integer | 0, 1 | Active mode |

### 2.6 Music Mode

Music mode enables **high-frequency, rate-limit-free** command sending by creating a reverse connection.

#### How Music Mode Works

1. **Application** creates a TCP listening socket on a chosen port
2. **Application** sends `set_music` command with its IP and port
3. **Device** connects back to the application
4. Old connection is closed; new connection is used
5. Commands are sent fire-and-forget (no responses)
6. **Unlimited command rate** (theoretically ~1000+ updates/second)

#### Music Mode Protocol Flow

```
[App]                              [Device]
  |                                   |
  |-- 1. Create listening socket      |
  |                                   |
  |-- 2. send_command("set_music") -->|
  |                                   |
  |                <-- 3. TCP connect |
  |                                   |
  |-- 4. Close old connection         |
  |                                   |
  |-- 5. Send rapid commands -------->|
  |-- 6. Send rapid commands -------->|
  |-- 7. Send rapid commands -------->|
  |                                   |
```

#### Advantages

- **No rate limiting**: Send unlimited commands
- **Fast updates**: Ideal for animations and real-time control
- **No response overhead**: Fire-and-forget operation

#### Disadvantages

- **No feedback**: Can't verify command success
- **More complex setup**: Requires listening socket
- **Network dependency**: Both devices must be on same network

#### Implementation Example

```python
import socket
import json
from yeelight import Bulb

# Method 1: Using python-yeelight library
bulb = Bulb("192.168.0.34")
bulb.start_music(port=0, ip=None)  # Auto-select port and IP

# Now send unlimited commands
for i in range(1000):
    bulb.set_rgb(255, 0, 0)
    bulb.set_rgb(0, 255, 0)

bulb.stop_music()

# Method 2: Manual implementation
class MusicModeConnection:
    def __init__(self, device_ip, device_port=55443):
        self.device_ip = device_ip
        self.device_port = device_port
        self.listen_socket = None
        self.music_socket = None

    def start(self):
        # Create listening socket
        self.listen_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.listen_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        self.listen_socket.bind(("0.0.0.0", 0))  # Auto-select port
        self.listen_socket.listen(1)

        listen_port = self.listen_socket.getsockname()[1]
        listen_ip = self._get_local_ip()

        # Connect to device and send music mode command
        temp_sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        temp_sock.connect((self.device_ip, self.device_port))

        cmd = {
            "id": 1,
            "method": "set_music",
            "params": [1, listen_ip, listen_port]
        }
        temp_sock.send((json.dumps(cmd) + "\r\n").encode())

        # Wait for device to connect back
        self.music_socket, addr = self.listen_socket.accept()
        temp_sock.close()

    def send_command(self, method, params):
        cmd = {"id": 0, "method": method, "params": params}
        self.music_socket.send((json.dumps(cmd) + "\r\n").encode())

    def stop(self):
        if self.music_socket:
            self.music_socket.close()
        if self.listen_socket:
            self.listen_socket.close()

    def _get_local_ip(self):
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.connect((self.device_ip, 80))
        ip = s.getsockname()[0]
        s.close()
        return ip
```

### 2.7 Rate Limiting

**Standard Mode:**
- **Maximum**: 60 commands per minute
- **Exceeding limit**: Commands may be dropped or device may disconnect
- **Recommendation**: Add delays between commands

**Music Mode:**
- **Maximum**: Unlimited (hardware limited only)
- **Practical limit**: ~1000-2000 updates per second
- **Use case**: Animations, real-time control, rapid color changes

---

## 3. Matrix Device Specific Implementation

### 3.1 Matrix Device Architecture

The Yeelight Matrix (also known as Canvas) is a modular LED panel system that can display custom images and patterns.

#### Module Types

**1. 5x5_clear** - 25 LEDs in 5x5 grid without blur effect
```
[LED] [LED] [LED] [LED] [LED]
[LED] [LED] [LED] [LED] [LED]
[LED] [LED] [LED] [LED] [LED]
[LED] [LED] [LED] [LED] [LED]
[LED] [LED] [LED] [LED] [LED]
```

**2. 5x5_blur** - 25 LEDs in 5x5 grid with diffusion/blur effect
- Same LED layout as 5x5_clear
- Diffuser creates softer, blurred appearance
- Better for gradients and smooth color transitions

**3. 1x1** - Single LED module
- One individually controllable LED
- Used for accent lighting or simple indicators

#### LED Addressing

LEDs in a 5x5 module are addressed in **row-major order** (left-to-right, top-to-bottom):

```
Position indices:
 0   1   2   3   4
 5   6   7   8   9
10  11  12  13  14
15  16  17  18  19
20  21  22  23  24
```

Color data must be provided in this exact order.

#### Physical Orientation

Matrix modules can be mounted in different orientations:

**Vertical Layout:**
```
[Module 0]
[Module 1]
[Module 2]
[Module 3]
```

**Horizontal Layout:**
```
[Module 0] [Module 1] [Module 2] [Module 3]
```

**Base Position:**
- **Bottom**: Modules numbered bottom-to-top
- **Top**: Modules numbered top-to-bottom
- **Left**: Modules numbered left-to-right
- **Right**: Modules numbered right-to-left

### 3.2 Matrix-Specific Commands

#### activate_fx_mode

Activates a special effect mode on the device. For Matrix devices, `"direct"` mode is required for manual LED control.

**Command:**
```json
{
    "id": 1,
    "method": "activate_fx_mode",
    "params": [{"mode": "direct"}]
}
```

**Modes:**
- `"direct"`: Manual LED control (required for custom patterns)
- Other modes may be device-specific

**Must be called before sending LED updates.**

#### update_leds

Updates all LEDs with new RGB data.

**Command:**
```json
{
    "id": 2,
    "method": "update_leds",
    "params": ["<base64_encoded_rgb_data>"]
}
```

**Parameter:**
- `base64_encoded_rgb_data`: Base64-encoded concatenated RGB bytes for all LEDs

### 3.3 Color Encoding for Matrix

#### Encoding Process

The Matrix requires color data to be **base64-encoded RGB bytes**.

**Step-by-Step Process:**

1. **Start with hex color**: `"#FF0000"` (red)
2. **Convert to RGB tuple**: `(255, 0, 0)`
3. **Convert to bytes**: `b'\xff\x00\x00'`
4. **Base64 encode**: `"/wAA"`
5. **Concatenate for all LEDs**

#### Color Encoding Example

```python
import base64

def encode_hex_color(hex_color):
    """Convert hex color to base64-encoded RGB."""
    # Remove '#' prefix if present
    hex_color = hex_color.lstrip("#")

    # Convert hex to RGB tuple
    rgb = tuple(int(hex_color[i:i+2], 16) for i in (0, 2, 4))

    # Convert to bytes
    rgb_bytes = bytes(rgb)

    # Base64 encode
    encoded = base64.b64encode(rgb_bytes).decode("ascii")

    return encoded

# Examples
encode_hex_color("#FF0000")  # "/wAA" (red)
encode_hex_color("#00FF00")  # "AP8A" (green)
encode_hex_color("#0000FF")  # "AAD/" (blue)
encode_hex_color("#FFFFFF")  # "////" (white)
encode_hex_color("#000000")  # "AAAA" (black/off)
```

#### 5x5 Module Format

A 5x5 module requires exactly **25 encoded colors** concatenated together.

**Data Structure:**
- 25 LEDs × 3 bytes per LED = 75 bytes total
- 75 bytes base64-encoded = 100 characters
- Order: row-major (positions 0-24)

**Example - All Red:**
```python
# 25 red LEDs
rgb_data = "".join([encode_hex_color("#FF0000") for _ in range(25)])
# Result: "/wAA/wAA/wAA..." (100 characters)
```

**Example - Checkerboard Pattern:**
```python
colors = []
for i in range(5):
    for j in range(5):
        if (i + j) % 2 == 0:
            colors.append("#FF0000")  # Red
        else:
            colors.append("#0000FF")  # Blue

rgb_data = "".join([encode_hex_color(c) for c in colors])
```

#### 1x1 Module Format

A 1x1 module requires exactly **1 encoded color**.

**Data Structure:**
- 1 LED × 3 bytes = 3 bytes
- 3 bytes base64-encoded = 4 characters

**Example:**
```python
rgb_data = encode_hex_color("#00FF00")  # "AP8A" (green)
```

#### Multiple Module Format

When controlling multiple modules, concatenate their RGB data in the correct order based on physical layout.

**Example - 4 Modules (Vertical, Bottom-to-Top):**
```python
module_0_data = "".join([encode_hex_color("#FF0000") for _ in range(25)])  # Red
module_1_data = "".join([encode_hex_color("#00FF00") for _ in range(25)])  # Green
module_2_data = "".join([encode_hex_color("#0000FF") for _ in range(25)])  # Blue
module_3_data = "".join([encode_hex_color("#FFFF00") for _ in range(25)])  # Yellow

# Concatenate in bottom-to-top order
combined_data = module_0_data + module_1_data + module_2_data + module_3_data

# Send to device
bulb.send_command("update_leds", [combined_data])
```

### 3.4 Setting Image Patterns

#### Complete Flow

To display an image on the Matrix device:

1. **Activate Direct Mode**
   ```python
   bulb.send_command("activate_fx_mode", [{"mode": "direct"}])
   ```

2. **Prepare Image Data**
   - Load image file
   - Resize to module dimensions (5×N or N×5 pixels per module)
   - Extract RGB values for each pixel
   - Apply rotation based on physical orientation

3. **Encode RGB Data**
   - Convert each RGB pixel to base64
   - Concatenate in proper module order

4. **Send to Device**
   ```python
   bulb.send_command("update_leds", [encoded_rgb_data])
   ```

5. **Optional: Set Brightness**
   ```python
   bulb.send_command("set_bright", [100])
   ```

#### Image Processing Pipeline

```python
from PIL import Image
import base64

def image_to_matrix_data(image_path, width=5, height=5):
    """
    Convert an image file to Matrix RGB data.

    Args:
        image_path: Path to image file
        width: Number of LEDs horizontally
        height: Number of LEDs vertically

    Returns:
        Base64-encoded RGB data string
    """
    # Load and resize image
    img = Image.open(image_path)
    img = img.resize((width, height), Image.Resampling.LANCZOS)

    # Convert to RGB (handle RGBA, grayscale, etc.)
    img = img.convert("RGB")

    # Extract RGB data
    rgb_data = ""
    for pixel in img.getdata():
        r, g, b = pixel
        # Encode each pixel
        encoded = base64.b64encode(bytes([r, g, b])).decode("ascii")
        rgb_data += encoded

    return rgb_data

# Usage
rgb_data = image_to_matrix_data("pattern.png", width=5, height=5)
bulb.send_command("update_leds", [rgb_data])
```

#### Multi-Module Image Display

For images spanning multiple modules:

```python
def multi_module_image(image_path, num_modules, orientation="vertical"):
    """
    Convert image to multi-module Matrix data.

    Args:
        image_path: Path to image file
        num_modules: Number of 5x5 modules
        orientation: "vertical" or "horizontal"

    Returns:
        Base64-encoded RGB data for all modules
    """
    from PIL import Image
    import base64

    # Calculate image dimensions
    if orientation == "vertical":
        img_width, img_height = 5, 5 * num_modules
    else:
        img_width, img_height = 5 * num_modules, 5

    # Load and resize
    img = Image.open(image_path)
    img = img.resize((img_width, img_height), Image.Resampling.LANCZOS)
    img = img.convert("RGB")

    # Split into modules
    all_module_data = ""

    if orientation == "vertical":
        for i in range(num_modules):
            # Crop 5x5 section
            module_img = img.crop((0, i*5, 5, (i+1)*5))

            # Encode module
            for pixel in module_img.getdata():
                r, g, b = pixel
                all_module_data += base64.b64encode(bytes([r, g, b])).decode("ascii")
    else:  # horizontal
        for i in range(num_modules):
            # Crop 5x5 section
            module_img = img.crop((i*5, 0, (i+1)*5, 5))

            # Encode module
            for pixel in module_img.getdata():
                r, g, b = pixel
                all_module_data += base64.b64encode(bytes([r, g, b])).decode("ascii")

    return all_module_data

# Usage for 4 vertical modules
rgb_data = multi_module_image("tall_art.png", num_modules=4, orientation="vertical")
bulb.send_command("update_leds", [rgb_data])
```

#### Module Ordering

The order of concatenated module data depends on physical mounting:

**Vertical Layout (Bottom-to-Top):**
```python
# Physical arrangement:
# [Module 3] ← Top
# [Module 2]
# [Module 1]
# [Module 0] ← Bottom

# Data order: bottom to top
data = module_0 + module_1 + module_2 + module_3
```

**Vertical Layout (Top-to-Bottom):**
```python
# Physical arrangement:
# [Module 0] ← Top
# [Module 1]
# [Module 2]
# [Module 3] ← Bottom

# Data order: top to bottom
data = module_0 + module_1 + module_2 + module_3
```

**Horizontal Layout (Left-to-Right):**
```python
# Physical arrangement:
# [Module 0] [Module 1] [Module 2] [Module 3]

# Data order: left to right
data = module_0 + module_1 + module_2 + module_3
```

---

## 4. Code Examples

### 4.1 Raw Protocol Examples

#### Example 1: Basic Connection and Command

```python
import socket
import json

# Connect to device
sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sock.settimeout(5)
sock.connect(("192.168.0.34", 55443))

# Turn on with smooth transition
command = {
    "id": 1,
    "method": "set_power",
    "params": ["on", "smooth", 500]
}
message = json.dumps(command, separators=(",", ":")) + "\r\n"
sock.send(message.encode("utf-8"))

# Read response
response = sock.recv(4096).decode("utf-8")
print(response)
# Output: {"id":1,"result":["ok"]}

sock.close()
```

#### Example 2: Activate Direct Mode and Update LEDs

```python
import socket
import json
import base64

# Connect
sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sock.settimeout(5)
sock.connect(("192.168.0.34", 55443))

# Activate direct mode
cmd1 = {
    "id": 1,
    "method": "activate_fx_mode",
    "params": [{"mode": "direct"}]
}
sock.send((json.dumps(cmd1) + "\r\n").encode("utf-8"))
response1 = sock.recv(4096)
print("Direct mode:", response1.decode())

# Helper function
def encode_color(r, g, b):
    return base64.b64encode(bytes([r, g, b])).decode("ascii")

# Create pattern: 25 red LEDs (one 5x5 module)
rgb_data = "".join([encode_color(255, 0, 0) for _ in range(25)])

# Send update
cmd2 = {
    "id": 2,
    "method": "update_leds",
    "params": [rgb_data]
}
sock.send((json.dumps(cmd2) + "\r\n").encode("utf-8"))
response2 = sock.recv(4096)
print("Update LEDs:", response2.decode())

sock.close()
```

#### Example 3: Query Properties

```python
import socket
import json

sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sock.connect(("192.168.0.34", 55443))

# Query multiple properties
cmd = {
    "id": 1,
    "method": "get_prop",
    "params": ["power", "bright", "ct", "rgb", "color_mode"]
}
sock.send((json.dumps(cmd) + "\r\n").encode())

response = sock.recv(4096).decode()
data = json.loads(response)
print("Properties:", data["result"])
# Output: ["on", "50", "3500", "16711680", "2"]

sock.close()
```

#### Example 4: Color Flow Animation

```python
import socket
import json

sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sock.connect(("192.168.0.34", 55443))

# Create flow: Red -> Green -> Blue (1 second each)
flow_expression = "1000,1,16711680,100,1000,1,65280,100,1000,1,255,100"

cmd = {
    "id": 1,
    "method": "start_cf",
    "params": [
        0,  # Loop forever
        1,  # Stay at last color when stopped
        flow_expression
    ]
}
sock.send((json.dumps(cmd) + "\r\n").encode())
response = sock.recv(4096)
print(response.decode())

sock.close()
```

### 4.2 Python-yeelight Library Examples

#### Example 1: Basic Connection and Control

```python
from yeelight import Bulb

# Connect to device
bulb = Bulb("192.168.0.34", port=55443)

# Turn on
bulb.turn_on()

# Set brightness
bulb.set_brightness(100)

# Set RGB color (red)
bulb.set_rgb(255, 0, 0)

# Set color temperature
bulb.set_color_temp(4700)

# Turn off with smooth transition
bulb.turn_off(effect="smooth", duration=1000)
```

#### Example 2: Using Custom Commands for Matrix

```python
from yeelight import Bulb
import base64

# Connect
bulb = Bulb("192.168.0.34", port=55443)

# Activate direct mode (required for Matrix)
bulb.send_command("activate_fx_mode", [{"mode": "direct"}])

# Set brightness
bulb.set_brightness(100)

# Helper function
def encode_hex_color(hex_color):
    hex_color = hex_color.lstrip("#")
    rgb = tuple(int(hex_color[i:i+2], 16) for i in (0, 2, 4))
    return base64.b64encode(bytes(rgb)).decode("ascii")

# Create solid red 5x5 module
colors = ["#FF0000"] * 25
rgb_data = "".join([encode_hex_color(c) for c in colors])

# Send to device
bulb.send_command("update_leds", [rgb_data])
```

#### Example 3: Matrix Control with Music Mode

```python
from yeelight import Bulb
import base64
import time

bulb = Bulb("192.168.0.34", port=55443)

# Enable music mode for fast updates
bulb.start_music()

# Activate direct mode
bulb.send_command("activate_fx_mode", [{"mode": "direct"}])

def encode_hex_color(hex_color):
    hex_color = hex_color.lstrip("#")
    rgb = tuple(int(hex_color[i:i+2], 16) for i in (0, 2, 4))
    return base64.b64encode(bytes(rgb)).decode("ascii")

# Create checkerboard pattern for 5x5 module
def create_checkerboard():
    colors = []
    for i in range(5):
        for j in range(5):
            if (i + j) % 2 == 0:
                colors.append("#FF0000")  # Red
            else:
                colors.append("#0000FF")  # Blue
    return "".join([encode_hex_color(c) for c in colors])

# Display checkerboard
rgb_data = create_checkerboard()
bulb.send_command("update_leds", [rgb_data])

# Animate: alternate between checkerboard and inverse
for _ in range(10):
    # Normal checkerboard
    rgb_data = create_checkerboard()
    bulb.send_command("update_leds", [rgb_data])
    time.sleep(0.5)

    # Inverse checkerboard (swap colors)
    colors_inv = []
    for i in range(5):
        for j in range(5):
            if (i + j) % 2 == 0:
                colors_inv.append("#0000FF")  # Blue
            else:
                colors_inv.append("#FF0000")  # Red
    rgb_data_inv = "".join([encode_hex_color(c) for c in colors_inv])
    bulb.send_command("update_leds", [rgb_data_inv])
    time.sleep(0.5)

# Stop music mode
bulb.stop_music()
```

#### Example 4: Setting Image from File (Single Module)

```python
from yeelight import Bulb
from PIL import Image
import base64

def image_to_matrix_data(image_path, width=5, height=5):
    """Convert image to matrix RGB data."""
    img = Image.open(image_path)
    img = img.resize((width, height), Image.Resampling.LANCZOS)
    img = img.convert("RGB")

    rgb_data = ""
    for pixel in img.getdata():
        r, g, b = pixel
        encoded = base64.b64encode(bytes([r, g, b])).decode("ascii")
        rgb_data += encoded

    return rgb_data

# Connect and setup
bulb = Bulb("192.168.0.34", port=55443)
bulb.start_music()
bulb.set_brightness(100)
bulb.send_command("activate_fx_mode", [{"mode": "direct"}])

# Load and display image
rgb_data = image_to_matrix_data("pattern.png", width=5, height=5)
bulb.send_command("update_leds", [rgb_data])

bulb.stop_music()
```

#### Example 5: Multiple Modules (Vertical Layout)

```python
from yeelight import Bulb
from PIL import Image
import base64

def load_multi_module_image(image_path, num_modules=4):
    """Load image and split for multiple 5x5 modules (vertical)."""
    img = Image.open(image_path)
    img = img.resize((5, 5 * num_modules), Image.Resampling.LANCZOS)
    img = img.convert("RGB")

    modules_data = []
    for i in range(num_modules):
        # Crop 5x5 section
        module_img = img.crop((0, i*5, 5, (i+1)*5))

        # Convert to base64 RGB
        rgb_data = ""
        for pixel in module_img.getdata():
            r, g, b = pixel
            rgb_data += base64.b64encode(bytes([r, g, b])).decode("ascii")

        modules_data.append(rgb_data)

    return "".join(modules_data)

# Setup
bulb = Bulb("192.168.0.34", port=55443)
bulb.start_music()
bulb.send_command("activate_fx_mode", [{"mode": "direct"}])
bulb.set_brightness(100)

# Display on 4 modules
rgb_data = load_multi_module_image("tall_art.png", num_modules=4)
bulb.send_command("update_leds", [rgb_data])

bulb.stop_music()
```

#### Example 6: Animated Gradient

```python
from yeelight import Bulb
import base64
import time
import colorsys

bulb = Bulb("192.168.0.34", port=55443)
bulb.start_music()
bulb.send_command("activate_fx_mode", [{"mode": "direct"}])

def create_gradient(hue_offset):
    """Create a rainbow gradient for 5x5 module."""
    colors = []
    for i in range(25):
        # Calculate hue based on position and offset
        hue = ((i / 25.0) + hue_offset) % 1.0
        r, g, b = colorsys.hsv_to_rgb(hue, 1.0, 1.0)

        # Convert to 0-255 range
        r, g, b = int(r * 255), int(g * 255), int(b * 255)

        # Encode
        encoded = base64.b64encode(bytes([r, g, b])).decode("ascii")
        colors.append(encoded)

    return "".join(colors)

# Animate gradient
for frame in range(100):
    hue_offset = frame / 100.0
    rgb_data = create_gradient(hue_offset)
    bulb.send_command("update_leds", [rgb_data])
    time.sleep(0.05)

bulb.stop_music()
```

#### Example 7: Device Discovery

```python
from yeelight import discover_bulbs

# Discover all Yeelight devices on network
bulbs = discover_bulbs(timeout=2)

for bulb_info in bulbs:
    print(f"Found device:")
    print(f"  IP: {bulb_info['ip']}")
    print(f"  Port: {bulb_info['port']}")
    print(f"  Model: {bulb_info['capabilities']['model']}")
    print(f"  Supports: {bulb_info['capabilities']['support']}")
    print()
```

### 4.3 YeelightMatrix Library Examples

The `yeelight_matrix` library provides high-level abstractions for controlling Matrix devices.

#### Example 1: Basic Usage

```python
from yeelight_matrix.cube_matrix import CubeMatrix
from yeelight_matrix.layout import Layout

# Connect to device
cube = CubeMatrix("192.168.0.34", 55443)
cube.set_fx_mode("direct")
cube.get_bulb().set_brightness(100)

# Define physical layout (4 modules vertical, bottom-to-top)
layout = Layout("vertical", "bottom")
layout.add_modules_list([
    "5x5_blur",
    "5x5_clear",
    "5x5_clear",
    "5x5_clear"
])

# Set solid colors on individual modules
layout.set_module_colors(0, ["#FF0000"] * 25)  # Bottom: red
layout.set_module_colors(1, ["#00FF00"] * 25)  # Green
layout.set_module_colors(2, ["#0000FF"] * 25)  # Blue
layout.set_module_colors(3, ["#FFFF00"] * 25)  # Top: yellow

# Send to device
cube.draw_matrices(layout.get_raw_rgb_data())
```

#### Example 2: Display Image Across Multiple Modules

```python
from yeelight_matrix.cube_matrix import CubeMatrix
from yeelight_matrix.layout import Layout

cube = CubeMatrix("192.168.0.34", 55443)
cube.set_fx_mode("direct")
cube.get_bulb().set_brightness(100)

# Define layout
layout = Layout("vertical", "bottom")
layout.add_modules_list([
    "5x5_clear",
    "5x5_clear",
    "5x5_clear",
    "5x5_clear"
])

# Load and display image across all modules
# Image will be resized to 5x20 pixels (5 wide, 20 tall for 4 modules)
layout.set_image("artwork.png", start=0, max=4)

# Send to device
cube.draw_matrices(layout.get_raw_rgb_data())
```

#### Example 3: Mixed Module Types with Custom Colors and Images

```python
from yeelight_matrix.cube_matrix import CubeMatrix
from yeelight_matrix.layout import Layout

cube = CubeMatrix("192.168.0.34", 55443)
cube.set_fx_mode("direct")
cube.get_bulb().set_brightness(100)

# Mixed layout: 1x1 on bottom, 3x 5x5 above
layout = Layout("vertical", "bottom")
layout.add_modules_list([
    "1x1",         # Module 0: Single LED
    "5x5_clear",   # Module 1
    "5x5_clear",   # Module 2
    "5x5_clear"    # Module 3
])

# Set 1x1 module to red
layout.set_module_colors(0, ["#FF0000"])

# Display image on 5x5 modules
layout.set_image("pattern.png", start=1, max=3)

# Send to device
cube.draw_matrices(layout.get_raw_rgb_data())
```

#### Example 4: Animation Loop

```python
from yeelight_matrix.cube_matrix import CubeMatrix
from yeelight_matrix.layout import Layout
import time

cube = CubeMatrix("192.168.0.34", 55443, music_mode=True)
cube.set_fx_mode("direct")
cube.get_bulb().set_brightness(100)

layout = Layout("vertical", "bottom")
layout.add_modules_list(["5x5_clear"] * 4)

# Animate: light up one module at a time
colors = ["#FF0000", "#00FF00", "#0000FF", "#FFFF00"]

for _ in range(10):  # 10 cycles
    for i in range(4):
        # Clear all modules
        for j in range(4):
            layout.set_module_colors(j, ["#000000"] * 25)

        # Light up current module
        layout.set_module_colors(i, [colors[i]] * 25)

        # Send to device
        cube.draw_matrices(layout.get_raw_rgb_data())
        time.sleep(0.2)
```

---

## 5. Troubleshooting and Best Practices

### 5.1 Common Issues

#### Connection Timeout

**Symptoms:**
- `socket.timeout` exception
- "Connection refused" error

**Solutions:**
1. Verify device IP address is correct
2. Check device is on and connected to network
3. Ensure device is on same subnet
4. Verify port 55443 is not blocked by firewall
5. Try increasing socket timeout

```python
sock.settimeout(10)  # Increase to 10 seconds
```

#### Rate Limiting Errors

**Symptoms:**
- Commands ignored or dropped
- Device disconnects after rapid commands
- "Limit exceeded" errors

**Solutions:**
1. Enable music mode for frequent updates
2. Add delays between commands in standard mode

```python
import time

bulb.set_rgb(255, 0, 0)
time.sleep(1)  # Wait 1 second
bulb.set_rgb(0, 255, 0)
```

3. Batch related operations where possible

#### Color Encoding Errors

**Symptoms:**
- Unexpected colors displayed
- Device returns error on `update_leds`
- "Invalid parameter" error

**Solutions:**
1. Verify base64 encoding is correct
2. Ensure RGB data length matches module count:
   - 5x5 module: 100 characters (25 LEDs × 4 chars)
   - 1x1 module: 4 characters (1 LED × 4 chars)
3. Check for correct byte order (RGB, not BGR)

```python
# Correct
encoded = base64.b64encode(bytes([r, g, b])).decode("ascii")

# Wrong - BGR order
encoded = base64.b64encode(bytes([b, g, r])).decode("ascii")  # ❌
```

#### Module Ordering Issues

**Symptoms:**
- Image appears upside down or mirrored
- Colors on wrong modules

**Solutions:**
1. Verify physical module orientation
2. Adjust base position in Layout
3. Reverse module order if needed

```python
# If image is upside down
layout = Layout("vertical", "top")  # Try "top" instead of "bottom"

# Or manually reverse module data
modules_data = [module_0, module_1, module_2, module_3]
combined = "".join(reversed(modules_data))
```

#### Image Not Displaying

**Symptoms:**
- LEDs stay black/off
- No response to `update_leds`

**Solutions:**
1. Ensure `activate_fx_mode` with `"direct"` was called first
2. Verify brightness is not 0
3. Check RGB data is not all black

```python
# Correct sequence
bulb.send_command("activate_fx_mode", [{"mode": "direct"}])
bulb.set_brightness(100)  # Set brightness AFTER direct mode
bulb.send_command("update_leds", [rgb_data])
```

### 5.2 Best Practices

#### Use Music Mode for Animations

For any rapid updates or animations, always use music mode:

```python
# Good - with music mode
bulb.start_music()
for frame in range(100):
    bulb.send_command("update_leds", [generate_frame(frame)])
bulb.stop_music()

# Bad - without music mode (slow and rate limited)
for frame in range(100):
    bulb.send_command("update_leds", [generate_frame(frame)])  # ❌
```

#### Keep Persistent Connections

Create one connection and reuse it:

```python
# Good
bulb = Bulb("192.168.0.34")
for i in range(10):
    bulb.set_rgb(i * 25, 0, 0)

# Bad - reconnects every time
for i in range(10):
    bulb = Bulb("192.168.0.34")  # ❌ Wasteful
    bulb.set_rgb(i * 25, 0, 0)
```

#### Handle Notifications Asynchronously

In production code, handle property notifications separately from responses:

```python
import socket
import json
import threading

def notification_listener(sock):
    """Background thread to handle notifications."""
    while True:
        data = sock.recv(4096).decode()
        for line in data.split("\r\n"):
            if not line:
                continue
            msg = json.loads(line)
            if msg.get("method") == "props":
                print(f"Property changed: {msg['params']}")

# Start listener thread
sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sock.connect(("192.168.0.34", 55443))
thread = threading.Thread(target=notification_listener, args=(sock,))
thread.daemon = True
thread.start()
```

#### Validate RGB Data Length

Before sending, verify data length matches expected:

```python
def validate_module_data(rgb_data, module_type):
    """Validate RGB data length for module type."""
    if module_type == "5x5":
        expected_length = 100  # 25 LEDs × 4 chars
    elif module_type == "1x1":
        expected_length = 4    # 1 LED × 4 chars
    else:
        raise ValueError(f"Unknown module type: {module_type}")

    if len(rgb_data) != expected_length:
        raise ValueError(
            f"Invalid data length: expected {expected_length}, got {len(rgb_data)}"
        )

    return True

# Usage
rgb_data = create_pattern()
validate_module_data(rgb_data, "5x5")
bulb.send_command("update_leds", [rgb_data])
```

#### Set Brightness Before Updating LEDs

For better visibility, set brightness before sending LED data:

```python
# Good
bulb.set_brightness(100)
bulb.send_command("update_leds", [rgb_data])

# Works but less ideal
bulb.send_command("update_leds", [rgb_data])
bulb.set_brightness(100)  # User sees dim colors first
```

#### Use Smooth Transitions for Better Effects

When changing states, use smooth transitions:

```python
# Good - smooth transition
bulb.turn_on(effect="smooth", duration=500)
bulb.set_rgb(255, 0, 0, effect="smooth", duration=1000)

# Works but jarring
bulb.turn_on(effect="sudden")
bulb.set_rgb(255, 0, 0, effect="sudden")
```

#### Error Handling

Always handle potential errors:

```python
import socket
import json

def safe_send_command(bulb, method, params):
    """Send command with error handling."""
    try:
        response = bulb.send_command(method, params)
        if "error" in response:
            print(f"Device error: {response['error']['message']}")
            return None
        return response
    except socket.timeout:
        print("Command timed out")
        return None
    except ConnectionError:
        print("Connection lost")
        return None
    except Exception as e:
        print(f"Unexpected error: {e}")
        return None

# Usage
result = safe_send_command(bulb, "set_rgb", [16711680, "smooth", 500])
if result:
    print("Success")
```

### 5.3 Performance Considerations

#### Update Rates

**Standard Mode:**
- **Maximum**: ~1 update/second (60/minute limit)
- **Practical**: Add 1-second delays between commands
- **Use case**: Status indicators, slow transitions

**Music Mode:**
- **Theoretical maximum**: No limit
- **Practical maximum**: ~1000-2000 updates/second
- **Network limited**: ~500-1000 updates/second typical
- **Hardware limited**: Device processing time
- **Use case**: Animations, real-time visualizations

#### Base64 Encoding Overhead

**Encoding Time:**
- Single LED: ~1-2 microseconds
- 5x5 module (25 LEDs): ~25-50 microseconds
- 4 modules (100 LEDs): ~100-200 microseconds

**Impact:** Negligible for most applications. Encoding is fast enough even for real-time animations at 60+ FPS.

#### Network Latency

**Typical Latencies:**
- Same subnet: 1-5ms
- Same WiFi network: 5-20ms
- Cross-router: 20-50ms

**Optimization:**
- Use wired connection if possible
- Minimize network hops
- Use music mode to eliminate response overhead
- Batch updates when possible

#### Memory Usage

**RGB Data Size:**
- 5x5 module: 100 characters = 100 bytes
- 4 modules: 400 bytes
- 10 modules: 1000 bytes = 1 KB

**Impact:** Minimal memory usage. Even complex multi-module setups use negligible memory.

#### Frame Rate Calculations

For smooth animations, target frame rates:

**Music Mode:**
```python
# 60 FPS (16.7ms per frame)
fps = 60
frame_time = 1.0 / fps

for frame in range(1000):
    start = time.time()

    # Generate and send frame
    rgb_data = generate_frame(frame)
    bulb.send_command("update_leds", [rgb_data])

    # Wait for next frame
    elapsed = time.time() - start
    if elapsed < frame_time:
        time.sleep(frame_time - elapsed)
```

**Achievable Frame Rates:**
- Simple patterns: 100+ FPS
- Complex image processing: 30-60 FPS
- Network limited: 50-100 FPS typical

---

## 6. Reference Tables

### 6.1 Command Reference

| Command | Parameters | Description | Example |
|---------|------------|-------------|---------|
| `set_power` | [power, effect, duration, mode] | Turn on/off | `["on", "smooth", 500]` |
| `toggle` | [] | Toggle power | `[]` |
| `set_bright` | [brightness, effect, duration] | Set brightness (1-100) | `[50, "smooth", 300]` |
| `set_rgb` | [rgb_value, effect, duration] | Set RGB color | `[16711680, "smooth", 300]` |
| `set_hsv` | [hue, sat, effect, duration] | Set HSV color | `[120, 100, "smooth", 300]` |
| `set_ct_abx` | [ct_value, effect, duration] | Set color temp (1700-6500K) | `[4700, "smooth", 300]` |
| `get_prop` | [prop1, prop2, ...] | Get properties | `["power", "bright", "rgb"]` |
| `set_scene` | [class, val1, val2, ...] | Set scene | `["color", 16711680, 50]` |
| `start_cf` | [count, action, flow_expr] | Start color flow | `[0, 0, "1000,1,16711680,100"]` |
| `stop_cf` | [] | Stop color flow | `[]` |
| `set_music` | [action, host, port] | Music mode control | `[1, "192.168.0.100", 54321]` |
| `set_default` | [] | Save current as default | `[]` |
| `set_name` | [name] | Set device name | `["Living Room"]` |
| `set_adjust` | [action, prop] | Adjust property | `["increase", "bright"]` |
| `cron_add` | [type, value] | Add cron timer | `[0, 10]` |
| `cron_get` | [type] | Get cron timer | `[0]` |
| `cron_del` | [type] | Delete cron timer | `[0]` |
| `activate_fx_mode` | [{"mode": mode}] | Activate effect mode (Matrix) | `[{"mode": "direct"}]` |
| `update_leds` | [rgb_data] | Update LEDs (Matrix) | `["<base64_data>"]` |

### 6.2 Property Reference

| Property | Type | Range/Values | Description |
|----------|------|--------------|-------------|
| `power` | string | "on", "off" | Power state |
| `bright` | integer | 1-100 | Brightness percentage |
| `ct` | integer | 1700-6500 | Color temperature (Kelvin) |
| `rgb` | integer | 0-16777215 | RGB value (r×65536 + g×256 + b) |
| `hue` | integer | 0-359 | Hue (degrees) |
| `sat` | integer | 0-100 | Saturation percentage |
| `color_mode` | integer | 1, 2, 3 | Color mode (1=RGB, 2=CT, 3=HSV) |
| `flowing` | integer | 0, 1 | Color flow active (0=no, 1=yes) |
| `delayoff` | integer | 0-60 | Delay off timer (minutes) |
| `flow_params` | string | - | Current flow parameters |
| `music_on` | integer | 0, 1 | Music mode active (0=no, 1=yes) |
| `name` | string | - | Device name |
| `bg_power` | string | "on", "off" | Background light power |
| `bg_bright` | integer | 1-100 | Background brightness |
| `bg_ct` | integer | 1700-6500 | Background color temp |
| `bg_rgb` | integer | 0-16777215 | Background RGB value |
| `bg_hue` | integer | 0-359 | Background hue |
| `bg_sat` | integer | 0-100 | Background saturation |
| `bg_flowing` | integer | 0, 1 | Background flow active |
| `nl_br` | integer | 1-100 | Night light brightness |
| `active_mode` | integer | 0, 1 | Active mode |

### 6.3 Error Codes

| Code | Message | Description | Solution |
|------|---------|-------------|----------|
| `-1` | general error | Unspecified error | Check command syntax and device state |
| `-2` | invalid parameter | Parameter out of range or wrong type | Verify parameter values match specification |
| `-3` | operation not supported | Device doesn't support this command | Check device capabilities in discovery response |
| `-4` | device in wrong state | Command not valid in current state | Check power state or current mode |
| `-5` | connection error | Network or socket error | Check network connection and device availability |

### 6.4 Effect Types

| Effect | Duration Required | Description |
|--------|-------------------|-------------|
| `"sudden"` | No (ignored) | Immediate change, no transition |
| `"smooth"` | Yes (30+ ms) | Gradual transition over specified duration |

### 6.5 Color Mode Values

| Value | Mode | Description |
|-------|------|-------------|
| `1` | RGB | Device using RGB color |
| `2` | CT | Device using color temperature |
| `3` | HSV | Device using HSV color |

### 6.6 Module Types (Matrix)

| Type | LEDs | Description |
|------|------|-------------|
| `"5x5_clear"` | 25 | 5×5 grid, no diffusion |
| `"5x5_blur"` | 25 | 5×5 grid, with diffuser |
| `"1x1"` | 1 | Single LED |

---

## 7. Appendix

### 7.1 Protocol Specification Links

**Official Documentation:**
- Yeelight API Specification (Chinese): https://www.yeelight.com/download/Yeelight_Inter-Operation_Spec.pdf
- Yeelight Developer Portal: https://www.yeelight.com/en_US/developer

**Open Source Libraries:**
- python-yeelight: https://github.com/skorokithakis/python-yeelight
- YeelightMatrix (reference implementation): Custom implementation for Matrix devices

**Community Resources:**
- Home Assistant Yeelight Integration: https://www.home-assistant.io/integrations/yeelight/
- Node.js Yeelight Library: https://github.com/samuelthomas2774/node-yeelight

### 7.2 RGB Color Conversion

**Hex to RGB Value:**
```python
def hex_to_rgb_value(hex_color):
    """Convert hex color to RGB integer value."""
    hex_color = hex_color.lstrip("#")
    r = int(hex_color[0:2], 16)
    g = int(hex_color[2:4], 16)
    b = int(hex_color[4:6], 16)
    return r * 65536 + g * 256 + b

# Example
rgb_value = hex_to_rgb_value("#FF0000")  # 16711680
```

**RGB Value to Hex:**
```python
def rgb_value_to_hex(rgb_value):
    """Convert RGB integer value to hex color."""
    r = (rgb_value >> 16) & 0xFF
    g = (rgb_value >> 8) & 0xFF
    b = rgb_value & 0xFF
    return f"#{r:02x}{g:02x}{b:02x}"

# Example
hex_color = rgb_value_to_hex(16711680)  # "#ff0000"
```

**HSV to RGB:**
```python
import colorsys

def hsv_to_rgb(h, s, v):
    """
    Convert HSV to RGB.
    h: 0-359 (degrees)
    s: 0-100 (percentage)
    v: 0-100 (percentage)
    Returns: (r, g, b) where each is 0-255
    """
    h_norm = h / 360.0
    s_norm = s / 100.0
    v_norm = v / 100.0

    r, g, b = colorsys.hsv_to_rgb(h_norm, s_norm, v_norm)

    return int(r * 255), int(g * 255), int(b * 255)
```

### 7.3 Example Color Values

| Color | Hex | RGB Value | Base64 (Matrix) |
|-------|-----|-----------|-----------------|
| Red | `#FF0000` | 16711680 | `/wAA` |
| Green | `#00FF00` | 65280 | `AP8A` |
| Blue | `#0000FF` | 255 | `AAD/` |
| Yellow | `#FFFF00` | 16776960 | `//8A` |
| Cyan | `#00FFFF` | 65535 | `AP//` |
| Magenta | `#FF00FF` | 16711935 | `/wD/` |
| White | `#FFFFFF` | 16777215 | `////` |
| Black (Off) | `#000000` | 0 | `AAAA` |
| Orange | `#FF8000` | 16744448 | `/4AA` |
| Purple | `#8000FF` | 8388863 | `gAD/` |

### 7.4 Color Temperature Reference

| Kelvin | Description | Appearance |
|--------|-------------|------------|
| 1700 | Candle flame | Very warm, orange |
| 2700 | Incandescent bulb | Warm, yellowish |
| 3000 | Warm white | Soft white |
| 3500 | Neutral white | Slightly warm |
| 4000 | Cool white | Natural white |
| 5000 | Daylight | Bright white |
| 6500 | Cool daylight | Bluish white |

### 7.5 Complete Matrix Display Example

Here's a complete, production-ready example for displaying images on a Matrix device:

```python
#!/usr/bin/env python3
"""
Complete Matrix Display Example
Displays images on Yeelight Matrix with error handling and logging.
"""

import base64
import logging
from PIL import Image
from yeelight import Bulb

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class MatrixDisplay:
    """High-level interface for Yeelight Matrix control."""

    def __init__(self, ip, port=55443, num_modules=4, orientation="vertical"):
        """
        Initialize Matrix display.

        Args:
            ip: Device IP address
            port: Device port (default 55443)
            num_modules: Number of 5x5 modules
            orientation: "vertical" or "horizontal"
        """
        self.ip = ip
        self.port = port
        self.num_modules = num_modules
        self.orientation = orientation
        self.bulb = None

    def connect(self):
        """Connect to device and initialize."""
        try:
            logger.info(f"Connecting to {self.ip}:{self.port}")
            self.bulb = Bulb(self.ip, self.port)

            # Enable music mode
            self.bulb.start_music()
            logger.info("Music mode enabled")

            # Activate direct mode
            self.bulb.send_command("activate_fx_mode", [{"mode": "direct"}])
            logger.info("Direct mode activated")

            # Set brightness
            self.bulb.set_brightness(100)
            logger.info("Connected successfully")

        except Exception as e:
            logger.error(f"Connection failed: {e}")
            raise

    def disconnect(self):
        """Disconnect from device."""
        if self.bulb:
            try:
                self.bulb.stop_music()
                logger.info("Disconnected")
            except Exception as e:
                logger.warning(f"Disconnect error: {e}")

    def display_image(self, image_path):
        """
        Load and display image on Matrix.

        Args:
            image_path: Path to image file
        """
        try:
            logger.info(f"Loading image: {image_path}")
            rgb_data = self._load_image(image_path)

            logger.info("Sending to device")
            self.bulb.send_command("update_leds", [rgb_data])

            logger.info("Image displayed successfully")

        except Exception as e:
            logger.error(f"Display error: {e}")
            raise

    def _load_image(self, image_path):
        """Load and encode image for Matrix."""
        # Calculate dimensions
        if self.orientation == "vertical":
            width, height = 5, 5 * self.num_modules
        else:
            width, height = 5 * self.num_modules, 5

        # Load and resize
        img = Image.open(image_path)
        img = img.resize((width, height), Image.Resampling.LANCZOS)
        img = img.convert("RGB")

        # Encode pixels
        rgb_data = ""
        for pixel in img.getdata():
            r, g, b = pixel
            encoded = base64.b64encode(bytes([r, g, b])).decode("ascii")
            rgb_data += encoded

        return rgb_data


def main():
    """Main entry point."""
    # Configuration
    DEVICE_IP = "192.168.0.34"
    IMAGE_PATH = "artwork.png"
    NUM_MODULES = 4

    # Create display
    display = MatrixDisplay(DEVICE_IP, num_modules=NUM_MODULES)

    try:
        # Connect
        display.connect()

        # Display image
        display.display_image(IMAGE_PATH)

        # Keep displayed
        input("Press Enter to exit...")

    except Exception as e:
        logger.error(f"Error: {e}")

    finally:
        # Cleanup
        display.disconnect()


if __name__ == "__main__":
    main()
```

---

## Document Information

**Version:** 1.0
**Last Updated:** 2025-12-16
**Author:** Technical Documentation
**License:** MIT

This document provides a comprehensive reference for the Yeelight protocol with a focus on Matrix/Canvas device control. For questions, issues, or contributions, please refer to the official Yeelight documentation and community resources listed in the Appendix.
