package main

import (
	"bytes"
	"compress/zlib"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
)

func mediaPtr(mt MediaType) *MediaType {
	return &mt
}

func TestParseMaster(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/master.m3u8", currentURL: "https://example.com/master.m3u8", baseURL: "https://example.com/master.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="aud",LANGUAGE="zh",NAME="中文",DEFAULT=YES,URI="audio.m3u8"
#EXT-X-STREAM-INF:BANDWIDTH=2000,CODECS="avc1,mp4a",RESOLUTION=1920x1080,AUDIO="aud"
video.m3u8
`
	streams, err := p.parseMaster(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 2 {
		t.Fatalf("want 2 streams, got %d", len(streams))
	}
	if streams[0].URL != "https://example.com/audio.m3u8" {
		t.Fatalf("audio url wrong: %s", streams[0].URL)
	}
	if streams[1].Resolution != "1920x1080" {
		t.Fatalf("resolution wrong: %s", streams[1].Resolution)
	}
	if streams[0].Default {
		t.Fatalf("EXT-X-MEDIA DEFAULT should remain unset like upstream parser bug, got true")
	}
}

func TestParseMasterAcceptsLongVariantURLLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/master.m3u8", currentURL: "https://example.com/master.m3u8", baseURL: "https://example.com/master.m3u8", rawFiles: map[string]string{}}
	token := strings.Repeat("a", 70*1024)
	raw := "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=2000,RESOLUTION=1920x1080\nvideo.m3u8?token=" + token + "\n"
	streams, err := p.parseMaster(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 || !strings.Contains(streams[0].URL, token) {
		t.Fatalf("long variant URL should be preserved, got %#v", streams)
	}
}

func TestParseMasterKeepsDuplicateVariantURLsLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/master.m3u8", currentURL: "https://example.com/master.m3u8", baseURL: "https://example.com/master.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=1000,RESOLUTION=640x360
video.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=2000,RESOLUTION=1280x720
video.m3u8
`
	streams, err := p.parseMaster(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 2 {
		t.Fatalf("duplicate variant URLs should be kept like upstream, got %#v", streams)
	}
	if streams[0].ID != 0 || streams[1].ID != 1 || streams[0].Bandwidth != 1000 || streams[1].Bandwidth != 2000 {
		t.Fatalf("duplicate variant metadata should remain ordered, got %#v", streams)
	}
}

func TestExtractMasterDedupesDuplicateURLsLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/master.m3u8", currentURL: "https://example.com/master.m3u8", baseURL: "https://example.com/master.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=1000,RESOLUTION=640x360
video.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=2000,RESOLUTION=1280x720
video.m3u8
`
	streams, _, err := p.extract(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 || streams[0].ID != 0 || streams[0].Bandwidth != 1000 {
		t.Fatalf("initial master extraction should DistinctBy URL like upstream, got %#v", streams)
	}
}

func TestParseMasterCharacteristicsUsesLastCommaThenLastDotLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/master.m3u8", currentURL: "https://example.com/master.m3u8", baseURL: "https://example.com/master.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-MEDIA:TYPE=SUBTITLES,GROUP-ID="subs",NAME="CC",URI="cc.m3u8",CHARACTERISTICS="public.accessibility.describes-music-and-sound,forced"
`
	streams, err := p.parseMaster(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 || streams[0].Characteristics != "forced" {
		t.Fatalf("CHARACTERISTICS should follow upstream last comma/last dot rule, got %#v", streams)
	}
}

func TestParseMasterInvalidNumericFieldsFailLikeUpstream(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "bandwidth",
			raw:  "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=bad,RESOLUTION=1280x720\nvideo.m3u8\n",
		},
		{
			name: "empty bandwidth",
			raw:  "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=,RESOLUTION=1280x720\nvideo.m3u8\n",
		},
		{
			name: "empty bandwidth suffix attribute",
			raw:  "#EXTM3U\n#EXT-X-STREAM-INF:X-BANDWIDTH=,RESOLUTION=1280x720\nvideo.m3u8\n",
		},
		{
			name: "frame rate",
			raw:  "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=2000,FRAME-RATE=bad\nvideo.m3u8\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := defaultOptions()
			p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/master.m3u8", currentURL: "https://example.com/master.m3u8", baseURL: "https://example.com/master.m3u8", rawFiles: map[string]string{}}
			if _, err := p.parseMaster(tt.raw); err == nil {
				t.Fatal("invalid master numeric field should fail like upstream Convert")
			}
		})
	}
}

func TestParseMasterNumericFieldsTrimWhitespaceLikeUpstreamConvert(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/master.m3u8", currentURL: "https://example.com/master.m3u8", baseURL: "https://example.com/master.m3u8", rawFiles: map[string]string{}}
	raw := "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=\" 2000 \",FRAME-RATE=\" 29.97 \",RESOLUTION=1280x720\nvideo.m3u8\n"
	streams, err := p.parseMaster(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 {
		t.Fatalf("want 1 stream, got %#v", streams)
	}
	if streams[0].Bandwidth != 2000 {
		t.Fatalf("BANDWIDTH whitespace should parse like upstream Convert.ToInt32, got %d", streams[0].Bandwidth)
	}
	if streams[0].FrameRate != 29.97 {
		t.Fatalf("FRAME-RATE whitespace should parse like upstream Convert.ToDouble, got %f", streams[0].FrameRate)
	}
}

func TestParseMasterMalformedQuotedAttributesFailLikeUpstream(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "stream codecs",
			raw:  "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=2000,CODECS=\"avc1\nvideo.m3u8\n",
		},
		{
			name: "media uri",
			raw:  "#EXTM3U\n#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID=\"aud\",URI=\"audio.m3u8\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := defaultOptions()
			p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/master.m3u8", currentURL: "https://example.com/master.m3u8", baseURL: "https://example.com/master.m3u8", rawFiles: map[string]string{}}
			if _, err := p.parseMaster(tt.raw); err == nil {
				t.Fatal("malformed quoted master attribute should fail like upstream GetAttribute")
			}
		})
	}
}

func TestParseMasterMissingBandwidthDefaultsToZeroLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/master.m3u8", currentURL: "https://example.com/master.m3u8", baseURL: "https://example.com/master.m3u8", rawFiles: map[string]string{}}
	raw := "#EXTM3U\n#EXT-X-STREAM-INF:RESOLUTION=1280x720\nvideo.m3u8\n"
	streams, err := p.parseMaster(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 || streams[0].Bandwidth != 0 {
		t.Fatalf("missing BANDWIDTH should parse as 0 like upstream Convert.ToInt32(null), got %#v", streams)
	}
}

func TestParseMasterUnknownMediaTypeKeepsNilMediaTypeLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/master.m3u8", currentURL: "https://example.com/master.m3u8", baseURL: "https://example.com/master.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-MEDIA:TYPE=DATA,GROUP-ID="data",NAME="Timed Metadata",URI="data.m3u8"
`
	streams, err := p.parseMaster(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 {
		t.Fatalf("unknown media TYPE with URI should still be kept like upstream, got %#v", streams)
	}
	if streams[0].MediaType != nil {
		t.Fatalf("unknown media TYPE should leave MediaType nil like upstream, got %#v", streams[0].MediaType)
	}
	if streams[0].URL != "https://example.com/data.m3u8" {
		t.Fatalf("unknown media TYPE URL wrong: %s", streams[0].URL)
	}
}

func TestParseMasterMediaTypeYESSetsDefaultLikeUpstreamBug(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/master.m3u8", currentURL: "https://example.com/master.m3u8", baseURL: "https://example.com/master.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-MEDIA:TYPE=YES,GROUP-ID="data",NAME="Timed Metadata",DEFAULT=NO,URI="data.m3u8"
`
	streams, err := p.parseMaster(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 || !streams[0].Default {
		t.Fatalf("TYPE=YES should set Default through upstream parser bug, got %#v", streams)
	}
	if streams[0].MediaType != nil {
		t.Fatalf("TYPE=YES should still leave MediaType nil, got %#v", streams[0].MediaType)
	}
}

