package main

import "strings"

type languageInfo struct {
	Code        string
	Description string
	AudioName   string
}

var (
	languageAliases = buildLanguageLookup()
	languageCodeMap = buildLanguageCodeMap()
)

func normalizeOutputFileLanguage(file outputFile) outputFile {
	code, name := normalizeLanguageAndName(file.Language, file.Name, file.MediaType)
	file.Language = code
	file.Name = name
	return file
}

func normalizeLanguageAndName(langCode, description string, mediaType *MediaType) (string, string) {
	original := strings.TrimSpace(langCode)
	if original == "" {
		return langCode, description
	}
	key := strings.ToLower(original)
	info, ok := languageAliases[key]
	if !ok {
		base := strings.Split(key, "-")[0]
		if code := languageCodeMap[base]; code != "" {
			info, ok = languageAliases[strings.ToLower(code)]
		}
	}
	if !ok {
		info = languageInfo{Code: "und"}
	}
	if description == "" {
		// long: 上游只在用户没有提供轨道名称时补默认语言名，保留显式 name 可避免覆盖用户在 m3u8 或 --mux-import 中的描述。
		if mediaType != nil && *mediaType == MediaSubtitles {
			description = info.Description
		} else {
			description = info.AudioName
		}
		if description == "" {
			description = original
		}
	}
	return info.Code, description
}

func buildLanguageLookup() map[string]languageInfo {
	out := map[string]languageInfo{}
	for _, line := range strings.Split(upstreamLanguageList, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ";")
		if len(parts) != 4 {
			continue
		}
		info := languageInfo{
			Code:        strings.TrimSpace(parts[1]),
			Description: strings.TrimSpace(parts[2]),
			AudioName:   strings.TrimSpace(parts[3]),
		}
		for _, token := range []string{parts[0], parts[1]} {
			key := strings.ToLower(strings.TrimSpace(token))
			if key == "" {
				continue
			}
			if _, exists := out[key]; !exists {
				// long: 上游用 FirstOrDefault 匹配语言表；保留第一次出现的映射，才能复现 chi、zho 这类重复代码的标题选择。
				out[key] = info
			}
		}
	}
	return out
}

func buildLanguageCodeMap() map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(upstreamLanguageCodeMap, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ";")
		if len(parts) != 2 {
			continue
		}
		out[strings.ToLower(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])
	}
	return out
}
