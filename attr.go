package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

func attr(line, key string) string {
	line = strings.TrimSpace(line)
	if key == "" {
		if i := strings.IndexByte(line, ':'); i >= 0 {
			return line[i+1:]
		}
		return ""
	}
	quotedPrefix := key + "=\""
	if i := strings.Index(line, quotedPrefix); i >= 0 {
		rest := line[i+len(quotedPrefix):]
		end := strings.IndexByte(rest, '"')
		if end < 0 {
			return rest
		}
		// long: 上游 ParserUtil.GetAttribute 会先查找带引号的 key，再查找裸 key；当 KEYFORMATURI=foo,URI="..." 同行出现时，真正的 quoted URI 应优先于前面的后缀属性。
		return rest[:end]
	}
	prefix := key + "="
	i := strings.Index(line, prefix)
	if i < 0 {
		return ""
	}
	rest := line[i+len(prefix):]
	if strings.HasPrefix(rest, "\"") {
		rest = rest[1:]
		end := strings.IndexByte(rest, '"')
		if end < 0 {
			return rest
		}
		return rest[:end]
	}
	end := strings.IndexByte(rest, ',')
	if end < 0 {
		return rest
	}
	return rest[:end]
}

func attrExists(line, key string) bool {
	prefix := key + "="
	for start := 0; start < len(line); {
		i := strings.Index(line[start:], prefix)
		if i < 0 {
			return false
		}
		idx := start + i
		if idx == 0 || line[idx-1] == ':' || line[idx-1] == ',' {
			return true
		}
		start = idx + len(prefix)
	}
	return false
}

func attrExistsLoose(line, key string) bool {
	return strings.Contains(strings.TrimSpace(line), key+"=")
}

func malformedQuotedAttr(line, key string) bool {
	prefix := key + "=\""
	i := strings.Index(line, prefix)
	if i < 0 {
		return false
	}
	rest := line[i+len(prefix):]
	return !strings.Contains(rest, "\"")
}

func ensureQuotedAttrsClosed(line string, keys ...string) error {
	for _, key := range keys {
		if malformedQuotedAttr(line, key) {
			// long: 原版 ParserUtil.GetAttribute 在被读取属性缺少闭合引号时会切片越界并中断；这里显式报错，避免坏 Master 被继续解析。
			return fmt.Errorf("%s quoted attribute is not closed", key)
		}
	}
	return nil
}

func parseByteRange(input string) (length int64, start *int64, err error) {
	parts := strings.Split(input, "@")
	if len(parts) > 2 {
		// long: 上游 GetRange 对多个 @ 的 BYTERANGE 直接返回 0/null；不能默默取第二段，否则会下载错误字节窗口。
		return 0, nil, nil
	}
	length, err = strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return 0, nil, err
	}
	if len(parts) > 1 {
		v, e := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if e != nil {
			return 0, nil, e
		}
		start = &v
	}
	return length, start, nil
}

func combineURL(baseURL, ref string) string {
	if strings.TrimSpace(baseURL) == "" {
		return ref
	}
	b, err := url.Parse(baseURL)
	if err != nil {
		return ref
	}
	r, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return b.ResolveReference(r).String()
}

func appendURLParams(target, source string) (string, error) {
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		return target, nil
	}
	tu, err := url.Parse(target)
	if err != nil {
		return target, err
	}
	su, err := url.Parse(source)
	if err != nil {
		return target, err
	}
	targetQuery := parseOrderedQuery(tu.RawQuery)
	sourceQuery := parseOrderedQuery(su.RawQuery)
	for _, sourceEntry := range sourceQuery {
		joined := strings.Join(sourceEntry.values, ",")
		if targetEntry := targetQuery.findByKindAndKey(sourceEntry.kind, sourceEntry.key); targetEntry != nil {
			// long: 上游 NameValueCollection.Set 会在原 key 位置替换，并把重复源参数通过 Get 合成逗号字符串。
			targetEntry.values = []string{joined}
		} else {
			targetQuery = append(targetQuery, orderedQueryEntry{kind: sourceEntry.kind, key: sourceEntry.key, values: []string{joined}})
		}
	}
	encoded := targetQuery.encode()
	if encoded == "" {
		return target, nil
	}
	out := *tu
	out.RawQuery = encoded
	out.Fragment = ""
	return out.String(), nil
}

