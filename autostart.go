//go:build !notray

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

func runAutostart(subcmd string, args []string) {
	var err error
	switch subcmd {
	case "enable":
		err = autostartEnable(args)
	case "disable":
		err = autostartDisable()
	case "status":
		autostartStatus()
		return
	default:
		fmt.Fprintln(os.Stderr, "Uso: call-recorder autostart <enable|disable|status> [opzioni tray]")
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "autostart: %v\n", err)
		os.Exit(1)
	}
}

// ── Enable ────────────────────────────────────────────────────────────────────

func autostartEnable(extraArgs []string) error {
	binary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("impossibile trovare il percorso del binario: %w", err)
	}
	switch runtime.GOOS {
	case "linux":
		return enableLinux(binary, extraArgs)
	case "darwin":
		return enableDarwin(binary, extraArgs)
	case "windows":
		return enableWindows(binary, extraArgs)
	default:
		return fmt.Errorf("autostart non supportato su %s", runtime.GOOS)
	}
}

// ── Disable ───────────────────────────────────────────────────────────────────

func autostartDisable() error {
	switch runtime.GOOS {
	case "linux":
		return disableLinux()
	case "darwin":
		return disableDarwin()
	case "windows":
		return disableWindows()
	default:
		return fmt.Errorf("autostart non supportato su %s", runtime.GOOS)
	}
}

// ── Status ────────────────────────────────────────────────────────────────────

func autostartStatus() {
	var enabled bool
	var detail string
	switch runtime.GOOS {
	case "linux":
		p := xdgDesktopPath()
		if _, err := os.Stat(p); err == nil {
			enabled, detail = true, p
		}
	case "darwin":
		p := launchAgentPath()
		if _, err := os.Stat(p); err == nil {
			enabled, detail = true, p
		}
	case "windows":
		out, err := exec.Command("reg", "query",
			`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
			"/v", "call-recorder").Output()
		if err == nil {
			enabled, detail = true, strings.TrimSpace(string(out))
		}
	}
	if enabled {
		fmt.Println("✓ Autostart abilitato:", detail)
	} else {
		fmt.Println("✗ Autostart non abilitato")
	}
}

// ── Linux (XDG autostart) ─────────────────────────────────────────────────────

func xdgDesktopPath() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "autostart", "call-recorder.desktop")
}

func enableLinux(binary string, args []string) error {
	path := xdgDesktopPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	cmdLine := binary + " tray"
	if len(args) > 0 {
		cmdLine += " " + strings.Join(args, " ")
	}
	content := "[Desktop Entry]\nType=Application\nName=call-recorder\nExec=" +
		cmdLine + "\nHidden=false\nNoDisplay=false\nX-GNOME-Autostart-enabled=true\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	fmt.Println("✓ Autostart abilitato:", path)
	return nil
}

func disableLinux() error {
	path := xdgDesktopPath()
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Autostart non era abilitato.")
			return nil
		}
		return err
	}
	fmt.Println("✓ Autostart disabilitato.")
	return nil
}

// ── macOS (LaunchAgent) ───────────────────────────────────────────────────────

const launchAgentTmpl = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>it.simultech.call-recorder</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{.Binary}}</string>
		<string>tray</string>{{range .Args}}
		<string>{{.}}</string>{{end}}
	</array>{{if .APIKey}}
	<key>EnvironmentVariables</key>
	<dict>
		<key>OPENAI_API_KEY</key>
		<string>{{.APIKey}}</string>
	</dict>{{end}}
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<false/>
</dict>
</plist>
`

func launchAgentPath() string {
	return filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "it.simultech.call-recorder.plist")
}

func enableDarwin(binary string, args []string) error {
	path := launchAgentPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data := struct {
		Binary string
		Args   []string
		APIKey string
	}{binary, args, os.Getenv("OPENAI_API_KEY")}

	tmpl, err := template.New("plist").Parse(launchAgentTmpl)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return err
	}
	exec.Command("launchctl", "load", path).Run()
	fmt.Println("✓ Autostart abilitato:", path)
	if data.APIKey != "" {
		fmt.Println("  OPENAI_API_KEY inclusa nel plist")
	}
	return nil
}

func disableDarwin() error {
	path := launchAgentPath()
	exec.Command("launchctl", "unload", path).Run()
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Autostart non era abilitato.")
			return nil
		}
		return err
	}
	fmt.Println("✓ Autostart disabilitato.")
	return nil
}

// ── Windows (Registry) ────────────────────────────────────────────────────────

func enableWindows(binary string, args []string) error {
	cmdLine := `"` + binary + `" tray`
	if len(args) > 0 {
		cmdLine += " " + strings.Join(args, " ")
	}
	out, err := exec.Command("reg", "add",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
		"/v", "call-recorder", "/t", "REG_SZ", "/d", cmdLine, "/f",
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("reg add: %w\n%s", err, out)
	}
	fmt.Println("✓ Autostart abilitato nel registro di sistema")
	return nil
}

func disableWindows() error {
	out, err := exec.Command("reg", "delete",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
		"/v", "call-recorder", "/f",
	).CombinedOutput()
	if err != nil {
		s := string(out)
		if strings.Contains(s, "non trovato") || strings.Contains(s, "not found") {
			fmt.Println("Autostart non era abilitato.")
			return nil
		}
		return fmt.Errorf("reg delete: %w\n%s", err, out)
	}
	fmt.Println("✓ Autostart disabilitato.")
	return nil
}
