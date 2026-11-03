package license

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// FingerprintCollection contains fingerprint collection results: raw items and their combined hash.
type FingerprintCollection struct {
	// Items contains the collected raw hardware items.
	Items map[string]string `json:"items"`
	// Hash is the SHA-256 of all combined items, used as the machine code for License binding.
	Hash string `json:"hash"`
}

// FingerprintRequest is a fingerprint request file submitted by the customer to the vendor for License issuance.
type FingerprintRequest struct {
	// Customer is the customer name.
	Customer string `json:"customer"`
	// IssuedAt is the collection time.
	IssuedAt time.Time `json:"issued_at"`
	// Items contains raw hardware items for verification by the issuer.
	Items map[string]string `json:"items"`
	// Hash is the combined machine-code hash.
	Hash string `json:"hash"`
}

// CollectFingerprint collects the local fingerprint by combining cross-platform base items and platform-specific items with SHA-256.
func CollectFingerprint() FingerprintCollection {
	items := map[string]string{}
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		items["hostname"] = hostname
	}
	items["cpu_cores"] = strconv.Itoa(runtime.NumCPU())
	if mac := firstPhysicalMAC(); mac != "" {
		items["mac"] = mac
	}
	for key, value := range collectPlatformItems() {
		if value != "" {
			items[key] = value
		}
	}
	return FingerprintCollection{Items: items, Hash: hashItems(items)}
}

// firstPhysicalMAC returns the physical address of the first active, non-loopback network interface.
func firstPhysicalMAC() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		if len(iface.HardwareAddr) == 0 {
			continue
		}
		return iface.HardwareAddr.String()
	}
	return ""
}

// hashItems computes SHA-256 over hardware items concatenated in order, ensuring that identical input produces the same machine code.
func hashItems(items map[string]string) string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(items[key])
		builder.WriteString(";")
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}
