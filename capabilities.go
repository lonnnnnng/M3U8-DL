package main

import (
	"encoding/json"
	"sort"
)

type capabilitiesReport struct {
	Version      versionInfo         `json:"version"`
	Scope        string              `json:"scope"`
	Capabilities map[string][]string `json:"capabilities"`
	Unsupported  []string            `json:"unsupported"`
}

func runCapabilitiesJSON() string {
	b, _ := json.MarshalIndent(buildCapabilitiesReport(), "", "  ")
	return string(b) + "\n"
}

func buildCapabilitiesReport() capabilitiesReport {
	// long: 桌面端和脚本都从这里读取真实能力边界，避免 UI 展示的开关和 CLI 当前支持项发生漂移。
	capabilities := map[string][]string{
		"protocols": {
			"HLS VOD",
			"HLS live",
			"HLS master playlist",
			"HLS media playlist",
		},
		"inputs": {
			"HTTP/HTTPS URL",
			"file URL",
			"local m3u8 file",
		},
		"hlsFeatures": {
			"EXT-X-STREAM-INF",
			"EXT-X-MEDIA",
			"EXTINF",
			"EXT-X-MAP",
			"EXT-X-BYTERANGE",
			"EXT-X-DISCONTINUITY",
			"EXT-X-PROGRAM-DATE-TIME",
			"EXT-X-ENDLIST",
		},
		"download": {
			"segment concurrency",
			"multi-track concurrency",
			"retry",
			"speed limit",
			"range download",
			"resume existing segments",
			"segment count check",
			"custom range",
		},
		"decryption": {
			"AES-128 CBC",
			"AES-128 ECB",
			"CHACHA20",
			"CENC external decrypt",
			"SAMPLE-AES external decrypt",
			"mp4decrypt",
			"shaka-packager",
			"ffmpeg",
		},
		"muxing": {
			"binary merge",
			"ffmpeg concat protocol",
			"ffmpeg concat demuxer",
			"final mp4 mux",
			"final mkv mux",
			"external mux imports",
		},
		"subtitles": {
			"VTT",
			"SRT conversion",
			"TTML",
			"MP4 WebVTT extract",
			"MP4 TTML extract",
			"bitmap subtitle PNG output",
		},
		"live": {
			"playlist polling",
			"record limit",
			"real-time append merge",
			"pipe mux",
			"VTT audio timeline fix",
		},
		"diagnostics": {
			"doctor text",
			"doctor json",
			"effective options json",
			"probe json",
			"progress json",
			"capabilities json",
		},
		"languages": {
			"zh-CN",
			"zh-TW",
			"en-US",
		},
	}
	for key := range capabilities {
		sort.Strings(capabilities[key])
	}
	return capabilitiesReport{
		Version:      currentVersionInfo(),
		Scope:        "hls-only",
		Capabilities: capabilities,
		Unsupported: []string{
			"DASH",
			"MSS",
			"Live TS",
			"Bilibili page parsing",
		},
	}
}