func TestParseMasterMediaTypeTrimsWhitespaceLikeUpstreamEnum(t *testing.T) {
	tests := []struct {
		name      string
		typeAttr  string
		wantCount int
		wantMedia *MediaType
		wantDef   bool
	}{
		{name: "audio", typeAttr: `" AUDIO "`, wantCount: 1, wantMedia: mediaPtr(MediaAudio)},
		{name: "closed captions", typeAttr: `" CLOSED-CAPTIONS "`, wantCount: 0},
		{name: "yes default bug", typeAttr: `" YES "`, wantCount: 1, wantDef: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := defaultOptions()
			p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/master.m3u8", currentURL: "https://example.com/master.m3u8", baseURL: "https://example.com/master.m3u8", rawFiles: map[string]string{}}
			raw := `#EXTM3U
#EXT-X-MEDIA:TYPE=` + tt.typeAttr + `,GROUP-ID="data",NAME="Timed Metadata",DEFAULT=NO,URI="data.m3u8"
`
			streams, err := p.parseMaster(raw)
			if err != nil {
				t.Fatal(err)
			}
			if len(streams) != tt.wantCount {
				t.Fatalf("unexpected stream count for TYPE=%s: got %#v", tt.typeAttr, streams)
			}
			if tt.wantCount == 0 {
				return
			}
			if tt.wantMedia == nil {
				if streams[0].MediaType != nil {
					t.Fatalf("expected nil MediaType, got %#v", streams[0].MediaType)
				}
			} else if streams[0].MediaType == nil || *streams[0].MediaType != *tt.wantMedia {
				t.Fatalf("wrong MediaType: got %#v want %#v", streams[0].MediaType, tt.wantMedia)
			}
			if streams[0].Default != tt.wantDef {
				t.Fatalf("wrong Default: got %v want %v", streams[0].Default, tt.wantDef)
			}
		})
	}
}

func TestParseMasterMediaMissingTypeFailsLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/master.m3u8", currentURL: "https://example.com/master.m3u8", baseURL: "https://example.com/master.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-MEDIA:GROUP-ID="data",NAME="Timed Metadata",URI="data.m3u8"
`
	if _, err := p.parseMaster(raw); err == nil {
		t.Fatal("EXT-X-MEDIA without TYPE should fail like upstream null Replace")
	}
}

func TestParseMasterLeadingSpaceTagIgnoredLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/master.m3u8", currentURL: "https://example.com/master.m3u8", baseURL: "https://example.com/master.m3u8", rawFiles: map[string]string{}}
	raw := "#EXTM3U\n #EXT-X-STREAM-INF:BANDWIDTH=2000,RESOLUTION=1280x720\nvideo.m3u8\n"
	streams, err := p.parseMaster(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 0 {
		t.Fatalf("leading-space master tags should be ignored like upstream StartsWith, got %#v", streams)
	}
}

func TestParseMediaAcceptsLongSegmentURLLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	token := strings.Repeat("b", 70*1024)
	raw := "#EXTM3U\n#EXT-X-TARGETDURATION:4\n#EXTINF:4,\nseg.ts?token=" + token + "\n#EXT-X-ENDLIST\n"
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 || !strings.Contains(segs[0].URL, token) {
		t.Fatalf("long segment URL should be preserved, got %#v", segs)
	}
}

func TestParseMediaLeadingSpaceTagIgnoredLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	raw := "#EXTM3U\n#EXT-X-TARGETDURATION:4\n #EXTINF:4,\nseg.ts\n#EXT-X-ENDLIST\n"
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(sortedSegments(pl)); got != 0 {
		t.Fatalf("leading-space media tags should be ignored like upstream StartsWith, got %d segments", got)
	}
}

func TestParseMediaPlaylistTypeVODTrailingSpaceMatchesUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	raw := "#EXTM3U\n#EXT-X-TARGETDURATION:4\n#EXT-X-PLAYLIST-TYPE:VOD   \n#EXTINF:4,\nseg.ts\n"
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if pl.IsLive {
		t.Fatal("PLAYLIST-TYPE VOD with trailing whitespace should be treated as VOD like upstream")
	}
	if len(pl.Parts) != 0 {
		t.Fatalf("VOD without ENDLIST should not append live tail part, got %#v", pl.Parts)
	}
}

func TestParseByteRangeTooManyAtSignsMatchesUpstream(t *testing.T) {
	length, start, err := parseByteRange("100@20@30")
	if err != nil {
		t.Fatal(err)
	}
	if length != 0 || start != nil {
		t.Fatalf("multiple @ BYTERANGE should return 0,nil like upstream, got length=%d start=%v", length, start)
	}
}

func TestParseMediaByteRangeWithoutOffsetBeforeFirstSegmentFailsLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	raw := "#EXTM3U\n#EXT-X-TARGETDURATION:4\n#EXT-X-BYTERANGE:100\n#EXTINF:4,\nseg.ts\n#EXT-X-ENDLIST\n"
	if _, err := p.parseMedia(context.Background(), raw); err == nil {
		t.Fatal("first BYTERANGE without offset should fail like upstream segments.Last()")
	}
}

func TestParseMediaMapEmptyByteRangeFailsLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	raw := "#EXTM3U\n#EXT-X-TARGETDURATION:4\n#EXT-X-MAP:URI=\"init.mp4\",BYTERANGE=\n#EXTINF:4,\nseg.m4s\n#EXT-X-ENDLIST\n"
	if _, err := p.parseMedia(context.Background(), raw); err == nil {
		t.Fatal("empty EXT-X-MAP BYTERANGE should fail like upstream GetRange")
	}
}

func TestParseSourceRetriesHTTPTextLikeUpstream(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			http.Error(w, "temporary", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\nseg.ts\n#EXT-X-ENDLIST\n"))
	}))
	defer srv.Close()

	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	streams, _, err := parseSource(context.Background(), srv.Client(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Fatalf("source playlist should retry until success, hits=%d", hits)
	}
	if len(streams) != 1 || streams[0].Playlist == nil {
		t.Fatalf("media playlist not parsed after retry: %#v", streams)
	}
}

func TestParseSourceSendsHTTPUtilHeadersLikeUpstream(t *testing.T) {
	var acceptEncoding, cacheControl string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptEncoding = r.Header.Get("Accept-Encoding")
		cacheControl = r.Header.Get("Cache-Control")
		_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\nseg.ts\n#EXT-X-ENDLIST\n"))
	}))
	defer srv.Close()

	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	if _, _, err := parseSource(context.Background(), srv.Client(), opt); err != nil {
		t.Fatal(err)
	}
	if acceptEncoding != "gzip, deflate" {
		t.Fatalf("playlist request should send upstream Accept-Encoding, got %q", acceptEncoding)
	}
	if cacheControl != "no-cache" {
		t.Fatalf("playlist request should send no-cache, got %q", cacheControl)
	}
}

func TestParseSourceKeepsCustomAcceptEncoding(t *testing.T) {
	var acceptEncoding string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptEncoding = r.Header.Get("Accept-Encoding")
		_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\nseg.ts\n#EXT-X-ENDLIST\n"))
	}))
	defer srv.Close()

	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	opt.Headers["accept-encoding"] = "identity"
	if _, _, err := parseSource(context.Background(), srv.Client(), opt); err != nil {
		t.Fatal(err)
	}
	if acceptEncoding != "identity" {
		t.Fatalf("custom Accept-Encoding should not be overwritten, got %q", acceptEncoding)
	}
}

func TestParseSourceDecodesHTTPCharsetLikeUpstream(t *testing.T) {
	gbkName := []byte{0xd6, 0xd0, 0xce, 0xc4}
	var raw []byte
	raw = append(raw, []byte("#EXTM3U\n#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID=\"aud\",LANGUAGE=\"zh\",NAME=\"")...)
	raw = append(raw, gbkName...)
	raw = append(raw, []byte("\",URI=\"audio.m3u8\"\n#EXT-X-STREAM-INF:BANDWIDTH=2000,AUDIO=\"aud\"\nvideo.m3u8\n")...)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl; charset=gbk")
		_, _ = w.Write(raw)
	}))
	defer srv.Close()

	opt := defaultOptions()
	opt.Input = srv.URL + "/master.m3u8"
	streams, _, err := parseSource(context.Background(), srv.Client(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) == 0 || streams[0].Name != "中文" {
		t.Fatalf("GBK playlist text should be decoded before parsing, got %#v", streams)
	}
}

func TestParseSourceDecodesDeflateHTTPTextLikeUpstream(t *testing.T) {
	raw := []byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\nseg.ts\n#EXT-X-ENDLIST\n")
	deflated := zlibBytes(t, raw)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "deflate")
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		_, _ = w.Write(deflated)
	}))
	defer srv.Close()

	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	streams, _, err := parseSource(context.Background(), srv.Client(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 || streams[0].Playlist == nil {
		t.Fatalf("deflate playlist should be decompressed before parsing, got %#v", streams)
	}
}

func TestParseMediaLoadsDeflateHLSKeyLikeUpstream(t *testing.T) {
	key := []byte("0123456789abcdef")
	deflatedKey := zlibBytes(t, key)
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/key.bin":
			w.Header().Set("Content-Encoding", "deflate")
			_, _ = w.Write(deflatedKey)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	opt := defaultOptions()
	p := &parser{opt: opt, client: srv.Client(), originalURL: base + "/main.m3u8", currentURL: base + "/main.m3u8", baseURL: base + "/main.m3u8", rawFiles: map[string]string{}}
	raw := "#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXT-X-KEY:METHOD=AES-128,URI=\"key.bin\"\n#EXTINF:1,\nseg.ts\n#EXT-X-ENDLIST\n"
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 || !bytes.Equal(segs[0].Encrypt.Key, key) {
		t.Fatalf("deflate HLS key should be decompressed before use, got %#v", segs)
	}
}

func TestParseMediaHLSKeySendsHTTPUtilHeadersLikeUpstream(t *testing.T) {
	var acceptEncoding, cacheControl string
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/key.bin":
			acceptEncoding = r.Header.Get("Accept-Encoding")
			cacheControl = r.Header.Get("Cache-Control")
			_, _ = w.Write([]byte("0123456789abcdef"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	opt := defaultOptions()
	p := &parser{opt: opt, client: srv.Client(), originalURL: base + "/main.m3u8", currentURL: base + "/main.m3u8", baseURL: base + "/main.m3u8", rawFiles: map[string]string{}}
	raw := "#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXT-X-KEY:METHOD=AES-128,URI=\"key.bin\"\n#EXTINF:1,\nseg.ts\n#EXT-X-ENDLIST\n"
	if _, err := p.parseMedia(context.Background(), raw); err != nil {
		t.Fatal(err)
	}
	if acceptEncoding != "gzip, deflate" {
		t.Fatalf("HLS key request should send upstream Accept-Encoding, got %q", acceptEncoding)
	}
	if cacheControl != "no-cache" {
		t.Fatalf("HLS key request should send no-cache, got %q", cacheControl)
	}
}

func TestParseSourceDecodesBrotliHTTPTextLikeUpstream(t *testing.T) {
	raw := []byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\nseg.ts\n#EXT-X-ENDLIST\n")
	compressed := brotliBytes(t, raw)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "br")
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		_, _ = w.Write(compressed)
	}))
	defer srv.Close()

	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	streams, _, err := parseSource(context.Background(), srv.Client(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 || streams[0].Playlist == nil {
		t.Fatalf("brotli playlist should be decompressed before parsing, got %#v", streams)
	}
}

func TestParseMediaLoadsBrotliHLSKeyLikeUpstream(t *testing.T) {
	key := []byte("0123456789abcdef")
	compressedKey := brotliBytes(t, key)
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/key.bin":
			w.Header().Set("Content-Encoding", "br")
			_, _ = w.Write(compressedKey)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	opt := defaultOptions()
	p := &parser{opt: opt, client: srv.Client(), originalURL: base + "/main.m3u8", currentURL: base + "/main.m3u8", baseURL: base + "/main.m3u8", rawFiles: map[string]string{}}
	raw := "#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXT-X-KEY:METHOD=AES-128,URI=\"key.bin\"\n#EXTINF:1,\nseg.ts\n#EXT-X-ENDLIST\n"
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 || !bytes.Equal(segs[0].Encrypt.Key, key) {
		t.Fatalf("brotli HLS key should be decompressed before use, got %#v", segs)
	}
}

func zlibBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	zw := zlib.NewWriter(&out)
	if _, err := zw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func brotliBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	bw := brotli.NewWriter(&out)
	if _, err := bw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := bw.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestPreProcessHLSContentMatchesUpstreamSiteFixes(t *testing.T) {
	t.Run("preserves surrounding whitespace", func(t *testing.T) {
		raw := " \n#EXTM3U\n#EXT-X-ENDLIST\n "
		got := preProcessHLSContent(raw, "https://example.com/main.m3u8")
		if got != raw {
			t.Fatalf("HLS preprocessing should not trim surrounding whitespace like upstream:\nwant %q\ngot  %q", raw, got)
		}
	})

	t.Run("carriage returns and YSP endlist", func(t *testing.T) {
		got := preProcessHLSContent("#EXTM3U\r#EXT-X-TARGETDURATION:1\r#EXTINF:1,\rseg.ts", "https://tlivecloud-playback-cdn.ysp.cctv.cn/live.m3u8?endtime=1")
		if !strings.Contains(got, "\n#EXTINF:1,") || !strings.HasSuffix(got, "#EXT-X-ENDLIST") {
			t.Fatalf("YSP content should normalize CR lines and append ENDLIST, got:\n%s", got)
		}
	})

	t.Run("YSP endlist is appended even when already present", func(t *testing.T) {
		got := preProcessHLSContent("#EXTM3U\n#EXT-X-ENDLIST", "https://tlivecloud-playback-cdn.ysp.cctv.cn/live.m3u8?endtime=1")
		if strings.Count(got, "#EXT-X-ENDLIST") != 2 {
			t.Fatalf("YSP replay should append ENDLIST unconditionally like upstream, got:\n%s", got)
		}
	})

	t.Run("key order", func(t *testing.T) {
		got := preProcessHLSContent("#EXTM3U\n#EXTINF:1,\n#EXT-X-KEY:METHOD=AES-128,URI=\"key.bin\"\nseg.ts", "https://example.com/main.m3u8")
		if !strings.Contains(got, "#EXT-X-KEY:METHOD=AES-128,URI=\"key.bin\"\n#EXTINF:1,\nseg.ts") {
			t.Fatalf("EXT-X-KEY should be moved before EXTINF like upstream, got:\n%s", got)
		}
	})

	t.Run("key order keeps whitespace separator", func(t *testing.T) {
		got := preProcessHLSContent("#EXTM3U\n#EXTINF:1,\n  \n#EXT-X-KEY:METHOD=AES-128,URI=\"key.bin\"\nseg.ts", "https://example.com/main.m3u8")
		if !strings.Contains(got, "#EXT-X-KEY:METHOD=AES-128,URI=\"key.bin\"\n  \n#EXTINF:1,\nseg.ts") {
			t.Fatalf("EXT-X-KEY should move across whitespace separator like upstream, got:\n%s", got)
		}
	})

	t.Run("youku dolby vision map", func(t *testing.T) {
		raw := "#EXTM3U\n#EXT-X-DISCONTINUITY\n#EXT-X-MAP:URI=\"https://ott.cibntv.net/init.mp4?ccode=0902\",BYTERANGE=\"100@0\"\n#EXTINF:4,\nseg.m4s"
		got := preProcessHLSContent(raw, "https://example.com/main.m3u8")
		if !strings.Contains(got, "#EXTINF:0.000000,\n#EXT-X-BYTERANGE:100@0\nhttps://ott.cibntv.net/init.mp4?ccode=0902") {
			t.Fatalf("Youku map should be converted to zero-duration segment, got:\n%s", got)
		}
	})

	t.Run("disney bumper media", func(t *testing.T) {
		raw := "#EXTM3U\n#EXT-X-MAP:URI=\"https://media.dssott.com/BUMPER/init.mp4\"\n#EXTINF:1,\nhttps://media.dssott.com/BUMPER/0.m4s\n#EXT-X-DISCONTINUITY\n#EXT-X-MAP:URI=\"main.mp4\"\n#EXTINF:1,\nmain.m4s"
		got := preProcessHLSContent(raw, "https://media.dssott.com/video/main.m3u8")
		if strings.Contains(got, "BUMPER") || !strings.Contains(got, "#XXX\n#EXT-X-MAP:URI=\"main.mp4\"") {
			t.Fatalf("Disney bumper media prelude should be removed, got:\n%s", got)
		}
	})

	t.Run("disney bumper media replaces first match only", func(t *testing.T) {
		raw := "#EXTM3U\n#EXT-X-MAP:URI=\"https://media.dssott.com/BUMPER/init-a.mp4\"\n#EXTINF:1,\nhttps://media.dssott.com/BUMPER/a.m4s\n#EXT-X-DISCONTINUITY\n#EXT-X-MAP:URI=\"https://media.dssott.com/BUMPER/init-b.mp4\"\n#EXTINF:1,\nhttps://media.dssott.com/BUMPER/b.m4s\n#EXT-X-DISCONTINUITY\n#EXT-X-MAP:URI=\"main.mp4\"\n#EXTINF:1,\nmain.m4s"
		got := preProcessHLSContent(raw, "https://media.dssott.com/video/main.m3u8")
		if strings.Count(got, "#XXX") != 1 {
			t.Fatalf("Disney media preprocessor should replace only first match like upstream, got:\n%s", got)
		}
		if !strings.Contains(got, "init-b.mp4") || !strings.Contains(got, "b.m4s") {
			t.Fatalf("second Disney media bumper block should remain like upstream, got:\n%s", got)
		}
	})

	t.Run("disney bumper subtitle", func(t *testing.T) {
		raw := "#EXTM3U\n#EXTINF:1,\nhttps://media.dssott.com/BUMPER/seg_00000.vtt\n#EXT-X-DISCONTINUITY\n#EXTINF:1,\nseg_00000.vtt"
		got := preProcessHLSContent(raw, "https://media.dssott.com/sub/main.m3u8")
		if strings.Contains(got, "BUMPER") || !strings.Contains(got, "#XXX\n#EXTINF:1,\nseg_00000.vtt") {
			t.Fatalf("Disney bumper subtitle prelude should be removed, got:\n%s", got)
		}
	})

	t.Run("disney bumper subtitle replaces first match only", func(t *testing.T) {
		raw := "#EXTM3U\n#EXTINF:1,\nhttps://media.dssott.com/BUMPER/a/seg_00000.vtt\n#EXT-X-DISCONTINUITY\n#EXTINF:1,\nhttps://media.dssott.com/BUMPER/b/seg_00000.vtt\n#EXT-X-DISCONTINUITY\n#EXTINF:1,\nseg_00000.vtt"
		got := preProcessHLSContent(raw, "https://media.dssott.com/sub/main.m3u8")
		if strings.Count(got, "#XXX") != 1 {
			t.Fatalf("Disney subtitle preprocessor should replace only first match like upstream, got:\n%s", got)
		}
		if !strings.Contains(got, "BUMPER/b/seg_00000.vtt") {
			t.Fatalf("second Disney subtitle bumper block should remain like upstream, got:\n%s", got)
		}
	})

	t.Run("apple tv encrypted part", func(t *testing.T) {
		raw := "#EXTM3U\n#EXT-X-MAP:URI=\"https://video.apple.com/init.mp4\"\n#EXT-X-KEY:METHOD=SAMPLE-AES,URI=\"skd://asset\"\n#EXTINF:4,\nenc.m4s\n#EXT-X-DISCONTINUITY\n#EXTINF:4,\nclear.m4s"
		got := preProcessHLSContent(raw, "https://video.apple.com/main.m3u8")
		want := "#EXTM3U\r\n#EXT-X-KEY:METHOD=SAMPLE-AES,URI=\"skd://asset\"\n#EXTINF:4,\nenc.m4s\n\r\n#EXT-X-ENDLIST"
		if got != want {
			t.Fatalf("AppleTV preprocessing mismatch:\nwant:\n%s\ngot:\n%s", want, got)
		}
	})

	t.Run("apple tv malformed key order follows upstream processor order", func(t *testing.T) {
		raw := "#EXTM3U\n#EXT-X-MAP:URI=\"https://video.apple.com/init.mp4\"\n#EXTINF:4,\n#EXT-X-KEY:METHOD=SAMPLE-AES,URI=\"skd://asset\"\nenc.m4s\n#EXT-X-DISCONTINUITY\n#EXTINF:4,\nclear.m4s"
		got := preProcessHLSContent(raw, "https://video.apple.com/main.m3u8")
		want := "#EXTM3U\r\n#EXT-X-KEY:METHOD=SAMPLE-AES,URI=\"skd://asset\"\nenc.m4s\n\r\n#EXT-X-ENDLIST"
		if got != want {
			t.Fatalf("AppleTV malformed key-order preprocessing mismatch:\nwant:\n%s\ngot:\n%s", want, got)
		}
	})
}

func TestParseSourceRawM3U8IsSavedBeforeHLSPreprocessLikeUpstream(t *testing.T) {
	raw := "#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\nseg.ts"
	opt := defaultOptions()
	opt.Input = "https://tlivecloud-playback-cdn.ysp.cctv.cn/live.m3u8?endtime=1"
	p := &parser{
		opt:         opt,
		client:      http.DefaultClient,
		originalURL: opt.Input,
		currentURL:  opt.Input,
		baseURL:     opt.Input,
		rawFiles:    map[string]string{},
	}
	streams, _, err := p.extract(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 || streams[0].Playlist == nil || streams[0].Playlist.IsLive {
		t.Fatalf("YSP replay should parse with appended ENDLIST, got %#v", streams)
	}
	if p.rawFiles["raw.m3u8"] != raw {
		t.Fatalf("raw.m3u8 should keep trimmed source before HLS preprocess like upstream:\nwant %q\ngot  %q", raw, p.rawFiles["raw.m3u8"])
	}
}

func TestParseSourceLocalRelativePathResolvesSegmentsFromPlaylistDir(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "media"), 0755); err != nil {
		t.Fatal(err)
	}
	playlist := "#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\nmedia/seg.ts\n#EXT-X-ENDLIST\n"
	if err := os.WriteFile(filepath.Join(tmp, "main.m3u8"), []byte(playlist), 0644); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	opt := defaultOptions()
	opt.Input = "main.m3u8"
	streams, _, err := parseSource(context.Background(), http.DefaultClient, opt)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(streams[0].Playlist)
	want, err := localFileURL(filepath.Join("media", "seg.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 1 || segs[0].URL != want {
		t.Fatalf("relative local segment should resolve from playlist dir, want %s got %#v", want, segs)
	}
}

func TestParseSourceLocalRelativeKeyLoadsFromPlaylistDir(t *testing.T) {
	tmp := t.TempDir()
	key := []byte("0123456789abcdef")
	if err := os.WriteFile(filepath.Join(tmp, "key.bin"), key, 0644); err != nil {
		t.Fatal(err)
	}
	playlist := "#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXT-X-KEY:METHOD=AES-128,URI=\"key.bin\",IV=0x00000000000000000000000000000000\n#EXTINF:1,\nseg.ts\n#EXT-X-ENDLIST\n"
	if err := os.WriteFile(filepath.Join(tmp, "main.m3u8"), []byte(playlist), 0644); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	opt := defaultOptions()
	opt.Input = "main.m3u8"
	streams, _, err := parseSource(context.Background(), http.DefaultClient, opt)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(streams[0].Playlist)
	if len(segs) != 1 || string(segs[0].Encrypt.Key) != string(key) {
		t.Fatalf("relative local key should load from playlist dir, got %#v", segs)
	}
}

func TestParseSourceRelativeKeyTrimsWhitespaceLikeUpstreamUri(t *testing.T) {
	tmp := t.TempDir()
	key := []byte("0123456789abcdef")
	if err := os.WriteFile(filepath.Join(tmp, "key.bin"), key, 0644); err != nil {
		t.Fatal(err)
	}
	playlist := "#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXT-X-KEY:METHOD=AES-128,URI=\" key.bin \",IV=0x00000000000000000000000000000000\n#EXTINF:1,\nseg.ts\n#EXT-X-ENDLIST\n"
	if err := os.WriteFile(filepath.Join(tmp, "main.m3u8"), []byte(playlist), 0644); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	opt := defaultOptions()
	opt.Input = "main.m3u8"
	streams, _, err := parseSource(context.Background(), http.DefaultClient, opt)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(streams[0].Playlist)
	if len(segs) != 1 || string(segs[0].Encrypt.Key) != string(key) {
		t.Fatalf("relative key URI whitespace should be trimmed like upstream Uri, got %#v", segs)
	}
}

func TestParseMediaAES(t *testing.T) {
	opt := defaultOptions()
	opt.CustomHLSKey = []byte("0123456789abcdef")
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/a/main.m3u8", currentURL: "https://example.com/a/main.m3u8", baseURL: "https://example.com/a/main.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:8
#EXT-X-MEDIA-SEQUENCE:7
#EXT-X-KEY:METHOD=AES-128,URI="key.bin"
#EXTINF:8.0,
seg.ts
#EXT-X-ENDLIST
`
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if segs[0].URL != "https://example.com/a/seg.ts" {
		t.Fatalf("segment url wrong: %s", segs[0].URL)
	}
	if !segs[0].IsEncrypted() || len(segs[0].Encrypt.IV) != 16 {
		t.Fatalf("encrypt info missing")
	}
}