type orderedQueryEntry struct {
	kind   queryKeyKind
	key    string
	values []string
}

type orderedQuery []orderedQueryEntry

type queryKeyKind int

const (
	queryKeyNormal queryKeyKind = iota
	queryKeyMissing
	queryKeyEmpty
)

func parseOrderedQuery(raw string) orderedQuery {
	if raw == "" {
		return nil
	}
	var out orderedQuery
	for _, part := range strings.Split(raw, "&") {
		key, value, hasEqual := strings.Cut(part, "=")
		kind := queryKeyNormal
		if !hasEqual {
			kind = queryKeyMissing
			value = key
			key = ""
		} else if key == "" {
			kind = queryKeyEmpty
		}
		decodedKey, err := url.QueryUnescape(key)
		if err != nil {
			decodedKey = key
		}
		decodedValue, err := url.QueryUnescape(value)
		if err != nil {
			decodedValue = value
		}
		if entry := out.findByKindAndKey(kind, decodedKey); entry != nil {
			entry.values = append(entry.values, decodedValue)
			continue
		}
		out = append(out, orderedQueryEntry{kind: kind, key: decodedKey, values: []string{decodedValue}})
	}
	return out
}

func (q orderedQuery) findByKindAndKey(kind queryKeyKind, key string) *orderedQueryEntry {
	for i := range q {
		if q[i].kind == kind && q[i].key == key {
			return &q[i]
		}
	}
	return nil
}

func (q orderedQuery) encode() string {
	parts := make([]string, 0, len(q))
	for _, entry := range q {
		value := strings.Join(entry.values, ",")
		if entry.kind == queryKeyMissing || entry.kind == queryKeyEmpty {
			parts = append(parts, queryEscapeLower(value))
			continue
		}
		parts = append(parts, queryEscapeLower(entry.key)+"="+queryEscapeLower(value))
	}
	return strings.Join(parts, "&")
}

func queryEscapeLower(value string) string {
	escaped := url.QueryEscape(value)
	var b strings.Builder
	b.Grow(len(escaped))
	for i := 0; i < len(escaped); i++ {
		if escaped[i] == '%' && i+2 < len(escaped) {
			b.WriteByte('%')
			b.WriteByte(byte(strings.ToLower(string(escaped[i+1]))[0]))
			b.WriteByte(byte(strings.ToLower(string(escaped[i+2]))[0]))
			i += 2
			continue
		}
		b.WriteByte(escaped[i])
	}
	return b.String()
}

