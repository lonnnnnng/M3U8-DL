package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type parser struct {
	opt         Options
	client      *http.Client
	originalURL string
	currentURL  string
	baseURL     string
	master      bool
	rawFiles    map[string]string
}

const hlsKeyRetryCount = 3
const maxHLSLineSize = 16 * 1024 * 1024

func newHLSScanner(raw string) *bufio.Scanner {
	sc := bufio.NewScanner(strings.NewReader(raw))
	// long: HLS 里常见带长签名的分片 URL 或 data URI key；上游逐行读取没有 64KB 限制，Go 版必须显式放大 Scanner 缓冲。
	sc.Buffer(make([]byte, 1024), maxHLSLineSize)
	return sc
}

func parseSource(ctx context.Context, client *http.Client, opt Options) ([]StreamSpec, *parser, error) {
	raw, finalURL, err := fetchText(ctx, client, opt.Input, opt.Headers)
	if err != nil {
		return nil, nil, err
	}
	p := &parser{opt: opt, client: client, originalURL: opt.Input, currentURL: finalURL, baseURL: finalURL, rawFiles: map[string]string{}}
	if opt.BaseURL != "" {
		p.baseURL = opt.BaseURL
	}
	return p.extract(ctx, raw)
}

func (p *parser) extract(ctx context.Context, raw string) ([]StreamSpec, *parser, error) {
	raw = strings.TrimSpace(raw)
	raw = preProcessHLSContent(raw, p.currentURL)
	if !strings.HasPrefix(raw, "#EXTM3U") {
		return nil, p, fmt.Errorf("当前 Go 版只支持 HLS m3u8")
	}
	if strings.Contains(raw, "#EXT-X-STREAM-INF") {
		p.master = true
		streams, err := p.parseMaster(raw)
		if err != nil {
			return nil, p, err
		}
		return distinctStreamsByURL(streams), p, nil
	}
	pl, err := p.parseMedia(ctx, raw)
	if err != nil {
		return nil, p, err
	}
	ext := "ts"
	if pl.MediaInit != nil {
		ext = "mp4"
	}
	return []StreamSpec{{ID: 0, URL: p.currentURL, OriginalURL: p.originalURL, Playlist: pl, Extension: ext}}, p, nil
}

func (p *parser) parseMaster(raw string) ([]StreamSpec, error) {
	p.rawFiles["raw.m3u8"] = raw
	var streams []StreamSpec
	sc := newHLSScanner(raw)
	expectPlaylist := false
	cur := StreamSpec{OriginalURL: p.originalURL}
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "#EXT-X-STREAM-INF"):
			cur = StreamSpec{OriginalURL: p.originalURL}
			bw := attr(line, "AVERAGE-BANDWIDTH")
			if bw == "" {
				bw = attr(line, "BANDWIDTH")
			}
			bandwidth, err := strconv.Atoi(bw)
			if err != nil {
				return nil, err
			}
			cur.Bandwidth = bandwidth
			cur.Codecs = attr(line, "CODECS")
			cur.Resolution = attr(line, "RESOLUTION")
			if frameRate := attr(line, "FRAME-RATE"); frameRate != "" {
				parsedFrameRate, err := strconv.ParseFloat(frameRate, 64)
				if err != nil {
					return nil, err
				}
				cur.FrameRate = parsedFrameRate
			}
			cur.AudioID = attr(line, "AUDIO")
			cur.VideoID = attr(line, "VIDEO")
			cur.SubtitleID = attr(line, "SUBTITLES")
			cur.VideoRange = attr(line, "VIDEO-RANGE")
			if cur.Codecs != "" && cur.AudioID != "" {
				cur.Codecs = strings.Split(cur.Codecs, ",")[0]
			}
			expectPlaylist = true
		case strings.HasPrefix(line, "#EXT-X-MEDIA"):
			mtText := strings.ReplaceAll(attr(line, "TYPE"), "-", "_")
			if mtText == "CLOSED_CAPTIONS" {
				continue
			}
			uri := attr(line, "URI")
			if uri == "" {
				continue
			}
			var mediaType *MediaType
			if mt := MediaType(mtText); mt == MediaVideo || mt == MediaAudio || mt == MediaSubtitles {
				// long: 上游使用 nullable enum，未知 TYPE 不会写入 MediaType，但轨道本身仍会保留给后续选择。
				mediaType = &mt
			}
			s := StreamSpec{
				OriginalURL:     p.originalURL,
				MediaType:       mediaType,
				URL:             p.preProcessURL(combineURL(p.baseURL, uri)),
				GroupID:         attr(line, "GROUP-ID"),
				Language:        attr(line, "LANGUAGE"),
				Name:            attr(line, "NAME"),
				Channels:        attr(line, "CHANNELS"),
				Characteristics: characteristicToken(attr(line, "CHARACTERISTICS")),
				Default:         strings.EqualFold(attr(line, "DEFAULT"), "YES"),
			}
			streams = append(streams, s)
		case strings.HasPrefix(line, "#"):
			continue
		case expectPlaylist:
			cur.URL = p.preProcessURL(combineURL(p.baseURL, line))
			streams = append(streams, cur)
			expectPlaylist = false
		}
	}
	for i := range streams {
		// long: 上游 Master 解析不会按 URL 去重，同一播放列表 URL 可以承载不同码率/元数据轨道，必须保留给后续选择器判断。
		streams[i].ID = i
	}
	return streams, sc.Err()
}

