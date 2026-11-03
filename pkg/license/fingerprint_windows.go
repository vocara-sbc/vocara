//go:build windows

package license

import (
	"os/exec"
	"strings"
)

// collectPlatformItems collects Windows hardware items through the built-in PowerShell without additional dependencies.
func collectPlatformItems() map[string]string {
	items := map[string]string{}
	if value := runPowerShell("(Get-CimInstance Win32_ComputerSystemProduct).UUID"); value != "" {
		items["product_uuid"] = value
	}
	if value := runPowerShell("(Get-CimInstance Win32_BaseBoard).SerialNumber"); value != "" {
		items["board_serial"] = value
	}
	if value := runPowerShell("(Get-CimInstance Win32_Processor).Name"); value != "" {
		items["cpu_model"] = value
	}
	return items
}

// runPowerShell executes a single PowerShell value command and returns the result with whitespace removed.
func runPowerShell(script string) string {
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