func TestParseProgramDateTimeAcceptsUpstreamDateTimeParseFormats(t *testing.T) {
	got, err := parseHLSProgramDateTime("2026-06-18T12:34:56.789+0000")
	if err != nil {
		t.Fatal(err)
	}
	if got.UTC().Format(time.RFC3339Nano) != "2026-06-18T12:34:56.789Z" {
		t.Fatalf("compact timezone offset parsed wrong: %s", got.UTC().Format(time.RFC3339Nano))
	}
	local, err := parseHLSProgramDateTime("2026-06-18 12:34:56")
	if err != nil {
		t.Fatal(err)
	}
	if local.Location() != time.Local {
		t.Fatalf("timezone-less timestamp should use local location, got %s", local.Location())
	}
}

func TestParseMediaProgramDateTimeCompactOffset(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:4
#EXT-X-PROGRAM-DATE-TIME:2026-06-18T12:34:56.789+0000
#EXTINF:4.0,
seg.ts
`
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 || segs[0].DateTime == nil {
		t.Fatalf("program date time should be parsed: %#v", segs)
	}
	if segs[0].DateTime.UTC().Format(time.RFC3339Nano) != "2026-06-18T12:34:56.789Z" {
		t.Fatalf("segment date time mismatch: %s", segs[0].DateTime.UTC().Format(time.RFC3339Nano))
	}
}

func TestParseMediaProgramDateTimeInvalidFailsLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:4
#EXT-X-PROGRAM-DATE-TIME:not-a-date
#EXTINF:4.0,
seg.ts
`
	if _, err := p.parseMedia(context.Background(), raw); err == nil {
		t.Fatal("invalid PROGRAM-DATE-TIME should fail like upstream DateTime.Parse")
	}
}