func distinctStreamsByURL(streams []StreamSpec) []StreamSpec {
	out := make([]StreamSpec, 0, len(streams))
	seen := map[string]bool{}
	for _, s := range streams {
		if seen[s.URL] {
			continue
		}
		// long: 原版对外 ExtractStreamsAsync 会 DistinctBy URL，但刷新 Master 时仍保留内部完整列表；这里把去重限制在初始导出层。
		s.ID = len(out)
		out = append(out, s)
		seen[s.URL] = true
	}
	return out
}

func (p *parser) fetchPlaylist(ctx context.Context, s *StreamSpec) error {
	raw, finalURL, err := fetchText(ctx, p.client, s.URL, p.opt.Headers)
	if err != nil {
		if !p.master {
			return err
		}
		if refreshErr := p.refreshPlaylistURLFromMaster(ctx, s); refreshErr != nil {
			return err
		}
		raw, finalURL, err = fetchText(ctx, p.client, s.URL, p.opt.Headers)
		if err != nil {
			return err
		}
	}
	oldURL, oldBase := p.currentURL, p.baseURL
	p.currentURL = finalURL
	p.baseURL = finalURL
	if p.opt.BaseURL != "" {
		p.baseURL = p.opt.BaseURL
	}
	pl, err := p.parseMedia(ctx, raw)
	p.currentURL, p.baseURL = oldURL, oldBase
	if err != nil {
		return err
	}
	if s.Playlist != nil && s.Playlist.MediaInit != nil {
		// long: fMP4 直播刷新时后续窗口常省略 EXT-X-MAP，上游只替换媒体分片并沿用首次解析到的 init。
		pl.MediaInit = s.Playlist.MediaInit
	}
	s.Playlist = pl
	if s.MediaType != nil && *s.MediaType == MediaSubtitles {
		s.Extension = subtitleExt(pl)
	} else if pl.MediaInit != nil {
		s.Extension = "m4s"
	} else {
		s.Extension = "ts"
	}
	return nil
}

func (p *parser) refreshPlaylistURLFromMaster(ctx context.Context, s *StreamSpec) error {
	raw, finalURL, err := fetchText(ctx, p.client, p.originalURL, p.opt.Headers)
	if err != nil {
		return err
	}
	oldURL, oldBase := p.currentURL, p.baseURL
	p.currentURL = finalURL
	p.baseURL = finalURL
	if p.opt.BaseURL != "" {
		p.baseURL = p.opt.BaseURL
	}
	streams, err := p.parseMaster(raw)
	p.currentURL, p.baseURL = oldURL, oldBase
	if err != nil {
		return err
	}
	key := streamRefreshKey(*s)
	for _, candidate := range streams {
		if streamRefreshKey(candidate) == key {
			s.URL = candidate.URL
			return nil
		}
	}
	return fmt.Errorf("刷新 master 后未找到匹配轨道: %s", s.Short())
}

