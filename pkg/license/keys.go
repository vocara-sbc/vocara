package license

import (
	"crypto/ed25519"
	"encoding/hex"
)

// defaultPublicKeyHex is the built-in hexadecimal public key for License signature verification, corresponding to the vendor's offline Ed25519 private key.
const defaultPublicKeyHex = "9451922590501aea8b83208ceac0b372acc604d0f9981afd178bb27b41ecc782"

// DefaultPublicKey returns the built-in public key for License signature verification.
func DefaultPublicKey() ed25519.PublicKey {
	raw, err := hex.DecodeString(defaultPublicKeyHex)
	if err != nil {
		panic("license: 内置公钥非法")
	}
	return ed25519.PublicKey(raw)
}

// Redis key conventions for runtime distribution: admin writes these keys, and other services read them during startup and periodic verification.
const (
	// LicenseSourceKey is the distribution key for the original License content.
	LicenseSourceKey = "vocara:license:current"
	// InstallIDSourceKey is the distribution key for the cluster installation identifier.
	InstallIDSourceKey = "vocara:license:install_id"
	// LicenseChangedChannel is the authorization change notification channel: admin publishes after importing or reloading, and other services subscribe to verify immediately.
	LicenseChangedChannel = "vocara:license:changed"
)