func TestParseMediaInvalidNumericFieldsFailLikeUpstream(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "target duration",
			raw:  "#EXTM3U\n#EXT-X-TARGETDURATION:bad\n#EXTINF:4,\nseg.ts\n",
		},
		{
			name: "media sequence",
			raw:  "#EXTM3U\n#EXT-X-TARGETDURATION:4\n#EXT-X-MEDIA-SEQUENCE:bad\n#EXTINF:4,\nseg.ts\n",
		},
		{
			name: "extinf",
			raw:  "#EXTM3U\n#EXT-X-TARGETDURATION:4\n#EXTINF:bad,\nseg.ts\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := defaultOptions()
			p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
			if _, err := p.parseMedia(context.Background(), tt.raw); err == nil {
				t.Fatal("invalid media numeric field should fail like upstream Convert")
			}
		})
	}
}

func TestParseMediaNumericFieldsTrimWhitespaceLikeUpstreamConvert(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	raw := "#EXTM3U\n" +
		"#EXT-X-TARGETDURATION: 4 \x20\n" +
		"#EXT-X-MEDIA-SEQUENCE: 7 \x20\n" +
		"#EXTINF: 8.5 ,\n" +
		"seg.ts\n" +
		"#EXT-X-ENDLIST\n"
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if pl.TargetDuration != 4 {
		t.Fatalf("TARGETDURATION whitespace should parse like upstream Convert.ToDouble, got %f", pl.TargetDuration)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %#v", segs)
	}
	if segs[0].Index != 7 {
		t.Fatalf("MEDIA-SEQUENCE whitespace should parse like upstream Convert.ToInt64, got %d", segs[0].Index)
	}
	if segs[0].Duration != 8.5 {
		t.Fatalf("EXTINF whitespace should parse like upstream Convert.ToDouble, got %f", segs[0].Duration)
	}
}

func TestParseMediaMarksOriginalLivePlaylist(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:4
#EXTINF:4.0,
seg.ts
`
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if !pl.IsLive || !pl.WasLive {
		t.Fatalf("live playlist should preserve original live state: %#v", pl)
	}
}

func TestParseMediaEmptyLivePlaylistKeepsEmptyPartLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:4
#EXT-X-MEDIA-SEQUENCE:42
`
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if !pl.IsLive || !pl.WasLive {
		t.Fatalf("empty live playlist should still be treated as live: %#v", pl)
	}
	if len(pl.Parts) != 1 || len(pl.Parts[0].Segments) != 0 {
		t.Fatalf("empty live playlist should keep one empty part like upstream, got %#v", pl.Parts)
	}
}

func TestParseMediaUplynkAdStateOnlyEndsOnSegmentMarkerLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:4
#UPLYNK-SEGMENT:0,segment
#EXTINF:4.0,
main0.ts
#UPLYNK-SEGMENT:1,ad
#UPLYNK-SEGMENT:1,cue
#EXTINF:4.0,
ad0.ts
#UPLYNK-SEGMENT:2,segment
#EXTINF:4.0,
main1.ts
#EXT-X-ENDLIST
`
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(pl)
	if len(segs) != 2 {
		t.Fatalf("Uplynk ad state should skip cue-only ad segment like upstream, got %#v", segs)
	}
	if segs[0].URL != "https://example.com/main0.ts" || segs[1].URL != "https://example.com/main1.ts" {
		t.Fatalf("unexpected Uplynk filtered segments: %#v", segs)
	}
}

func TestParseMediaYkAdDiscontinuityDoesNotSplitCurrentPartLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:4
#EXTINF:4.0,
main0.ts
#EXTINF:1.0,
https://yk.example.com/ad/0.ts?ccode=0902&duration=1
#EXT-X-DISCONTINUITY
#EXTINF:4.0,
main1.ts
#EXT-X-ENDLIST
`
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(pl.Parts) != 1 {
		t.Fatalf("YK ad discontinuity should keep one media part like upstream, got %#v", pl.Parts)
	}
	segs := sortedSegments(pl)
	if len(segs) != 2 {
		t.Fatalf("YK ad should be removed while preserving main segments, got %#v", segs)
	}
	if segs[0].URL != "https://example.com/main0.ts" || segs[1].URL != "https://example.com/main1.ts" {
		t.Fatalf("unexpected YK filtered segments: %#v", segs)
	}
}

func TestParseMediaCachesRepeatedKeyLine(t *testing.T) {
	var keyHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/key.bin" {
			http.NotFound(w, r)
			return
		}
		keyHits++
		_, _ = w.Write([]byte("0123456789abcdef"))
	}))
	defer srv.Close()

	opt := defaultOptions()
	p := &parser{
		opt:         opt,
		client:      srv.Client(),
		originalURL: srv.URL + "/main.m3u8",
		currentURL:  srv.URL + "/main.m3u8",
		baseURL:     srv.URL + "/main.m3u8",
		rawFiles:    map[string]string{},
	}
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:8
#EXT-X-KEY:METHOD=AES-128,URI="key.bin"
#EXTINF:8.0,
0.ts
#EXT-X-KEY:METHOD=AES-128,URI="key.bin"
#EXTINF:8.0,
1.ts
#EXT-X-ENDLIST
`
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(sortedSegments(pl)); got != 2 {
		t.Fatalf("want 2 segments, got %d", got)
	}
	if keyHits != 1 {
		t.Fatalf("expected one key request for repeated key line, got %d", keyHits)
	}
}

func TestParseMediaKeyLoadFailureDowngradesToUnknown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	opt := defaultOptions()
	p := &parser{
		opt:         opt,
		client:      srv.Client(),
		originalURL: srv.URL + "/main.m3u8",
		currentURL:  srv.URL + "/main.m3u8",
		baseURL:     srv.URL + "/main.m3u8",
		rawFiles:    map[string]string{},
	}
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:8
#EXT-X-KEY:METHOD=AES-128,URI="missing-key.bin"
#EXTINF:8.0,
0.ts
#EXT-X-ENDLIST
`
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if segs[0].Encrypt.Method != EncryptUnknown {
		t.Fatalf("missing key should downgrade encryption to UNKNOWN, got %s", segs[0].Encrypt.Method)
	}
}

func TestParseMediaKeyMissingURIDowngradesToUnknownLikeUpstream(t *testing.T) {
	tests := []struct {
		name   string
		method EncryptMethod
	}{
		{name: "aes128", method: EncryptAES128},
		{name: "none", method: EncryptNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := defaultOptions()
			p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
			raw := `#EXTM3U
#EXT-X-TARGETDURATION:8
#EXT-X-KEY:METHOD=` + string(tt.method) + `
#EXTINF:8.0,
0.ts
#EXT-X-ENDLIST
`
			pl, err := p.parseMedia(context.Background(), raw)
			if err != nil {
				t.Fatal(err)
			}
			segs := sortedSegments(pl)
			if len(segs) != 1 || segs[0].Encrypt.Method != EncryptUnknown {
				t.Fatalf("key without URI should downgrade to UNKNOWN like upstream, got %#v", segs)
			}
		})
	}
}