func streamRefreshKey(s StreamSpec) string {
	mt := "VIDEO"
	if s.MediaType != nil {
		mt = string(*s.MediaType)
	}
	switch mt {
	case string(MediaAudio):
		return strings.Join([]string{mt, s.GroupID, strconv.Itoa(s.Bandwidth), s.Name, s.Codecs, s.Language, s.Channels}, "|")
	case string(MediaSubtitles):
		return strings.Join([]string{mt, s.GroupID, s.Language, s.Name, s.Codecs}, "|")
	default:
		return strings.Join([]string{mt, s.Resolution, strconv.Itoa(s.Bandwidth), s.GroupID, fmt.Sprintf("%g", s.FrameRate), s.Codecs, s.VideoRange}, "|")
	}
}

func (p *parser) parseMedia(ctx context.Context, raw string) (*Playlist, error) {
	p.rawFiles["raw.m3u8"] = raw
	pl := &Playlist{}
	current := EncryptInfo{Method: EncryptNone}
	if p.opt.CustomHLSMethod != "" {
		current.Method = p.opt.CustomHLSMethod
	}
	if len(p.opt.CustomHLSKey) > 0 {
		current.Key = p.opt.CustomHLSKey
	}
	if len(p.opt.CustomHLSIV) > 0 {
		current.IV = p.opt.CustomHLSIV
	}
	var parts []MediaPart
	var segs []Segment
	var seg Segment
	var seq int64
	expectSegment := false
	isEnd := false
	hasAd := false
	isAd := false
	lastKeyLine := ""
	sc := newHLSScanner(raw)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "#EXT-X-BYTERANGE"):
			l, start, err := parseByteRange(attr(line, ""))
			if err != nil {
				return nil, err
			}
			seg.ExpectLength = &l
			if start != nil {
				seg.StartRange = start
			} else if len(segs) > 0 && segs[len(segs)-1].StartRange != nil && segs[len(segs)-1].ExpectLength != nil {
				v := *segs[len(segs)-1].StartRange + *segs[len(segs)-1].ExpectLength
				seg.StartRange = &v
			}
			expectSegment = true
		case strings.HasPrefix(line, "#EXT-X-PLAYLIST-TYPE"):
			isEnd = strings.HasSuffix(line, "VOD")
		case strings.HasPrefix(line, "#UPLYNK-SEGMENT"):
			if strings.Contains(line, ",ad") {
				isAd = true
			} else if strings.Contains(line, ",segment") {
				// long: Uplynk 广告区中间可能穿插普通元数据行，只有明确回到 segment 状态时才结束广告过滤。
				isAd = false
			}
		case isAd:
			continue
		case strings.HasPrefix(line, "#EXT-X-TARGETDURATION"):
			targetDuration, err := strconv.ParseFloat(attr(line, ""), 64)
			if err != nil {
				return nil, err
			}
			pl.TargetDuration = targetDuration
		case strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE"):
			parsedSeq, err := strconv.ParseInt(attr(line, ""), 10, 64)
			if err != nil {
				return nil, err
			}
			seq = parsedSeq
		case strings.HasPrefix(line, "#EXT-X-PROGRAM-DATE-TIME"):
			t, err := parseHLSProgramDateTime(attr(line, ""))
			if err != nil {
				return nil, err
			}
			seg.DateTime = &t
		case strings.HasPrefix(line, "#EXT-X-DISCONTINUITY"):
			if hasAd && len(parts) > 0 {
				segs = parts[len(parts)-1].Segments
				parts = parts[:len(parts)-1]
				hasAd = false
				continue
			}
			if hasAd || len(segs) < 1 {
				continue
			}
			if len(segs) > 0 {
				parts = append(parts, MediaPart{Segments: segs})
				segs = nil
			}
		case strings.HasPrefix(line, "#EXT-X-KEY"):
			if line != lastKeyLine {
				ei, err := p.parseKey(ctx, line)
				if err != nil {
					return nil, err
				}
				// long: 真实 HLS 源常重复输出同一条 EXT-X-KEY，上游会复用上次解析结果，避免同一 key URI 被反复请求。
				current = ei
			}
			lastKeyLine = line
		case strings.HasPrefix(line, "#EXTINF"):
			// long: 原版 Convert.ToDouble/ToInt64 遇到坏数值会中断解析，不能静默置 0 继续下载错误清单。
			d, err := strconv.ParseFloat(strings.Split(attr(line, ""), ",")[0], 64)
			if err != nil {
				return nil, err
			}
			seg.Duration = d
			seg.Index = seq
			if current.Method != "" && current.Method != EncryptNone {
				seg.Encrypt = current
				// long: HLS 没有显式 IV 时，业务上必须用媒体序号补成 16 字节 IV，否则 AES-128 节目会解密出错。
				if len(seg.Encrypt.IV) == 0 {
					seg.Encrypt.IV = sequenceIV(seq)
				}
			}
			expectSegment = true
			seq++
		case strings.HasPrefix(line, "#EXT-X-ENDLIST"):
			if len(segs) > 0 {
				parts = append(parts, MediaPart{Segments: segs})
				segs = nil
			}
			isEnd = true
		case strings.HasPrefix(line, "#EXT-X-MAP"):
			if pl.MediaInit == nil || hasAd {
				initSeg := Segment{URL: p.preProcessURL(combineURL(p.baseURL, attr(line, "URI"))), Index: -1}
				if br := attr(line, "BYTERANGE"); br != "" {
					l, start, err := parseByteRange(br)
					if err != nil {
						return nil, err
					}
					initSeg.ExpectLength = &l
					if start == nil {
						v := int64(0)
						start = &v
					}
					initSeg.StartRange = start
				}
				if current.Method != EncryptNone {
					initSeg.Encrypt = current
					// long: fMP4 init 分片也可能被同一条 EXT-X-KEY 保护，提前写入 IV 能让下载器统一走分片解密路径。
					if len(initSeg.Encrypt.IV) == 0 {
						initSeg.Encrypt.IV = sequenceIV(seq)
					}
				}
				pl.MediaInit = &initSeg
			} else {
				if len(segs) > 0 {
					parts = append(parts, MediaPart{Segments: segs})
					segs = nil
				}
				if !p.opt.AllowHLSMultiExtMap {
					isEnd = true
					goto done
				}
			}
		case strings.HasPrefix(line, "#"):
			continue
		case expectSegment:
			seg.URL = p.preProcessURL(combineURL(p.baseURL, line))
			if p.shouldDropAd(seg.URL) {
				seq--
				hasAd = true
			} else {
				segs = append(segs, seg)
			}
			seg = Segment{}
			expectSegment = false
		}
	}