func preProcessHLSContent(content, m3u8URL string) string {
	content = strings.TrimSpace(content)
	if strings.Contains(content, "\r") && !strings.Contains(content, "\n") {
		content = strings.ReplaceAll(content, "\r", "\n")
	}
	if strings.Contains(m3u8URL, "tlivecloud-playback-cdn.ysp.cctv.cn") && strings.Contains(m3u8URL, "endtime=") {
		// long: 原版对 YSP 回放会无条件补 ENDLIST，即使源内容已经带有结束标记；这里保留重复标记以贴近上游预处理输出。
		content += "\n#EXT-X-ENDLIST"
	}
	if strings.Contains(content, "#EXT-X-DISCONTINUITY") && strings.Contains(content, "#EXT-X-MAP") && strings.Contains(content, "ott.cibntv.net") && strings.Contains(content, "ccode=") {
		yk := regexp.MustCompile(`#EXT-X-DISCONTINUITY\s+#EXT-X-MAP:URI="(.*?)",BYTERANGE="(.*?)"`)
		content = yk.ReplaceAllString(content, "#EXTINF:0.000000,\n#EXT-X-BYTERANGE:$2\n$1")
	}
	if strings.Contains(content, "#EXT-X-DISCONTINUITY") && strings.Contains(content, "#EXT-X-MAP") && strings.Contains(m3u8URL, "media.dssott.com/") {
		dnsp := regexp.MustCompile(`#EXT-X-MAP:URI=".*?BUMPER/[\s\S]+?#EXT-X-DISCONTINUITY`)
		content = replaceFirstRegexp(content, dnsp, "#XXX")
	}
	if strings.Contains(content, "#EXT-X-DISCONTINUITY") && strings.Contains(content, "seg_00000.vtt") && strings.Contains(m3u8URL, "media.dssott.com/") {
		dnspSub := regexp.MustCompile(`#EXTINF:.*?,\s+.*BUMPER.*\s+?#EXT-X-DISCONTINUITY`)
		content = replaceFirstRegexp(content, dnspSub, "#XXX")
	}
	if strings.Contains(content, "#EXT-X-DISCONTINUITY") && strings.Contains(content, "#EXT-X-MAP") && (strings.Contains(m3u8URL, ".apple.com/") || regexp.MustCompile(`#EXT-X-MAP.*\.apple\.com/`).MatchString(content)) {
		atv := regexp.MustCompile(`(#EXT-X-KEY:[\s\S]*?)(#EXT-X-DISCONTINUITY|#EXT-X-ENDLIST)`)
		if m := atv.FindStringSubmatch(content); len(m) > 1 {
			// long: 原版 AppleTV 裁剪分支显式插入 CRLF，raw.m3u8 落盘和后续逐行读取都应保留这个混合换行形态。
			content = "#EXTM3U\r\n" + m[1] + "\r\n#EXT-X-ENDLIST"
		}
	}
	// long: 上游先完成 YK/Disney/AppleTV 等站点修正，最后才修复 KEY/EXTINF 顺序；顺序不同会改变 AppleTV 裁剪后的内容。
	re := regexp.MustCompile(`(#EXTINF[^\n\r]*)(\s+)(#EXT-X-KEY[^\n\r]*)`)
	content = re.ReplaceAllString(content, "$3$2$1")
	return content
}

func replaceFirstRegexp(input string, re *regexp.Regexp, replacement string) string {
	match := re.FindStringSubmatchIndex(input)
	if match == nil {
		return input
	}
	out := make([]byte, 0, len(input))
	out = append(out, input[:match[0]]...)
	// long: 上游 Disney+ 修正规则只替换第一个 Match，后续 BUMPER 块应保持原样，避免把原版不会动的内容一并删掉。
	out = re.ExpandString(out, replacement, input, match)
	out = append(out, input[match[1]:]...)
	return string(out)
}

func splitComplex(input string) map[string]string {
	out := map[string]string{}
	for _, part := range splitRespectQuotes(input, ':') {
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			out[part] = part
			continue
		}
		out[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), "\"'")
	}
	return out
}

func splitRespectQuotes(input string, sep rune) []string {
	var res []string
	var b strings.Builder
	var quote rune
	for _, r := range input {
		if r == '"' || r == '\'' {
			if quote == 0 {
				quote = r
			} else if quote == r {
				quote = 0
			}
			b.WriteRune(r)
			continue
		}
		if r == sep && quote == 0 {
			current := b.String()
			if strings.HasSuffix(current, `\`) {
				// long: 上游复杂参数允许用 \: 表示值里的冒号，例如外部轨道标题或工具路径，不能在这里误拆成下一个参数。
				b.Reset()
				b.WriteString(strings.TrimSuffix(current, `\`))
				b.WriteRune(r)
				continue
			}
			res = append(res, b.String())
			b.Reset()
			continue
		}
		b.WriteRune(r)
	}
	res = append(res, b.String())
	return res
}
