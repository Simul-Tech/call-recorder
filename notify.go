package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func notify(title, body string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("notify-send", "-a", "call-recorder", title, body)
	case "darwin":
		t := strings.ReplaceAll(title, `"`, `\"`)
		b := strings.ReplaceAll(body, `"`, `\"`)
		cmd = exec.Command("osascript", "-e",
			fmt.Sprintf(`display notification "%s" with title "%s"`, b, t))
	case "windows":
		t := strings.ReplaceAll(title, "'", "''")
		b := strings.ReplaceAll(body, "'", "''")
		ps := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$n = New-Object System.Windows.Forms.NotifyIcon
$n.Icon = [System.Drawing.SystemIcons]::Application
$n.Visible = $true
$n.BalloonTipTitle = '%s'
$n.BalloonTipText = '%s'
$n.BalloonTipIcon = 'Info'
$n.ShowBalloonTip(5000)
Start-Sleep 2
$n.Dispose()`, t, b)
		cmd = exec.Command("powershell", "-WindowStyle", "Hidden", "-NonInteractive", "-Command", ps)
	default:
		return
	}
	_ = cmd.Start()
}