done:
	if !isEnd {
		// long: 直播窗口可能暂时没有分片，上游仍保留一个空 MediaPart；这样后续直播刷新逻辑能把它当作“已解析的直播轨道”继续处理。
		parts = append(parts, MediaPart{Segments: segs})
	}
	pl.Parts = parts
	pl.IsLive = !isEnd
	if pl.IsLive {
		pl.WasLive = true
		td := pl.TargetDuration
		if td <= 0 {
			td = 5
		}
		pl.RefreshIntervalMS = int(td * 2 * 1000)
	}
	return pl, sc.Err()
}

func (p *parser) preProcessURL(rawURL string) string {
	if !p.opt.AppendURLParams {
		return rawURL
	}
	out, err := appendURLParams(rawURL, p.currentURL)
	if err != nil {
		return rawURL
	}
	return out
}

func (p *parser) parseKey(ctx context.Context, line string) (EncryptInfo, error) {
	method := normalizeEncryptMethod(attr(line, "METHOD"))
	if method == "" || !isKnownHLSMethod(method) {
		method = EncryptUnknown
	}
	ei := EncryptInfo{Method: method}
	if iv := attr(line, "IV"); iv != "" {
		b, err := hex.DecodeString(strings.TrimPrefix(strings.ToLower(iv), "0x"))
		if err != nil {
			return ei, err
		}
		ei.IV = b
	}
	if len(p.opt.CustomHLSIV) > 0 {
		ei.IV = p.opt.CustomHLSIV
	}
	if len(p.opt.CustomHLSKey) > 0 {
		ei.Key = p.opt.CustomHLSKey
	} else if uri := attr(line, "URI"); uri != "" {
		key, err := p.loadHLSKey(ctx, uri)
		if err != nil {
			// long: 上游 key 加载失败时不会中断解析，而是把该加密标成 UNKNOWN，让下载阶段保留原始分片供后续外部处理。
			ei.Method = EncryptUnknown
		} else {
			ei.Key = key
		}
	}
	if p.opt.CustomHLSMethod != "" {
		ei.Method = p.opt.CustomHLSMethod
	}
	return ei, nil
}