func TestParseMediaKeyEmptyURIKeepsMethodLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:8
#EXT-X-KEY:METHOD=AES-128,URI=""
#EXTINF:8.0,
0.ts
#EXT-X-ENDLIST
`
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %#v", segs)
	}
	if segs[0].Encrypt.Method != EncryptAES128 {
		t.Fatalf("empty URI should keep parsed METHOD like upstream, got %#v", segs[0].Encrypt)
	}
	if len(segs[0].Encrypt.Key) != 0 {
		t.Fatalf("empty URI should not load a key, got %x", segs[0].Encrypt.Key)
	}
}

func TestParseMediaKeyLooseURISuffixMatchesUpstream(t *testing.T) {
	tests := []struct {
		name    string
		attr    string
		wantKey string
	}{
		{name: "empty suffix uri", attr: `KEYFORMATURI=""`},
		{name: "base64 suffix uri", attr: `KEYFORMATURI="base64:MDEyMzQ1Njc4OWFiY2RlZg=="`, wantKey: "0123456789abcdef"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := defaultOptions()
			p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
			raw := `#EXTM3U
#EXT-X-TARGETDURATION:8
#EXT-X-KEY:METHOD=AES-128,` + tt.attr + `
#EXTINF:8.0,
0.ts
#EXT-X-ENDLIST
`
			pl, err := p.parseMedia(context.Background(), raw)
			if err != nil {
				t.Fatal(err)
			}
			segs := sortedSegments(pl)
			if len(segs) != 1 || segs[0].Encrypt.Method != EncryptAES128 {
				t.Fatalf("URI suffix attribute should keep AES-128 like upstream, got %#v", segs)
			}
			if string(segs[0].Encrypt.Key) != tt.wantKey {
				t.Fatalf("URI suffix key mismatch: got %q want %q", string(segs[0].Encrypt.Key), tt.wantKey)
			}
		})
	}
}

func TestParseMediaKeyQuotedURIWinsAfterLooseSuffixLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:8
#EXT-X-KEY:METHOD=AES-128,KEYFORMATURI=foo,URI="base64:MDEyMzQ1Njc4OWFiY2RlZg=="
#EXTINF:8.0,
0.ts
#EXT-X-ENDLIST
`
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %#v", segs)
	}
	if segs[0].Encrypt.Method != EncryptAES128 {
		t.Fatalf("quoted URI should keep AES-128 like upstream, got %#v", segs[0].Encrypt)
	}
	if string(segs[0].Encrypt.Key) != "0123456789abcdef" {
		t.Fatalf("quoted URI should beat earlier KEYFORMATURI=foo, got key %q", string(segs[0].Encrypt.Key))
	}
}

func TestParseMediaKeyInlineBase64IgnoresWhitespaceLikeUpstream(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{name: "base64", uri: "base64: MDEy\tMzQ1 Njc4OWFiY2RlZg== "},
		{name: "data text plain", uri: "data:text/plain;base64, MDEy\tMzQ1 Njc4OWFiY2RlZg== "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := defaultOptions()
			p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
			raw := `#EXTM3U
#EXT-X-TARGETDURATION:8
#EXT-X-KEY:METHOD=AES-128,URI="` + tt.uri + `"
#EXTINF:8.0,
0.ts
#EXT-X-ENDLIST
`
			pl, err := p.parseMedia(context.Background(), raw)
			if err != nil {
				t.Fatal(err)
			}
			segs := sortedSegments(pl)
			if len(segs) != 1 || segs[0].Encrypt.Method != EncryptAES128 {
				t.Fatalf("inline base64 key should keep AES-128 like upstream, got %#v", segs)
			}
			if string(segs[0].Encrypt.Key) != "0123456789abcdef" {
				t.Fatalf("inline base64 key should ignore whitespace like upstream, got %q", string(segs[0].Encrypt.Key))
			}
		})
	}
}

func TestParseMediaKeyMethodIsCaseSensitiveLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:8
#EXT-X-KEY:METHOD=aes-128,URI="base64:MDEyMzQ1Njc4OWFiY2RlZg=="
#EXTINF:8.0,
0.ts
#EXT-X-ENDLIST
`
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 || segs[0].Encrypt.Method != EncryptUnknown {
		t.Fatalf("lowercase HLS METHOD should parse as UNKNOWN like upstream Enum.TryParse, got %#v", segs)
	}
	if string(segs[0].Encrypt.Key) != "0123456789abcdef" {
		t.Fatalf("key bytes should still load before method downgrade is observed, got %#v", segs)
	}
}

func TestParseMediaKeyMethodTrimsWhitespaceLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:8
#EXT-X-KEY:METHOD=" AES-128 ",URI="base64:MDEyMzQ1Njc4OWFiY2RlZg=="
#EXTINF:8.0,
0.ts
#EXT-X-ENDLIST
`
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 || segs[0].Encrypt.Method != EncryptAES128 {
		t.Fatalf("HLS METHOD should trim whitespace like upstream Enum.TryParse, got %#v", segs)
	}
}

func TestParseMediaKeyMethodNumericEnumValuesLikeUpstream(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want EncryptMethod
	}{
		{name: "zero none", raw: "0", want: EncryptNone},
		{name: "leading zero aes128", raw: "01", want: EncryptAES128},
		{name: "explicit plus aes128", raw: "+1", want: EncryptAES128},
		{name: "seven unknown", raw: "7", want: EncryptUnknown},
		{name: "undefined positive", raw: "8", want: EncryptMethod("8")},
		{name: "explicit plus undefined positive", raw: "+8", want: EncryptMethod("8")},
		{name: "negative rejected after dash replacement", raw: "-1", want: EncryptUnknown},
		{name: "int32 overflow rejected", raw: "2147483648", want: EncryptUnknown},
		{name: "uint max rejected", raw: "4294967295", want: EncryptUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := defaultOptions()
			p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
			ei, err := p.parseKey(context.Background(), `#EXT-X-KEY:METHOD="`+tt.raw+`",URI="base64:MDEyMzQ1Njc4OWFiY2RlZg=="`)
			if err != nil {
				t.Fatal(err)
			}
			if ei.Method != tt.want {
				t.Fatalf("numeric HLS METHOD %q should parse as %s like upstream Enum.TryParse, got %s", tt.raw, tt.want, ei.Method)
			}
		})
	}
}

func TestParseMediaKeyLoadRetriesBeforeDowngrade(t *testing.T) {
	var keyHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/key.bin" {
			http.NotFound(w, r)
			return
		}
		keyHits++
		if keyHits < 3 {
			http.Error(w, "temporary key failure", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("0123456789abcdef"))
	}))
	defer srv.Close()

	opt := defaultOptions()
	p := &parser{
		opt:         opt,
		client:      srv.Client(),
		originalURL: srv.URL + "/main.m3u8",
		currentURL:  srv.URL + "/main.m3u8",
		baseURL:     srv.URL + "/main.m3u8",
		rawFiles:    map[string]string{},
	}
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:8
#EXT-X-KEY:METHOD=AES-128,URI="key.bin"
#EXTINF:8.0,
0.ts
#EXT-X-ENDLIST
`
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 || segs[0].Encrypt.Method != EncryptAES128 || string(segs[0].Encrypt.Key) != "0123456789abcdef" {
		t.Fatalf("key retry should eventually keep AES-128 key, got %#v", segs)
	}
	if keyHits != 3 {
		t.Fatalf("expected key endpoint to be retried until success, got %d hits", keyHits)
	}
}

func TestParseMediaKeyLoadFailureUsesUpstreamRetryCount(t *testing.T) {
	var keyHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/key.bin" {
			http.NotFound(w, r)
			return
		}
		keyHits++
		http.Error(w, "temporary key failure", http.StatusInternalServerError)
	}))
	defer srv.Close()

	opt := defaultOptions()
	p := &parser{
		opt:         opt,
		client:      srv.Client(),
		originalURL: srv.URL + "/main.m3u8",
		currentURL:  srv.URL + "/main.m3u8",
		baseURL:     srv.URL + "/main.m3u8",
		rawFiles:    map[string]string{},
	}
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:8
#EXT-X-KEY:METHOD=AES-128,URI="key.bin"
#EXTINF:8.0,
0.ts
#EXT-X-ENDLIST
`
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 || segs[0].Encrypt.Method != EncryptUnknown {
		t.Fatalf("key failure after retries should downgrade to UNKNOWN, got %#v", segs)
	}
	if keyHits != hlsKeyRetryCount+1 {
		t.Fatalf("expected upstream retry count to make %d key attempts, got %d", hlsKeyRetryCount+1, keyHits)
	}
}

