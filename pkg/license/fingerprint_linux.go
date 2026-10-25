//go:build linux

package license

import (
	"os"
	"strings"
)

// collectPlatformItems 采集 Linux 平台硬件项。
func collectPlatformItems() map[string]string {
	items := map[string]string{}
	if raw, err := os.ReadFile("/sys/class/dmi/id/product_uuid"); err == nil {
		items["product_uuid"] = strings.TrimSpace(string(raw))
	}
	if raw, err := os.ReadFile("/sys/class/dmi/id/board_serial"); err == nil {
		items["board_serial"] = strings.TrimSpace(string(raw))
	}
	if model := readCPUModel(); model != "" {
		items["cpu_model"] = model
	}
	return items
}

// readCPUModel 从 /proc/cpuinfo 读取首个 CPU 型号。
func readCPUModel() string {
	raw, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "model name") {
			if idx := strings.Index(line, ":"); idx >= 0 {
				return strings.TrimSpace(line[idx+1:])
			}
		}
	}
	return ""
}
