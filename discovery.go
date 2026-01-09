package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

type DeviceInfo struct {
	Location  string
	ID        string
	Model     string
	FwVer     string
	Support   string
	Power     string
	Bright    string
	ColorMode string
	CT        string
	RGB       string
	Hue       string
	Sat       string
	Name      string
}

const (
	multicastAddr = "239.255.255.250:1982"
	searchMessage = "M-SEARCH * HTTP/1.1\r\n" +
		"HOST: 239.255.255.250:1982\r\n" +
		"MAN: \"ssdp:discover\"\r\n" +
		"ST: wifi_bulb\r\n" +
		"\r\n"
)

func DiscoverDevices() ([]*DeviceInfo, error) {
	addr, err := net.ResolveUDPAddr("udp4", multicastAddr)
	if err != nil {
		return nil, fmt.Errorf("error resolving address: %w", err)
	}

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, fmt.Errorf("error creating UDP connection: %w", err)
	}
	defer conn.Close()

	if _, err = conn.WriteToUDP([]byte(searchMessage), addr); err != nil {
		return nil, fmt.Errorf("error sending search request: %w", err)
	}

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	buffer := make([]byte, 2048)
	discoveredDevices := make(map[string]*DeviceInfo)

	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			continue
		}

		deviceInfo := parseDeviceInfo(string(buffer[:n]))
		if discoveredDevices[deviceInfo.Location] == nil {
			discoveredDevices[deviceInfo.Location] = deviceInfo
		}

		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	}

	devices := make([]*DeviceInfo, 0, len(discoveredDevices))
	for _, device := range discoveredDevices {
		if device.Model == "CubeLite" {
			devices = append(devices, device)
		}
	}

	return devices, nil
}

func parseDeviceInfo(response string) *DeviceInfo {
	device := &DeviceInfo{}
	scanner := bufio.NewScanner(strings.NewReader(response))

	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 2)
		if len(parts) != 2 {
			continue
		}

		header := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch header {
		case "location":
			device.Location = value
		case "id":
			device.ID = value
		case "model":
			device.Model = value
		case "fw_ver":
			device.FwVer = value
		case "support":
			device.Support = value
		case "power":
			device.Power = value
		case "bright":
			device.Bright = value
		case "color_mode":
			device.ColorMode = value
		case "ct":
			device.CT = value
		case "rgb":
			device.RGB = value
		case "hue":
			device.Hue = value
		case "sat":
			device.Sat = value
		case "name":
			device.Name = value
		}
	}

	return device
}