func TestLoadHLSKeyUnsupportedSchemeFailsWithoutFileFallbackLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	if _, err := p.loadHLSKey(context.Background(), "skd://asset"); err == nil || !strings.Contains(err.Error(), "scheme is not supported") {
		t.Fatalf("unsupported key scheme should fail like upstream HttpClient, got %v", err)
	}
}

func TestParseMediaKeyURLAppliesAppendURLParamsLikeUpstream(t *testing.T) {
	var keyHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/key.bin" {
			http.NotFound(w, r)
			return
		}
		keyHits++
		if r.URL.Query().Get("token") != "abc" {
			http.Error(w, "missing token", http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte("0123456789abcdef"))
	}))
	defer srv.Close()

	opt := defaultOptions()
	opt.AppendURLParams = true
	p := &parser{
		opt:         opt,
		client:      srv.Client(),
		originalURL: srv.URL + "/main.m3u8?token=abc",
		currentURL:  srv.URL + "/main.m3u8?token=abc",
		baseURL:     srv.URL + "/main.m3u8?token=abc",
		rawFiles:    map[string]string{},
	}
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:8
#EXT-X-KEY:METHOD=AES-128,URI="key.bin"
#EXTINF:8.0,
0.ts
#EXT-X-ENDLIST
`
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 || segs[0].Encrypt.Method != EncryptAES128 || string(segs[0].Encrypt.Key) != "0123456789abcdef" {
		t.Fatalf("key URL should inherit playlist params and keep AES key, got %#v", segs)
	}
	if keyHits != 1 {
		t.Fatalf("expected key endpoint to be requested once, got %d", keyHits)
	}
}

func TestParseMediaInvalidInlineKeyDowngradesToUnknown(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:8
#EXT-X-KEY:METHOD=AES-128,URI="base64:not-valid"
#EXTINF:8.0,
0.ts
#EXT-X-ENDLIST
`
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 || segs[0].Encrypt.Method != EncryptUnknown {
		t.Fatalf("invalid inline key should become UNKNOWN, got %#v", segs)
	}
}

func TestParseMediaMalformedQuotedKeyAttributesFailLikeUpstream(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "uri",
			raw: `#EXTM3U
#EXT-X-TARGETDURATION:8
#EXT-X-KEY:METHOD=AES-128,URI="base64:MDEyMzQ1Njc4OWFiY2RlZg==
#EXTINF:8.0,
0.ts
#EXT-X-ENDLIST
`,
		},
		{
			name: "iv",
			raw: `#EXTM3U
#EXT-X-TARGETDURATION:8
#EXT-X-KEY:METHOD=AES-128,URI="base64:MDEyMzQ1Njc4OWFiY2RlZg==",IV="0x00000000000000000000000000000001
#EXTINF:8.0,
0.ts
#EXT-X-ENDLIST
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := defaultOptions()
			p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
			if _, err := p.parseMedia(context.Background(), tt.raw); err == nil {
				t.Fatal("malformed quoted key attribute should fail like upstream GetAttribute")
			}
		})
	}
}

func TestParseMediaHLSIVTrimsWhitespaceLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/main.m3u8", currentURL: "https://example.com/main.m3u8", baseURL: "https://example.com/main.m3u8", rawFiles: map[string]string{}}
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:8
#EXT-X-KEY:METHOD=AES-128,URI="base64:MDEyMzQ1Njc4OWFiY2RlZg==",IV=" 0x00000000000000000000000000000001 "
#EXTINF:8.0,
0.ts
#EXT-X-ENDLIST
`
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 || len(segs[0].Encrypt.IV) != 16 || segs[0].Encrypt.IV[15] != 1 {
		t.Fatalf("HLS IV should trim whitespace before hex parsing like upstream, got %#v", segs)
	}
}

