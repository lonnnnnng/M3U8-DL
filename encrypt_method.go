package main

import (
	"strconv"
	"strings"
)

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
	normalized := strings.ReplaceAll(strings.TrimSpace(input), "-", "_")
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
		if isUnsignedDecimal(normalized) {
			if n, err := strconv.Atoi(normalized); err == nil {
				switch n {
				case 0:
					return EncryptNone
				case 1:
					return EncryptAES128
				case 2:
					return EncryptAES128ECB
				case 3:
					return EncryptSampleAES
				case 4:
					return EncryptSampleCTR
				case 5:
					return EncryptCENC
				case 6:
					return EncryptChaCha20
				case 7:
					return EncryptUnknown
				default:
					// long: C# Enum.TryParse 会接受未定义的非负数字枚举值并保留为数字；这类 METHOD 不等同于 UNKNOWN，下载阶段也不会误走已知解密分支。
					return EncryptMethod(strconv.Itoa(n))
				}
			}
		}
		// long: HLS 清单里的 METHOD 走上游 EncryptInfo.ParseMethod，Enum.TryParse 默认大小写敏感；小写 aes-128 会降级 UNKNOWN。
		return EncryptUnknown
	}
}

func isUnsignedDecimal(input string) bool {
	if input == "" {
		return false
	}
	for _, r := range input {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
