package main

import (
	"fmt"
	"strings"

	"github.com/gen2brain/malgo"
)

func listDevices(ctx *malgo.AllocatedContext) error {
	captures, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return err
	}
	playbacks, err := ctx.Devices(malgo.Playback)
	if err != nil {
		return err
	}

	fmt.Println("=== INPUT DEVICES (microphone + loopback candidates) ===")
	for i, d := range captures {
		tag := ""
		if isMonitorDevice(d.Name()) {
			tag = " [MONITOR/LOOPBACK ✓]"
		}
		def := ""
		if d.IsDefault != 0 {
			def = " [default]"
		}
		fmt.Printf("  [%d] %s%s%s\n", i, d.Name(), tag, def)
	}

	fmt.Println()
	fmt.Println("=== OUTPUT DEVICES ===")
	for i, d := range playbacks {
		def := ""
		if d.IsDefault != 0 {
			def = " [default]"
		}
		fmt.Printf("  [%d] %s%s\n", i, d.Name(), def)
	}

	fmt.Println()
	fmt.Println("TIPS:")
	fmt.Println("  Linux:   cerca un device con [MONITOR/LOOPBACK ✓] per l'audio di sistema")
	fmt.Println("           oppure: pactl list sources short")
	fmt.Println("  Windows: abilita 'Stereo Mix' oppure installa VB-Audio Cable")
	fmt.Println("  macOS:   installa BlackHole: https://github.com/ExistentialAudio/BlackHole")
	return nil
}

func isMonitorDevice(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, ".monitor") ||
		strings.Contains(lower, "monitor of") ||
		strings.Contains(lower, "loopback") ||
		strings.Contains(lower, "stereo mix") ||
		strings.Contains(lower, "what u hear") ||
		strings.Contains(lower, "blackhole") ||
		strings.Contains(lower, "soundflower") ||
		strings.Contains(lower, "virtual cable") ||
		strings.Contains(lower, "cable output")
}

func findCaptureDevice(ctx *malgo.AllocatedContext, partial string) (*malgo.DeviceInfo, error) {
	devices, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return nil, err
	}
	lower := strings.ToLower(partial)
	for i := range devices {
		if strings.Contains(strings.ToLower(devices[i].Name()), lower) {
			return &devices[i], nil
		}
	}
	return nil, fmt.Errorf("device not found: %q (run 'call-recorder list' to see available devices)", partial)
}

func autoDetectLoopback(ctx *malgo.AllocatedContext) (*malgo.DeviceInfo, error) {
	captures, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return nil, err
	}
	playbacks, err := ctx.Devices(malgo.Playback)
	if err != nil {
		return nil, err
	}

	// Find the default output device name.
	var defaultOutName string
	for i := range playbacks {
		if playbacks[i].IsDefault != 0 {
			defaultOutName = strings.ToLower(playbacks[i].Name())
			break
		}
	}

	var fallback *malgo.DeviceInfo

	for i := range captures {
		name := captures[i].Name()
		if !isMonitorDevice(name) {
			continue
		}
		lower := strings.ToLower(name)

		// Best match: monitor of the default output.
		if defaultOutName != "" && strings.Contains(lower, defaultOutName) {
			return &captures[i], nil
		}

		// Skip HDMI/DisplayPort monitors — almost always inactive.
		if isPassiveMonitor(lower) {
			continue
		}

		if fallback == nil {
			fallback = &captures[i]
		}
	}

	return fallback, nil
}

// isPassiveMonitor returns true for monitor sources that are typically
// attached to inactive outputs (HDMI, DisplayPort, S/PDIF).
func isPassiveMonitor(lowerName string) bool {
	passive := []string{"hdmi", "displayport", "dp ", "s/pdif", "spdif", "optical"}
	for _, kw := range passive {
		if strings.Contains(lowerName, kw) {
			return true
		}
	}
	return false
}