func (p *parser) loadHLSKey(ctx context.Context, uri string) ([]byte, error) {
	lower := strings.ToLower(uri)
	switch {
	case strings.HasPrefix(lower, "base64:"):
		return base64.StdEncoding.DecodeString(uri[7:])
	case strings.HasPrefix(lower, "data:;base64,"):
		return base64.StdEncoding.DecodeString(uri[13:])
	case strings.HasPrefix(lower, "data:text/plain;base64,"):
		return base64.StdEncoding.DecodeString(uri[23:])
	case fileExists(uri):
		return os.ReadFile(uri)
	default:
		// long: HLS key URI 和媒体分片一样可能依赖 playlist URL 上的鉴权参数，上游会对 key URL 也执行 URL 预处理。
		keyURL := p.preProcessURL(combineURL(p.baseURL, uri))
		var lastErr error
		for try := 0; try <= hlsKeyRetryCount; try++ {
			key, err := fetchBytes(ctx, p.client, keyURL, p.opt.Headers)
			if err == nil {
				return key, nil
			}
			lastErr = err
		}
		return nil, lastErr
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func isKnownHLSMethod(method EncryptMethod) bool {
	switch method {
	case EncryptNone, EncryptAES128, EncryptAES128ECB, EncryptCENC, EncryptSampleAES, EncryptSampleCTR, EncryptChaCha20, EncryptUnknown:
		return true
	default:
		return false
	}
}

func (p *parser) shouldDropAd(segURL string) bool {
	if strings.Contains(segURL, "ccode=") && strings.Contains(segURL, "/ad/") && strings.Contains(segURL, "duration=") {
		return true
	}
	if strings.Contains(segURL, "ccode=0902") && strings.Contains(segURL, "duration=") {
		return true
	}
	return false
}

func sequenceIV(seq int64) []byte {
	out := make([]byte, 16)
	for i := 15; i >= 0 && seq > 0; i-- {
		out[i] = byte(seq)
		seq >>= 8
	}
	return out
}

func parseHLSProgramDateTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t, nil
	}
	layouts := []string{
		"2006-01-02T15:04:05.999999999-0700",
		"2006-01-02T15:04:05.999999-0700",
		"2006-01-02T15:04:05.999-0700",
		"2006-01-02T15:04:05-0700",
		"2006-01-02 15:04:05.999999999 -0700",
		"2006-01-02 15:04:05.999 -0700",
		"2006-01-02 15:04:05 -0700",
	}
	var lastErr error
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	// long: 上游使用 DateTime.Parse，部分站点会给无时区时间；Go 版把这类值按本地时区解释，尽量保住直播同步和去重依据。
	for _, layout := range []string{
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05.999",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05.999",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	if lastErr != nil {
		return time.Time{}, lastErr
	}
	return time.Time{}, fmt.Errorf("invalid PROGRAM-DATE-TIME: %s", raw)
}

func subtitleExt(pl *Playlist) string {
	for _, part := range pl.Parts {
		for _, seg := range part.Segments {
			u := strings.ToLower(seg.URL)
			if strings.Contains(u, ".ttml") {
				return "ttml"
			}
			if strings.Contains(u, ".vtt") || strings.Contains(u, ".webvtt") {
				return "vtt"
			}
		}
	}
	return "vtt"
}

func lastToken(input, sep string) string {
	if input == "" {
		return ""
	}
	parts := strings.Split(input, sep)
	return parts[len(parts)-1]
}

func characteristicToken(input string) string {
	if input == "" {
		return ""
	}
	// long: 上游先取 CHARACTERISTICS 逗号列表最后一项，再取点号命名空间最后一段；不能直接按点号切整个字符串，否则最后项无命名空间时会混入前一项。
	return lastToken(lastToken(input, ","), ".")
}
