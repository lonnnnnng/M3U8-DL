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

func parseHLSPlaylistEncryptMethod(input string) EncryptMethod {
	normalized := strings.ReplaceAll(input, "-", "_")
	switch normalized {
	case "NONE":
		return EncryptNone
	case "AES_128":
		return EncryptAES128
	case "AES_128_ECB":
		return EncryptAES128ECB
	case "CENC":
		return EncryptCENC
	case "SAMPLE_AES":
		return EncryptSampleAES
	case "SAMPLE_AES_CTR":
		return EncryptSampleCTR
	case "CHACHA20":
		return EncryptChaCha20
	case "UNKNOWN":
		return EncryptUnknown
	default:
		// long: HLS 清单里的 METHOD 走上游 EncryptInfo.ParseMethod，Enum.TryParse 默认大小写敏感且不会修剪属性值；小写或带空格都会降级 UNKNOWN。
		return EncryptUnknown
	}
}
