package main

import "strings"

func normalizeEncryptMethod(input string) EncryptMethod {
	normalized := strings.ToUpper(strings.TrimSpace(input))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	switch normalized {
	case "":
		return ""
	case string(EncryptAES128):
		return EncryptAES128
	case string(EncryptAES128ECB):
		return EncryptAES128ECB
	case string(EncryptCENC):
		return EncryptCENC
	case string(EncryptSampleAES):
		return EncryptSampleAES
	case string(EncryptSampleCTR):
		return EncryptSampleCTR
	case string(EncryptChaCha20):
		return EncryptChaCha20
	case string(EncryptNone):
		return EncryptNone
	case string(EncryptUnknown):
		return EncryptUnknown
	default:
		return EncryptMethod(normalized)
	}
}
