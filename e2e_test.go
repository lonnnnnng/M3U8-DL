package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDownloadAndBinaryMerge(t *testing.T) {
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/main.m3u8":
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
			_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXT-X-MEDIA-SEQUENCE:0\n#EXTINF:1,\n" + base + "/0.ts\n#EXTINF:1,\n" + base + "/1.ts\n#EXT-X-ENDLIST\n"))
		case "/0.ts":
			_, _ = w.Write([]byte("hello"))
		case "/1.ts":
			_, _ = w.Write([]byte("world"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	tmp := t.TempDir()
	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	opt.AutoSelect = true
	opt.BinaryMerge = true
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "sample"

	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	streams, p, err := parseSource(context.Background(), client, opt)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := chooseStreams(streams, opt)
	if err != nil {
		t.Fatal(err)
	}
	if selected[0].Playlist == nil {
		if err := p.fetchPlaylist(context.Background(), &selected[0]); err != nil {
			t.Fatal(err)
		}
	}
	outs, err := downloadAll(context.Background(), client, selected, opt)
	if err != nil {
		t.Fatal(err)
	}
	if len(outs) != 1 {
		t.Fatalf("want 1 output, got %d", len(outs))
	}
	b, err := os.ReadFile(outs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "helloworld" {
		t.Fatalf("merged content wrong: %q", string(b))
	}
}

func TestDownloadLiveRealTimeMergeUsesAppendPath(t *testing.T) {
	tmp := t.TempDir()
	segA := filepath.Join(tmp, "a.ts")
	segB := filepath.Join(tmp, "b.ts")
	if err := os.WriteFile(segA, []byte("live-a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(segB, []byte("live-b"), 0644); err != nil {
		t.Fatal(err)
	}
	video := MediaVideo
	stream := StreamSpec{
		ID:        1,
		Extension: "ts",
		MediaType: &video,
		Playlist: &Playlist{IsLive: true, Parts: []MediaPart{{Segments: []Segment{
			{Index: 1, URL: (&url.URL{Scheme: "file", Path: segA}).String(), Duration: 1},
			{Index: 2, URL: (&url.URL{Scheme: "file", Path: segB}).String(), Duration: 1},
		}}}},
	}
	opt := defaultOptions()
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "live"
	opt.LiveRealTimeMerge = true
	opt.LiveKeepSegments = false
	opt.DelAfterDone = false
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	outs, err := downloadAll(context.Background(), client, []StreamSpec{stream}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if len(outs) != 1 {
		t.Fatalf("want 1 output, got %d", len(outs))
	}
	got, err := os.ReadFile(outs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "live-alive-b" {
		t.Fatalf("live realtime merge output mismatch: %q", got)
	}
	if matches, err := filepath.Glob(filepath.Join(opt.TmpDir, "*_tmp", "*.ts")); err != nil {
		t.Fatal(err)
	} else if len(matches) != 0 {
		t.Fatalf("live realtime merge should delete downloaded media segments when keep=false: %#v", matches)
	}
}

func TestDownloadLiveRealtimeRefreshesAndAppendsIncrementally(t *testing.T) {
	var playlistHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/main.m3u8":
			playlistHits++
			if playlistHits == 1 {
				_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXT-X-MEDIA-SEQUENCE:0\n#EXTINF:1,\n0.ts\n"))
				return
			}
			_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXT-X-MEDIA-SEQUENCE:0\n#EXTINF:1,\n0.ts\n#EXTINF:1,\n1.ts\n"))
		case "/0.ts":
			_, _ = w.Write([]byte("a"))
		case "/1.ts":
			_, _ = w.Write([]byte("b"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tmp := t.TempDir()
	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	opt.AutoSelect = true
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "live-refresh"
	opt.LiveRealTimeMerge = true
	opt.LiveKeepSegments = false
	limit := 2 * time.Second
	opt.LiveRecordLimit = &limit
	wait := 0
	opt.LiveWaitTime = &wait
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	streams, p, err := parseSource(context.Background(), client, opt)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := chooseStreams(streams, opt)
	if err != nil {
		t.Fatal(err)
	}
	outs, handled, err := downloadLiveRealtimeIfNeeded(context.Background(), client, selected, p, opt)
	if err != nil {
		t.Fatal(err)
	}
	if !handled {
		t.Fatal("live realtime record path should handle this stream")
	}
	if playlistHits < 2 {
		t.Fatalf("live recorder should refresh playlist while downloading, hits=%d", playlistHits)
	}
	if len(outs) != 1 {
		t.Fatalf("want 1 output, got %d", len(outs))
	}
	got, err := os.ReadFile(outs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ab" {
		t.Fatalf("incremental live output mismatch: %q", got)
	}
}

func TestDownloadLivePipeMuxWritesRefreshBatchesToPipe(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this fake ffmpeg shell script test is Unix-only; Windows named pipe runtime is covered by live_pipe_windows_test.go")
	}
	var playlistHits int
	var segHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/main.m3u8":
			playlistHits++
			if playlistHits == 1 {
				_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXT-X-MEDIA-SEQUENCE:0\n#EXTINF:1,\n0.ts\n"))
				return
			}
			_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXT-X-MEDIA-SEQUENCE:0\n#EXTINF:1,\n0.ts\n#EXTINF:1,\n1.ts\n"))
		case "/0.ts":
			segHits++
			_, _ = w.Write([]byte("a"))
		case "/1.ts":
			segHits++
			_, _ = w.Write([]byte("b"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tmp := t.TempDir()
	tool := filepath.Join(tmp, "ffmpeg")
	script := `#!/bin/sh
pipes=""
last=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "-i" ]; then
    pipes="$pipes $arg"
  fi
  prev="$arg"
  last="$arg"
done
: > "$last"
for p in $pipes; do
  cat "$p" >> "$last"
done
`
	if err := os.WriteFile(tool, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envLivePipeTmpDir, tmp)
	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	opt.AutoSelect = true
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "pipe-refresh"
	opt.LiveRealTimeMerge = true
	opt.LivePipeMux = true
	opt.LiveKeepSegments = false
	opt.FFmpegBinaryPath = tool
	limit := 2 * time.Second
	opt.LiveRecordLimit = &limit
	wait := 0
	opt.LiveWaitTime = &wait
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	streams, p, err := parseSource(context.Background(), client, opt)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := chooseStreams(streams, opt)
	if err != nil {
		t.Fatal(err)
	}
	outs, handled, err := downloadLiveRealtimeIfNeeded(context.Background(), client, selected, p, opt)
	if err != nil {
		t.Fatal(err)
	}
	if !handled {
		t.Fatal("live pipe mux path should handle this stream")
	}
	if len(outs) != 1 {
		t.Fatalf("pipe mux should replace live media outputs, got %#v", outs)
	}
	got, err := os.ReadFile(outs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ab" {
		t.Fatalf("pipe mux output mismatch: %q", got)
	}
	if playlistHits < 2 {
		t.Fatalf("live pipe mux should refresh playlist, hits=%d", playlistHits)
	}
	if segHits != 2 {
		t.Fatalf("live pipe mux should not re-download media after pipe write, segment hits=%d", segHits)
	}
}

func TestSingleLargeSegmentUsesRangeSplitting(t *testing.T) {
	payload := make([]byte, largeSingleFileSplitSize+17)
	for i := range payload {
		payload[i] = byte(i % 251)
	}
	var base string
	var mu sync.Mutex
	var ranges []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/main.m3u8":
			_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\n" + base + "/single.ts\n#EXT-X-ENDLIST\n"))
		case "/single.ts":
			if r.Method == http.MethodHead {
				w.Header().Set("Accept-Ranges", "bytes")
				w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
				return
			}
			rangeHeader := r.Header.Get("Range")
			mu.Lock()
			ranges = append(ranges, rangeHeader)
			mu.Unlock()
			if rangeHeader == "" {
				t.Errorf("single large segment GET should use Range")
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			start, end, err := parseTestRange(rangeHeader)
			if err != nil {
				t.Errorf("bad range %q: %v", rangeHeader, err)
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(payload)))
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(payload[start : end+1])
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	tmp := t.TempDir()
	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	opt.AutoSelect = true
	opt.BinaryMerge = true
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "single-large"
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	streams, _, err := parseSource(context.Background(), client, opt)
	if err != nil {
		t.Fatal(err)
	}
	outs, err := downloadAll(context.Background(), client, streams, opt)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("range split output does not match original payload")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(ranges) < 2 {
		t.Fatalf("expected at least two range requests, got %#v", ranges)
	}
	if !containsString(ranges, "bytes=0-10485759") {
		t.Fatalf("expected first split range, got %#v", ranges)
	}
}

func TestProbeRangeSizeAcceptsAnyAcceptRangesHeaderLikeUpstream(t *testing.T) {
	payloadSize := int64(largeSingleFileSplitSize + 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Accept-Ranges", "none")
		w.Header().Set("Content-Length", strconv.FormatInt(payloadSize, 10))
	}))
	defer srv.Close()

	size, ok, err := probeRangeSize(context.Background(), srv.Client(), srv.URL+"/single.ts", defaultOptions().Headers)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || size != payloadSize {
		t.Fatalf("Accept-Ranges presence should be enough like upstream, ok=%v size=%d", ok, size)
	}
}

func TestConcurrentDownloadKeepsInputStreamOrder(t *testing.T) {
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/slow.ts":
			time.Sleep(150 * time.Millisecond)
			_, _ = w.Write([]byte("slow-first"))
		case "/fast.ts":
			_, _ = w.Write([]byte("fast-second"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	tmp := t.TempDir()
	opt := defaultOptions()
	opt.ConcurrentDownload = true
	opt.ThreadCount = 2
	opt.BinaryMerge = true
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "ordered"
	streams := []StreamSpec{
		{ID: 0, Language: "video", Extension: "ts", Playlist: &Playlist{Parts: []MediaPart{{Segments: []Segment{{Index: 0, URL: base + "/slow.ts", Duration: 1}}}}}},
		{ID: 1, Language: "audio", Extension: "ts", Playlist: &Playlist{Parts: []MediaPart{{Segments: []Segment{{Index: 0, URL: base + "/fast.ts", Duration: 1}}}}}},
	}
	outs, err := downloadAll(context.Background(), srv.Client(), streams, opt)
	if err != nil {
		t.Fatal(err)
	}
	if len(outs) != 2 {
		t.Fatalf("expected 2 outputs, got %#v", outs)
	}
	first, err := os.ReadFile(outs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(outs[1].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != "slow-first" || string(second) != "fast-second" {
		t.Fatalf("concurrent output order should follow input stream order, got %q then %q", first, second)
	}
}

func TestDownloadRawGzipSegmentDecompresses(t *testing.T) {
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	if _, err := zw.Write([]byte("ddp-audio-payload")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/main.m3u8":
			_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\n" + base + "/audio.bin\n#EXT-X-ENDLIST\n"))
		case "/audio.bin":
			_, _ = w.Write(gz.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	tmp := t.TempDir()
	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	opt.AutoSelect = true
	opt.BinaryMerge = true
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "raw-gzip"
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	streams, _, err := parseSource(context.Background(), client, opt)
	if err != nil {
		t.Fatal(err)
	}
	outs, err := downloadAll(context.Background(), client, streams, opt)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ddp-audio-payload" {
		t.Fatalf("gzip payload not decompressed: %q", got)
	}
}

func TestDownloadDeflateEncodedSegmentDecompressesLikeUpstream(t *testing.T) {
	deflated := zlibBytes(t, []byte("deflate-http-payload"))
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/main.m3u8":
			_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\n" + base + "/audio.bin\n#EXT-X-ENDLIST\n"))
		case "/audio.bin":
			w.Header().Set("Content-Encoding", "deflate")
			_, _ = w.Write(deflated)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	tmp := t.TempDir()
	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	opt.AutoSelect = true
	opt.BinaryMerge = true
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "deflate-segment"
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	streams, _, err := parseSource(context.Background(), client, opt)
	if err != nil {
		t.Fatal(err)
	}
	outs, err := downloadAll(context.Background(), client, streams, opt)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "deflate-http-payload" {
		t.Fatalf("deflate response payload should be decompressed: %q", got)
	}
}

func TestDownloadBrotliEncodedSegmentDecompressesLikeUpstream(t *testing.T) {
	compressed := brotliBytes(t, []byte("brotli-http-payload"))
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/main.m3u8":
			_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\n" + base + "/audio.bin\n#EXT-X-ENDLIST\n"))
		case "/audio.bin":
			w.Header().Set("Content-Encoding", "br")
			_, _ = w.Write(compressed)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	tmp := t.TempDir()
	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	opt.AutoSelect = true
	opt.BinaryMerge = true
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "brotli-segment"
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	streams, _, err := parseSource(context.Background(), client, opt)
	if err != nil {
		t.Fatal(err)
	}
	outs, err := downloadAll(context.Background(), client, streams, opt)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "brotli-http-payload" {
		t.Fatalf("brotli response payload should be decompressed: %q", got)
	}
}

func TestDownloadImageHeaderSegmentStripsGIFHeader(t *testing.T) {
	header := append([]byte("GIF8"), bytes.Repeat([]byte{0}, 38)...)
	wrapped := append(header, []byte("ts-payload")...)
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/main.m3u8":
			_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\n" + base + "/image.ts\n#EXT-X-ENDLIST\n"))
		case "/image.ts":
			_, _ = w.Write(wrapped)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	tmp := t.TempDir()
	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	opt.AutoSelect = true
	opt.BinaryMerge = true
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "image-header"
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	streams, _, err := parseSource(context.Background(), client, opt)
	if err != nil {
		t.Fatal(err)
	}
	outs, err := downloadAll(context.Background(), client, streams, opt)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ts-payload" {
		t.Fatalf("image header not stripped: %q", got)
	}
}

func TestDownloadImageHeaderSegmentStripsPNGFallbackAtTSSync(t *testing.T) {
	header := append([]byte{137, 80, 78, 71}, bytes.Repeat([]byte{0}, 21)...)
	payload := tsSyncPayload([]byte("png-ts-payload"))
	wrapped := append(header, payload...)
	got := stripImageHeader(wrapped)
	if !bytes.Equal(got, payload) {
		t.Fatalf("png fallback did not strip to TS sync: got offset payload %q", got[:min(len(got), len(payload))])
	}
}

func TestDownloadImageHeaderSegmentStripsJPEGFallbackAtTSSync(t *testing.T) {
	header := append([]byte{0xff, 0xd8, 0xff, 0xe0}, bytes.Repeat([]byte{0}, 17)...)
	payload := tsSyncPayload([]byte("jpeg-ts-payload"))
	wrapped := append(header, payload...)
	got := stripImageHeader(wrapped)
	if !bytes.Equal(got, payload) {
		t.Fatalf("jpeg fallback did not strip to TS sync: got offset payload %q", got[:min(len(got), len(payload))])
	}
}

func tsSyncPayload(tail []byte) []byte {
	packet := bytes.Repeat([]byte{0}, 188)
	packet[0] = 0x47
	payload := append(append(append([]byte{}, packet...), packet...), packet...)
	return append(payload, tail...)
}

func TestDownloadBase64AndHexSegments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/main.m3u8":
			_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\nbase64://aGVsbG8=\n#EXTINF:1,\nhex://776f726c64\n#EXT-X-ENDLIST\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tmp := t.TempDir()
	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	opt.AutoSelect = true
	opt.BinaryMerge = true
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "inline"
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	streams, _, err := parseSource(context.Background(), client, opt)
	if err != nil {
		t.Fatal(err)
	}
	outs, err := downloadAll(context.Background(), client, streams, opt)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "helloworld" {
		t.Fatalf("inline segments not merged: %q", got)
	}
}

func TestDownloadMissingSegmentFailsWhenCheckEnabled(t *testing.T) {
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/main.m3u8":
			_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\n" + base + "/0.ts\n#EXTINF:1,\n" + base + "/missing.ts\n#EXT-X-ENDLIST\n"))
		case "/0.ts":
			_, _ = w.Write([]byte("ok"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	tmp := t.TempDir()
	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	opt.AutoSelect = true
	opt.BinaryMerge = true
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "strict"
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	streams, _, err := parseSource(context.Background(), client, opt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := downloadAll(context.Background(), client, streams, opt); err == nil {
		t.Fatal("expected missing segment to fail when check is enabled")
	}
}

func TestDownloadByteRangeShortReadFails(t *testing.T) {
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/main.m3u8":
			_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXT-X-BYTERANGE:10@0\n#EXTINF:1,\n" + base + "/media.ts\n#EXT-X-ENDLIST\n"))
		case "/media.ts":
			if r.Header.Get("Range") != "bytes=0-9" {
				t.Errorf("unexpected range header: %s", r.Header.Get("Range"))
			}
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write([]byte("short"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	tmp := t.TempDir()
	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	opt.AutoSelect = true
	opt.BinaryMerge = true
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "short-range"
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	streams, _, err := parseSource(context.Background(), client, opt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := downloadAll(context.Background(), client, streams, opt); err == nil {
		t.Fatal("expected short BYTERANGE response to fail")
	}
}

func TestDownloadSegmentReusesExistingDecryptedFile(t *testing.T) {
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		http.Error(w, "should not fetch", http.StatusInternalServerError)
	}))
	defer srv.Close()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "000.m4s")
	dec := filepath.Join(tmp, "000_dec.m4s")
	if err := os.WriteFile(dec, []byte("decrypted"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := downloadSegment(context.Background(), srv.Client(), Segment{URL: srv.URL + "/seg.m4s"}, path, defaultOptions(), nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != dec {
		t.Fatalf("expected decrypted segment path, got %s", got)
	}
	if hit {
		t.Fatal("existing decrypted segment should skip network fetch")
	}
}

func TestDownloadSubtitleFixRemovesSourceSegmentsLikeUpstream(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "source.vtt")
	if err := os.WriteFile(source, []byte("WEBVTT\n\n00:00:00.000 --> 00:00:01.000\nhello\n\n"), 0644); err != nil {
		t.Fatal(err)
	}
	sourceURL, err := localFileURL(source)
	if err != nil {
		t.Fatal(err)
	}
	sub := MediaSubtitles
	stream := StreamSpec{
		ID:        1,
		MediaType: &sub,
		Extension: "vtt",
		Playlist:  &Playlist{Parts: []MediaPart{{Segments: []Segment{{Index: 0, URL: sourceURL, Duration: 1}}}}},
	}
	opt := defaultOptions()
	opt.SaveDir = filepath.Join(tmp, "out")
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "sub-clean"
	opt.DelAfterDone = false
	out, err := downloadStream(context.Background(), http.DefaultClient, stream, opt, newRateLimiter(0), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out.Path); err != nil {
		t.Fatal(err)
	}
	if files := listRegularFiles(t, opt.TmpDir); len(files) != 0 {
		t.Fatalf("fixed VTT source segments should be removed like upstream, got %#v", files)
	}
}

func TestDownloadTTMLFixHonorsKeepImageSegmentsEnv(t *testing.T) {
	t.Setenv("RE_KEEP_IMAGE_SEGMENTS", "1")
	tmp := t.TempDir()
	source := filepath.Join(tmp, "source.ttml")
	xml := `<tt xmlns="http://www.w3.org/ns/ttml"><body><div><p begin="00:00:00.000" end="00:00:01.000">hello</p></div></body></tt>`
	if err := os.WriteFile(source, []byte(xml), 0644); err != nil {
		t.Fatal(err)
	}
	sourceURL, err := localFileURL(source)
	if err != nil {
		t.Fatal(err)
	}
	sub := MediaSubtitles
	stream := StreamSpec{
		ID:        2,
		MediaType: &sub,
		Extension: "ttml",
		Playlist:  &Playlist{Parts: []MediaPart{{Segments: []Segment{{Index: 0, URL: sourceURL, Duration: 1}}}}},
	}
	opt := defaultOptions()
	opt.SaveDir = filepath.Join(tmp, "out")
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "sub-keep"
	opt.DelAfterDone = false
	out, err := downloadStream(context.Background(), http.DefaultClient, stream, opt, newRateLimiter(0), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out.Path); err != nil {
		t.Fatal(err)
	}
	if files := listRegularFiles(t, opt.TmpDir); len(files) == 0 {
		t.Fatal("RE_KEEP_IMAGE_SEGMENTS=1 should preserve TTML source segment like upstream")
	}
}

func listRegularFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return files
}

func TestDownloadRedirectPreservesHeadersAndRange(t *testing.T) {
	var base string
	var hitTarget bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/main.m3u8":
			_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXT-X-BYTERANGE:4@1\n#EXTINF:1,\n" + base + "/redirect.ts\n#EXT-X-ENDLIST\n"))
		case "/redirect.ts":
			http.Redirect(w, r, "/target.ts", http.StatusFound)
		case "/target.ts":
			hitTarget = true
			if r.Header.Get("Range") != "bytes=1-4" {
				t.Errorf("redirected request lost Range header: %s", r.Header.Get("Range"))
			}
			if r.Header.Get("Authorization") != "Bearer abc" {
				t.Errorf("redirected request lost Authorization header: %s", r.Header.Get("Authorization"))
			}
			if r.Header.Get("Cookie") != "sid=xyz" {
				t.Errorf("redirected request lost Cookie header: %s", r.Header.Get("Cookie"))
			}
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write([]byte("bcde"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	tmp := t.TempDir()
	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	opt.AutoSelect = true
	opt.BinaryMerge = true
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "redirect-range"
	opt.Headers["authorization"] = "Bearer abc"
	opt.Headers["cookie"] = "sid=xyz"
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	streams, _, err := parseSource(context.Background(), client, opt)
	if err != nil {
		t.Fatal(err)
	}
	outs, err := downloadAll(context.Background(), client, streams, opt)
	if err != nil {
		t.Fatal(err)
	}
	if !hitTarget {
		t.Fatal("redirect target was not requested")
	}
	got, err := os.ReadFile(outs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "bcde" {
		t.Fatalf("unexpected redirected range payload: %q", got)
	}
}

func TestDownloadMissingSegmentContinuesWhenCheckDisabled(t *testing.T) {
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/main.m3u8":
			_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\n" + base + "/0.ts\n#EXTINF:1,\n" + base + "/missing.ts\n#EXTINF:1,\n" + base + "/1.ts\n#EXT-X-ENDLIST\n"))
		case "/0.ts":
			_, _ = w.Write([]byte("hello"))
		case "/1.ts":
			_, _ = w.Write([]byte("world"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	tmp := t.TempDir()
	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	opt.AutoSelect = true
	opt.BinaryMerge = true
	opt.CheckSegmentsCount = false
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "tolerant"
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	streams, _, err := parseSource(context.Background(), client, opt)
	if err != nil {
		t.Fatal(err)
	}
	outs, err := downloadAll(context.Background(), client, streams, opt)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "helloworld" {
		t.Fatalf("unexpected tolerant merge: %q", got)
	}
}

func TestDownloadAES128(t *testing.T) {
	key := []byte("0123456789abcdef")
	iv := make([]byte, aes.BlockSize)
	plain := []byte("encrypted-segment")
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	encrypted := pkcs7Pad(plain, aes.BlockSize)
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(encrypted, encrypted)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/main.m3u8":
			_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXT-X-KEY:METHOD=AES-128,URI=\"/key.bin\",IV=0x00000000000000000000000000000000\n#EXTINF:1,\nseg.ts\n#EXT-X-ENDLIST\n"))
		case "/key.bin":
			_, _ = w.Write(key)
		case "/seg.ts":
			_, _ = w.Write(encrypted)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tmp := t.TempDir()
	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	opt.AutoSelect = true
	opt.BinaryMerge = true
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "aes"
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	streams, _, err := parseSource(context.Background(), client, opt)
	if err != nil {
		t.Fatal(err)
	}
	outs, err := downloadAll(context.Background(), client, streams, opt)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("decrypt got %q", got)
	}
}

func TestDownloadAES128GzipSegmentProcessesPayloadAfterDecrypt(t *testing.T) {
	key := []byte("0123456789abcdef")
	iv := make([]byte, aes.BlockSize)
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	if _, err := zw.Write([]byte("decrypted-gzip-payload")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	encrypted := pkcs7Pad(gz.Bytes(), aes.BlockSize)
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(encrypted, encrypted)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/main.m3u8":
			_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXT-X-KEY:METHOD=AES-128,URI=\"/key.bin\",IV=0x00000000000000000000000000000000\n#EXTINF:1,\nseg.ts\n#EXT-X-ENDLIST\n"))
		case "/key.bin":
			_, _ = w.Write(key)
		case "/seg.ts":
			_, _ = w.Write(encrypted)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tmp := t.TempDir()
	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	opt.AutoSelect = true
	opt.BinaryMerge = true
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "aes-gzip"
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	streams, _, err := parseSource(context.Background(), client, opt)
	if err != nil {
		t.Fatal(err)
	}
	outs, err := downloadAll(context.Background(), client, streams, opt)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "decrypted-gzip-payload" {
		t.Fatalf("AES gzip payload should be processed after decrypt, got %q", got)
	}
}

func TestDownloadUnknownHLSEncryptionKeepsRawAndForcesBinaryMerge(t *testing.T) {
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/main.m3u8":
			_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXT-X-KEY:METHOD=FOO\n#EXTINF:1,\n" + base + "/0.ts\n#EXTINF:1,\n" + base + "/1.ts\n#EXT-X-ENDLIST\n"))
		case "/0.ts":
			_, _ = w.Write([]byte("raw-"))
		case "/1.ts":
			_, _ = w.Write([]byte("segment"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	tmp := t.TempDir()
	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	opt.AutoSelect = true
	opt.BinaryMerge = false
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "unknown"
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	streams, _, err := parseSource(context.Background(), client, opt)
	if err != nil {
		t.Fatal(err)
	}
	if !hasUnknownEncryption(streams) {
		t.Fatal("unknown HLS encryption should be detected after parsing")
	}
	if !opt.BinaryMerge {
		// long: 主流程遇到 UNKNOWN 会改写 Options；测试中手动复现这一步，避免依赖 ffmpeg 合并路径掩盖未知加密行为。
		opt.BinaryMerge = true
	}
	outs, err := downloadAll(context.Background(), client, streams, opt)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "raw-segment" {
		t.Fatalf("unknown encryption should keep raw segment bytes, got %q", got)
	}
}

func TestRealtimeExternalDecrypt(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("shell helper is unix-only")
	}
	kidBytes := []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20}
	kid := "1112131415161718191a1b1c1d1e1f20"
	tencPayload := append([]byte{0, 0, 0, 0, 0, 0, 0, 0}, kidBytes...)
	initData := mustMP4Box("moov", mustMP4Box("trak", mustMP4Box("mdia", mustMP4Box("minf", mustMP4Box("stbl", mustMP4Box("encv", mustMP4Box("sinf", mustMP4Box("schi", mustMP4Box("tenc", tencPayload)))))))))
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/main.m3u8":
			_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-KEY:METHOD=CENC,URI=\"data:;base64,AA==\"\n#EXT-X-MAP:URI=\"" + base + "/init.mp4\"\n#EXTINF:1,\n" + base + "/seg.m4s\n#EXT-X-ENDLIST\n"))
		case "/init.mp4":
			_, _ = w.Write(initData)
		case "/seg.m4s":
			_, _ = w.Write([]byte("encrypted-seg"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	tmp := t.TempDir()
	tool := filepath.Join(tmp, "mp4decrypt")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nlast=\"\"\nprev=\"\"\nfor arg in \"$@\"; do prev=\"$last\"; last=\"$arg\"; done\nprintf 'decrypted:%s;' \"$prev\" > \"$last\"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	opt := defaultOptions()
	opt.Input = srv.URL + "/main.m3u8"
	opt.AutoSelect = true
	opt.BinaryMerge = true
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "rt"
	opt.MP4RealTimeDecryption = true
	opt.DecryptionBinaryPath = tool
	opt.Keys = []string{kid + ":00112233445566778899aabbccddeeff"}

	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	streams, _, err := parseSource(context.Background(), client, opt)
	if err != nil {
		t.Fatal(err)
	}
	outs, err := downloadAll(context.Background(), client, streams, opt)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(got), "decrypted:") != 2 {
		t.Fatalf("realtime decrypt should include decrypted init and media segment, got %q", got)
	}
}

func TestDownloadMP4StppSubtitleDetectedFromInit(t *testing.T) {
	stsdPayload := append([]byte{0, 0, 0, 0, 0, 0, 0, 1}, mustMP4Box("stpp", nil)...)
	init := mustMP4Box("moov", mustMP4Box("trak", mustMP4Box("mdia", mustMP4Box("minf", mustMP4Box("stbl", mustMP4Box("stsd", stsdPayload))))))
	xml := `<tt xmlns="http://www.w3.org/ns/ttml"><body><div><p begin="00:00:00.000" end="00:00:01.000">download stpp</p></div></body></tt>`
	segData := mustMP4Box("mdat", []byte(xml))
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/init.mp4":
			_, _ = w.Write(init)
		case "/seg.m4s":
			_, _ = w.Write(segData)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	mt := MediaSubtitles
	stream := StreamSpec{
		ID:        1,
		MediaType: &mt,
		Extension: "m4s",
		Playlist: &Playlist{
			MediaInit: &Segment{Index: -1, URL: base + "/init.mp4"},
			Parts:     []MediaPart{{Segments: []Segment{{Index: 0, URL: base + "/seg.m4s", Duration: 1}}}},
		},
	}
	tmp := t.TempDir()
	opt := defaultOptions()
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "subtitle-stpp"
	opt.SubFormat = "SRT"
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	outs, err := downloadAll(context.Background(), client, []StreamSpec{stream}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if len(outs) != 1 || !strings.HasSuffix(outs[0].Path, ".srt") {
		t.Fatalf("expected fixed srt output, got %#v", outs)
	}
	b, err := os.ReadFile(outs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "download stpp") || !strings.Contains(string(b), "00:00:00,000") {
		t.Fatalf("bad downloaded stpp subtitle:\n%s", b)
	}
}

func parseTestRange(header string) (int64, int64, error) {
	raw := strings.TrimPrefix(header, "bytes=")
	left, right, ok := strings.Cut(raw, "-")
	if !ok {
		return 0, 0, fmt.Errorf("missing dash")
	}
	start, err := strconv.ParseInt(left, 10, 64)
	if err != nil {
		return 0, 0, err
	}
	end, err := strconv.ParseInt(right, 10, 64)
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	pad := blockSize - len(data)%blockSize
	out := append([]byte{}, data...)
	out = append(out, bytes.Repeat([]byte{byte(pad)}, pad)...)
	return out
}
