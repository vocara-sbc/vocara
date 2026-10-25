//go:build windows

package license

import (
	"os/exec"
	"strings"
)

// collectPlatformItems 通过 PowerShell（系统自带）采集 Windows 硬件项，无额外依赖。
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

// runPowerShell 执行单条 PowerShell 取值命令并返回去除空白的结果。
func runPowerShell(script string) string {
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
