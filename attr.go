package main

import (
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
	tq := tu.Query()
	for k, vals := range su.Query() {
		tq.Del(k)
		for _, v := range vals {
			tq.Add(k, v)
		}
	}
	tu.RawQuery = tq.Encode()
	return tu.String(), nil
}

func preProcessHLSContent(content, m3u8URL string) string {
	content = strings.TrimSpace(content)
	if strings.Contains(content, "\r") && !strings.Contains(content, "\n") {
		content = strings.ReplaceAll(content, "\r", "\n")
	}
	if strings.Contains(m3u8URL, "tlivecloud-playback-cdn.ysp.cctv.cn") && strings.Contains(m3u8URL, "endtime=") && !strings.Contains(content, "#EXT-X-ENDLIST") {
		content += "\n#EXT-X-ENDLIST"
	}
	// long: 少数站点会把 EXT-X-KEY 放在 EXTINF 后面，提前调正顺序才能让后续分片继承正确密钥。
	re := regexp.MustCompile(`(?m)(#EXTINF[^\n\r]*)([\r\n]+)(#EXT-X-KEY[^\n\r]*)`)
	content = re.ReplaceAllString(content, "$3$2$1")
	if strings.Contains(content, "#EXT-X-DISCONTINUITY") && strings.Contains(content, "#EXT-X-MAP") && strings.Contains(content, "ott.cibntv.net") && strings.Contains(content, "ccode=") {
		yk := regexp.MustCompile(`#EXT-X-DISCONTINUITY\s+#EXT-X-MAP:URI="(.*?)",BYTERANGE="(.*?)"`)
		content = yk.ReplaceAllString(content, "#EXTINF:0.000000,\n#EXT-X-BYTERANGE:$2\n$1")
	}
	if strings.Contains(content, "#EXT-X-DISCONTINUITY") && strings.Contains(content, "#EXT-X-MAP") && strings.Contains(m3u8URL, "media.dssott.com/") {
		dnsp := regexp.MustCompile(`#EXT-X-MAP:URI=".*?BUMPER/[\s\S]+?#EXT-X-DISCONTINUITY`)
		content = dnsp.ReplaceAllString(content, "#XXX")
	}
	if strings.Contains(content, "#EXT-X-DISCONTINUITY") && strings.Contains(content, "seg_00000.vtt") && strings.Contains(m3u8URL, "media.dssott.com/") {
		dnspSub := regexp.MustCompile(`#EXTINF:.*?,\s+.*BUMPER.*\s+?#EXT-X-DISCONTINUITY`)
		content = dnspSub.ReplaceAllString(content, "#XXX")
	}
	if strings.Contains(content, "#EXT-X-DISCONTINUITY") && strings.Contains(content, "#EXT-X-MAP") && (strings.Contains(m3u8URL, ".apple.com/") || regexp.MustCompile(`#EXT-X-MAP.*\.apple\.com/`).MatchString(content)) {
		atv := regexp.MustCompile(`(#EXT-X-KEY:[\s\S]*?)(#EXT-X-DISCONTINUITY|#EXT-X-ENDLIST)`)
		if m := atv.FindStringSubmatch(content); len(m) > 1 {
			content = "#EXTM3U\n" + m[1] + "\n#EXT-X-ENDLIST"
		}
	}
	return content
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