func TestParseMediaRelativeURLsTrimWhitespaceLikeUpstreamUri(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/a/main.m3u8", currentURL: "https://example.com/a/main.m3u8", baseURL: "https://example.com/a/main.m3u8", rawFiles: map[string]string{}}
	raw := "#EXTM3U\n" +
		"#EXT-X-TARGETDURATION:4\n" +
		"#EXT-X-MAP:URI=\" init.mp4 \"\n" +
		"#EXTINF:4.0,\n" +
		"\tseg.m4s \n" +
		"#EXT-X-ENDLIST\n"
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if pl.MediaInit == nil || pl.MediaInit.URL != "https://example.com/a/init.mp4" {
		t.Fatalf("EXT-X-MAP URI whitespace should be trimmed like upstream Uri, got %#v", pl.MediaInit)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 || segs[0].URL != "https://example.com/a/seg.m4s" {
		t.Fatalf("segment URL whitespace should be trimmed like upstream Uri, got %#v", segs)
	}
}

func TestParseMediaRelativeBackslashURLsMatchUpstreamUri(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/a/main.m3u8", currentURL: "https://example.com/a/main.m3u8", baseURL: "https://example.com/a/main.m3u8", rawFiles: map[string]string{}}
	raw := "#EXTM3U\n" +
		"#EXT-X-TARGETDURATION:4\n" +
		"#EXT-X-MAP:URI=\".\\init.mp4?token=a\\b\"\n" +
		"#EXTINF:4.0,\n" +
		"dir\\seg.m4s\n" +
		"#EXT-X-ENDLIST\n"
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if pl.MediaInit == nil || pl.MediaInit.URL != `https://example.com/a/init.mp4?token=a\b` {
		t.Fatalf("EXT-X-MAP path backslashes should normalize but query should stay untouched, got %#v", pl.MediaInit)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 || segs[0].URL != "https://example.com/a/dir/seg.m4s" {
		t.Fatalf("segment path backslashes should match upstream Uri, got %#v", segs)
	}
}

func TestParseMediaSameSchemeRelativeURLsMatchUpstreamUri(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/a/main.m3u8", currentURL: "https://example.com/a/main.m3u8", baseURL: "https://example.com/a/main.m3u8", rawFiles: map[string]string{}}
	raw := "#EXTM3U\n" +
		"#EXT-X-TARGETDURATION:4\n" +
		"#EXT-X-MAP:URI=\"https:/init.mp4\"\n" +
		"#EXTINF:4.0,\n" +
		"https:dir/seg.m4s\n" +
		"#EXT-X-ENDLIST\n"
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if pl.MediaInit == nil || pl.MediaInit.URL != "https://example.com/init.mp4" {
		t.Fatalf("same-scheme root-relative map URI should match upstream Uri, got %#v", pl.MediaInit)
	}
	segs := sortedSegments(pl)
	if len(segs) != 1 || segs[0].URL != "https://example.com/a/dir/seg.m4s" {
		t.Fatalf("same-scheme relative segment URI should match upstream Uri, got %#v", segs)
	}
}

func TestParseMediaURLDisplayEscapesMatchUpstreamUri(t *testing.T) {
	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/a/main.m3u8", currentURL: "https://example.com/a/main.m3u8", baseURL: "https://example.com/a/main.m3u8", rawFiles: map[string]string{}}
	raw := "#EXTM3U\n" +
		"#EXT-X-TARGETDURATION:4\n" +
		"#EXT-X-MAP:URI=\"init%20文件%7E%41.mp4?token=a%20b&safe=a%2Fb\"\n" +
		"#EXTINF:4.0,\n" +
		"seg%20中文.ts?x=a%20b&path=a%2Fb&next=a%3Fb&frag=a%23b&pct=a%25b\n" +
		"#EXT-X-ENDLIST\n"
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if pl.MediaInit == nil || pl.MediaInit.URL != "https://example.com/a/init 文件~A.mp4?token=a b&safe=a%2Fb" {
		t.Fatalf("EXT-X-MAP display escapes should match upstream Uri.ToString, got %#v", pl.MediaInit)
	}
	segs := sortedSegments(pl)
	want := "https://example.com/a/seg 中文.ts?x=a b&path=a%2Fb&next=a%3Fb&frag=a%23b&pct=a%25b"
	if len(segs) != 1 || segs[0].URL != want {
		t.Fatalf("segment URL display escapes should match upstream Uri.ToString:\nwant %s\ngot  %#v", want, segs)
	}
}

func TestParseMediaMultipleExtMapMatchesUpstream(t *testing.T) {
	raw := `#EXTM3U
#EXT-X-TARGETDURATION:4
#EXT-X-MAP:URI="init-a.mp4"
#EXTINF:4.0,
a.m4s
#EXT-X-MAP:URI="init-b.mp4"
#EXTINF:4.0,
b.m4s
#EXT-X-ENDLIST
`

	opt := defaultOptions()
	p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/v/main.m3u8", currentURL: "https://example.com/v/main.m3u8", baseURL: "https://example.com/v/main.m3u8", rawFiles: map[string]string{}}
	pl, err := p.parseMedia(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(sortedSegments(pl)); got != 1 {
		t.Fatalf("default parser should stop at the second EXT-X-MAP, got %d segments", got)
	}
	if len(pl.Parts) != 1 || len(pl.Parts[0].Segments) != 1 {
		t.Fatalf("unexpected default parts: %#v", pl.Parts)
	}

	opt.AllowHLSMultiExtMap = true
	opt.UILanguage = "en-US"
	p = &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/v/main.m3u8", currentURL: "https://example.com/v/main.m3u8", baseURL: "https://example.com/v/main.m3u8", rawFiles: map[string]string{}}
	output := captureStdout(t, func() {
		pl, err = p.parseMedia(context.Background(), raw)
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "Multiple #EXT-X-MAP tags are now allowed for detection.") {
		t.Fatalf("allow-hls-multi-ext-map should warn like upstream, got %q", output)
	}
	if got := len(sortedSegments(pl)); got != 2 {
		t.Fatalf("allow-hls-multi-ext-map should keep later segments, got %d", got)
	}
	if len(pl.Parts) != 2 || len(pl.Parts[0].Segments) != 1 || len(pl.Parts[1].Segments) != 1 {
		t.Fatalf("multi-map playlists should split parts at the new EXT-X-MAP like upstream: %#v", pl.Parts)
	}
}

func TestParseMediaMalformedQuotedMapAttributesFailLikeUpstream(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "uri",
			raw: `#EXTM3U
#EXT-X-TARGETDURATION:4
#EXT-X-MAP:URI="init.mp4
#EXTINF:4.0,
seg.m4s
#EXT-X-ENDLIST
`,
		},
		{
			name: "byterange",
			raw: `#EXTM3U
#EXT-X-TARGETDURATION:4
#EXT-X-MAP:URI="init.mp4",BYTERANGE="100@0
#EXTINF:4.0,
seg.m4s
#EXT-X-ENDLIST
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := defaultOptions()
			p := &parser{opt: opt, client: http.DefaultClient, originalURL: "https://example.com/v/main.m3u8", currentURL: "https://example.com/v/main.m3u8", baseURL: "https://example.com/v/main.m3u8", rawFiles: map[string]string{}}
			if _, err := p.parseMedia(context.Background(), tt.raw); err == nil {
				t.Fatal("malformed quoted EXT-X-MAP attribute should fail like upstream GetAttribute")
			}
		})
	}
}

func TestFetchPlaylistRefreshesURLFromMasterOnFailure(t *testing.T) {
	var masterHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/master.m3u8":
			masterHits++
			child := "old.m3u8"
			if masterHits > 1 {
				child = "new.m3u8"
			}
			_, _ = w.Write([]byte(`#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=2000,RESOLUTION=1280x720,CODECS="avc1.4d401f"
` + child + `
`))
		case "/old.m3u8":
			http.NotFound(w, r)
		case "/new.m3u8":
			_, _ = w.Write([]byte(`#EXTM3U
#EXT-X-TARGETDURATION:4
#EXTINF:4.0,
seg.ts
#EXT-X-ENDLIST
`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	opt := defaultOptions()
	opt.Input = srv.URL + "/master.m3u8"
	streams, p, err := parseSource(context.Background(), srv.Client(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 || streams[0].URL != srv.URL+"/old.m3u8" {
		t.Fatalf("unexpected initial streams: %#v", streams)
	}
	if err := p.fetchPlaylist(context.Background(), &streams[0]); err != nil {
		t.Fatal(err)
	}
	if streams[0].URL != srv.URL+"/new.m3u8" {
		t.Fatalf("playlist URL should refresh from master, got %s", streams[0].URL)
	}
	if got := len(sortedSegments(streams[0].Playlist)); got != 1 {
		t.Fatalf("expected refreshed playlist to parse one segment, got %d", got)
	}
	if masterHits != 2 {
		t.Fatalf("expected master to be fetched twice, got %d", masterHits)
	}
}

func TestFetchPlaylistPreProcessesChildPlaylistLikeUpstream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/master.m3u8":
			_, _ = w.Write([]byte(`#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=2000,RESOLUTION=1280x720,CODECS="avc1.4d401f"
child.m3u8
`))
		case "/child.m3u8":
			_, _ = w.Write([]byte(`#EXTM3U
#EXT-X-TARGETDURATION:4
#EXTINF:4.0,
#EXT-X-KEY:METHOD=AES-128,URI="data:;base64,AAAAAAAAAAAAAAAAAAAAAA=="
seg.ts
#EXT-X-ENDLIST
`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	opt := defaultOptions()
	opt.Input = srv.URL + "/master.m3u8"
	streams, p, err := parseSource(context.Background(), srv.Client(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 {
		t.Fatalf("unexpected initial streams: %#v", streams)
	}
	if err := p.fetchPlaylist(context.Background(), &streams[0]); err != nil {
		t.Fatal(err)
	}
	segs := sortedSegments(streams[0].Playlist)
	if len(segs) != 1 {
		t.Fatalf("expected one child playlist segment, got %#v", segs)
	}
	if segs[0].Encrypt.Method != EncryptAES128 {
		t.Fatalf("child playlist should run HLS preprocessor before parse so KEY applies to EXTINF, got %#v", segs[0].Encrypt)
	}
}

func TestFetchPlaylistPreservesExistingMediaInitLikeUpstream(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits == 1 {
			_, _ = w.Write([]byte(`#EXTM3U
#EXT-X-TARGETDURATION:4
#EXT-X-MAP:URI="init-a.mp4"
#EXTINF:4.0,
a.m4s
`))
			return
		}
		_, _ = w.Write([]byte(`#EXTM3U
#EXT-X-TARGETDURATION:9
#EXT-X-MEDIA-SEQUENCE:1
#EXTINF:4.0,
b.m4s
`))
	}))
	defer srv.Close()

	opt := defaultOptions()
	p := &parser{opt: opt, client: srv.Client(), originalURL: srv.URL + "/live.m3u8", currentURL: srv.URL + "/live.m3u8", baseURL: srv.URL + "/live.m3u8", rawFiles: map[string]string{}}
	stream := StreamSpec{URL: srv.URL + "/live.m3u8"}
	if err := p.fetchPlaylist(context.Background(), &stream); err != nil {
		t.Fatal(err)
	}
	if stream.Playlist == nil || stream.Playlist.MediaInit == nil {
		t.Fatalf("first playlist should contain init: %#v", stream.Playlist)
	}
	firstInit := stream.Playlist.MediaInit.URL
	firstTargetDuration := stream.Playlist.TargetDuration
	if err := p.fetchPlaylist(context.Background(), &stream); err != nil {
		t.Fatal(err)
	}
	if stream.Playlist == nil || stream.Playlist.MediaInit == nil || stream.Playlist.MediaInit.URL != firstInit {
		t.Fatalf("refresh should preserve existing init, got %#v want %s", stream.Playlist, firstInit)
	}
	if stream.Playlist.TargetDuration != firstTargetDuration {
		t.Fatalf("refresh with existing init should only replace media parts like upstream, target duration got %v want %v", stream.Playlist.TargetDuration, firstTargetDuration)
	}
	if stream.Extension != "m4s" {
		t.Fatalf("refresh without EXT-X-MAP should still keep fMP4 extension, got %s", stream.Extension)
	}
	segs := sortedSegments(stream.Playlist)
	if len(segs) != 1 || !strings.HasSuffix(segs[0].URL, "/b.m4s") {
		t.Fatalf("refresh should replace media parts with new segments, got %#v", segs)
	}
}

func TestSubtitleExtPrefersVTTWhenPlaylistContainsBothLikeUpstream(t *testing.T) {
	pl := &Playlist{Parts: []MediaPart{{Segments: []Segment{
		{URL: "https://example.com/subtitle.ttml"},
		{URL: "https://example.com/subtitle.vtt"},
	}}}}
	if got := subtitleExt(pl); got != "vtt" {
		t.Fatalf("subtitle extension should let VTT override TTML like upstream, got %s", got)
	}
}
