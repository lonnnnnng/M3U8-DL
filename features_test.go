package main

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestAppendURLParams(t *testing.T) {
	got, err := appendURLParams("https://cdn.example.com/seg.ts?token=old&x=1&x=2", "https://cdn.example.com/main.m3u8?token=new&hmac=abc&token=second")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://cdn.example.com/seg.ts?token=new%2csecond&x=1%2c2&hmac=abc"
	if got != want {
		t.Fatalf("query not merged like upstream:\nwant %s\ngot  %s", want, got)
	}
	got, err = appendURLParams("https://cdn.example.com/seg.ts?z=hello world&token=old#frag", "https://cdn.example.com/main.m3u8?token=a,b&hmac=a b")
	if err != nil {
		t.Fatal(err)
	}
	want = "https://cdn.example.com/seg.ts?z=hello+world&token=a%2cb&hmac=a+b"
	if got != want {
		t.Fatalf("query encoding not merged like upstream:\nwant %s\ngot  %s", want, got)
	}
	got, err = appendURLParams("https://user:pass@cdn.example.com:8443/a%20b/seg.ts?x=1#frag", "https://cdn.example.com/main.m3u8?token=abc")
	if err != nil {
		t.Fatal(err)
	}
	want = "https://user:pass@cdn.example.com:8443/a%20b/seg.ts?x=1&token=abc"
	if got != want {
		t.Fatalf("URL authority/path should match upstream append semantics:\nwant %s\ngot  %s", want, got)
	}
	tests := []struct {
		name   string
		target string
		source string
		want   string
	}{
		{
			name:   "missing key replaced by missing key",
			target: "https://cdn.example.com/seg.ts?flag&x=1",
			source: "https://cdn.example.com/main.m3u8?token&y=2",
			want:   "https://cdn.example.com/seg.ts?token&x=1&y=2",
		},
		{
			name:   "empty key replaced by empty key",
			target: "https://cdn.example.com/seg.ts?=flag&x=1",
			source: "https://cdn.example.com/main.m3u8?=empty&y=2",
			want:   "https://cdn.example.com/seg.ts?empty&x=1&y=2",
		},
		{
			name:   "missing and empty keys do not replace each other",
			target: "https://cdn.example.com/seg.ts?flag&x=1",
			source: "https://cdn.example.com/main.m3u8?=empty&y=2",
			want:   "https://cdn.example.com/seg.ts?flag&x=1&empty&y=2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := appendURLParams(tt.target, tt.source)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("query keyless merge mismatch:\nwant %s\ngot  %s", tt.want, got)
			}
		})
	}
}

func TestParseArgsBooleanExplicitFalse(t *testing.T) {
	opt, err := parseArgs([]string{"--check-segments-count", "false", "--auto-select", "false", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.CheckSegmentsCount {
		t.Fatal("check-segments-count false was not parsed")
	}
	if opt.AutoSelect {
		t.Fatal("auto-select false was not parsed")
	}
	if opt.Input != "https://example.com/main.m3u8" {
		t.Fatalf("input parsed incorrectly: %s", opt.Input)
	}
}

func TestConsoleRedirectDefaultsFollowUpstream(t *testing.T) {
	tmp := t.TempDir()
	redirected, err := os.Create(filepath.Join(tmp, "stdout.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer redirected.Close()
	opt := defaultOptions()
	if !applyConsoleRedirectDefaults(&opt, redirected, nil) {
		t.Fatal("regular file stdout should be treated as redirected")
	}
	if !opt.ForceANSIConsole || !opt.NoANSIColor {
		t.Fatalf("redirected output should force ansi console and clear colors: %#v", opt)
	}
	opt = defaultOptions()
	if applyConsoleRedirectDefaults(&opt, nil, nil) {
		t.Fatal("nil console files should not be treated as redirected")
	}
	if opt.ForceANSIConsole || opt.NoANSIColor {
		t.Fatalf("non-redirected console defaults should remain unchanged: %#v", opt)
	}
}

func TestParseArgsBooleanImplicitTrue(t *testing.T) {
	opt, err := parseArgs([]string{"--auto-select", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if !opt.AutoSelect {
		t.Fatal("auto-select should default to true when value is omitted")
	}
	if opt.Input != "https://example.com/main.m3u8" {
		t.Fatalf("input parsed incorrectly: %s", opt.Input)
	}
}

func TestParseArgsHTTPRequestTimeoutAcceptsFractionalSecondsLikeUpstream(t *testing.T) {
	opt, err := parseArgs([]string{"--http-request-timeout", "1.5", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.HTTPRequestTimeout != 1.5 {
		t.Fatalf("timeout should preserve fractional seconds, got %g", opt.HTTPRequestTimeout)
	}
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	if client.Timeout != 1500*time.Millisecond {
		t.Fatalf("HTTP client timeout should use fractional seconds, got %s", client.Timeout)
	}

	if _, err := parseArgs([]string{"--http-request-timeout", "abc", "https://example.com/main.m3u8"}); err == nil || !strings.Contains(err.Error(), "error in parse HttpRequestTimeout") {
		t.Fatalf("expected invalid timeout to be rejected, got %v", err)
	}
}

func TestParseArgsNumericOptionsRejectInvalidIntegersLikeUpstream(t *testing.T) {
	opt, err := parseArgs([]string{
		"--thread-count", "8",
		"--download-retry-count", "5",
		"--live-wait-time", "12",
		"--live-take-count", "4",
		"https://example.com/main.m3u8",
	})
	if err != nil {
		t.Fatal(err)
	}
	if opt.ThreadCount != 8 || opt.DownloadRetryCount != 5 || opt.LiveWaitTime == nil || *opt.LiveWaitTime != 12 || opt.LiveTakeCount != 4 {
		t.Fatalf("integer options not parsed correctly: %#v", opt)
	}

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"--thread-count", "many", "https://example.com/main.m3u8"}, "error in parse ThreadCount"},
		{[]string{"--download-retry-count", "twice", "https://example.com/main.m3u8"}, "error in parse DownloadRetryCount"},
		{[]string{"--live-wait-time", "soon", "https://example.com/main.m3u8"}, "error in parse LiveWaitTime"},
		{[]string{"--live-take-count", "last", "https://example.com/main.m3u8"}, "error in parse LiveTakeCount"},
	}
	for _, tc := range cases {
		if _, err := parseArgs(tc.args); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("expected %q for args %#v, got %v", tc.want, tc.args, err)
		}
	}
}

func TestParseArgsValueOptionsRequireValuesLikeUpstream(t *testing.T) {
	cases := []string{
		"--morehelp",
		"--tmp-dir",
		"--sub-format",
		"--custom-hls-key",
		"--custom-range",
		"--mux-after-done",
		"--select-video",
		"--max-speed",
	}
	for _, option := range cases {
		if _, err := parseArgs([]string{option}); err == nil || !strings.Contains(err.Error(), option+" 缺少参数值") {
			t.Fatalf("expected missing value error for %s, got %v", option, err)
		}
	}
}

func TestParseArgsSaveNameSanitizesLikeUpstream(t *testing.T) {
	opt, err := parseArgs([]string{"--save-name", `bad name:ok?.mp4`, "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.SaveName != "bad name_ok_.mp4" {
		t.Fatalf("save-name should be sanitized before use, got %q", opt.SaveName)
	}

	opt, err = parseArgs([]string{"--save-name", `<>:"/\|?*`, "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.SaveName != "_________" {
		t.Fatalf("all invalid filename chars should become replacement chars like upstream, got %q", opt.SaveName)
	}
}

func TestParseArgsSaveNameRejectsEmptyAfterSanitize(t *testing.T) {
	if _, err := parseArgs([]string{"--save-name", `...`, "https://example.com/main.m3u8"}); err == nil || !strings.Contains(err.Error(), "Invalid save name!") {
		t.Fatalf("expected invalid save-name to be rejected like upstream, got %v", err)
	}
}

func TestParseArgsLogFilePathSanitizesLikeUpstream(t *testing.T) {
	tmp := t.TempDir()
	opt, err := parseArgs([]string{"--log-file-path", filepath.Join(tmp, `bad:name?.log`), "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(filepath.Join(tmp, "bad_name_.log"))
	if err != nil {
		t.Fatal(err)
	}
	if opt.LogFilePath != want {
		t.Fatalf("log file path should be absolute and sanitized, got %q want %q", opt.LogFilePath, want)
	}
}

func TestParseArgsLogFilePathRejectsEmptyFileNameLikeUpstream(t *testing.T) {
	if _, err := parseArgs([]string{"--log-file-path", "...", "https://example.com/main.m3u8"}); err == nil || !strings.Contains(err.Error(), "Invalid log file name!") {
		t.Fatalf("expected invalid log file name to be rejected like upstream, got %v", err)
	}
}

func TestParseArgsHeaderAcceptsMultipleValuesAfterOneFlag(t *testing.T) {
	opt, err := parseArgs([]string{
		"--header",
		"Cookie: token=abc",
		"Referer: https://example.com/watch?id=1",
		"not-a-header",
		"https://example.com/main.m3u8",
	})
	if err != nil {
		t.Fatal(err)
	}
	if opt.Input != "https://example.com/main.m3u8" {
		t.Fatalf("input should not be consumed as header: %s", opt.Input)
	}
	if opt.Headers["cookie"] != "token=abc" {
		t.Fatalf("cookie header not parsed: %#v", opt.Headers)
	}
	if opt.Headers["referer"] != "https://example.com/watch?id=1" {
		t.Fatalf("referer header with URL value not parsed: %#v", opt.Headers)
	}
	if _, ok := opt.Headers["not-a-header"]; ok {
		t.Fatalf("header token without colon should be ignored like upstream: %#v", opt.Headers)
	}
}

func TestParseArgsMaxSpeedMatchesUpstreamUnits(t *testing.T) {
	opt, err := parseArgs([]string{"--max-speed", "1.5M", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.MaxSpeed != int64(1.5*1024*1024) {
		t.Fatalf("unexpected max speed: %d", opt.MaxSpeed)
	}
	opt, err = parseArgs([]string{"--max-speed", "1MB", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.MaxSpeed != int64(1024*1024) {
		t.Fatalf("speed parser should use first M/K match like upstream, got %d", opt.MaxSpeed)
	}
	if _, err := parseArgs([]string{"--max-speed", "1024", "https://example.com/main.m3u8"}); err == nil || !strings.Contains(err.Error(), "error in parse SpeedLimit") {
		t.Fatalf("expected bare speed to be rejected like upstream, got %v", err)
	}
	if _, err := parseArgs([]string{"--max-speed", "1G", "https://example.com/main.m3u8"}); err == nil || !strings.Contains(err.Error(), "error in parse SpeedLimit") {
		t.Fatalf("expected invalid speed unit to be rejected, got %v", err)
	}
}

func TestParseArgsCustomProxyValidatesURLLikeUpstream(t *testing.T) {
	opt, err := parseArgs([]string{"--custom-proxy", "http://user:pass@127.0.0.1:8080", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.CustomProxy != "http://user:pass@127.0.0.1:8080" {
		t.Fatalf("unexpected proxy: %s", opt.CustomProxy)
	}
	if _, err := parseArgs([]string{"--custom-proxy", "127.0.0.1:8080", "https://example.com/main.m3u8"}); err == nil || !strings.Contains(err.Error(), "error in parse proxy") {
		t.Fatalf("expected invalid proxy to be rejected like upstream, got %v", err)
	}
}

func TestParseArgsKeyMatchesUpstreamFormats(t *testing.T) {
	opt, err := parseArgs([]string{"--key", "ABCDEFABCDEFABCDEFABCDEFABCDEFAB", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if got := opt.Keys[0]; got != "ABCDEFABCDEFABCDEFABCDEFABCDEFAB" {
		t.Fatalf("direct hex key should preserve casing like upstream, got %s", got)
	}

	directPair := "ABCDEFABCDEFABCDEFABCDEFABCDEFAB:00112233445566778899AABBCCDDEEFF"
	opt, err = parseArgs([]string{"--key", directPair, "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if got := opt.Keys[0]; got != directPair {
		t.Fatalf("direct KID:KEY should preserve casing like upstream, got %s", got)
	}

	kidB64 := base64.StdEncoding.EncodeToString([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15})
	keyB64 := base64.StdEncoding.EncodeToString([]byte{16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31})
	opt, err = parseArgs([]string{"--key", kidB64 + ":" + keyB64, "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if got := opt.Keys[0]; got != "000102030405060708090a0b0c0d0e0f:101112131415161718191a1b1c1d1e1f" {
		t.Fatalf("base64 KID:KEY should be converted to hex, got %s", got)
	}

	if _, err := parseArgs([]string{"--key", "not-a-valid-key", "https://example.com/main.m3u8"}); err == nil || !strings.Contains(err.Error(), "error in parse custom key") {
		t.Fatalf("expected invalid key to be rejected like upstream, got %v", err)
	}
}

func TestParseArgsKeyDropsEmptyPartsLikeUpstream(t *testing.T) {
	key := "00112233445566778899aabbccddeeff"
	kid := "abcdefabcdefabcdefabcdefabcdefab"

	opt, err := parseArgs([]string{"--key", ":" + key, "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.Keys[0] != key {
		t.Fatalf("leading empty KID should be dropped like upstream, got %s", opt.Keys[0])
	}

	opt, err = parseArgs([]string{"--key", strings.ToUpper(key) + ":", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.Keys[0] != key {
		t.Fatalf("trailing empty KEY should leave single key like upstream, got %s", opt.Keys[0])
	}

	opt, err = parseArgs([]string{"--key", kid + "::" + key, "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.Keys[0] != kid+":"+key {
		t.Fatalf("empty middle key part should be dropped like upstream, got %s", opt.Keys[0])
	}
}

func TestParseArgsKeyAcceptsMultipleValuesAfterOneFlag(t *testing.T) {
	opt, err := parseArgs([]string{
		"--key",
		"00000000000000000000000000000000:00112233445566778899aabbccddeeff",
		"11111111111111111111111111111111:aabbccddeeff00112233445566778899",
		"https://example.com/main.m3u8",
	})
	if err != nil {
		t.Fatal(err)
	}
	if opt.Input != "https://example.com/main.m3u8" {
		t.Fatalf("input should not be consumed as key: %s", opt.Input)
	}
	if len(opt.Keys) != 2 {
		t.Fatalf("expected two keys from one --key flag, got %#v", opt.Keys)
	}
	if _, err := parseArgs([]string{
		"--key",
		"00000000000000000000000000000000:00112233445566778899aabbccddeeff",
		"bad-second-key",
		"https://example.com/main.m3u8",
	}); err == nil || !strings.Contains(err.Error(), "error in parse custom key") {
		t.Fatalf("expected invalid extra key to be rejected, got %v", err)
	}
}

func TestParseArgsCustomHLSMethodUsesUpstreamEnumNames(t *testing.T) {
	opt, err := parseArgs([]string{"--custom-hls-method", "AES_128_ECB", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.CustomHLSMethod != EncryptAES128ECB {
		t.Fatalf("AES_128_ECB should normalize to %s, got %s", EncryptAES128ECB, opt.CustomHLSMethod)
	}
	opt, err = parseArgs([]string{"--custom-hls-method", "sample_aes_ctr", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.CustomHLSMethod != EncryptSampleCTR {
		t.Fatalf("sample_aes_ctr should normalize to %s, got %s", EncryptSampleCTR, opt.CustomHLSMethod)
	}
}

func TestParseArgsCustomHLSKeyAndIVLikeUpstream(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "hls.key")
	fileBytes := []byte("0123456789abcdef")
	if err := os.WriteFile(keyFile, fileBytes, 0644); err != nil {
		t.Fatal(err)
	}

	opt, err := parseArgs([]string{
		"--custom-hls-key", "00112233445566778899aabbccddeeff",
		"--custom-hls-iv", base64.StdEncoding.EncodeToString(fileBytes),
		"https://example.com/main.m3u8",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", opt.CustomHLSKey); got != "00112233445566778899aabbccddeeff" {
		t.Fatalf("hex custom HLS key not parsed: %s", got)
	}
	if string(opt.CustomHLSIV) != string(fileBytes) {
		t.Fatalf("base64 custom HLS iv not parsed: %x", opt.CustomHLSIV)
	}

	opt, err = parseArgs([]string{"--custom-hls-key", keyFile, "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if string(opt.CustomHLSKey) != string(fileBytes) {
		t.Fatalf("file custom HLS key not read: %x", opt.CustomHLSKey)
	}

	opt, err = parseArgs([]string{"--custom-hls-key", "", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.CustomHLSKey != nil {
		t.Fatalf("empty custom HLS key should be ignored like upstream, got %x", opt.CustomHLSKey)
	}

	if _, err := parseArgs([]string{"--custom-hls-iv", "not-valid-key", "https://example.com/main.m3u8"}); err == nil || !strings.Contains(err.Error(), "error in parse hls custom key: not-valid-key") {
		t.Fatalf("expected upstream custom HLS key parser error, got %v", err)
	}
}

func TestParseCustomRangeOpenEndedLikeUpstream(t *testing.T) {
	cr, err := parseCustomRange("-2")
	if err != nil {
		t.Fatal(err)
	}
	if cr.StartSeg == nil || cr.EndSeg == nil || *cr.StartSeg != 0 || *cr.EndSeg != 2 {
		t.Fatalf("unexpected left-open segment range: %#v", cr)
	}
	cr, err = parseCustomRange("2-")
	if err != nil {
		t.Fatal(err)
	}
	if cr.StartSeg == nil || cr.EndSeg == nil || *cr.StartSeg != 2 || *cr.EndSeg != math.MaxInt64 {
		t.Fatalf("unexpected right-open segment range: %#v", cr)
	}
	cr, err = parseCustomRange("-00:20")
	if err != nil {
		t.Fatal(err)
	}
	if cr.StartSec == nil || cr.EndSec == nil || *cr.StartSec != 0 || *cr.EndSec != 20 {
		t.Fatalf("unexpected left-open time range: %#v", cr)
	}
	cr, err = parseCustomRange("00:10-")
	if err != nil {
		t.Fatal(err)
	}
	if cr.StartSec == nil || cr.EndSec == nil || *cr.StartSec != 10 || *cr.EndSec != math.MaxFloat64 {
		t.Fatalf("unexpected right-open time range: %#v", cr)
	}
}

func TestParseArgsCustomRangeErrorsLikeUpstream(t *testing.T) {
	opt, err := parseArgs([]string{"--custom-range", "", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.CustomRange != nil {
		t.Fatalf("empty custom-range should be ignored like upstream, got %#v", opt.CustomRange)
	}

	if _, err := parseArgs([]string{"--custom-range", "bad", "https://example.com/main.m3u8"}); err == nil || !strings.Contains(err.Error(), "error in parse CustomRange: Bad format!") {
		t.Fatalf("expected upstream custom-range format error, got %v", err)
	}
	if _, err := parseArgs([]string{"--custom-range", "1-2-3", "https://example.com/main.m3u8"}); err == nil || !strings.Contains(err.Error(), "error in parse CustomRange: Bad format!") {
		t.Fatalf("expected upstream custom-range multiple dash error, got %v", err)
	}
}

func TestParseArgsCustomRangeUsesFirstSegmentRangeMatchLikeUpstream(t *testing.T) {
	opt, err := parseArgs([]string{"--custom-range", "prefix1-2suffix", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.CustomRange == nil || opt.CustomRange.StartSeg == nil || opt.CustomRange.EndSeg == nil || *opt.CustomRange.StartSeg != 1 || *opt.CustomRange.EndSeg != 2 {
		t.Fatalf("custom-range should use first numeric range match like upstream, got %#v", opt.CustomRange)
	}

	opt, err = parseArgs([]string{"--custom-range", "x-y", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.CustomRange == nil || opt.CustomRange.StartSeg == nil || opt.CustomRange.EndSeg == nil || *opt.CustomRange.StartSeg != 0 || *opt.CustomRange.EndSeg != math.MaxInt64 {
		t.Fatalf("empty numeric match should become open range like upstream, got %#v", opt.CustomRange)
	}
}

func TestParseDurationColonUsesUpstreamDayHourMinuteSecondOrder(t *testing.T) {
	got, err := parseDuration("1:02:03:04")
	if err != nil {
		t.Fatal(err)
	}
	want := 24*time.Hour + 2*time.Hour + 3*time.Minute + 4*time.Second
	if got != want {
		t.Fatalf("colon duration should be parsed as days:hours:minutes:seconds like upstream, got %s want %s", got, want)
	}
	got, err = parseDuration("01：02：03")
	if err != nil {
		t.Fatal(err)
	}
	want = time.Hour + 2*time.Minute + 3*time.Second
	if got != want {
		t.Fatalf("fullwidth colon duration should match upstream ParseDur, got %s want %s", got, want)
	}
}

func TestParseDurationIgnoresExtraLeadingPartsLikeUpstream(t *testing.T) {
	got, err := parseDuration("9:1:02:03:04")
	if err != nil {
		t.Fatal(err)
	}
	want := 24*time.Hour + 2*time.Hour + 3*time.Minute + 4*time.Second
	if got != want {
		t.Fatalf("extra leading duration parts should be ignored after validation, got %s want %s", got, want)
	}
	if _, err := parseDuration("bad:1:02:03:04"); err == nil {
		t.Fatal("extra leading duration parts should still be numeric like upstream Convert.ToInt32")
	}
}

func TestParseDurationBareNumberMeansSecondsLikeUpstream(t *testing.T) {
	got, err := parseDuration("30")
	if err != nil {
		t.Fatal(err)
	}
	if got != 30*time.Second {
		t.Fatalf("bare number duration should mean seconds like upstream, got %s", got)
	}
	opt, err := parseArgs([]string{"--live-record-limit", "30", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.LiveRecordLimit == nil || *opt.LiveRecordLimit != 30*time.Second {
		t.Fatalf("live-record-limit bare number should be 30s, got %#v", opt.LiveRecordLimit)
	}
	if _, err := parseArgs([]string{"--live-record-limit", "not-duration", "https://example.com/main.m3u8"}); err == nil || !strings.Contains(err.Error(), "error in parse LiveRecordLimit: not-duration") {
		t.Fatalf("expected upstream live-record-limit parse error, got %v", err)
	}
}

func TestParseArgsStreamFilterPlaylistDurationUsesParseSecondsLikeUpstream(t *testing.T) {
	opt, err := parseArgs([]string{"-sv", "plistDurMin=1h20m30s:plistDurMax=2h:for=all", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.VideoFilter == nil || opt.VideoFilter.PlaylistMin == nil || *opt.VideoFilter.PlaylistMin != 4830 {
		t.Fatalf("playlist min duration should use h/m/s ParseSeconds format, got %#v", opt.VideoFilter)
	}
	if opt.VideoFilter.PlaylistMax == nil || *opt.VideoFilter.PlaylistMax != 7200 {
		t.Fatalf("playlist max duration should use h/m/s ParseSeconds format, got %#v", opt.VideoFilter)
	}

	for _, tc := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "bare number", input: "plistDurMin=30", want: "plistDurMin=30 not valid"},
		{name: "colon duration", input: "plistDurMin=00:10", want: "plistDurMin=00 not valid"},
		{name: "unsupported suffix", input: "plistDurMin=1d", want: "plistDurMin=1d not valid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseArgs([]string{"-sv", tc.input, "https://example.com/main.m3u8"})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestParseArgsTaskStartAtMatchesUpstreamFormat(t *testing.T) {
	opt, err := parseArgs([]string{"--task-start-at", "20260618123456", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.TaskStartAt == nil || opt.TaskStartAt.Format("20060102150405") != "20260618123456" {
		t.Fatalf("task-start-at should parse yyyyMMddHHmmss like upstream, got %#v", opt.TaskStartAt)
	}

	if _, err := parseArgs([]string{"--task-start-at", "2026-06-18 12:34:56", "https://example.com/main.m3u8"}); err == nil || !strings.Contains(err.Error(), "error in parse TaskStartTime: 2026-06-18 12:34:56") {
		t.Fatalf("expected upstream task-start-at parse error, got %v", err)
	}
}

func TestTaskStartAtWaitsBeforeDerivingDefaultSaveName(t *testing.T) {
	start := time.Date(2026, 6, 18, 23, 59, 59, 0, time.Local)
	target := time.Date(2026, 6, 19, 0, 0, 1, 0, time.Local)
	opt := defaultOptions()
	opt.Input = "https://example.com/main.m3u8"
	opt.TaskStartAt = &target
	var slept time.Duration
	var messages []string
	current := start
	waitForTaskStart(opt, func() time.Time { return current }, func(d time.Duration) {
		slept = d
		current = target
	}, func(msg string) {
		messages = append(messages, msg)
	})
	if slept != 2*time.Second {
		t.Fatalf("task-start-at should sleep until target time, got %s", slept)
	}
	if len(messages) != 1 || !strings.Contains(messages[0], "2026-06-19 00:00:01") {
		t.Fatalf("task-start-at message missing target time: %#v", messages)
	}
	applyDerivedDefaults(&opt, current)
	if opt.SaveName != "main_2026-06-19_00-00-01" {
		t.Fatalf("default save name should be derived after waiting like upstream, got %s", opt.SaveName)
	}
}

func TestParseArgsMuxAfterDoneStrictValidation(t *testing.T) {
	cases := []struct {
		name string
		mux  string
		want string
	}{
		{name: "format", mux: "format=avi", want: "format=avi not valid"},
		{name: "muxer", mux: "format=mkv:muxer=bad", want: "muxer=bad not valid"},
		{name: "mkvmerge mp4", mux: "format=mp4:muxer=mkvmerge", want: "mkvmerge can not do mp4"},
		{name: "empty bin_path", mux: "format=mp4:bin_path=", want: "bin_path= not valid"},
		{name: "keep", mux: "format=mp4:keep=yes", want: "keep=yes not valid"},
		{name: "skip_sub", mux: "format=mp4:skip_sub=yes", want: "skip_sub=yes not valid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseArgs([]string{"-M", tc.mux, "https://example.com/main.m3u8"})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestParseArgsMuxAfterDoneAcceptsBinPathAuto(t *testing.T) {
	opt, err := parseArgs([]string{"-M", "format=mkv:muxer=mkvmerge:bin_path=auto:keep=true:skip_sub=false", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.MuxAfterDone == nil || opt.MuxAfterDone.BinPath != "" || !opt.MuxAfterDone.Keep || opt.MuxAfterDone.SkipSubtitle {
		t.Fatalf("unexpected mux options: %#v", opt.MuxAfterDone)
	}
}

func TestParseArgsStreamFilterRejectsInvalidValuesLikeUpstream(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{name: "for", args: []string{"-sv", "for=middle", "https://example.com/main.m3u8"}, want: "for=middle not valid"},
		{name: "regex", args: []string{"-sa", "lang=[", "https://example.com/main.m3u8"}, want: "filter lang 正则无效"},
		{name: "segsMin", args: []string{"-ss", "segsMin=many", "https://example.com/main.m3u8"}, want: "segsMin=many not valid"},
		{name: "playlist duration", args: []string{"-dv", "plistDurMin=soon", "https://example.com/main.m3u8"}, want: "plistDurMin=soon not valid"},
		{name: "bandwidth", args: []string{"-da", "bwMin=fast", "https://example.com/main.m3u8"}, want: "bwMin=fast not valid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseArgs(tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestParseArgsStreamFilterLooseKeySearchMatchesUpstream(t *testing.T) {
	opt, err := parseArgs([]string{"-sv", `videoid=aud:filename=Part\:One`, "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.VideoFilter == nil || opt.VideoFilter.GroupID != "aud" || opt.VideoFilter.Name != "Part:One" {
		t.Fatalf("stream filter should use upstream loose key search, got %#v", opt.VideoFilter)
	}
}

func TestParseArgsMuxAfterDoneBareBoolFlagsMatchUpstreamSuffixRule(t *testing.T) {
	tests := []struct {
		name    string
		mux     string
		keep    bool
		skipSub bool
	}{
		{name: "keep suffix", mux: "format=mp4:keep", keep: true},
		{name: "skip suffix", mux: "format=mp4:skip_sub", skipSub: true},
		{name: "only trailing skip_sub wins", mux: "format=mp4:keep:skip_sub", skipSub: true},
		{name: "only trailing keep wins", mux: "format=mp4:skip_sub:keep", keep: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt, err := parseArgs([]string{"-M", tt.mux, "https://example.com/main.m3u8"})
			if err != nil {
				t.Fatal(err)
			}
			if opt.MuxAfterDone == nil || opt.MuxAfterDone.Keep != tt.keep || opt.MuxAfterDone.SkipSubtitle != tt.skipSub {
				t.Fatalf("bare keep/skip_sub should follow upstream suffix rule, got %#v", opt.MuxAfterDone)
			}
		})
	}
}

func TestParseArgsMuxImportAcceptsMultipleValuesAfterOneFlag(t *testing.T) {
	tmp := t.TempDir()
	en := filepath.Join(tmp, "extra-en.srt")
	ja := filepath.Join(tmp, "extra-ja.srt")
	for _, path := range []string{en, ja} {
		if err := os.WriteFile(path, []byte("1\n00:00:00,000 --> 00:00:01,000\nhi\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	opt, err := parseArgs([]string{
		"-M", "format=mp4",
		"--mux-import",
		"path=" + en + ":lang=en:name=English",
		"path=" + ja + ":lang=ja:name=Japanese",
		"https://example.com/main.m3u8",
	})
	if err != nil {
		t.Fatal(err)
	}
	if opt.Input != "https://example.com/main.m3u8" {
		t.Fatalf("input should not be consumed as mux import: %s", opt.Input)
	}
	if len(opt.MuxImports) != 2 {
		t.Fatalf("expected two mux imports from one flag, got %#v", opt.MuxImports)
	}
}

func TestParseArgsMuxImportRejectsMissingPathLikeUpstream(t *testing.T) {
	for _, tc := range []struct {
		name string
		arg  string
	}{
		{name: "empty", arg: "path="},
		{name: "missing file", arg: "path=/no/such/file.srt:lang=zh"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseArgs([]string{
				"-M", "format=mp4",
				"--mux-import", tc.arg,
				"https://example.com/main.m3u8",
			})
			if err == nil || !strings.Contains(err.Error(), "path empty or file not exists!") {
				t.Fatalf("expected upstream mux-import path error, got %v", err)
			}
		})
	}
}

func TestSplitComplexEscapedColonAndSingleQuotesLikeUpstream(t *testing.T) {
	p := splitComplex(`path=extra.srt:lang='zh-Hans':name=Part\:One`)
	if p["path"] != "extra.srt" || p["lang"] != "zh-Hans" || p["name"] != "Part:One" {
		t.Fatalf("complex params should preserve escaped colon and trim single quotes, got %#v", p)
	}
}

func TestSplitComplexQuotedColonStopsLikeUpstream(t *testing.T) {
	p := splitComplex(`path=extra.srt:name="Part:One"`)
	if p["path"] != "extra.srt" || p["name"] != "Part" {
		t.Fatalf("quoted colon should still split like upstream ComplexParamParser, got %#v", p)
	}
}

func TestMoreHelpIncludesMuxImportLikeUpstream(t *testing.T) {
	help := moreHelp("mux-import")
	for _, want := range []string{"--mux-import", "path=PATH", "lang=CODE", "name=NAME"} {
		if !strings.Contains(help, want) {
			t.Fatalf("mux-import morehelp missing %q:\n%s", want, help)
		}
	}
	if !strings.Contains(usage(), "mux-import") {
		t.Fatalf("usage should advertise mux-import morehelp:\n%s", usage())
	}
}

func TestParseArgsAdKeywordAcceptsMultipleValuesAfterOneFlag(t *testing.T) {
	opt, err := parseArgs([]string{
		"--ad-keyword",
		`/ad\d+\.ts$`,
		`BUMPER`,
		"https://example.com/main.m3u8",
	})
	if err != nil {
		t.Fatal(err)
	}
	if opt.Input != "https://example.com/main.m3u8" {
		t.Fatalf("input should not be consumed as ad keyword: %s", opt.Input)
	}
	if len(opt.AdKeywords) != 2 || opt.AdKeywords[0] != `/ad\d+\.ts$` || opt.AdKeywords[1] != "BUMPER" {
		t.Fatalf("unexpected ad keywords: %#v", opt.AdKeywords)
	}
}

func TestWriteMetaWritesAllAndSelected(t *testing.T) {
	tmp := t.TempDir()
	opt := defaultOptions()
	opt.TmpDir = tmp
	opt.SaveName = "job"
	p := &parser{rawFiles: map[string]string{"raw.m3u8": "#EXTM3U\n"}}
	all := []StreamSpec{{ID: 1, URL: "video.m3u8"}, {ID: 2, URL: "audio.m3u8"}}
	selected := []StreamSpec{{ID: 1, URL: "video.m3u8"}}
	if err := writeMeta(opt, p, all, selected); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(tmp, "job")
	meta, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		t.Fatal(err)
	}
	metaSelected, err := os.ReadFile(filepath.Join(dir, "meta_selected.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(meta), "audio.m3u8") {
		t.Fatalf("meta.json should contain all streams: %s", meta)
	}
	if strings.Contains(string(metaSelected), "audio.m3u8") || !strings.Contains(string(metaSelected), "video.m3u8") {
		t.Fatalf("meta_selected.json should contain only selected streams: %s", metaSelected)
	}
	if _, err := os.Stat(filepath.Join(dir, "raw.m3u8")); err != nil {
		t.Fatalf("raw m3u8 not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "raw.m3u8")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("raw m3u8 should be written under task temp dir like upstream, root err=%v", err)
	}
}

func TestTaskTempDirUsesSaveNameLikeUpstream(t *testing.T) {
	opt := defaultOptions()
	opt.TmpDir = "/tmp/root"
	opt.SaveName = "job"
	if got := taskTempDir(opt); got != filepath.Join("/tmp/root", "job") {
		t.Fatalf("task temp dir mismatch: %s", got)
	}
}

func TestWriteMetaDisabledSkipsRawAndMetaFiles(t *testing.T) {
	tmp := t.TempDir()
	opt := defaultOptions()
	opt.TmpDir = tmp
	opt.WriteMetaJSON = false
	p := &parser{rawFiles: map[string]string{"raw.m3u8": "#EXTM3U\n"}}
	if err := writeMeta(opt, p, []StreamSpec{{ID: 1, URL: "video.m3u8"}}, []StreamSpec{{ID: 1, URL: "video.m3u8"}}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("write-meta-json=false should not create raw/meta files, got %d entries", len(entries))
	}
}

func TestWriteMetaDoesNotOverwriteExistingFiles(t *testing.T) {
	tmp := t.TempDir()
	opt := defaultOptions()
	opt.TmpDir = tmp
	opt.SaveName = "job"
	p := &parser{rawFiles: map[string]string{"raw.m3u8": "#EXTM3U\n#EXTINF:1,\nnew.ts\n"}}
	dir := filepath.Join(tmp, "job")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	existing := map[string]string{
		"raw.m3u8":           "existing raw",
		"meta.json":          "existing all",
		"meta_selected.json": "existing selected",
	}
	for name, content := range existing {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeMeta(opt, p, []StreamSpec{{ID: 1, URL: "video.m3u8"}}, []StreamSpec{{ID: 1, URL: "video.m3u8"}}); err != nil {
		t.Fatal(err)
	}
	for name, want := range existing {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Fatalf("%s should keep existing content, got %q want %q", name, got, want)
		}
	}
}

func TestCleanupRawMetaAfterDownloadMatchesUpstream(t *testing.T) {
	tmp := t.TempDir()
	opt := defaultOptions()
	opt.TmpDir = tmp
	opt.SaveName = "job"
	p := &parser{rawFiles: map[string]string{"raw.m3u8": "#EXTM3U\n"}}
	if err := writeMeta(opt, p, []StreamSpec{{ID: 1, URL: "video.m3u8"}}, []StreamSpec{{ID: 1, URL: "video.m3u8"}}); err != nil {
		t.Fatal(err)
	}
	dir := rawMetaDir(opt)
	if _, err := os.Stat(filepath.Join(dir, "raw.m3u8")); err != nil {
		t.Fatal(err)
	}
	if err := cleanupRawMetaAfterDownload(opt, p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("empty raw/meta task dir should be removed like upstream, err=%v", err)
	}
	if _, err := os.Stat(tmp); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("empty tmp root should also be removed by upstream SafeDeleteDir recursion, err=%v", err)
	}
}

func TestCleanupRawMetaKeepsFilesWhenSkipMerge(t *testing.T) {
	tmp := t.TempDir()
	opt := defaultOptions()
	opt.TmpDir = tmp
	opt.SaveName = "job"
	opt.SkipMerge = true
	p := &parser{rawFiles: map[string]string{"raw.m3u8": "#EXTM3U\n"}}
	if err := writeMeta(opt, p, []StreamSpec{{ID: 1, URL: "video.m3u8"}}, []StreamSpec{{ID: 1, URL: "video.m3u8"}}); err != nil {
		t.Fatal(err)
	}
	if err := cleanupRawMetaAfterDownload(opt, p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(rawMetaDir(opt), "raw.m3u8")); err != nil {
		t.Fatalf("skip-merge should keep raw/meta files like upstream, err=%v", err)
	}
}

func TestCleanupRawMetaPreservesNonEmptyTaskDir(t *testing.T) {
	tmp := t.TempDir()
	opt := defaultOptions()
	opt.TmpDir = tmp
	opt.SaveName = "job"
	p := &parser{rawFiles: map[string]string{"raw.m3u8": "#EXTM3U\n"}}
	if err := writeMeta(opt, p, []StreamSpec{{ID: 1, URL: "video.m3u8"}}, []StreamSpec{{ID: 1, URL: "video.m3u8"}}); err != nil {
		t.Fatal(err)
	}
	dir := rawMetaDir(opt)
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("note"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := cleanupRawMetaAfterDownload(opt, p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "keep.txt")); err != nil {
		t.Fatalf("non raw/meta task file should be kept, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "raw.m3u8")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("raw/meta files should still be removed from non-empty task dir, err=%v", err)
	}
}

func TestCleanupDownloadedTempDirPreservesUnexpectedFilesLikeUpstream(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "task")
	streamDir := filepath.Join(root, "stream")
	if err := os.MkdirAll(streamDir, 0755); err != nil {
		t.Fatal(err)
	}
	segment := filepath.Join(streamDir, "000.ts")
	note := filepath.Join(streamDir, "note.txt")
	for path, content := range map[string]string{
		segment:                                "segment",
		filepath.Join(streamDir, "concat.txt"): "concat",
		note:                                   "keep",
	} {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := cleanupDownloadedTempDir(streamDir, []string{segment}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(segment); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("known segment should be removed, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(streamDir, "concat.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Go concat helper file should be removed, err=%v", err)
	}
	if _, err := os.Stat(note); err != nil {
		t.Fatalf("unexpected user file should keep non-empty dir alive, err=%v", err)
	}
}

func TestCleanupDownloadedTempDirDeletesEmptyParentsLikeUpstream(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "task")
	streamDir := filepath.Join(root, "stream")
	if err := os.MkdirAll(streamDir, 0755); err != nil {
		t.Fatal(err)
	}
	segment := filepath.Join(streamDir, "000.ts")
	if err := os.WriteFile(segment, []byte("segment"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := cleanupDownloadedTempDir(streamDir, []string{segment}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(streamDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("empty stream dir should be removed, err=%v", err)
	}
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("empty task root should be removed recursively, err=%v", err)
	}
	if _, err := os.Stat(tmp); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("empty tmp root should be removed recursively, err=%v", err)
	}
}

func TestCleanupRawMetaHonorsWriteMetaJSONFlag(t *testing.T) {
	tmp := t.TempDir()
	opt := defaultOptions()
	opt.TmpDir = tmp
	opt.SaveName = "job"
	opt.WriteMetaJSON = false
	p := &parser{rawFiles: map[string]string{"raw.m3u8": "#EXTM3U\n"}}
	dir := rawMetaDir(opt)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"raw.m3u8":           "old raw",
		"meta.json":          "old all",
		"meta_selected.json": "old selected",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := cleanupRawMetaAfterDownload(opt, p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "raw.m3u8")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("raw file should be removed because rawFiles tracks it, err=%v", err)
	}
	for _, name := range []string{"meta.json", "meta_selected.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("%s should be preserved when write-meta-json=false, err=%v", name, err)
		}
	}
}

func TestDeriveSaveNameFromInputURL(t *testing.T) {
	now := time.Date(2026, 6, 18, 12, 34, 56, 0, time.Local)
	got := deriveSaveNameFromInput("https://example.com/path/master.m3u8?token=abc", now)
	if got != "master_2026-06-18_12-34-56" {
		t.Fatalf("unexpected save name: %s", got)
	}
}

func TestApplyDerivedDefaultsKeepsExplicitSaveName(t *testing.T) {
	opt := Options{Input: "https://example.com/a.m3u8", SaveName: "explicit"}
	applyDerivedDefaults(&opt, time.Date(2026, 6, 18, 12, 34, 56, 0, time.Local))
	if opt.SaveName != "explicit" {
		t.Fatalf("explicit save name changed: %s", opt.SaveName)
	}
}

func TestDecryptSegmentUnknownKeepsRawBytes(t *testing.T) {
	raw := []byte("raw-encrypted-segment")
	got, err := decryptSegment(raw, EncryptInfo{Method: EncryptUnknown})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(raw) {
		t.Fatalf("unknown encryption should keep raw bytes, got %q", got)
	}
}

func TestAutoBinaryMergeDetectionForCENCAndFMP4(t *testing.T) {
	video := MediaVideo
	sub := MediaSubtitles
	cencStreams := []StreamSpec{{
		Playlist: &Playlist{Parts: []MediaPart{{Segments: []Segment{{Encrypt: EncryptInfo{Method: EncryptCENC}}}}}},
	}}
	if !hasCENCEncryption(cencStreams) {
		t.Fatal("CENC encryption should be detected")
	}
	fmp4Video := []StreamSpec{{
		MediaType: &video,
		Playlist:  &Playlist{MediaInit: &Segment{URL: "init.mp4"}},
	}}
	if !hasFMP4Media(fmp4Video) {
		t.Fatal("fMP4 video should force binary merge")
	}
	fmp4Subtitle := []StreamSpec{{
		MediaType: &sub,
		Playlist:  &Playlist{MediaInit: &Segment{URL: "sub-init.mp4"}},
	}}
	if hasFMP4Media(fmp4Subtitle) {
		t.Fatal("fMP4 subtitles should not use the media fMP4 auto merge rule")
	}
}

func TestDefaultNameSavePatternFrameRate(t *testing.T) {
	opt := Options{SavePattern: "<SaveName>_<FrameRate>_<Resolution>", SaveName: "movie"}
	got := defaultName(opt, StreamSpec{FrameRate: 23.976, Resolution: "1920x1080"}, "fallback")
	if got != "movie_23.976_1920x1080" {
		t.Fatalf("unexpected pattern name: %s", got)
	}
}

func TestDownloadSavePatternIdUsesTaskOrderLikeUpstream(t *testing.T) {
	tmp := t.TempDir()
	opt := defaultOptions()
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "movie"
	opt.SavePattern = "<SaveName>_<Id>"
	opt.BinaryMerge = true
	opt.DelAfterDone = false
	stream := StreamSpec{
		ID:        7,
		Extension: "ts",
		Playlist: &Playlist{Parts: []MediaPart{{Segments: []Segment{{
			Index:    0,
			URL:      "base64://c2VnbWVudA==",
			Duration: 1,
		}}}}},
	}
	outs, err := downloadAll(context.Background(), http.DefaultClient, []StreamSpec{stream}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if len(outs) != 1 {
		t.Fatalf("unexpected outputs: %#v", outs)
	}
	if got := filepath.Base(outs[0].Path); got != "movie_0.ts" {
		t.Fatalf("<Id> should use task order instead of original stream id, got %s", got)
	}
	if _, err := os.Stat(filepath.Join(opt.TmpDir, opt.SaveName, "0___0")); err != nil {
		t.Fatalf("task temp dir should also use task order like upstream, err=%v", err)
	}
}

func TestDefaultNameSavePatternCleansEmptySeparators(t *testing.T) {
	opt := Options{SavePattern: "__<SaveName>__<Language>..<Resolution>..", SaveName: "movie"}
	got := defaultName(opt, StreamSpec{}, "fallback")
	if got != "movie_" {
		t.Fatalf("empty optional fields should be cleaned like upstream, got %s", got)
	}
}

func TestDefaultNameSavePatternPreservesSpacesLikeUpstream(t *testing.T) {
	opt := Options{SavePattern: "<SaveName> <Language>", SaveName: "my movie"}
	got := defaultName(opt, StreamSpec{Language: "zh Hans"}, "fallback")
	if got != "my movie zh Hans" {
		t.Fatalf("save pattern should preserve spaces like upstream GetValidFileName, got %q", got)
	}
}

func TestDefaultNameSavePatternMediaTypeUsesEnumValue(t *testing.T) {
	audio := MediaAudio
	opt := Options{SavePattern: "<SaveName>_<MediaType>", SaveName: "movie"}
	got := defaultName(opt, StreamSpec{MediaType: &audio, Resolution: "1080p"}, "fallback")
	if got != "movie_AUDIO" {
		t.Fatalf("media type pattern should use enum value, got %s", got)
	}
}

func TestOutputBaseNamePreservesSaveNameSpacesLikeUpstream(t *testing.T) {
	if got := outputBaseName("my movie.zh Hans", "fallback"); got != "my movie.zh Hans" {
		t.Fatalf("output basename should preserve save-name spaces like upstream, got %q", got)
	}
	if got := outputBaseName("...", "fallback name"); got != "fallback_name" {
		t.Fatalf("empty output basename should use safe fallback, got %q", got)
	}
}

func TestOutputExtSubtitleUsesSubFormatWhenAutoFixEnabled(t *testing.T) {
	sub := MediaSubtitles
	s := StreamSpec{MediaType: &sub, Extension: "ttml"}
	opt := defaultOptions()
	opt.SubFormat = "SRT"
	if got := outputExt(s, opt); got != ".srt" {
		t.Fatalf("subtitle auto-fix should use SRT extension, got %s", got)
	}
	opt.SubFormat = "VTT"
	if got := outputExt(s, opt); got != ".vtt" {
		t.Fatalf("subtitle auto-fix should use VTT extension, got %s", got)
	}
	opt.AutoSubtitleFix = false
	if got := outputExt(s, opt); got != ".ttml" {
		t.Fatalf("without auto-fix, subtitle extension should stay original, got %s", got)
	}
}

func TestCollisionPathForStreamUsesVideoMetadata(t *testing.T) {
	tmp := t.TempDir()
	original := filepath.Join(tmp, "movie.mp4")
	if err := os.WriteFile(original, []byte("exists"), 0644); err != nil {
		t.Fatal(err)
	}
	video := MediaVideo
	got := collisionPathForStream(original, StreamSpec{MediaType: &video, Resolution: "1920x1080", Bandwidth: 5000000})
	want := filepath.Join(tmp, "movie.1920x1080.mp4")
	if got != want {
		t.Fatalf("video collision should use resolution first, got %s", got)
	}
}

func TestCollisionPathForStreamUsesAudioMetadata(t *testing.T) {
	tmp := t.TempDir()
	original := filepath.Join(tmp, "movie.m4a")
	if err := os.WriteFile(original, []byte("exists"), 0644); err != nil {
		t.Fatal(err)
	}
	audio := MediaAudio
	got := collisionPathForStream(original, StreamSpec{MediaType: &audio, Language: "en", Channels: "2", Bandwidth: 128000})
	want := filepath.Join(tmp, "movie.en.m4a")
	if got != want {
		t.Fatalf("audio collision should use language first, got %s", got)
	}
}

func TestCollisionPathForStreamFallsBackToCopyChain(t *testing.T) {
	tmp := t.TempDir()
	for _, name := range []string{"movie.vtt", "movie.zh.vtt"} {
		if err := os.WriteFile(filepath.Join(tmp, name), []byte("exists"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	sub := MediaSubtitles
	got := collisionPathForStream(filepath.Join(tmp, "movie.vtt"), StreamSpec{MediaType: &sub, Language: "zh"})
	want := filepath.Join(tmp, "movie.copy.vtt")
	if got != want {
		t.Fatalf("subtitle collision should fall back to copy chain, got %s", got)
	}
}

func TestAutoSelectKeepsAllSubtitleAndBestAudioPerLanguage(t *testing.T) {
	audio := MediaAudio
	sub := MediaSubtitles
	streams := []StreamSpec{
		{ID: 0, Bandwidth: 1000, Resolution: "1280x720"},
		{ID: 1, Bandwidth: 4000, Resolution: "1920x1080"},
		{ID: 2, MediaType: &audio, Language: "en", Bandwidth: 128},
		{ID: 3, MediaType: &audio, Language: "en", Bandwidth: 256},
		{ID: 4, MediaType: &audio, Language: "ja", Bandwidth: 192},
		{ID: 5, MediaType: &sub, Language: "en", Name: "English"},
		{ID: 6, MediaType: &sub, Language: "zh", Name: "Chinese"},
	}
	selected := autoSelect(streams, Options{AutoSelect: true})
	if len(selected) != 5 {
		t.Fatalf("want video + 2 audio + 2 subs, got %#v", selected)
	}
	wantIDs := map[int]bool{1: true, 3: true, 4: true, 5: true, 6: true}
	for _, s := range selected {
		if !wantIDs[s.ID] {
			t.Fatalf("unexpected selected stream: %#v", s)
		}
		delete(wantIDs, s.ID)
	}
	if len(wantIDs) != 0 {
		t.Fatalf("missing selected streams: %#v", wantIDs)
	}
}

func TestAutoSelectBreaksAudioBandwidthTieByChannelsLikeUpstream(t *testing.T) {
	audio := MediaAudio
	streams := []StreamSpec{
		{ID: 1, Bandwidth: 5000, Resolution: "2160p"},
		{ID: 2, MediaType: &audio, Language: "en", Bandwidth: 256, Channels: "2"},
		{ID: 3, MediaType: &audio, Language: "en", Bandwidth: 256, Channels: "6"},
	}
	selected := autoSelect(streams, Options{AutoSelect: true})
	if len(selected) != 2 {
		t.Fatalf("want video + best en audio, got %#v", selected)
	}
	if selected[1].ID != 3 {
		t.Fatalf("same-bandwidth audio should prefer larger CHANNELS like upstream, got %#v", selected)
	}
}

func TestAutoSelectPrefersBasicStreamBeforeMediaVideoLikeUpstream(t *testing.T) {
	video := MediaVideo
	streams := []StreamSpec{
		{ID: 1, Bandwidth: 1000, Resolution: "720p"},
		{ID: 2, MediaType: &video, Bandwidth: 8000, Resolution: "2160p"},
	}
	selected := autoSelect(streams, Options{AutoSelect: true})
	if len(selected) != 1 || selected[0].ID != 1 {
		t.Fatalf("basic stream should stay before EXT-X-MEDIA TYPE=VIDEO like upstream, got %#v", selected)
	}
}

func TestApplyFiltersKeepsOnlyRequestedTypeAndForCount(t *testing.T) {
	audio := MediaAudio
	sub := MediaSubtitles
	streams := []StreamSpec{
		{ID: 1, Bandwidth: 5000, Resolution: "2160p"},
		{ID: 2, Bandwidth: 3000, Resolution: "1080p"},
		{ID: 3, Bandwidth: 1000, Resolution: "720p"},
		{ID: 4, MediaType: &audio, Language: "en", Bandwidth: 128},
		{ID: 5, MediaType: &sub, Language: "en"},
	}
	got := applyFilters(streams, Options{VideoFilter: parseFilter("for=best2")})
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 2 {
		t.Fatalf("unexpected video filter result: %#v", got)
	}
}

func TestApplyFiltersForBestUsesUpstreamQualitySort(t *testing.T) {
	streams := []StreamSpec{
		{ID: 1, Bandwidth: 1000, Resolution: "720p"},
		{ID: 2, Bandwidth: 5000, Resolution: "2160p"},
		{ID: 3, Bandwidth: 3000, Resolution: "1080p"},
	}
	got := applyFilters(streams, Options{VideoFilter: parseFilter("for=best2")})
	if len(got) != 2 || got[0].ID != 2 || got[1].ID != 3 {
		t.Fatalf("best2 should use upstream sorted quality order, got %#v", got)
	}
}

func TestApplyFiltersVideoBestPrefersBasicStreamBeforeMediaVideoLikeUpstream(t *testing.T) {
	video := MediaVideo
	streams := []StreamSpec{
		{ID: 1, Bandwidth: 1000, Resolution: "720p"},
		{ID: 2, MediaType: &video, Bandwidth: 8000, Resolution: "2160p"},
	}
	got := applyFilters(streams, Options{VideoFilter: parseFilter("for=best")})
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("video best should keep basic stream before TYPE=VIDEO rendition like upstream, got %#v", got)
	}
}

func TestApplyFiltersBandwidthAndPlaylistDuration(t *testing.T) {
	streams := []StreamSpec{
		{ID: 1, Bandwidth: 1000, Playlist: &Playlist{Parts: []MediaPart{{Segments: []Segment{{Duration: 4}, {Duration: 4}}}}}},
		{ID: 2, Bandwidth: 5000, Playlist: &Playlist{Parts: []MediaPart{{Segments: []Segment{{Duration: 12}, {Duration: 12}}}}}},
		{ID: 3, Bandwidth: 9000, Playlist: &Playlist{Parts: []MediaPart{{Segments: []Segment{{Duration: 30}}}}}},
	}
	got := applyFilters(streams, Options{VideoFilter: parseFilter("bwMin=4:bwMax=6:plistDurMin=20s:plistDurMax=30s:for=all")})
	if len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("unexpected bandwidth/duration filter result: %#v", got)
	}
}

func TestApplyFiltersRoleLikeUpstream(t *testing.T) {
	streams := []StreamSpec{
		{ID: 1, Bandwidth: 3000, Role: "Main"},
		{ID: 2, Bandwidth: 2000, Role: "Commentary"},
		{ID: 3, Bandwidth: 1000, Role: "Subtitle"},
	}
	got := applyFilters(streams, Options{VideoFilter: parseFilter("role=commentary:for=all")})
	if len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("role filter should match upstream enum case-insensitively, got %#v", got)
	}
	got = applyFilters(streams, Options{VideoFilter: parseFilter("role=unknown:for=all")})
	if len(got) != 3 {
		t.Fatalf("invalid role should be ignored like upstream Enum.TryParse, got %#v", got)
	}
}

func TestApplyFiltersSkipsSegmentCountWhenPlaylistsMissing(t *testing.T) {
	streams := []StreamSpec{
		{ID: 1, Bandwidth: 1000, Resolution: "720p"},
		{ID: 2, Bandwidth: 3000, Resolution: "1080p"},
	}
	got := applyFilters(streams, Options{VideoFilter: parseFilter("segsMin=10:for=all")})
	if len(got) != 2 {
		t.Fatalf("segsMin should be ignored until all candidates have playlist segment counts, got %#v", got)
	}
}

func TestApplyDropsHonorsForBest(t *testing.T) {
	streams := []StreamSpec{
		{ID: 1, Bandwidth: 5000, Resolution: "2160p"},
		{ID: 2, Bandwidth: 3000, Resolution: "1080p"},
		{ID: 3, Bandwidth: 1000, Resolution: "720p"},
	}
	got := applyDrops(streams, Options{DropVideoFilter: parseFilter("for=best")})
	if len(got) != 2 || got[0].ID != 2 || got[1].ID != 3 {
		t.Fatalf("unexpected drop best result: %#v", got)
	}
}

func TestApplyDropsHonorsAudioWorstWithLanguage(t *testing.T) {
	audio := MediaAudio
	streams := []StreamSpec{
		{ID: 1, Bandwidth: 5000, Resolution: "2160p"},
		{ID: 2, MediaType: &audio, Language: "en", Bandwidth: 256},
		{ID: 3, MediaType: &audio, Language: "en", Bandwidth: 128},
		{ID: 4, MediaType: &audio, Language: "ja", Bandwidth: 96},
	}
	got := applyDrops(streams, Options{DropAudioFilter: parseFilter("lang=en:for=worst")})
	if len(got) != 3 {
		t.Fatalf("unexpected drop count: %#v", got)
	}
	for _, s := range got {
		if s.ID == 3 {
			t.Fatalf("worst en audio should be dropped: %#v", got)
		}
	}
}

func TestFilterStreamsAppliesDropBeforeKeep(t *testing.T) {
	streams := []StreamSpec{
		{ID: 1, Bandwidth: 5000, Resolution: "2160p"},
		{ID: 2, Bandwidth: 3000, Resolution: "1080p"},
		{ID: 3, Bandwidth: 1000, Resolution: "720p"},
	}
	selected := filterStreams(streams, Options{
		VideoFilter:     parseFilter("for=best2"),
		DropVideoFilter: parseFilter("for=worst"),
	})
	if len(selected) != 2 || selected[0].ID != 1 || selected[1].ID != 2 {
		t.Fatalf("drop should run before keep, got %#v", selected)
	}
}

func TestApplyCustomRangeSkippedDurationOnlyCountsBeforeRange(t *testing.T) {
	start, end := int64(1), int64(2)
	s := StreamSpec{Playlist: &Playlist{Parts: []MediaPart{{Segments: []Segment{
		{Index: 0, Duration: 10},
		{Index: 1, Duration: 10},
		{Index: 2, Duration: 10},
		{Index: 3, Duration: 10},
	}}}}}
	applyCustomRange(&s, &CustomRange{StartSeg: &start, EndSeg: &end})
	if len(s.Playlist.Parts[0].Segments) != 2 || s.Playlist.Parts[0].Segments[0].Index != 1 || s.Playlist.Parts[0].Segments[1].Index != 2 {
		t.Fatalf("unexpected custom range segments: %#v", s.Playlist.Parts[0].Segments)
	}
	if s.SkippedDuration != 10 {
		t.Fatalf("skipped duration should only count before selected range, got %.0f", s.SkippedDuration)
	}
}

func TestApplyCustomRangeTimeUsesSegmentStartLikeUpstream(t *testing.T) {
	start, end := 15.0, 25.0
	s := StreamSpec{Playlist: &Playlist{Parts: []MediaPart{{Segments: []Segment{
		{Index: 0, Duration: 10},
		{Index: 1, Duration: 10},
		{Index: 2, Duration: 10},
		{Index: 3, Duration: 10},
	}}}}}
	applyCustomRange(&s, &CustomRange{StartSec: &start, EndSec: &end})
	got := s.Playlist.Parts[0].Segments
	if len(got) != 1 || got[0].Index != 2 {
		t.Fatalf("time range should keep only segment starting inside range, got %#v", got)
	}
	if s.SkippedDuration != 20 {
		t.Fatalf("time skipped duration should be 20, got %.0f", s.SkippedDuration)
	}
}

func TestPrepareSelectedStreamsSkipsCustomRangeForLive(t *testing.T) {
	start, end := int64(1), int64(1)
	streams := []StreamSpec{{
		Playlist: &Playlist{IsLive: true, Parts: []MediaPart{{Segments: []Segment{
			{Index: 0, Duration: 1},
			{Index: 1, Duration: 1},
			{Index: 2, Duration: 1},
		}}}},
	}}
	opt := defaultOptions()
	opt.CustomRange = &CustomRange{StartSeg: &start, EndSeg: &end}
	prepareSelectedStreams(streams, &opt)
	if got := streams[0].Playlist.Parts[0].Segments; len(got) != 3 {
		t.Fatalf("live stream should not apply custom range, got %#v", got)
	}
}

func TestPrepareSelectedStreamsLiveForcesRecordOptionsLikeUpstream(t *testing.T) {
	streams := []StreamSpec{{
		Playlist: &Playlist{IsLive: true, Parts: []MediaPart{{Segments: []Segment{{Index: 0, Duration: 1}}}}},
	}}
	opt := defaultOptions()
	prepareSelectedStreams(streams, &opt)
	if !opt.ConcurrentDownload {
		t.Fatal("live recording should force concurrent download like upstream")
	}
	if !opt.MP4RealTimeDecryption {
		t.Fatal("live recording should force MP4 real-time decryption like upstream")
	}

	opt = defaultOptions()
	opt.LivePerformAsVOD = true
	prepareSelectedStreams(streams, &opt)
	if opt.ConcurrentDownload || opt.MP4RealTimeDecryption {
		t.Fatal("live-perform-as-vod should not enable live recording-only options")
	}
}

func TestPrepareSelectedStreamsDisablesLiveVTTFixWithoutAudioLikeUpstream(t *testing.T) {
	sub := MediaSubtitles
	streams := []StreamSpec{{
		MediaType: &sub,
		Playlist:  &Playlist{IsLive: true, Parts: []MediaPart{{Segments: []Segment{{Index: 0, Duration: 1}}}}},
	}}
	opt := defaultOptions()
	opt.LiveFixVTTByAudio = true
	prepareSelectedStreams(streams, &opt)
	if opt.LiveFixVTTByAudio {
		t.Fatal("live-fix-vtt-by-audio should be disabled when no audio stream is selected")
	}
}

func TestDownloadSubtitleWithLiveVTTFixNoAudioTrackerDoesNotPanic(t *testing.T) {
	tmp := t.TempDir()
	sub := MediaSubtitles
	opt := defaultOptions()
	opt.SaveDir = tmp
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveName = "sub"
	opt.LiveFixVTTByAudio = true
	stream := StreamSpec{
		ID:        0,
		MediaType: &sub,
		Extension: "vtt",
		Playlist: &Playlist{Parts: []MediaPart{{Segments: []Segment{{
			Index:    0,
			URL:      "base64://V0VCVlRUCgowMDowMDowMC4wMDAgLS0+IDAwOjAwOjAxLjAwMApIaQo=",
			Duration: 1,
		}}}}},
	}
	if _, err := downloadStream(context.Background(), http.DefaultClient, stream, opt, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestPrepareSelectedStreamsWarnsRealtimeDecryptEngineLikeUpstream(t *testing.T) {
	streams := []StreamSpec{{
		Playlist: &Playlist{Parts: []MediaPart{{Segments: []Segment{{Index: 0, Duration: 1}}}}},
	}}
	opt := defaultOptions()
	opt.UILanguage = "en-US"
	opt.MP4RealTimeDecryption = true
	opt.DecryptionEngine = "MP4DECRYPT"
	opt.Keys = []string{"00112233445566778899aabbccddeeff"}
	messages := prepareSelectedStreams(streams, &opt)
	if len(messages) != 1 || messages[0] != "When enabling real-time decryption, it is recommended to use shaka-packager instead of mp4decrypt/ffmpeg" {
		t.Fatalf("unexpected realtime decrypt warning: %#v", messages)
	}

	opt.DecryptionEngine = "SHAKA_PACKAGER"
	messages = prepareSelectedStreams(streams, &opt)
	if len(messages) != 0 {
		t.Fatalf("shaka realtime decrypt should not warn like upstream: %#v", messages)
	}

	opt = defaultOptions()
	opt.MP4RealTimeDecryption = true
	opt.KeyTextFile = "keys.txt"
	messages = prepareSelectedStreams(streams, &opt)
	if len(messages) != 0 {
		t.Fatalf("key-text-file alone should not trigger Keys.Length warning like upstream: %#v", messages)
	}
}

func TestPrepareSelectedStreamsUnknownEncryptionBeforeCustomRange(t *testing.T) {
	start, end := int64(1), int64(1)
	streams := []StreamSpec{{
		Playlist: &Playlist{Parts: []MediaPart{{Segments: []Segment{
			{Index: 0, Duration: 1, Encrypt: EncryptInfo{Method: EncryptUnknown}},
			{Index: 1, Duration: 1},
		}}}},
	}}
	opt := defaultOptions()
	opt.CustomRange = &CustomRange{StartSeg: &start, EndSeg: &end}
	msgs := prepareSelectedStreams(streams, &opt)
	if !opt.BinaryMerge {
		t.Fatal("unknown encryption should force binary merge before custom range removes the segment")
	}
	if len(msgs) != 1 || !strings.Contains(msgs[0], "无法识别") {
		t.Fatalf("unexpected messages: %#v", msgs)
	}
	if got := streams[0].Playlist.Parts[0].Segments; len(got) != 1 || got[0].Index != 1 {
		t.Fatalf("custom range should still be applied after binary decision, got %#v", got)
	}
}

func TestCleanAdSegmentsUsesRegexAndDropsEmptyParts(t *testing.T) {
	streams := []StreamSpec{{
		Playlist: &Playlist{Parts: []MediaPart{
			{Segments: []Segment{{Index: 0, URL: "https://cdn.example.com/main0.ts"}, {Index: 1, URL: "https://cdn.example.com/ad12.ts"}}},
			{Segments: []Segment{{Index: 2, URL: "https://cdn.example.com/ad13.ts"}}},
			{Segments: []Segment{{Index: 3, URL: "https://cdn.example.com/cadence.ts"}}},
		}},
	}}
	cleanAdSegments(streams, []string{`/ad\d+\.ts$`})
	parts := streams[0].Playlist.Parts
	if len(parts) != 2 {
		t.Fatalf("empty ad-only part should be removed, got %#v", parts)
	}
	got := []string{parts[0].Segments[0].URL, parts[1].Segments[0].URL}
	if got[0] != "https://cdn.example.com/main0.ts" || got[1] != "https://cdn.example.com/cadence.ts" {
		t.Fatalf("regex ad cleanup removed wrong segments: %#v", got)
	}
}

func TestChooseStreamsSubOnlyIgnoresKeepFilters(t *testing.T) {
	sub := MediaSubtitles
	streams := []StreamSpec{
		{ID: 1, Bandwidth: 5000, Resolution: "2160p"},
		{ID: 2, MediaType: &sub, Language: "en", Name: "English"},
		{ID: 3, MediaType: &sub, Language: "zh", Name: "Chinese"},
	}
	selected, err := chooseStreams(streams, Options{SubOnly: true, SubtitleFilter: parseFilter("lang=zh")})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 2 {
		t.Fatalf("sub-only should ignore keep filters like upstream, got %#v", selected)
	}
}

func TestChooseStreamsAutoSelectPrecedence(t *testing.T) {
	audio := MediaAudio
	sub := MediaSubtitles
	streams := []StreamSpec{
		{ID: 1, Bandwidth: 5000, Resolution: "2160p"},
		{ID: 2, Bandwidth: 3000, Resolution: "1080p"},
		{ID: 3, MediaType: &audio, Language: "en", Bandwidth: 256},
		{ID: 4, MediaType: &sub, Language: "zh", Name: "Chinese"},
	}
	selected, err := chooseStreams(streams, Options{
		AutoSelect:      true,
		SubOnly:         true,
		VideoFilter:     parseFilter("res=1080p"),
		DropVideoFilter: parseFilter("res=2160p"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 3 || selected[0].ID != 2 || selected[1].ID != 3 || selected[2].ID != 4 {
		t.Fatalf("auto-select should ignore sub-only/keep but honor drop, got %#v", selected)
	}
}

func TestChooseStreamsKeepFiltersReturnDirectlyLikeUpstream(t *testing.T) {
	audio := MediaAudio
	streams := []StreamSpec{
		{ID: 1, Bandwidth: 5000, Resolution: "2160p"},
		{ID: 2, Bandwidth: 3000, Resolution: "1080p"},
		{ID: 3, MediaType: &audio, Language: "en", Bandwidth: 256},
		{ID: 4, MediaType: &audio, Language: "ja", Bandwidth: 192},
	}
	selected, err := chooseStreams(streams, Options{
		VideoFilter: parseFilter("res=1080p"),
		AudioFilter: parseFilter("lang=ja"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 2 || selected[0].ID != 2 || selected[1].ID != 4 {
		t.Fatalf("keep filters should return selected streams without interactive fallback, got %#v", selected)
	}
}

func TestDefaultInteractiveSelectionUsesReferencedAudioAndSubtitleLikeUpstream(t *testing.T) {
	audio := MediaAudio
	sub := MediaSubtitles
	streams := []StreamSpec{
		{ID: 1, Bandwidth: 5000, Resolution: "2160p", AudioID: "aud-main", SubtitleID: "sub-main"},
		{ID: 2, MediaType: &audio, GroupID: "aud-main", Language: "en", Bandwidth: 256},
		{ID: 3, MediaType: &audio, GroupID: "aud-alt", Language: "ja", Bandwidth: 192},
		{ID: 4, MediaType: &sub, GroupID: "sub-main", Language: "zh", Name: "Chinese"},
		{ID: 5, MediaType: &sub, GroupID: "sub-alt", Language: "en", Name: "English"},
	}
	got := defaultInteractiveSelection(sortStreamsLikeUpstream(streams))
	if len(got) != 3 || got[0].ID != 1 || got[1].ID != 2 || got[2].ID != 4 {
		t.Fatalf("direct-enter interactive default should follow first stream AUDIO/SUBTITLES groups, got %#v", got)
	}
}

func TestMergeVTTToSRT(t *testing.T) {
	tmp := t.TempDir()
	a := filepath.Join(tmp, "a.vtt")
	b := filepath.Join(tmp, "b.vtt")
	out := filepath.Join(tmp, "out.srt")
	if err := os.WriteFile(a, []byte("WEBVTT\n\n00:00:00.000 --> 00:00:01.000\nhello\n\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nworld\n\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := mergeVTTFiles([]string{a, b}, out, 0, "SRT"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "1\n00:00:00,000 --> 00:00:01,000\nhello") || !strings.Contains(string(got), "2\n00:00:01,000 --> 00:00:02,000\nworld") {
		t.Fatalf("bad srt:\n%s", got)
	}
}

func TestCuesToSRTEmptySubtitleMatchesUpstream(t *testing.T) {
	got := cuesToSRT(nil)
	if got != "1\r\n00:00:00,000 --> 00:00:01,000" {
		t.Fatalf("empty SRT fallback mismatch: %q", got)
	}
}

func TestMergeVTTFilesWithSegmentOffsets(t *testing.T) {
	tmp := t.TempDir()
	a := filepath.Join(tmp, "a.vtt")
	b := filepath.Join(tmp, "b.vtt")
	out := filepath.Join(tmp, "out.vtt")
	if err := os.WriteFile(a, []byte("WEBVTT\n\n00:00:00.000 --> 00:00:01.000\nfirst\n\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("WEBVTT\n\n00:00:00.000 --> 00:00:01.000\nsecond\n\n"), 0644); err != nil {
		t.Fatal(err)
	}
	segments := []Segment{{Index: 0, Duration: 2}, {Index: 1, Duration: 2}}
	if err := mergeVTTFilesWithSegments([]string{a, b}, segments, out, 0, "VTT"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	// long: 部分 HLS 字幕分片都会从 0 秒起表；合并时必须按分片时长恢复全局时间线，避免第二片覆盖第一片。
	if !strings.Contains(string(got), "00:00:00.000 --> 00:00:01.000\nfirst") || !strings.Contains(string(got), "00:00:02.000 --> 00:00:03.000\nsecond") {
		t.Fatalf("bad vtt offsets:\n%s", got)
	}
}

func TestMergeVTTFilesWithAudioStartOffset(t *testing.T) {
	tmp := t.TempDir()
	a := filepath.Join(tmp, "a.vtt")
	out := filepath.Join(tmp, "out.vtt")
	if err := os.WriteFile(a, []byte("WEBVTT\n\n00:00:02.500 --> 00:00:03.500\nlate\n\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := mergeVTTFilesWithSegmentsAndOffset([]string{a}, nil, out, 1500*time.Millisecond, "VTT"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "00:00:01.000 --> 00:00:02.000\nlate") {
		t.Fatalf("audio start offset should shift VTT cues left:\n%s", got)
	}
}

func TestMergeVTTImageSubtitleWritesPNG(t *testing.T) {
	tmp := t.TempDir()
	a := filepath.Join(tmp, "a.vtt")
	out := filepath.Join(tmp, "out.vtt")
	if err := os.WriteFile(filepath.Join(tmp, "0.png"), []byte("exists"), 0644); err != nil {
		t.Fatal(err)
	}
	imagePayload := base64.StdEncoding.EncodeToString([]byte{0x89, 0x50, 0x4e, 0x47})
	if err := os.WriteFile(a, []byte("WEBVTT\n\n00:00:00.000 --> 00:00:01.000\nBase64::"+imagePayload+"\n\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := mergeVTTFilesWithSegments([]string{a}, nil, out, 0, "VTT"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	// long: 上游会把 Base64 图形字幕落盘成 PNG，再让字幕 cue 引用文件名，避免大块图片数据污染 VTT/SRT 文本。
	if !strings.Contains(string(got), "00:00:00.000 --> 00:00:01.000\n1.png") {
		t.Fatalf("subtitle should reference generated png:\n%s", got)
	}
	png, err := os.ReadFile(filepath.Join(tmp, "1.png"))
	if err != nil {
		t.Fatal(err)
	}
	if string(png) != string([]byte{0x89, 0x50, 0x4e, 0x47}) {
		t.Fatalf("generated png content mismatch: %v", png)
	}
}

func TestMergeVTTFilesUsesTimestampMapBeforeSegmentOffsets(t *testing.T) {
	tmp := t.TempDir()
	a := filepath.Join(tmp, "a.vtt")
	b := filepath.Join(tmp, "b.vtt")
	out := filepath.Join(tmp, "out.vtt")
	first := "WEBVTT\nX-TIMESTAMP-MAP=LOCAL:00:00:00.000,MPEGTS:900000\n\n00:00:00.000 --> 00:00:01.000\nfirst\n\n"
	second := "WEBVTT\nX-TIMESTAMP-MAP=LOCAL:00:00:00.000,MPEGTS:1080000\n\n00:00:00.000 --> 00:00:01.500\nsecond\n\n"
	if err := os.WriteFile(a, []byte(first), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte(second), 0644); err != nil {
		t.Fatal(err)
	}
	segments := []Segment{{Index: 0, Duration: 10}, {Index: 1, Duration: 10}}
	if err := mergeVTTFilesWithSegments([]string{a, b}, segments, out, 0, "VTT"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	// long: X-TIMESTAMP-MAP 是 HLS WebVTT 的真实时间轴；存在 MPEGTS 时不能再用分片时长把第二段误推到 10 秒。
	if !strings.Contains(string(got), "00:00:02.000 --> 00:00:03.500\nsecond") || strings.Contains(string(got), "00:00:10.000 --> 00:00:11.500\nsecond") {
		t.Fatalf("bad timestamp-map vtt offsets:\n%s", got)
	}
}

func TestAppendNewLiveSegments(t *testing.T) {
	cur := StreamSpec{Playlist: &Playlist{IsLive: true, Parts: []MediaPart{{Segments: []Segment{{Index: 10, URL: "a.ts"}}}}}}
	next := StreamSpec{Playlist: &Playlist{IsLive: true, RefreshIntervalMS: 1000, Parts: []MediaPart{{Segments: []Segment{{Index: 10, URL: "a.ts"}, {Index: 11, URL: "b.ts"}}}}}}
	added := appendNewLiveSegments(&cur, next)
	segs := cur.Playlist.Parts[0].Segments
	if len(segs) != 2 || segs[1].URL != "b.ts" {
		t.Fatalf("live append failed: %#v", segs)
	}
	if added != 0 {
		t.Fatalf("zero-duration segment should add zero duration, got %f", added)
	}
	if cur.Playlist.RefreshIntervalMS != 1000 {
		t.Fatalf("refresh interval not copied")
	}
}

func TestAppendNewLiveSegmentsUsesIndexForRepeatedHLSURL(t *testing.T) {
	cur := StreamSpec{Playlist: &Playlist{IsLive: true, Parts: []MediaPart{{Segments: []Segment{{Index: 10, URL: "segment.ts"}}}}}}
	next := StreamSpec{Playlist: &Playlist{IsLive: true, Parts: []MediaPart{{Segments: []Segment{
		{Index: 10, URL: "segment.ts"},
		{Index: 11, URL: "segment.ts"},
		{Index: 12, URL: "segment.ts"},
	}}}}}
	added := appendNewLiveSegments(&cur, next)
	segs := cur.Playlist.Parts[0].Segments
	if len(segs) != 3 || segs[1].Index != 11 || segs[2].Index != 12 {
		t.Fatalf("HLS live repeated URLs should be deduped by index like upstream, got %#v", segs)
	}
	if added != 0 {
		t.Fatalf("zero-duration repeated URL additions should add zero duration, got %f", added)
	}
}

func TestAppendNewLiveSegmentsReturnsAddedDuration(t *testing.T) {
	cur := StreamSpec{Playlist: &Playlist{IsLive: true, Parts: []MediaPart{{Segments: []Segment{{Index: 1, URL: "1.ts", Duration: 2}}}}}}
	next := StreamSpec{Playlist: &Playlist{IsLive: true, Parts: []MediaPart{{Segments: []Segment{
		{Index: 1, URL: "1.ts", Duration: 2},
		{Index: 2, URL: "2.ts", Duration: 3.5},
		{Index: 3, URL: "3.ts", Duration: 4.5},
	}}}}}
	if got := appendNewLiveSegments(&cur, next); got != 8 {
		t.Fatalf("new live duration should sum appended segments, got %f", got)
	}
}

func TestLiveSegmentTempPathUsesProgramDateTimeLikeUpstream(t *testing.T) {
	t0 := time.Unix(1781784000, 0).UTC()
	got := segmentTempPath("/tmp/live", Segment{Index: 7, DateTime: &t0}, 3, 4, "ts", true, true)
	if got != filepath.Join("/tmp/live", "1781784000.ts") {
		t.Fatalf("live date-time segment name mismatch: %s", got)
	}
}

func TestLiveSegmentTempPathFallsBackToIndexAndInitName(t *testing.T) {
	media := segmentTempPath("/tmp/live", Segment{Index: 42}, 3, 4, "m4s", true, false)
	if media != filepath.Join("/tmp/live", "42.m4s") {
		t.Fatalf("live index segment name mismatch: %s", media)
	}
	init := segmentTempPath("/tmp/live", Segment{Index: -1}, 0, 4, "mp4", true, false)
	if init != filepath.Join("/tmp/live", "_init.mp4") {
		t.Fatalf("live init segment name mismatch: %s", init)
	}
}

func TestShouldUseLiveSegmentNamesHonorsPerformAsVOD(t *testing.T) {
	stream := StreamSpec{Playlist: &Playlist{WasLive: true}}
	if !shouldUseLiveSegmentNames(stream, defaultOptions()) {
		t.Fatal("original live playlist should use upstream live segment names")
	}
	opt := defaultOptions()
	opt.LivePerformAsVOD = true
	if shouldUseLiveSegmentNames(stream, opt) {
		t.Fatal("live-perform-as-vod should keep normal VOD ordinal segment names")
	}
}

func TestLiveRecordLimitUsesInitialRefreshedDuration(t *testing.T) {
	video := MediaVideo
	audio := MediaAudio
	streams := []StreamSpec{
		{MediaType: &video, Playlist: &Playlist{IsLive: true, Parts: []MediaPart{{Segments: []Segment{{Duration: 2}, {Duration: 3}}}}}},
		{MediaType: &audio, Playlist: &Playlist{IsLive: true, Parts: []MediaPart{{Segments: []Segment{{Duration: 5}}}}}},
	}
	refreshed := liveInitialRefreshedDurations(streams)
	if len(refreshed) != 2 || refreshed[0] != 5 || refreshed[1] != 5 {
		t.Fatalf("initial live durations mismatch: %#v", refreshed)
	}
	if !liveRecordLimitReached(streams, refreshed, 5*time.Second) {
		t.Fatal("initial live window should count toward record limit like upstream")
	}
	refreshed[1] = 4.9
	if liveRecordLimitReached(streams, refreshed, 5*time.Second) {
		t.Fatal("all live tracks must reach record limit before stopping")
	}
}

func TestTrimLiveInitial(t *testing.T) {
	s := StreamSpec{Playlist: &Playlist{IsLive: true, Parts: []MediaPart{{Segments: []Segment{{Index: 1}, {Index: 2}, {Index: 3}}}}}}
	trimLiveInitial(&s, 2)
	if len(s.Playlist.Parts[0].Segments) != 2 || s.Playlist.Parts[0].Segments[0].Index != 2 {
		t.Fatalf("trim failed: %#v", s.Playlist.Parts[0].Segments)
	}
}

func TestSyncLiveStreamsAlignsByProgramDateTime(t *testing.T) {
	t0 := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Second)
	t2 := t0.Add(2 * time.Second)
	streams := []StreamSpec{
		{Playlist: &Playlist{IsLive: true, Parts: []MediaPart{{Segments: []Segment{
			{Index: 10, DateTime: &t0},
			{Index: 11, DateTime: &t1},
			{Index: 12, DateTime: &t2},
		}}}}},
		{Playlist: &Playlist{IsLive: true, Parts: []MediaPart{{Segments: []Segment{
			{Index: 20, DateTime: &t1},
			{Index: 21, DateTime: &t2},
		}}}}},
	}
	syncLiveStreams(streams, 0)
	if got := streams[0].Playlist.Parts[0].Segments; len(got) != 2 || got[0].Index != 11 {
		t.Fatalf("date sync should drop old video segment, got %#v", got)
	}
	if got := streams[1].Playlist.Parts[0].Segments; len(got) != 2 || got[0].Index != 20 {
		t.Fatalf("date sync should keep audio start, got %#v", got)
	}
}

func TestSyncLiveStreamsAlignsByIndexAndTakeWindow(t *testing.T) {
	streams := []StreamSpec{
		{Playlist: &Playlist{IsLive: true, Parts: []MediaPart{{Segments: []Segment{{Index: 5}, {Index: 6}, {Index: 7}, {Index: 8}}}}}},
		{Playlist: &Playlist{IsLive: true, Parts: []MediaPart{{Segments: []Segment{{Index: 6}, {Index: 7}, {Index: 8}}}}}},
	}
	syncLiveStreams(streams, 2)
	for i, stream := range streams {
		got := stream.Playlist.Parts[0].Segments
		if len(got) != 1 || got[0].Index != 8 {
			t.Fatalf("stream %d should keep newest aligned window like upstream, got %#v", i, got)
		}
	}
}

func TestLiveRefreshWaitDurationMatchesUpstreamDefault(t *testing.T) {
	streams := []StreamSpec{
		{Playlist: &Playlist{IsLive: true, Parts: []MediaPart{{Segments: []Segment{{Duration: 8}, {Duration: 8}}}}}},
		{Playlist: &Playlist{IsLive: true, Parts: []MediaPart{{Segments: []Segment{{Duration: 5}, {Duration: 5}}}}}},
	}
	got := liveRefreshWaitDuration(streams, defaultOptions())
	if got != 3*time.Second {
		t.Fatalf("wait should be shortest live window / 2 - 2, got %s", got)
	}
}

func TestLiveRefreshWaitDurationHonorsExplicitOption(t *testing.T) {
	wait := 9
	opt := defaultOptions()
	opt.LiveWaitTime = &wait
	got := liveRefreshWaitDuration(nil, opt)
	if got != 9*time.Second {
		t.Fatalf("explicit live wait time not honored: %s", got)
	}
	wait = 0
	got = liveRefreshWaitDuration(nil, opt)
	if got != time.Second {
		t.Fatalf("non-positive live wait should clamp to one second like upstream, got %s", got)
	}
}

func TestLiveSegmentKeyUsesDateTime(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 6, time.UTC)
	if liveSegmentKey(Segment{DateTime: &now, URL: "x"}) != fmt.Sprintf("%d", now.Unix()) {
		t.Fatalf("date key not used")
	}
	if liveSegmentKey(Segment{Index: 42, URL: "x"}) != "42" {
		t.Fatalf("index key should be used when PROGRAM-DATE-TIME is absent")
	}
}

func TestLiveRealtimeMergeFilesAppendsAndDeletesNonInit(t *testing.T) {
	tmp := t.TempDir()
	initFile := filepath.Join(tmp, "000.mp4")
	seg1 := filepath.Join(tmp, "001.m4s")
	seg2 := filepath.Join(tmp, "002.m4s")
	output := filepath.Join(tmp, "live.mp4")
	for path, content := range map[string]string{
		initFile: "init",
		seg1:     "one",
		seg2:     "two",
	} {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	files := []string{initFile, seg1, seg2}
	segments := []Segment{{Index: -1}, {Index: 1}, {Index: 2}}
	if err := liveRealtimeMergeFiles(files, segments, output, false); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "initonetwo" {
		t.Fatalf("live realtime merge should append files in order, got %q", got)
	}
	if _, err := os.Stat(initFile); err != nil {
		t.Fatalf("init segment should be kept: %v", err)
	}
	for _, path := range []string{seg1, seg2} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("non-init live segment should be deleted, path=%s err=%v", path, err)
		}
	}
	next := filepath.Join(tmp, "003.m4s")
	if err := os.WriteFile(next, []byte("three"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := liveRealtimeMergeFiles([]string{next}, []Segment{{Index: 3}}, output, true); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "initonetwothree" {
		t.Fatalf("second realtime merge should append instead of overwrite, got %q", got)
	}
	if _, err := os.Stat(next); err != nil {
		t.Fatalf("keepSegments=true should preserve media segment: %v", err)
	}
}

func TestCENCDeferToExternalDecrypt(t *testing.T) {
	data := []byte("mp4-fragment")
	got, err := decryptSegment(data, EncryptInfo{Method: EncryptCENC})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Fatalf("unexpected data mutation")
	}
}

func TestExternalMP4DecryptCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	tmp := t.TempDir()
	src := filepath.Join(tmp, "enc.mp4")
	if err := os.WriteFile(src, []byte("encrypted"), 0644); err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(tmp, "mp4decrypt")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\ncp \"$3\" \"$4\"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	opt := defaultOptions()
	opt.DecryptionBinaryPath = tool
	opt.Keys = []string{"00000000000000000000000000000000:00112233445566778899aabbccddeeff"}
	out, err := decryptMP4Output(src, opt)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "encrypted" {
		t.Fatalf("bad decrypted output: %q", b)
	}
}

func TestMP4DecryptUsesTempNamesWorkingDirectoryAndRelativeInit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	tmp := t.TempDir()
	workDir := filepath.Join(tmp, "中文 路径")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(workDir, "源 文件.mp4")
	initFile := filepath.Join(workDir, "init 文件.mp4")
	if err := os.WriteFile(src, []byte("encrypted"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(initFile, []byte("init"), 0644); err != nil {
		t.Fatal(err)
	}
	argsLog := filepath.Join(tmp, "args.txt")
	pwdLog := filepath.Join(tmp, "pwd.txt")
	tool := filepath.Join(tmp, "mp4decrypt")
	script := "#!/bin/sh\npwd > \"" + pwdLog + "\"\nprintf '%s\\n' \"$@\" > \"" + argsLog + "\"\nprev=''\nlast=''\nfor arg in \"$@\"; do prev=\"$last\"; last=\"$arg\"; done\ncp \"$prev\" \"$last\"\n"
	if err := os.WriteFile(tool, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	opt := defaultOptions()
	opt.DecryptionBinaryPath = tool
	opt.Keys = []string{"11111111111111111111111111111111:00112233445566778899aabbccddeeff"}
	if _, err := decryptMP4File(src, opt, "11111111111111111111111111111111", initFile); err != nil {
		t.Fatal(err)
	}
	pwdBytes, err := os.ReadFile(pwdLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(pwdBytes)) != workDir {
		t.Fatalf("mp4decrypt should run in media directory, got %q want %q", strings.TrimSpace(string(pwdBytes)), workDir)
	}
	argsBytes, err := os.ReadFile(argsLog)
	if err != nil {
		t.Fatal(err)
	}
	args := string(argsBytes)
	if strings.Contains(args, filepath.Base(src)) {
		t.Fatalf("mp4decrypt should receive temp media name instead of original path, args:\n%s", args)
	}
	if !strings.Contains(args, "\n--fragments-info\n"+filepath.Base(initFile)+"\n") {
		t.Fatalf("same-directory init should be passed as relative filename, args:\n%s", args)
	}
	if strings.Contains(args, initFile) {
		t.Fatalf("same-directory init should not use absolute path, args:\n%s", args)
	}
	got, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "encrypted" {
		t.Fatalf("source path should contain decrypted replacement, got %q", got)
	}
	if matches, err := filepath.Glob(filepath.Join(workDir, "n_m3u8dl_*")); err != nil {
		t.Fatal(err)
	} else if len(matches) != 0 {
		t.Fatalf("temporary mp4decrypt files should be cleaned: %#v", matches)
	}
}

func TestMP4DecryptUsesTrackOneForWidevinePSSHMultiDRM(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	tmp := t.TempDir()
	detectedKID := "bb0102030405060708090a0b0c0d0ecc"
	key := "00112233445566778899aabbccddeeff"
	src := filepath.Join(tmp, "multi.mp4")
	if err := os.WriteFile(src, mustMultiDRMInitWithWidevineKID([]byte{0xbb, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 0xcc}), 0644); err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(tmp, "keys.txt")
	if err := os.WriteFile(keyFile, []byte(detectedKID+":"+key+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	argsLog := filepath.Join(tmp, "mp4decrypt-args.txt")
	tool := filepath.Join(tmp, "mp4decrypt")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"" + argsLog + "\"\nprev=''\nlast=''\nfor arg in \"$@\"; do prev=\"$last\"; last=\"$arg\"; done\ncp \"$prev\" \"$last\"\n"
	if err := os.WriteFile(tool, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	opt := defaultOptions()
	opt.DecryptionBinaryPath = tool
	opt.KeyTextFile = keyFile
	if _, err := decryptMP4File(src, opt, "", ""); err != nil {
		t.Fatal(err)
	}
	argsBytes, err := os.ReadFile(argsLog)
	if err != nil {
		t.Fatal(err)
	}
	args := "\n" + string(argsBytes) + "\n"
	if !strings.Contains(args, "\n--key\n1:"+key+"\n") {
		t.Fatalf("multiDRM should use mp4decrypt track id 1, args:\n%s", args)
	}
	if strings.Contains(args, detectedKID+":"+key) {
		t.Fatalf("multiDRM should not pass detected KID directly to mp4decrypt, args:\n%s", args)
	}
}

func TestShakaDetectsKIDWhenMP4ParsingFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	tmp := t.TempDir()
	src := filepath.Join(tmp, "enc.webm")
	if err := os.WriteFile(src, []byte("encrypted"), 0644); err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(tmp, "keys.txt")
	detectedKID := "abcdefabcdefabcdefabcdefabcdefab"
	key := "00112233445566778899aabbccddeeff"
	if err := os.WriteFile(keyFile, []byte(detectedKID+":"+key+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	argsLog := filepath.Join(tmp, "shaka-args.txt")
	tool := filepath.Join(tmp, "shaka-packager")
	script := `#!/bin/sh
printf -- '---\n%s\n' "$*" >> __ARGSLOG__
case "$*" in
  *key_id=00000000000000000000000000000000:key=00000000000000000000000000000000*)
    echo 'Key for key_id=__KID__ was not found' >&2
    exit 1
    ;;
esac
input=''
output=''
for arg in "$@"; do
  case "$arg" in
    input=*)
      rest=${arg#input=}
      input=${rest%%,*}
      ;;
  esac
  case "$arg" in
    *output=*)
      rest=${arg#*output=}
      output=${rest%%,*}
      ;;
  esac
done
cp "$input" "$output"
`
	script = strings.ReplaceAll(script, "__ARGSLOG__", argsLog)
	script = strings.ReplaceAll(script, "__KID__", detectedKID)
	if err := os.WriteFile(tool, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	opt := defaultOptions()
	opt.DecryptionEngine = "SHAKA_PACKAGER"
	opt.DecryptionBinaryPath = tool
	opt.KeyTextFile = keyFile
	out, err := decryptMP4File(src, opt, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if out != src {
		t.Fatalf("decrypt should replace in place, got %s", out)
	}
	got, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "encrypted" {
		t.Fatalf("bad shaka decrypted content: %q", got)
	}
	args, err := os.ReadFile(argsLog)
	if err != nil {
		t.Fatal(err)
	}
	joined := string(args)
	if !strings.Contains(joined, "key_id="+detectedKID+":key="+key) {
		t.Fatalf("shaka decrypt should use detected kid from probe, args:\n%s", joined)
	}
}

func TestRealtimeInitUsesShakaDetectedKIDButSkipsStandaloneInitDecrypt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	tmp := t.TempDir()
	initSrc := filepath.Join(tmp, "init-src.mp4")
	if err := os.WriteFile(initSrc, []byte("encrypted-init"), 0644); err != nil {
		t.Fatal(err)
	}
	detectedKID := "1234567890abcdef1234567890abcdef"
	key := "00112233445566778899aabbccddeeff"
	keyFile := filepath.Join(tmp, "keys.txt")
	if err := os.WriteFile(keyFile, []byte(detectedKID+":"+key+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	argsLog := filepath.Join(tmp, "shaka-init-args.txt")
	tool := filepath.Join(tmp, "shaka-packager")
	script := `#!/bin/sh
printf -- '---\n%s\n' "$*" >> __ARGSLOG__
case "$*" in
  *key_id=00000000000000000000000000000000:key=00000000000000000000000000000000*)
    echo 'Key for key_id=__KID__ was not found' >&2
    exit 1
    ;;
esac
input=''
output=''
for arg in "$@"; do
  case "$arg" in
    input=*)
      rest=${arg#input=}
      input=${rest%%,*}
      ;;
  esac
  case "$arg" in
    *output=*)
      rest=${arg#*output=}
      output=${rest%%,*}
      ;;
  esac
done
printf 'decrypted-init:%s' "$(cat "$input")" > "$output"
`
	script = strings.ReplaceAll(script, "__ARGSLOG__", argsLog)
	script = strings.ReplaceAll(script, "__KID__", detectedKID)
	if err := os.WriteFile(tool, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	opt := defaultOptions()
	opt.MP4RealTimeDecryption = true
	opt.DecryptionEngine = "SHAKA_PACKAGER"
	opt.DecryptionBinaryPath = tool
	opt.KeyTextFile = keyFile
	actual, kid, err := downloadRealtimeInitSegment(
		context.Background(),
		&http.Client{},
		Segment{URL: initSrc, Index: -1, Encrypt: EncryptInfo{Method: EncryptCENC}},
		filepath.Join(tmp, "init.mp4"),
		opt,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if kid != detectedKID {
		t.Fatalf("expected shaka detected kid %s, got %s", detectedKID, kid)
	}
	got, err := os.ReadFile(actual)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "encrypted-init" {
		t.Fatalf("shaka should leave standalone init untouched for later init+fragment decrypt, got %q", got)
	}
	args, err := os.ReadFile(argsLog)
	if err != nil {
		t.Fatal(err)
	}
	joined := string(args)
	if !strings.Contains(joined, "key_id="+zeroKID+":key="+zeroKID) {
		t.Fatalf("shaka probe was not invoked, args:\n%s", joined)
	}
	if strings.Contains(joined, "key_id="+detectedKID+":key="+key) {
		t.Fatalf("shaka should not decrypt standalone init with matched key, args:\n%s", joined)
	}
}

func TestFFmpegDecryptUsesKeyMatchingDetectedKID(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	tmp := t.TempDir()
	src := filepath.Join(tmp, "enc.mp4")
	if err := os.WriteFile(src, []byte("encrypted"), 0644); err != nil {
		t.Fatal(err)
	}
	argsLog := filepath.Join(tmp, "args.txt")
	tool := filepath.Join(tmp, "ffmpeg")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"" + argsLog + "\"\ninput=\"\"\nprev=\"\"\nlast=\"\"\nfor arg in \"$@\"; do if [ \"$prev\" = \"-i\" ]; then input=\"$arg\"; fi; prev=\"$arg\"; last=\"$arg\"; done\ncp \"$input\" \"$last\"\n"
	if err := os.WriteFile(tool, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	opt := defaultOptions()
	opt.DecryptionEngine = "FFMPEG"
	opt.DecryptionBinaryPath = tool
	opt.Keys = []string{
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	if _, err := decryptMP4File(src, opt, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ""); err != nil {
		t.Fatal(err)
	}
	argsBytes, err := os.ReadFile(argsLog)
	if err != nil {
		t.Fatal(err)
	}
	args := string(argsBytes)
	if !strings.Contains(args, "\n-decryption_key\nbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n") {
		t.Fatalf("ffmpeg should use key matching detected KID, args:\n%s", args)
	}
	if strings.Contains(args, "\n-decryption_key\naaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n") {
		t.Fatalf("ffmpeg used first non-matching key, args:\n%s", args)
	}
}

func TestNormalizeMP4DecryptKeyUsesTrackIDForZeroKID(t *testing.T) {
	key := "00112233445566778899aabbccddeeff"
	if got := normalizeMP4DecryptKey(zeroKID+":"+key, zeroKID); got != "1:"+key {
		t.Fatalf("zero kid pair should use mp4decrypt track id, got %s", got)
	}
	if got := normalizeMP4DecryptKey(key, zeroKID); got != "1:"+key {
		t.Fatalf("single key with zero kid should use mp4decrypt track id, got %s", got)
	}
	if got := normalizeMP4DecryptKey("11111111111111111111111111111111:"+key, "11111111111111111111111111111111"); got != "11111111111111111111111111111111:"+key {
		t.Fatalf("non-zero kid should keep kid:key form, got %s", got)
	}
	if got := normalizeMP4DecryptKey(key, "11111111111111111111111111111111"); got != "11111111111111111111111111111111:"+key {
		t.Fatalf("single key with detected kid should use kid:key, got %s", got)
	}
}

func TestSelectDecryptKeyPairMatchesCurrentKIDLikeUpstream(t *testing.T) {
	keys := []string{
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	got, ok := selectDecryptKeyPair(keys, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	if !ok || got != keys[1] {
		t.Fatalf("expected matching KID key, got %q ok=%v", got, ok)
	}
	if got, ok := selectDecryptKeyPair(keys, "cccccccccccccccccccccccccccccccc"); ok || got != "" {
		t.Fatalf("multiple non-matching KID keys should not pick the first key, got %q ok=%v", got, ok)
	}
	upperKIDKey := strings.ToUpper("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb") + ":bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if got, ok := selectDecryptKeyPair([]string{upperKIDKey}, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"); ok || got != "" {
		t.Fatalf("runtime KID matching should be case-sensitive like upstream StartsWith, got %q ok=%v", got, ok)
	}
	got, ok = selectDecryptKeyPair([]string{"00112233445566778899aabbccddeeff"}, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	if !ok || got != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb:00112233445566778899aabbccddeeff" {
		t.Fatalf("single raw key should be paired with detected KID, got %q ok=%v", got, ok)
	}
}

func TestNormalizeKeyForShakaUsesLabelForZeroKID(t *testing.T) {
	key := "00112233445566778899aabbccddeeff"
	label, keyID, gotKey := normalizeKeyForShaka("11111111111111111111111111111111:"+key, zeroKID)
	if label != "1" || keyID != zeroKID || gotKey != key {
		t.Fatalf("zero kid shaka key should use label=1 and zero key id, got label=%s keyID=%s key=%s", label, keyID, gotKey)
	}
	label, keyID, gotKey = normalizeKeyForShaka("11111111111111111111111111111111:"+key, "11111111111111111111111111111111")
	if label != "" || keyID != "11111111111111111111111111111111" || gotKey != key {
		t.Fatalf("non-zero shaka key should keep kid:key, got label=%s keyID=%s key=%s", label, keyID, gotKey)
	}
}

func TestMuxOutputsByMkvmergeCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	tmp := t.TempDir()
	in := filepath.Join(tmp, "in.mp4")
	if err := os.WriteFile(in, []byte("video"), 0644); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(tmp, "mkvmerge-args.txt")
	tool := filepath.Join(tmp, "mkvmerge")
	script := fmt.Sprintf("#!/bin/sh\nlog=%q\n: > \"$log\"\nout=\"\"\nwhile [ $# -gt 0 ]; do printf '%%s\\n' \"$1\" >> \"$log\"; if [ \"$1\" = \"--output\" ]; then shift; printf '%%s\\n' \"$1\" >> \"$log\"; out=\"$1\"; fi; shift; done\nprintf mkv > \"$out\"\n", logPath)
	if err := os.WriteFile(tool, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	opt := defaultOptions()
	opt.SaveDir = tmp
	opt.SaveName = "mux"
	opt.MuxAfterDone = &MuxOptions{Muxer: "mkvmerge", BinPath: tool, Keep: true}
	out, err := muxOutputsByMkvmerge([]outputFile{{Path: in}}, opt)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "mkv" {
		t.Fatalf("bad mux output: %q", b)
	}
	argsBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	args := "\n" + string(argsBytes)
	for _, want := range []string{
		"\n--no-chapters\n",
		"\n--language\n0:und\n",
	} {
		if !strings.Contains(args, want) {
			t.Fatalf("mkvmerge args missing %q:\n%s", want, args)
		}
	}
}

func TestMuxOutputsByFFmpegUsesUpstreamMetadataAndDispositionArgs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	tmp := t.TempDir()
	videoPath := filepath.Join(tmp, "video.mp4")
	audio1Path := filepath.Join(tmp, "audio1.m4a")
	audio2Path := filepath.Join(tmp, "audio2.m4a")
	subPath := filepath.Join(tmp, "sub.srt")
	for _, p := range []string{videoPath, audio1Path, audio2Path, subPath} {
		if err := os.WriteFile(p, []byte("track"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	logPath := filepath.Join(tmp, "ffmpeg-args.txt")
	tool := filepath.Join(tmp, "ffmpeg")
	script := fmt.Sprintf("#!/bin/sh\nlog=%q\n: > \"$log\"\nlast=\"\"\nfor arg in \"$@\"; do printf '%%s\\n' \"$arg\" >> \"$log\"; last=\"$arg\"; done\nprintf mux > \"$last\"\n", logPath)
	if err := os.WriteFile(tool, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	video := MediaVideo
	audio := MediaAudio
	sub := MediaSubtitles
	opt := defaultOptions()
	opt.SaveDir = tmp
	opt.SaveName = "muxargs"
	opt.FFmpegBinaryPath = tool
	opt.MuxAfterDone = &MuxOptions{Format: "mp4", Keep: true}
	out, err := muxOutputs([]outputFile{
		{Path: videoPath, MediaType: &video},
		{Path: audio1Path, MediaType: &audio, Language: "en"},
		{Path: audio2Path, MediaType: &audio, Language: "ja-JP"},
		{Path: subPath, MediaType: &sub, Language: "zh-Hans"},
	}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatal(err)
	}
	argsBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	args := "\n" + string(argsBytes)
	for _, want := range []string{
		"\n-map_metadata\n-1\n",
		"\n-ignore_unknown\n",
		"\n-copy_unknown\n",
		"\n-disposition:v:0\ndefault\n",
		"\n-disposition:a:0\ndefault\n",
		"\n-disposition:a:1\n0\n",
		"\n-disposition:s\n0\n",
		"\n-metadata:s:0\nlanguage=und\n",
		"\n-metadata:s:1\nlanguage=eng\n",
		"\n-metadata:s:2\nlanguage=jpn\n",
		"\n-metadata:s:3\nlanguage=chi\n",
		"\n-metadata:s:1\ntitle=English\n",
		"\n-metadata:s:2\ntitle=Japanese (Japan)\n",
		"\n-metadata:s:3\ntitle=中文（简体）\n",
	} {
		if !strings.Contains(args, want) {
			t.Fatalf("ffmpeg args missing %q:\n%s", want, args)
		}
	}
}

func TestMuxOutputsByFFmpegMetadataUsesOutputStreamIndex(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	tmp := t.TempDir()
	videoPath := filepath.Join(tmp, "video-with-audio.mp4")
	audioPath := filepath.Join(tmp, "audio.m4a")
	subPath := filepath.Join(tmp, "sub.srt")
	for _, p := range []string{videoPath, audioPath, subPath} {
		if err := os.WriteFile(p, []byte("track"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	logPath := filepath.Join(tmp, "ffmpeg-args.txt")
	tool := filepath.Join(tmp, "ffmpeg")
	script := fmt.Sprintf("#!/bin/sh\nlog=%q\n: > \"$log\"\nlast=\"\"\nfor arg in \"$@\"; do printf '%%s\\n' \"$arg\" >> \"$log\"; last=\"$arg\"; done\nprintf mux > \"$last\"\n", logPath)
	if err := os.WriteFile(tool, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	video := MediaVideo
	audio := MediaAudio
	sub := MediaSubtitles
	opt := defaultOptions()
	opt.SaveDir = tmp
	opt.SaveName = "mux-stream-index"
	opt.FFmpegBinaryPath = tool
	opt.MuxAfterDone = &MuxOptions{Format: "mp4", Keep: true}
	_, err := muxOutputs([]outputFile{
		{Path: videoPath, MediaType: &video, StreamCount: 2},
		{Path: audioPath, MediaType: &audio, Language: "en"},
		{Path: subPath, MediaType: &sub, Language: "zh-Hans"},
	}, opt)
	if err != nil {
		t.Fatal(err)
	}
	argsBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	args := "\n" + string(argsBytes)
	// long: ffmpeg 的 -metadata:s:N 使用输出流序号；首个输入若含多条流，后续轨道不能继续用输入文件下标。
	for _, want := range []string{
		"\n-map_metadata\n-1\n",
		"\n-metadata:s:2\nlanguage=eng\n",
		"\n-metadata:s:3\nlanguage=chi\n",
	} {
		if !strings.Contains(args, want) {
			t.Fatalf("ffmpeg args missing %q:\n%s", want, args)
		}
	}
	if strings.Contains(args, "\n-metadata:s:1\nlanguage=eng\n") {
		t.Fatalf("audio metadata should not use input index when previous input has two streams:\n%s", args)
	}
}

func TestMuxDispositionArgsFollowUpstreamAudioSubtitleCondition(t *testing.T) {
	audio := MediaAudio
	sub := MediaSubtitles
	gotAudioOnly := muxDispositionArgs([]outputFile{{MediaType: &audio}})
	if !containsArgPair(gotAudioOnly, "-disposition:s", "0") {
		t.Fatalf("upstream adds subtitle disposition when audio exists, got %#v", gotAudioOnly)
	}
	gotSubOnly := muxDispositionArgs([]outputFile{{MediaType: &sub}})
	if containsArgPair(gotSubOnly, "-disposition:s", "0") {
		t.Fatalf("upstream typo does not add subtitle disposition for subtitle-only inputs, got %#v", gotSubOnly)
	}
}

func containsArgPair(args []string, key, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == key && args[i+1] == value {
			return true
		}
	}
	return false
}

func TestProbeMediaInfoUsesSiblingFFprobe(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	tmp := t.TempDir()
	ffmpeg := filepath.Join(tmp, "ffmpeg")
	ffprobe := filepath.Join(tmp, "ffprobe")
	if err := os.WriteFile(ffmpeg, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nprintf '%s\\n' '{\"streams\":[{\"codec_type\":\"subtitle\",\"codec_name\":\"webvtt\",\"codec_tag_string\":\"[0][0][0][0]\"}]}'\n"
	if err := os.WriteFile(ffprobe, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	media := filepath.Join(tmp, "seg.ts")
	if err := os.WriteFile(media, []byte("probe"), 0644); err != nil {
		t.Fatal(err)
	}
	opt := defaultOptions()
	opt.FFmpegBinaryPath = ffmpeg
	infos := probeMediaInfo(media, opt)
	if len(infos) != 1 || infos[0].Type != "subtitle" || infos[0].CodecName != "webvtt" {
		t.Fatalf("unexpected media info: %#v", infos)
	}
}

func TestProbeMediaInfoReadsAudioStartTime(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	tmp := t.TempDir()
	ffmpeg := filepath.Join(tmp, "ffmpeg")
	ffprobe := filepath.Join(tmp, "ffprobe")
	if err := os.WriteFile(ffmpeg, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nprintf '%s\\n' '{\"streams\":[{\"codec_type\":\"audio\",\"codec_name\":\"aac\",\"codec_tag_string\":\"mp4a\",\"start_time\":\"1.234\"}]}'\n"
	if err := os.WriteFile(ffprobe, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	media := filepath.Join(tmp, "audio.ts")
	if err := os.WriteFile(media, []byte("probe"), 0644); err != nil {
		t.Fatal(err)
	}
	opt := defaultOptions()
	opt.FFmpegBinaryPath = ffmpeg
	infos := probeMediaInfo(media, opt)
	if len(infos) != 1 {
		t.Fatalf("unexpected media info: %#v", infos)
	}
	if got := infos[0].StartTime; got != 1234*time.Millisecond {
		t.Fatalf("unexpected audio start time: %s", got)
	}
	if start, ok := mediaInfosAudioStart(infos); !ok || start != 1234*time.Millisecond {
		t.Fatalf("audio start helper failed start=%s ok=%v", start, ok)
	}
}

func TestApplyMediaInfoConvertsSubtitleTSToVTT(t *testing.T) {
	stream := StreamSpec{Extension: "ts"}
	opt := defaultOptions()
	applyMediaInfoToStream(&stream, &opt, []mediaInfo{{Type: "subtitle", CodecName: "webvtt"}})
	if stream.MediaType == nil || *stream.MediaType != MediaSubtitles {
		t.Fatalf("stream should become subtitle: %#v", stream.MediaType)
	}
	if stream.Extension != "vtt" {
		t.Fatalf("subtitle ts extension should become vtt, got %s", stream.Extension)
	}
}

func TestApplyMediaInfoDolbyVisionForcesBinaryMerge(t *testing.T) {
	stream := StreamSpec{}
	opt := defaultOptions()
	opt.MuxAfterDone = &MuxOptions{Format: "mp4"}
	applyMediaInfoToStream(&stream, &opt, []mediaInfo{{Type: "video", CodecName: "dvhe.05.06", DolbyVision: true}})
	if !opt.BinaryMerge {
		t.Fatal("Dolby Vision should force binary merge like upstream")
	}
	if opt.MuxAfterDone != nil {
		t.Fatal("Dolby Vision should disable final mux like upstream")
	}
}

func TestLiveAudioStartTrackerWaitsForStartTime(t *testing.T) {
	tracker := &liveAudioStartTracker{}
	done := make(chan struct{})
	go func() {
		time.Sleep(20 * time.Millisecond)
		tracker.set(2 * time.Second)
		close(done)
	}()
	got, ok := tracker.wait(time.Second)
	if !ok || got != 2*time.Second {
		t.Fatalf("tracker wait got %s ok=%v", got, ok)
	}
	<-done
}

func TestMediaInfosUseAACFilterRequiresAACAudio(t *testing.T) {
	cases := []struct {
		name  string
		infos []mediaInfo
		want  bool
	}{
		{name: "aac audio", infos: []mediaInfo{{Type: "audio", CodecName: "aac"}}, want: true},
		{name: "aac latm audio", infos: []mediaInfo{{Type: "audio", CodecName: "aac_latm"}}, want: true},
		{name: "aac with video", infos: []mediaInfo{{Type: "video", CodecName: "h264"}, {Type: "audio", CodecName: "aac"}}, want: true},
		{name: "eac3 audio", infos: []mediaInfo{{Type: "audio", CodecName: "eac3"}}, want: false},
		{name: "mixed audio", infos: []mediaInfo{{Type: "audio", CodecName: "aac"}, {Type: "audio", CodecName: "ac3"}}, want: false},
		{name: "video only", infos: []mediaInfo{{Type: "video", CodecName: "h264"}}, want: false},
		{name: "empty", infos: nil, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mediaInfosUseAACFilter(tc.infos); got != tc.want {
				t.Fatalf("mediaInfosUseAACFilter()=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestNormalizeLanguageUsesFullUpstreamTable(t *testing.T) {
	audio := MediaAudio
	sub := MediaSubtitles
	cases := []struct {
		name      string
		lang      string
		mediaType *MediaType
		wantCode  string
		wantName  string
	}{
		{name: "regional english audio", lang: "en-US", mediaType: &audio, wantCode: "eng", wantName: "English (United States)"},
		{name: "filipino subtitle", lang: "fil-PH", mediaType: &sub, wantCode: "fil", wantName: "Filipino (Philippines)"},
		{name: "cantonese audio alias", lang: "Cantonese", mediaType: &audio, wantCode: "chi", wantName: "粵語"},
		{name: "two letter fallback", lang: "pt-XX", mediaType: &sub, wantCode: "por", wantName: "Portuguese"},
		{name: "unknown becomes und", lang: "xx-Test", mediaType: &audio, wantCode: "und", wantName: "xx-Test"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, name := normalizeLanguageAndName(tc.lang, "", tc.mediaType)
			if code != tc.wantCode || name != tc.wantName {
				t.Fatalf("unexpected language normalization code=%q name=%q", code, name)
			}
		})
	}
}

func TestMuxOutputsByFFmpegMKVSubtitleCodecMatchesUpstream(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	tmp := t.TempDir()
	videoPath := filepath.Join(tmp, "video.mp4")
	srtPath := filepath.Join(tmp, "sub.srt")
	vttPath := filepath.Join(tmp, "sub.vtt")
	for _, p := range []string{videoPath, srtPath, vttPath} {
		if err := os.WriteFile(p, []byte("track"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	logPath := filepath.Join(tmp, "ffmpeg-args.txt")
	tool := filepath.Join(tmp, "ffmpeg")
	script := fmt.Sprintf("#!/bin/sh\nlog=%q\n: > \"$log\"\nlast=\"\"\nfor arg in \"$@\"; do printf '%%s\\n' \"$arg\" >> \"$log\"; last=\"$arg\"; done\nprintf mux > \"$last\"\n", logPath)
	if err := os.WriteFile(tool, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	sub := MediaSubtitles
	opt := defaultOptions()
	opt.SaveDir = tmp
	opt.SaveName = "mkv-srt"
	opt.FFmpegBinaryPath = tool
	opt.MuxAfterDone = &MuxOptions{Format: "mkv", Keep: true}
	if _, err := muxOutputs([]outputFile{{Path: videoPath}, {Path: srtPath, MediaType: &sub}}, opt); err != nil {
		t.Fatal(err)
	}
	argsBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains("\n"+string(argsBytes), "\n-c:s\nsrt\n") {
		t.Fatalf("mkv with srt input should use -c:s srt, args:\n%s", argsBytes)
	}
	opt.SaveName = "mkv-vtt"
	if _, err := muxOutputs([]outputFile{{Path: videoPath}, {Path: vttPath, MediaType: &sub}}, opt); err != nil {
		t.Fatal(err)
	}
	argsBytes, err = os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains("\n"+string(argsBytes), "\n-c:s\nwebvtt\n") {
		t.Fatalf("mkv without srt input should use -c:s webvtt, args:\n%s", argsBytes)
	}
}

func TestMuxOutputsByFFmpegUsesMuxBinPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	tmp := t.TempDir()
	in := filepath.Join(tmp, "video.mp4")
	if err := os.WriteFile(in, []byte("track"), 0644); err != nil {
		t.Fatal(err)
	}
	globalLog := filepath.Join(tmp, "global-ffmpeg.txt")
	muxLog := filepath.Join(tmp, "mux-ffmpeg.txt")
	globalTool := filepath.Join(tmp, "global-ffmpeg")
	muxTool := filepath.Join(tmp, "mux-ffmpeg")
	globalScript := fmt.Sprintf("#!/bin/sh\nprintf global > %q\nexit 2\n", globalLog)
	if err := os.WriteFile(globalTool, []byte(globalScript), 0755); err != nil {
		t.Fatal(err)
	}
	muxScript := fmt.Sprintf("#!/bin/sh\nprintf mux > %q\nlast=\"\"\nfor arg in \"$@\"; do last=\"$arg\"; done\nprintf muxed > \"$last\"\n", muxLog)
	if err := os.WriteFile(muxTool, []byte(muxScript), 0755); err != nil {
		t.Fatal(err)
	}
	opt := defaultOptions()
	opt.SaveDir = tmp
	opt.SaveName = "mux-bin-path"
	opt.FFmpegBinaryPath = globalTool
	opt.MuxAfterDone = &MuxOptions{Format: "mp4", Muxer: "ffmpeg", BinPath: muxTool, Keep: true}
	out, err := muxOutputs([]outputFile{{Path: in}}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(globalLog); !os.IsNotExist(err) {
		t.Fatalf("global ffmpeg should not be invoked, stat err=%v", err)
	}
	b, err := os.ReadFile(muxLog)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "mux" {
		t.Fatalf("mux bin_path was not invoked: %q", b)
	}
}

func TestMuxOutputsByFFmpegCleansOnlyActualInputsLikeUpstream(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	tmp := t.TempDir()
	videoPath := filepath.Join(tmp, "video.mp4")
	skippedSubPath := filepath.Join(tmp, "downloaded-sub.srt")
	importSubPath := filepath.Join(tmp, "import-sub.srt")
	for _, p := range []string{videoPath, skippedSubPath, importSubPath} {
		if err := os.WriteFile(p, []byte("track"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	tool := filepath.Join(tmp, "ffmpeg")
	script := "#!/bin/sh\nlast=\"\"\nfor arg in \"$@\"; do last=\"$arg\"; done\nprintf muxed > \"$last\"\n"
	if err := os.WriteFile(tool, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	sub := MediaSubtitles
	opt := defaultOptions()
	opt.SaveDir = tmp
	opt.SaveName = "mux-cleanup"
	opt.FFmpegBinaryPath = tool
	opt.MuxAfterDone = &MuxOptions{Format: "mp4", Muxer: "ffmpeg", Keep: false, SkipSubtitle: true}
	opt.MuxImports = []string{"path=" + importSubPath + ":lang=en:name=External"}
	if _, err := muxOutputs([]outputFile{
		{Path: videoPath},
		{Path: skippedSubPath, MediaType: &sub},
	}, opt); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(videoPath); !os.IsNotExist(err) {
		t.Fatalf("actual video input should be cleaned, stat err=%v", err)
	}
	if _, err := os.Stat(importSubPath); !os.IsNotExist(err) {
		t.Fatalf("mux-import input should be cleaned like upstream, stat err=%v", err)
	}
	if _, err := os.Stat(skippedSubPath); err != nil {
		t.Fatalf("skip_sub subtitle should be kept, stat err=%v", err)
	}
}

func TestFFmpegMergeSupportsAdditionalUpstreamFormats(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	tmp := t.TempDir()
	in := filepath.Join(tmp, "clip.ts")
	if err := os.WriteFile(in, []byte("track"), 0644); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(tmp, "merge-args.txt")
	tool := filepath.Join(tmp, "ffmpeg")
	script := fmt.Sprintf("#!/bin/sh\nlog=%q\nprintf '%%s\\n' \"$@\" >> \"$log\"\nlast=\"\"\nfor arg in \"$@\"; do last=\"$arg\"; done\nprintf merged > \"$last\"\n", logPath)
	if err := os.WriteFile(tool, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	opt := defaultOptions()
	opt.FFmpegBinaryPath = tool
	cases := []struct {
		format string
		ext    string
	}{
		{format: "flv", ext: ".flv"},
		{format: "eac3", ext: ".eac3"},
		{format: "ac3", ext: ".ac3"},
		{format: "ts", ext: ".ts"},
		{format: "m4a", ext: ".m4a"},
		{format: "aac", ext: ".m4a"},
	}
	for _, tc := range cases {
		out, err := ffmpegMerge([]string{in}, filepath.Join(tmp, "out_"+tc.format), tc.format, opt, true)
		if err != nil {
			t.Fatalf("%s merge failed: %v", tc.format, err)
		}
		if filepath.Ext(out) != tc.ext {
			t.Fatalf("%s output extension mismatch: %s", tc.format, out)
		}
		got, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "merged" {
			t.Fatalf("%s output content mismatch: %q", tc.format, got)
		}
	}
	argsBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	args := "\n" + string(argsBytes)
	for _, want := range []string{
		"\n-bsf:a\n",
		"\n0:a\n",
		"\n-f\nmpegts\n",
		"\n-bsf:v\nh264_mp4toannexb\n",
		"\n-f\nmp4\n",
	} {
		if !strings.Contains(args, want) {
			t.Fatalf("ffmpeg merge args missing %q:\n%s", want, args)
		}
	}
}

func TestFFmpegMergeAACFilterIsConditional(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	tmp := t.TempDir()
	in := filepath.Join(tmp, "clip.ts")
	if err := os.WriteFile(in, []byte("track"), 0644); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(tmp, "merge-args.txt")
	tool := filepath.Join(tmp, "ffmpeg")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %q\nlast=\"\"\nfor arg in \"$@\"; do last=\"$arg\"; done\nprintf merged > \"$last\"\n", logPath)
	if err := os.WriteFile(tool, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	opt := defaultOptions()
	opt.FFmpegBinaryPath = tool

	if _, err := ffmpegMerge([]string{in}, filepath.Join(tmp, "no-aac"), "mp4", opt, false); err != nil {
		t.Fatal(err)
	}
	argsBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(argsBytes), "aac_adtstoasc") {
		t.Fatalf("non-AAC media should not use AAC bitstream filter:\n%s", argsBytes)
	}

	if _, err := ffmpegMerge([]string{in}, filepath.Join(tmp, "with-aac"), "mp4", opt, true); err != nil {
		t.Fatal(err)
	}
	argsBytes, err = os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains("\n"+string(argsBytes), "\n-bsf:a\naac_adtstoasc\n") {
		t.Fatalf("AAC media should use AAC bitstream filter:\n%s", argsBytes)
	}

	if _, err := ffmpegMerge([]string{in}, filepath.Join(tmp, "aac-format"), "aac", opt, true); err != nil {
		t.Fatal(err)
	}
	argsBytes, err = os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	args := "\n" + string(argsBytes)
	for _, want := range []string{"\n-map\n0:a\n", "\n-c\ncopy\n"} {
		if !strings.Contains(args, want) {
			t.Fatalf("AAC output should match upstream audio-only copy args, missing %q:\n%s", want, argsBytes)
		}
	}
	if strings.Contains(string(argsBytes), "aac_adtstoasc") {
		t.Fatalf("AAC output format should not add AAC bitstream filter like upstream:\n%s", argsBytes)
	}
}

func TestFFmpegMergeMetadataMatchesUpstreamMP4Only(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	tmp := t.TempDir()
	in := filepath.Join(tmp, "clip.ts")
	if err := os.WriteFile(in, []byte("track"), 0644); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(tmp, "merge-args.txt")
	tool := filepath.Join(tmp, "ffmpeg")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %q\nlast=\"\"\nfor arg in \"$@\"; do last=\"$arg\"; done\nprintf merged > \"$last\"\n", logPath)
	if err := os.WriteFile(tool, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	opt := defaultOptions()
	opt.FFmpegBinaryPath = tool

	if _, err := ffmpegMerge([]string{in}, filepath.Join(tmp, "single-mp4"), "mp4", opt, false); err != nil {
		t.Fatal(err)
	}
	argsBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	mp4Args := "\n" + string(argsBytes)
	for _, want := range []string{
		"\n-metadata\ndate=",
		"\n-metadata\nencoding_tool=\n",
		"\n-metadata\ntitle=\n",
		"\n-metadata\ncopyright=\n",
		"\n-metadata\ncomment=\n",
		"\n-metadata:s:a:0\ntitle=\n",
		"\n-metadata:s:a:0\nhandler=\n",
	} {
		if !strings.Contains(mp4Args, want) {
			t.Fatalf("MP4 ffmpeg merge args should include upstream metadata %q:\n%s", want, mp4Args)
		}
	}

	if _, err := ffmpegMerge([]string{in}, filepath.Join(tmp, "single-mkv"), "mkv", opt, false); err != nil {
		t.Fatal(err)
	}
	argsBytes, err = os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	mkvArgs := "\n" + string(argsBytes)
	for _, unexpected := range []string{
		"\n-metadata\ndate=",
		"\n-metadata\nencoding_tool=\n",
		"\n-metadata:s:a:0\nhandler=\n",
	} {
		if strings.Contains(mkvArgs, unexpected) {
			t.Fatalf("non-MP4 ffmpeg merge should not receive MP4-only metadata %q:\n%s", unexpected, mkvArgs)
		}
	}
}

func TestPartialCombineMultipleFilesMatchesUpstreamChunking(t *testing.T) {
	tmp := t.TempDir()
	var files []string
	for i := 0; i < 205; i++ {
		p := filepath.Join(tmp, fmt.Sprintf("%04d.ts", i))
		if err := os.WriteFile(p, []byte(fmt.Sprintf("%03d", i)), 0644); err != nil {
			t.Fatal(err)
		}
		files = append(files, p)
	}
	out, err := partialCombineMultipleFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Fatalf("expected 3 chunk files, got %#v", out)
	}
	for _, wantBase := range []string{"T0000.ts", "T0001.ts", "T0002.ts"} {
		if _, err := os.Stat(filepath.Join(tmp, wantBase)); err != nil {
			t.Fatalf("missing chunk %s: %v", wantBase, err)
		}
	}
	b0, err := os.ReadFile(out[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b0), "000001002") || !strings.HasSuffix(string(b0), "097098099") {
		t.Fatalf("first chunk content order wrong: %q", b0)
	}
	if _, err := os.Stat(files[0]); !os.IsNotExist(err) {
		t.Fatalf("source files should be deleted after partial combine, stat err=%v", err)
	}
}

func TestFFmpegMergePartCombinesLongSegmentLists(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell helper is unix-only")
	}
	tmp := t.TempDir()
	var files []string
	for i := 0; i < partialMergeThreshold; i++ {
		p := filepath.Join(tmp, fmt.Sprintf("%04d.ts", i))
		if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		files = append(files, p)
	}
	logPath := filepath.Join(tmp, "ffmpeg-args.txt")
	tool := filepath.Join(tmp, "ffmpeg")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %q\nlast=\"\"\nfor arg in \"$@\"; do last=\"$arg\"; done\nprintf merged > \"$last\"\n", logPath)
	if err := os.WriteFile(tool, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	opt := defaultOptions()
	opt.FFmpegBinaryPath = tool
	out, err := ffmpegMerge(files, filepath.Join(tmp, "long"), "mp4", opt, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatal(err)
	}
	argsBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	args := string(argsBytes)
	if !strings.Contains(args, "T0000.ts|T0001.ts") {
		t.Fatalf("ffmpeg should receive partial chunk files, args:\n%s", args)
	}
	if strings.Contains(args, "0000.ts|0001.ts") {
		t.Fatalf("ffmpeg should not receive original long file list, args:\n%s", args)
	}
	if _, err := os.Stat(files[0]); !os.IsNotExist(err) {
		t.Fatalf("original first segment should be removed, stat err=%v", err)
	}
}

func TestReadAllWithLimit(t *testing.T) {
	got, err := readAllWithLimit(strings.NewReader("abc"), 1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "abc" {
		t.Fatalf("bad read: %q", got)
	}
}

func TestSharedRateLimiter(t *testing.T) {
	limiter := newRateLimiter(100)
	start := time.Now()
	limiter.Wait(50)
	limiter.Wait(50)
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond {
		t.Fatalf("limiter did not wait enough: %s", elapsed)
	}
}

func TestExtractDefaultKID(t *testing.T) {
	kid := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	tencPayload := append([]byte{0, 0, 0, 0, 0, 0, 0, 0}, kid...)
	mp4 := mustMP4Box("moov", mustMP4Box("trak", mustMP4Box("mdia", mustMP4Box("minf", mustMP4Box("stbl", mustMP4Box("encv", mustMP4Box("sinf", mustMP4Box("schi", mustMP4Box("tenc", tencPayload)))))))))
	got := extractDefaultKID(mp4)
	if got != "000102030405060708090a0b0c0d0e0f" {
		t.Fatalf("kid wrong: %s", got)
	}
}

func TestExtractWidevinePSSHKID(t *testing.T) {
	kid := []byte{0xaa, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 0xbb}
	psshData := append([]byte{0x08, 0x01, 0x12, 0x10}, kid...)
	payload := append([]byte{0, 0, 0, 0}, widevineSystemID...)
	payload = append(payload, []byte{0, 0, 0, byte(len(psshData))}...)
	payload = append(payload, psshData...)
	mp4 := mustMP4Box("moov", mustMP4Box("pssh", payload))
	got := extractDefaultKID(mp4)
	if got != "aa0102030405060708090a0b0c0d0ebb" {
		t.Fatalf("pssh kid wrong: %s", got)
	}
}

func TestReadMP4InfoReportsSchemeAndWidevinePSSH(t *testing.T) {
	kid := []byte{0xaa, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 0xbb}
	psshData := append([]byte{0x08, 0x01, 0x12, 0x10}, kid...)
	psshPayload := append([]byte{0, 0, 0, 0}, widevineSystemID...)
	psshPayload = append(psshPayload, []byte{0, 0, 0, byte(len(psshData))}...)
	psshPayload = append(psshPayload, psshData...)
	tencPayload := append([]byte{0, 0, 0, 0, 0, 0, 0, 0}, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}...)
	schmPayload := append([]byte{0, 0, 0, 0}, []byte("cenc")...)
	schmPayload = append(schmPayload, 0, 0, 0, 0)
	mp4 := mustMP4Box("moov", concatBytes(
		mustMP4Box("pssh", psshPayload),
		mustMP4Box("trak", mustMP4Box("mdia", mustMP4Box("minf", mustMP4Box("stbl", mustMP4Box("encv", mustMP4Box("sinf", concatBytes(
			mustMP4Box("schm", schmPayload),
			mustMP4Box("schi", mustMP4Box("tenc", tencPayload)),
		))))))),
	))
	info, err := readMP4Info(mp4)
	if err != nil {
		t.Fatal(err)
	}
	if info.Scheme != "cenc" {
		t.Fatalf("scheme wrong: %q", info.Scheme)
	}
	if info.PSSH != base64.StdEncoding.EncodeToString(psshData) {
		t.Fatalf("widevine pssh data wrong: %q", info.PSSH)
	}
	if info.KID != "000102030405060708090a0b0c0d0e0f" {
		t.Fatalf("tenc kid should keep priority over pssh, got %s", info.KID)
	}
	if info.MultiDRM {
		t.Fatal("non-zero tenc KID should not be treated as MultiDRM")
	}
}

func TestReadMP4InfoRejectsUnsupportedPSSHVersion(t *testing.T) {
	payload := append([]byte{2, 0, 0, 0}, widevineSystemID...)
	payload = append(payload, 0, 0, 0, 0)
	_, err := readMP4Info(mustMP4Box("moov", mustMP4Box("pssh", payload)))
	if err == nil || !strings.Contains(err.Error(), "PSSH version can only be 0 or 1") {
		t.Fatalf("expected unsupported PSSH version error, got %v", err)
	}
}

func TestExtractWidevinePSSHVersionOneUsesListedKID(t *testing.T) {
	kid := []byte{0xcc, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 0xdd}
	payload := append([]byte{1, 0, 0, 0}, widevineSystemID...)
	payload = append(payload, []byte{0, 0, 0, 1}...)
	payload = append(payload, kid...)
	payload = append(payload, 0, 0, 0, 0)
	got := extractDefaultKID(mustMP4Box("moov", mustMP4Box("pssh", payload)))
	if got != "cc0102030405060708090a0b0c0d0edd" {
		t.Fatalf("version 1 pssh listed kid wrong: %s", got)
	}
}

func TestExtractDefaultKIDUsesWidevinePSSHWhenTencKIDIsZero(t *testing.T) {
	kid := []byte{0xbb, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 0xcc}
	mp4 := mustMultiDRMInitWithWidevineKID(kid)
	got := extractDefaultKID(mp4)
	if got != "bb0102030405060708090a0b0c0d0ecc" {
		t.Fatalf("zero tenc should fall back to Widevine PSSH KID, got %s", got)
	}
	info, err := readMP4Info(mp4)
	if err != nil {
		t.Fatal(err)
	}
	if !info.MultiDRM {
		t.Fatal("zero tenc with Widevine PSSH should be marked as MultiDRM")
	}
	if info.PSSH == "" {
		t.Fatal("widevine pssh data should be exposed")
	}
}

func mustMultiDRMInitWithWidevineKID(kid []byte) []byte {
	zeroTenc := append([]byte{0, 0, 0, 0, 0, 0, 0, 0}, make([]byte, 16)...)
	psshData := append([]byte{0x08, 0x01, 0x12, 0x10}, kid...)
	psshPayload := append([]byte{0, 0, 0, 0}, widevineSystemID...)
	psshPayload = append(psshPayload, []byte{0, 0, 0, byte(len(psshData))}...)
	psshPayload = append(psshPayload, psshData...)
	return mustMP4Box("moov", concatBytes(
		mustMP4Box("pssh", psshPayload),
		mustMP4Box("trak", mustMP4Box("mdia", mustMP4Box("minf", mustMP4Box("stbl", mustMP4Box("encv", mustMP4Box("sinf", mustMP4Box("schi", mustMP4Box("tenc", zeroTenc)))))))),
	))
}

func TestExtractPlayReadyPSSHKID(t *testing.T) {
	rawKID := []byte{0x03, 0x02, 0x01, 0x00, 0x05, 0x04, 0x07, 0x06, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}
	xml := "<WRMHEADER><DATA><KID>" + base64.StdEncoding.EncodeToString(rawKID) + "</KID></DATA></WRMHEADER>"
	mp4 := mustPlayReadyPSSH(xml)
	got := extractDefaultKID(mp4)
	if got != "000102030405060708090a0b0c0d0e0f" {
		t.Fatalf("playready pssh kid wrong: %s", got)
	}
}

func TestExtractPlayReadyPSSHKIDFromValueAttribute(t *testing.T) {
	rawKID := []byte{0x03, 0x02, 0x01, 0x00, 0x05, 0x04, 0x07, 0x06, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}
	xml := `<WRMHEADER><DATA><PROTECTINFO><KID ALGID="AESCTR" VALUE="` + base64.StdEncoding.EncodeToString(rawKID) + `" /></PROTECTINFO></DATA></WRMHEADER>`
	mp4 := mustPlayReadyPSSH(xml)
	got := extractDefaultKID(mp4)
	if got != "000102030405060708090a0b0c0d0e0f" {
		t.Fatalf("playready VALUE kid wrong: %s", got)
	}
}

func mustPlayReadyPSSH(xml string) []byte {
	var prData []byte
	for _, b := range []byte(xml) {
		prData = append(prData, b, 0)
	}
	payload := append([]byte{0, 0, 0, 0}, playReadySystemID...)
	payload = append(payload, []byte{0, 0, 0, byte(len(prData))}...)
	payload = append(payload, prData...)
	return mustMP4Box("moov", mustMP4Box("pssh", payload))
}

func TestCollectDecryptKeysByKID(t *testing.T) {
	tmp := t.TempDir()
	keyFile := filepath.Join(tmp, "keys.txt")
	if err := os.WriteFile(keyFile, []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa:11111111111111111111111111111111\nbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb:22222222222222222222222222222222\n"), 0644); err != nil {
		t.Fatal(err)
	}
	opt := defaultOptions()
	opt.KeyTextFile = keyFile
	keys := collectDecryptKeys(opt, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	if len(keys) != 1 || !strings.Contains(keys[0], "222222") {
		t.Fatalf("unexpected keys: %#v", keys)
	}
}

func TestCollectDecryptKeysRequiresKIDLikeUpstream(t *testing.T) {
	tmp := t.TempDir()
	keyFile := filepath.Join(tmp, "keys.txt")
	if err := os.WriteFile(keyFile, []byte("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb:22222222222222222222222222222222\n"), 0644); err != nil {
		t.Fatal(err)
	}
	opt := defaultOptions()
	opt.KeyTextFile = keyFile
	opt.Keys = []string{"00112233445566778899aabbccddeeff"}
	keys := collectDecryptKeys(opt, "")
	if len(keys) != 1 || keys[0] != opt.Keys[0] {
		t.Fatalf("key-text-file should be ignored when KID is empty like upstream, got %#v", keys)
	}
}

func TestCollectDecryptKeysMatchesKIDPrefixCaseSensitivelyLikeUpstream(t *testing.T) {
	tmp := t.TempDir()
	keyFile := filepath.Join(tmp, "keys.txt")
	if err := os.WriteFile(keyFile, []byte("BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB:11111111111111111111111111111111\nbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb:22222222222222222222222222222222\n"), 0644); err != nil {
		t.Fatal(err)
	}
	opt := defaultOptions()
	opt.KeyTextFile = keyFile
	keys := collectDecryptKeys(opt, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	if len(keys) != 1 || !strings.Contains(keys[0], "222222") {
		t.Fatalf("key-text-file KID matching should be case-sensitive like upstream, got %#v", keys)
	}
}

func TestExtractMP4WebVTTFiles(t *testing.T) {
	tmp := t.TempDir()
	seg := filepath.Join(tmp, "seg.m4s")
	vttc := mustMP4Box("vttc", mustMP4Box("payl", []byte("hello mp4 vtt")))
	mdat := mustMP4Box("mdat", vttc)
	if err := os.WriteFile(seg, mdat, 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(tmp, "out.vtt")
	ok, err := extractMP4WebVTTFiles([]string{seg}, out, "VTT")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("expected extraction")
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "hello mp4 vtt") {
		t.Fatalf("bad vtt output:\n%s", b)
	}
}

func TestExtractMP4WebVTTFilesPreservesCueIDLikeUpstream(t *testing.T) {
	tmp := t.TempDir()
	seg := filepath.Join(tmp, "seg.m4s")
	vttc := mustMP4Box("vttc", concatBytes(
		mustMP4Box("iden", []byte("cue-42")),
		mustMP4Box("payl", []byte("hello with id")),
	))
	mdat := mustMP4Box("mdat", vttc)
	if err := os.WriteFile(seg, mdat, 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(tmp, "out.vtt")
	ok, err := extractMP4WebVTTFiles([]string{seg}, out, "VTT")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("expected extraction")
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "\ncue-42\n00:00:00.000 --> 00:00:01.000\nhello with id") {
		t.Fatalf("mp4 vtt iden cue id should be preserved like upstream:\n%s", got)
	}
}

func TestExtractMP4WebVTTFilesUsesMP4Timing(t *testing.T) {
	tmp := t.TempDir()
	init := filepath.Join(tmp, "_init.mp4")
	seg := filepath.Join(tmp, "seg.m4s")
	out := filepath.Join(tmp, "out.vtt")
	if err := os.WriteFile(init, makeTestWVTTInit(1000), 0644); err != nil {
		t.Fatal(err)
	}
	first := mustMP4Box("vttc", mustMP4Box("payl", []byte("first")))
	second := mustMP4Box("vttc", mustMP4Box("payl", []byte("second")))
	trunPayload := concatBytes(
		u32be(2),
		u32be(1500), u32be(uint32(len(first))),
		u32be(500), u32be(uint32(len(second))),
	)
	media := concatBytes(
		mustMP4Box("moof", mustMP4Box("traf", concatBytes(
			mustMP4FullBox("tfdt", 0, 0, u32be(2000)),
			mustMP4FullBox("tfhd", 0, 0, u32be(1)),
			mustMP4FullBox("trun", 0, 0x000300, trunPayload),
		))),
		mustMP4Box("mdat", concatBytes(first, second)),
	)
	if err := os.WriteFile(seg, media, 0644); err != nil {
		t.Fatal(err)
	}
	ok, err := extractMP4WebVTTFiles([]string{init, seg}, out, "VTT")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("expected extraction")
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	// long: MP4 WebVTT 的权威时间来自 tfdt/trun 与 init 的 mdhd timescale，不能再用每条字幕 1 秒的兜底时间。
	if !strings.Contains(string(got), "00:00:02.000 --> 00:00:03.500\nfirst") || !strings.Contains(string(got), "00:00:03.500 --> 00:00:04.000\nsecond") {
		t.Fatalf("bad timed mp4 vtt output:\n%s", got)
	}
}

func TestMergeTTMLFilesToVTT(t *testing.T) {
	tmp := t.TempDir()
	ttml := filepath.Join(tmp, "sub.ttml")
	out := filepath.Join(tmp, "out.vtt")
	xml := `<tt xmlns="http://www.w3.org/ns/ttml"><body><div><p begin="00:00:01.000" end="00:00:02.000">你好<br/>世界</p></div></body></tt>`
	if err := os.WriteFile(ttml, []byte(xml), 0644); err != nil {
		t.Fatal(err)
	}
	if err := mergeTTMLFiles([]string{ttml}, out, 0, "VTT"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "你好\n世界") {
		t.Fatalf("bad ttml vtt:\n%s", b)
	}
}

func TestMergeTTMLFilesWithSegmentOffsets(t *testing.T) {
	tmp := t.TempDir()
	a := filepath.Join(tmp, "a.ttml")
	b := filepath.Join(tmp, "b.ttml")
	out := filepath.Join(tmp, "out.srt")
	first := `<tt xmlns="http://www.w3.org/ns/ttml"><body><div><p begin="00:00:00.000" end="00:00:01.000">first</p></div></body></tt>`
	second := `<tt xmlns="http://www.w3.org/ns/ttml"><body><div><p begin="00:00:00.000" end="00:00:01.000">second</p></div></body></tt>`
	if err := os.WriteFile(a, []byte(first), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte(second), 0644); err != nil {
		t.Fatal(err)
	}
	segments := []Segment{{Index: 0, Duration: 2}, {Index: 1, Duration: 2}}
	if err := mergeTTMLFilesWithSegments([]string{a, b}, segments, out, 0, "SRT"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	// long: TTML 分片同样可能只携带片内时间；这里复刻上游用 HLS 分片时长拼回连续字幕轴的行为。
	if !strings.Contains(string(got), "1\n00:00:00,000 --> 00:00:01,000\nfirst") || !strings.Contains(string(got), "2\n00:00:02,000 --> 00:00:03,000\nsecond") {
		t.Fatalf("bad ttml offsets:\n%s", got)
	}
}

func TestExtractMP4TTMLFiles(t *testing.T) {
	tmp := t.TempDir()
	seg := filepath.Join(tmp, "seg.m4s")
	out := filepath.Join(tmp, "out.srt")
	xml := `<tt xmlns="http://www.w3.org/ns/ttml"><body><div><p begin="00:00:00.000" end="00:00:01.000">stpp text</p></div></body></tt>`
	if err := os.WriteFile(seg, mustMP4Box("mdat", []byte(xml)), 0644); err != nil {
		t.Fatal(err)
	}
	ok, err := extractMP4TTMLFiles([]string{seg}, out, 0, "SRT")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected stpp extraction")
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "stpp text") || !strings.Contains(string(b), "00:00:00,000") {
		t.Fatalf("bad stpp srt:\n%s", b)
	}
}

func TestFindMP4BoxesSkipsRealSTSDHeader(t *testing.T) {
	sample := mustMP4Box("stpp", []byte{})
	stsdPayload := append([]byte{0, 0, 0, 0, 0, 0, 0, 1}, sample...)
	init := mustMP4Box("moov", mustMP4Box("trak", mustMP4Box("mdia", mustMP4Box("minf", mustMP4Box("stbl", mustMP4Box("stsd", stsdPayload))))))
	if got := len(findMP4Boxes(init, "stpp")); got != 1 {
		t.Fatalf("expected stpp sample entry inside real stsd fullbox, got %d", got)
	}
}

func makeTestWVTTInit(timescale uint32) []byte {
	mdhd := mustMP4FullBox("mdhd", 0, 0, concatBytes(u32be(0), u32be(0), u32be(timescale), u32be(0)))
	stsd := mustMP4FullBox("stsd", 0, 0, concatBytes(u32be(1), mustMP4Box("wvtt", nil)))
	return mustMP4Box("moov", mustMP4Box("trak", mustMP4Box("mdia", concatBytes(
		mdhd,
		mustMP4Box("minf", mustMP4Box("stbl", stsd)),
	))))
}

func mustMP4FullBox(typ string, version byte, flags uint32, payload []byte) []byte {
	header := []byte{version, byte(flags >> 16), byte(flags >> 8), byte(flags)}
	return mustMP4Box(typ, append(header, payload...))
}

func u32be(v uint32) []byte {
	out := make([]byte, 4)
	binary.BigEndian.PutUint32(out, v)
	return out
}

func concatBytes(parts ...[]byte) []byte {
	var out []byte
	for _, part := range parts {
		out = append(out, part...)
	}
	return out
}

func TestSetupLoggingWritesFile(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "run.log")
	cleanup, actual, err := setupLogging(Options{LogFilePath: logPath}, []string{"n-m3u8dl-go-hls", "--version"})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("日志探针")
	if err := cleanup(); err != nil {
		t.Fatal(err)
	}
	if actual != logPath {
		t.Fatalf("log path mismatch: %s", actual)
	}
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if !strings.Contains(text, "Task CommandLine: n-m3u8dl-go-hls --version") || !strings.Contains(text, "日志探针") {
		t.Fatalf("log content missing:\n%s", text)
	}
}

func TestSetupLoggingHonorsLogLevel(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "filtered.log")
	cleanup, _, err := setupLogging(Options{LogFilePath: logPath, LogLevel: "WARN"}, []string{"n-m3u8dl-go-hls"})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("INFO: hidden")
	fmt.Println("plain hidden")
	fmt.Println("DEBUG: hidden")
	fmt.Fprintln(os.Stderr, "WARN: keep")
	fmt.Fprintln(os.Stderr, "ERROR: keep")
	if err := cleanup(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, hidden := range []string{"INFO: hidden", "plain hidden", "DEBUG: hidden"} {
		if strings.Contains(text, hidden) {
			t.Fatalf("log level WARN should filter %q:\n%s", hidden, text)
		}
	}
	for _, kept := range []string{"WARN: keep", "ERROR: keep"} {
		if !strings.Contains(text, kept) {
			t.Fatalf("log level WARN should keep %q:\n%s", kept, text)
		}
	}
}

func TestSetupLoggingNoLog(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "disabled.log")
	cleanup, actual, err := setupLogging(Options{NoLog: true, LogFilePath: logPath}, []string{"n-m3u8dl-go-hls"})
	if err != nil {
		t.Fatal(err)
	}
	if actual != "" {
		t.Fatalf("no-log should not return a path: %s", actual)
	}
	if err := cleanup(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("no-log should not create file, stat err=%v", err)
	}
}

func TestCheckLatestReleaseFromRedirect(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, srv.URL+"/nilaoda/N_m3u8DL-RE/releases/tag/v9.9.9", http.StatusFound)
	}))
	defer srv.Close()
	client := srv.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	latest, newer, err := checkLatestRelease(context.Background(), client, srv.URL+"/latest", "v0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if latest != "v9.9.9" || !newer {
		t.Fatalf("unexpected update result latest=%q newer=%v", latest, newer)
	}
}

func TestCompareVersionTags(t *testing.T) {
	if compareVersionTags("v1.2.10", "v1.2.9") <= 0 {
		t.Fatal("expected v1.2.10 to be newer")
	}
	if compareVersionTags("v1.2", "v1.2.0") != 0 {
		t.Fatal("expected missing patch to compare as zero")
	}
	if currentVersionTag("N_m3u8DL-GO-HLS 0.1.0") != "v0.1.0" {
		t.Fatal("current version tag extraction failed")
	}
}

func TestLivePipeMuxForcesRealtimeMerge(t *testing.T) {
	opt := Options{LivePipeMux: true}
	if !applyOptionImplications(&opt) {
		t.Fatal("expected implication to be applied")
	}
	if !opt.LiveRealTimeMerge {
		t.Fatal("live pipe mux should force realtime merge")
	}
}

func TestBuildLivePipeMuxArgsMatchesUpstreamDefaults(t *testing.T) {
	now := time.Date(2026, 6, 18, 12, 0, 0, 123, time.UTC)
	args := buildLivePipeMuxArgs([]string{"v.pipe", "a.pipe"}, "/out/live.ts", now, livePipeEnv{TmpDir: "/tmp/re-pipes"}, false)
	joined := "\n" + strings.Join(args, "\n") + "\n"
	for _, want := range []string{
		"\n-y\n",
		"\n-fflags\n+genpts\n",
		"\n-loglevel\nquiet\n",
		"\n-i\n/tmp/re-pipes/v.pipe\n",
		"\n-i\n/tmp/re-pipes/a.pipe\n",
		"\n-map\n0\n",
		"\n-map\n1\n",
		"\n-strict\nunofficial\n",
		"\n-c\ncopy\n",
		"\n-metadata\ndate=2026-06-18T12:00:00.000000123Z\n",
		"\n-ignore_unknown\n-copy_unknown\n",
		"\n-f\nmpegts\n-shortest\n/out/live.ts\n",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("pipe mux args missing %q:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "\n-re\n") {
		t.Fatalf("default pipe mux should not add -re:\n%s", joined)
	}
}

func TestBuildLivePipeMuxArgsHonorsEnvironmentOptions(t *testing.T) {
	now := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	args := buildLivePipeMuxArgs([]string{"v"}, "/ignored.ts", now, livePipeEnv{Options: "/custom/out.ts", TmpDir: "/pipes"}, false)
	joined := "\n" + strings.Join(args, "\n") + "\n"
	if !strings.Contains(joined, "\n-re\n") || !strings.Contains(joined, "\n-f\nmpegts\n-shortest\n/custom/out.ts\n") {
		t.Fatalf("custom pipe destination should add -re and target output:\n%s", joined)
	}

	args = buildLivePipeMuxArgs([]string{"v"}, "/ignored.ts", now, livePipeEnv{Options: "-f flv rtmp://example/live", TmpDir: "/pipes"}, false)
	joined = "\n" + strings.Join(args, "\n") + "\n"
	for _, want := range []string{"\n-re\n", "\n-f\nflv\nrtmp://example/live\n"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("raw pipe ffmpeg options missing %q:\n%s", want, joined)
		}
	}
}

func TestBuildLivePipeMuxArgsPreservesQuotedEnvironmentOptions(t *testing.T) {
	now := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	args := buildLivePipeMuxArgs([]string{"v"}, "/ignored.ts", now, livePipeEnv{Options: `-metadata title="Live Stream" -f flv 'rtmp://example/live app'`, TmpDir: "/pipes"}, false)
	joined := "\n" + strings.Join(args, "\n") + "\n"
	for _, want := range []string{
		"\n-re\n",
		"\n-metadata\ntitle=Live Stream\n",
		"\n-f\nflv\nrtmp://example/live app\n",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("quoted raw pipe ffmpeg option missing %q:\n%s", want, joined)
		}
	}
}

func TestSplitLivePipeOptionArgsFallsBackOnBrokenQuotes(t *testing.T) {
	got := splitLivePipeOptionArgs(`-metadata title="Live Stream`)
	want := []string{"-metadata", `title="Live`, "Stream"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("broken quoted pipe option should fall back to Fields, got %#v", got)
	}
}

func TestSplitLivePipeOptionArgsPreservesWindowsBackslashes(t *testing.T) {
	got := splitLivePipeOptionArgs(`-f mpegts C:\out\live.ts "D:\media output\live stream.ts" title=Live\ Stream`)
	want := []string{"-f", "mpegts", `C:\out\live.ts`, `D:\media output\live stream.ts`, "title=Live Stream"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("windows path backslashes should be preserved, got %#v", got)
	}
}

func TestBuildLivePipeMuxArgsWindowsPipePath(t *testing.T) {
	args := buildLivePipeMuxArgs([]string{"videoPipe"}, "out.ts", time.Unix(0, 0).UTC(), livePipeEnv{}, true)
	joined := "\n" + strings.Join(args, "\n") + "\n"
	if !strings.Contains(joined, "\n-i\n\\\\.\\pipe\\videoPipe\n") {
		t.Fatalf("windows pipe path mismatch:\n%s", joined)
	}
}

func TestCreateLivePipeCreatesFIFO(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("FIFO mode check is Unix-only; Windows named pipe runtime is covered by live_pipe_windows_test.go")
	}
	tmp := t.TempDir()
	file, path, err := createLivePipe("video.pipe", livePipeEnv{TmpDir: tmp})
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if path != filepath.Join(tmp, "video.pipe") {
		t.Fatalf("pipe path mismatch: %s", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeNamedPipe == 0 {
		t.Fatalf("expected FIFO mode, got %s", info.Mode())
	}
}

func TestCreateLivePipeRejectsExistingRegularFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("regular FIFO path collision check is Unix-only")
	}
	tmp := t.TempDir()
	path := filepath.Join(tmp, "video.pipe")
	if err := os.WriteFile(path, []byte("not fifo"), 0644); err != nil {
		t.Fatal(err)
	}
	file, gotPath, err := createLivePipe("video.pipe", livePipeEnv{TmpDir: tmp})
	if err == nil {
		if file != nil {
			_ = file.Close()
		}
		t.Fatal("expected regular file rejection")
	}
	if gotPath != path || !strings.Contains(err.Error(), "not a FIFO") {
		t.Fatalf("unexpected error path=%s err=%v", gotPath, err)
	}
}

func TestPrepareLivePipeMuxCreatesPipesAndStartsMux(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this fake ffmpeg shell script test is Unix-only")
	}
	tmp := t.TempDir()
	argsLog := filepath.Join(tmp, "ffmpeg-args.txt")
	tool := filepath.Join(tmp, "ffmpeg")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %q\n", argsLog)
	if err := os.WriteFile(tool, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	session, err := prepareLivePipeMux(tool, 2, filepath.Join(tmp, "live.mp4"), livePipeEnv{TmpDir: tmp})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if session.OutputPath != filepath.Join(tmp, "live.ts") {
		t.Fatalf("pipe mux output should use ts extension, got %s", session.OutputPath)
	}
	if len(session.Pipes) != 2 || len(session.PipeNames) != 2 || len(session.PipePaths) != 2 {
		t.Fatalf("unexpected session pipes: %#v", session)
	}
	for _, path := range session.PipePaths {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&os.ModeNamedPipe == 0 {
			t.Fatalf("expected FIFO mode for %s, got %s", path, info.Mode())
		}
	}
	if err := session.Cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	argsBytes, err := os.ReadFile(argsLog)
	if err != nil {
		t.Fatal(err)
	}
	args := "\n" + string(argsBytes) + "\n"
	for _, path := range session.PipePaths {
		if !strings.Contains(args, "\n-i\n"+path+"\n") {
			t.Fatalf("ffmpeg args should include pipe path %s:\n%s", path, args)
		}
	}
	if !strings.Contains(args, "\n-f\nmpegts\n-shortest\n"+session.OutputPath+"\n") {
		t.Fatalf("ffmpeg args should target ts output:\n%s", args)
	}
}

func TestWriteLivePipeMuxFilesWritesTracksAndWaits(t *testing.T) {
	tmp := t.TempDir()
	r1, w1, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r1.Close()
	r2, w2, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Close()
	readDone := make(chan []byte, 2)
	go func() {
		b, _ := io.ReadAll(r1)
		readDone <- b
	}()
	go func() {
		b, _ := io.ReadAll(r2)
		readDone <- b
	}()

	init := filepath.Join(tmp, "_init.mp4")
	v1 := filepath.Join(tmp, "v1.m4s")
	a1 := filepath.Join(tmp, "a1.m4s")
	for path, content := range map[string]string{
		init: "INIT",
		v1:   "VIDEO",
		a1:   "AUDIO",
	} {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	session := &livePipeSession{Pipes: []*os.File{w1, w2}}
	if err := writeLivePipeMuxFiles(session, [][]string{{init, v1}, {a1}}, false); err != nil {
		t.Fatal(err)
	}
	got := []string{string(<-readDone), string(<-readDone)}
	if !(containsPipeWrite(got, "INITVIDEO") && containsPipeWrite(got, "AUDIO")) {
		t.Fatalf("pipe writes mismatch: %#v", got)
	}
	if _, err := os.Stat(init); err != nil {
		t.Fatalf("init segment should be kept: %v", err)
	}
	for _, path := range []string{v1, a1} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("non-init segment should be removed after pipe write, path=%s err=%v", path, err)
		}
	}
}

func TestCopyFilesToOpenLivePipeKeepsPipeWritableAcrossBatches(t *testing.T) {
	tmp := t.TempDir()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	readDone := make(chan []byte, 1)
	go func() {
		b, _ := io.ReadAll(r)
		readDone <- b
	}()
	a := filepath.Join(tmp, "a.ts")
	b := filepath.Join(tmp, "b.ts")
	for path, content := range map[string]string{a: "A", b: "B"} {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := copyFilesToOpenLivePipe(w, []string{a}, false); err != nil {
		t.Fatal(err)
	}
	if err := copyFilesToOpenLivePipe(w, []string{b}, false); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if got := string(<-readDone); got != "AB" {
		t.Fatalf("open pipe writes mismatch: %q", got)
	}
	for _, path := range []string{a, b} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("pipe chunk write should delete media segment %s, err=%v", path, err)
		}
	}
}

func TestLiveRealtimeDownloadStateWritesBatchesToPipe(t *testing.T) {
	tmp := t.TempDir()
	srcA := filepath.Join(tmp, "src-a.ts")
	srcB := filepath.Join(tmp, "src-b.ts")
	if err := os.WriteFile(srcA, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcB, []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	readDone := make(chan []byte, 1)
	go func() {
		b, _ := io.ReadAll(r)
		readDone <- b
	}()
	video := MediaVideo
	opt := defaultOptions()
	opt.TmpDir = filepath.Join(tmp, "tmp")
	opt.SaveDir = tmp
	opt.SaveName = "pipe-live"
	opt.LiveKeepSegments = false
	client, err := newHTTPClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	stream := StreamSpec{
		ID:        1,
		Extension: "ts",
		MediaType: &video,
		Playlist: &Playlist{WasLive: true, Parts: []MediaPart{{Segments: []Segment{
			{Index: 1, URL: (&url.URL{Scheme: "file", Path: srcA}).String(), Duration: 1},
			{Index: 2, URL: (&url.URL{Scheme: "file", Path: srcB}).String(), Duration: 1},
		}}}},
	}
	state, _, err := newLiveRealtimeDownloadState(client, stream, opt, newRateLimiter(0))
	if err != nil {
		t.Fatal(err)
	}
	state.pipe = w
	if err := state.downloadAndAppend(context.Background(), []Segment{stream.Playlist.Parts[0].Segments[0]}); err != nil {
		t.Fatal(err)
	}
	if err := state.downloadAndAppend(context.Background(), []Segment{stream.Playlist.Parts[0].Segments[1]}); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if got := string(<-readDone); got != "ab" {
		t.Fatalf("live pipe batch writes mismatch: %q", got)
	}
	matches, err := filepath.Glob(filepath.Join(taskTempDir(opt), "*", "*.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("live pipe batch writes should delete temp media segments: %#v", matches)
	}
}

func TestFinishLivePipeMuxOutputsReplacesNonSubtitleOutputs(t *testing.T) {
	tmp := t.TempDir()
	videoFile := filepath.Join(tmp, "video.mp4")
	audioFile := filepath.Join(tmp, "audio.m4a")
	subFile := filepath.Join(tmp, "sub.srt")
	for path, content := range map[string]string{
		videoFile: "video",
		audioFile: "audio",
		subFile:   "subtitle",
	} {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	r1, w1, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r1.Close()
	r2, w2, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Close()
	readDone := make(chan []byte, 2)
	go func() {
		b, _ := io.ReadAll(r1)
		readDone <- b
	}()
	go func() {
		b, _ := io.ReadAll(r2)
		readDone <- b
	}()

	video := MediaVideo
	audio := MediaAudio
	sub := MediaSubtitles
	session := &livePipeSession{
		Pipes:      []*os.File{w1, w2},
		OutputPath: filepath.Join(tmp, "pipe.ts"),
	}
	got, err := finishLivePipeMuxOutputs(
		[]outputFile{{Path: videoFile, MediaType: &video}, {Path: audioFile, MediaType: &audio}},
		[]outputFile{{Path: subFile, MediaType: &sub}},
		session,
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	writes := []string{string(<-readDone), string(<-readDone)}
	if !(containsPipeWrite(writes, "video") && containsPipeWrite(writes, "audio")) {
		t.Fatalf("pipe writes mismatch: %#v", writes)
	}
	if len(got) != 2 || got[0].Path != session.OutputPath || got[1].Path != subFile {
		t.Fatalf("live pipe outputs should replace non-subtitle tracks and keep subtitles: %#v", got)
	}
	for _, path := range []string{videoFile, audioFile} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("non-subtitle output should be removed when keep=false, path=%s err=%v", path, err)
		}
	}
	if _, err := os.Stat(subFile); err != nil {
		t.Fatalf("subtitle output should stay outside pipe mux: %v", err)
	}
}

func containsPipeWrite(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestValidateOptionsRejectsMissingFFmpegBinary(t *testing.T) {
	opt := defaultOptions()
	opt.FFmpegBinaryPath = "/no/such/ffmpeg"
	err := validateOptions(opt)
	if err == nil || !strings.Contains(err.Error(), "找不到 ffmpeg") {
		t.Fatalf("expected missing ffmpeg error, got %v", err)
	}
}

func TestValidateOptionsRejectsInvalidEnumValuesLikeUpstream(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "ui language", args: []string{"--ui-language", "fr-FR", "https://example.com/main.m3u8"}, want: "--ui-language"},
		{name: "log level", args: []string{"--log-level", "TRACE", "https://example.com/main.m3u8"}, want: "--log-level"},
		{name: "subtitle format", args: []string{"--sub-format", "ASS", "https://example.com/main.m3u8"}, want: "--sub-format"},
		{name: "decryption engine", args: []string{"--decryption-engine", "BENTO4", "https://example.com/main.m3u8"}, want: "--decryption-engine"},
		{name: "custom hls method", args: []string{"--custom-hls-method", "AES_256", "https://example.com/main.m3u8"}, want: "--custom-hls-method"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opt, err := parseArgs(tc.args)
			if err != nil {
				t.Fatal(err)
			}
			err = validateOptions(opt)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %s validation error, got %v", tc.want, err)
			}
		})
	}
}

func TestValidateOptionsAcceptsEnumValuesCaseInsensitively(t *testing.T) {
	opt, err := parseArgs([]string{
		"--ui-language", "zh-CN",
		"--log-level", "warn",
		"--sub-format", "vtt",
		"--decryption-engine", "ffmpeg",
		"--custom-hls-method", "sample_aes",
		"https://example.com/main.m3u8",
	})
	if err != nil {
		t.Fatal(err)
	}
	opt.FFmpegBinaryPath = ""
	if err := validateOptions(opt); err != nil {
		t.Fatalf("expected lowercase enum inputs to pass validation, got %v", err)
	}
}

func TestParseArgsUseShakaPackagerBoolMatchesUpstream(t *testing.T) {
	opt, err := parseArgs([]string{"--decryption-engine", "ffmpeg", "--use-shaka-packager", "false", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.ToUpper(opt.DecryptionEngine) != "FFMPEG" {
		t.Fatalf("explicit false should not override decryption engine like upstream, got %s", opt.DecryptionEngine)
	}

	opt, err = parseArgs([]string{"--use-shaka-packager", "https://example.com/main.m3u8"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.DecryptionEngine != "SHAKA_PACKAGER" {
		t.Fatalf("bare --use-shaka-packager should select Shaka, got %s", opt.DecryptionEngine)
	}
}

func TestCoreMessagesFollowUILanguage(t *testing.T) {
	opt := defaultOptions()
	opt.UILanguage = "en-US"
	if got := tr(opt, "streamsParsed", 3); got != "Parsed 3 streams" {
		t.Fatalf("english streamsParsed wrong: %q", got)
	}
	opt.UILanguage = "zh-TW"
	if got := tr(opt, "skipDownload"); got != "已按 --skip-download 跳過下載" {
		t.Fatalf("traditional skipDownload wrong: %q", got)
	}
	opt.UILanguage = "zh-CN"
	if got := tr(opt, "downloadProgress", "VIDEO", 1, 2); got != "VIDEO 下载进度 1/2" {
		t.Fatalf("simplified downloadProgress wrong: %q", got)
	}
}

func TestOptionImplicationMessagesFollowUILanguage(t *testing.T) {
	opt := defaultOptions()
	opt.UILanguage = "en-US"
	opt.MuxAfterDone = &MuxOptions{Format: "mp4"}
	messages := applyOptionImplicationsWithMessages(&opt)
	if len(messages) != 1 || messages[0] != "MuxAfterDone detected, forced enable BinaryMerge" {
		t.Fatalf("english implication message wrong: %#v", messages)
	}
	opt = defaultOptions()
	opt.UILanguage = "zh-TW"
	opt.LivePipeMux = true
	messages = applyOptionImplicationsWithMessages(&opt)
	if len(messages) != 1 || messages[0] != "檢測到 LivePipeMux，已強制啟用 LiveRealTimeMerge" {
		t.Fatalf("traditional implication message wrong: %#v", messages)
	}
}

func TestPrepareSelectedStreamsMessagesFollowUILanguage(t *testing.T) {
	opt := defaultOptions()
	opt.UILanguage = "en-US"
	streams := []StreamSpec{{
		Playlist: &Playlist{Parts: []MediaPart{{Segments: []Segment{{Encrypt: EncryptInfo{Method: EncryptCENC}}}}}},
	}}
	messages := prepareSelectedStreams(streams, &opt)
	if len(messages) != 1 || messages[0] != "When CENC encryption is detected, binary merging is automatically enabled" {
		t.Fatalf("english CENC binary merge message wrong: %#v", messages)
	}
	opt = defaultOptions()
	opt.UILanguage = "zh-TW"
	streams = []StreamSpec{{Playlist: &Playlist{MediaInit: &Segment{URL: "init.mp4"}}}}
	messages = prepareSelectedStreams(streams, &opt)
	if len(messages) != 1 || messages[0] != "檢測到fMP4，自動開啟二進位制合併" {
		t.Fatalf("traditional fMP4 binary merge message wrong: %#v", messages)
	}
}

func TestValidateOptionsRejectsMuxImportWithoutMuxAfterDone(t *testing.T) {
	err := validateOptions(Options{MuxImports: []string{"path=extra.srt"}})
	if err == nil || !strings.Contains(err.Error(), "MuxAfterDone disabled") {
		t.Fatalf("expected mux import validation error, got %v", err)
	}
}

func TestValidateOptionsRejectsMissingMuxImportPath(t *testing.T) {
	err := validateOptions(Options{MuxAfterDone: &MuxOptions{Format: "mp4"}, MuxImports: []string{"path=/no/such/file.srt:lang=zh"}})
	if err == nil || !strings.Contains(err.Error(), "path empty or file not exists") {
		t.Fatalf("expected missing mux import path error, got %v", err)
	}
}

func TestValidateOptionsAcceptsExistingMuxImportPath(t *testing.T) {
	tmp := t.TempDir()
	sub := filepath.Join(tmp, "extra.srt")
	if err := os.WriteFile(sub, []byte("1\n00:00:00,000 --> 00:00:01,000\nhi\n"), 0644); err != nil {
		t.Fatal(err)
	}
	err := validateOptions(Options{MuxAfterDone: &MuxOptions{Format: "mp4"}, MuxImports: []string{"path=" + sub + ":lang=en:name=English"}})
	if err != nil {
		t.Fatalf("expected mux import path to pass, got %v", err)
	}
}

func TestValidateOptionsRejectsMissingMuxFFmpegBinPath(t *testing.T) {
	err := validateOptions(Options{MuxAfterDone: &MuxOptions{Format: "mp4", Muxer: "ffmpeg", BinPath: "/no/such/ffmpeg"}})
	if err == nil || !strings.Contains(err.Error(), "找不到 ffmpeg") {
		t.Fatalf("expected missing mux ffmpeg error, got %v", err)
	}
}

func TestValidateOptionsRejectsMissingMkvmergeBinPath(t *testing.T) {
	err := validateOptions(Options{MuxAfterDone: &MuxOptions{Format: "mkv", Muxer: "mkvmerge", BinPath: "/no/such/mkvmerge"}})
	if err == nil || !strings.Contains(err.Error(), "找不到 mkvmerge") {
		t.Fatalf("expected missing mkvmerge error, got %v", err)
	}
}

func TestValidateOptionsRejectsInvalidAdKeywordRegex(t *testing.T) {
	opt := defaultOptions()
	opt.AdKeywords = []string{"["}
	err := validateOptions(opt)
	if err == nil || !strings.Contains(err.Error(), "ad-keyword 正则无效") {
		t.Fatalf("expected invalid ad keyword regex error, got %v", err)
	}
}

func TestValidateOptionsRejectsInvalidFilterFor(t *testing.T) {
	err := validateOptions(Options{VideoFilter: parseFilter("for=middle")})
	if err == nil || !strings.Contains(err.Error(), "for=middle not valid") {
		t.Fatalf("expected invalid filter for error, got %v", err)
	}
}

func TestValidateOptionsRejectsInvalidFilterRegex(t *testing.T) {
	err := validateOptions(Options{AudioFilter: parseFilter("lang=[")})
	if err == nil || !strings.Contains(err.Error(), "filter lang 正则无效") {
		t.Fatalf("expected invalid filter regex error, got %v", err)
	}
}

func TestValidateOptionsRejectsMissingDecryptionBinary(t *testing.T) {
	err := validateOptions(Options{
		Keys:                 []string{"00000000000000000000000000000000:00112233445566778899aabbccddeeff"},
		DecryptionEngine:     "MP4DECRYPT",
		DecryptionBinaryPath: "/no/such/mp4decrypt",
	})
	if err == nil || !strings.Contains(err.Error(), "/no/such/mp4decrypt") {
		t.Fatalf("expected missing decryption binary error, got %v", err)
	}
}

func TestValidateOptionsAcceptsExistingDecryptionBinary(t *testing.T) {
	tmp := t.TempDir()
	tool := filepath.Join(tmp, "mp4decrypt")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	err := validateOptions(Options{
		Keys:                 []string{"00000000000000000000000000000000:00112233445566778899aabbccddeeff"},
		DecryptionEngine:     "MP4DECRYPT",
		DecryptionBinaryPath: tool,
	})
	if err != nil {
		t.Fatalf("expected existing decryption binary to pass, got %v", err)
	}
}

func TestMuxAfterDoneForcesBinaryMerge(t *testing.T) {
	opt := Options{MuxAfterDone: &MuxOptions{Format: "mp4"}}
	msgs := applyOptionImplicationsWithMessages(&opt)
	if !opt.BinaryMerge {
		t.Fatal("mux-after-done should force binary merge")
	}
	if len(msgs) != 1 || !strings.Contains(msgs[0], "BinaryMerge") {
		t.Fatalf("unexpected implication messages: %#v", msgs)
	}
}

func TestSkipMergeDoesNotTriggerMuxAfterDone(t *testing.T) {
	opt := Options{SkipMerge: true, MuxAfterDone: &MuxOptions{Format: "mp4"}}
	if shouldMuxAfterDownload(opt, []outputFile{{Path: "/tmp/segments_tmp"}}) {
		t.Fatal("skip-merge leaves segment directories and should not trigger final mux")
	}
	opt.SkipMerge = false
	if !shouldMuxAfterDownload(opt, []outputFile{{Path: "/tmp/output.mp4"}}) {
		t.Fatal("merged output files should trigger final mux")
	}
}

func TestLivePipeMuxDoesNotTriggerFinalMuxAfterDone(t *testing.T) {
	opt := Options{LivePipeMux: true, MuxAfterDone: &MuxOptions{Format: "mp4"}}
	if shouldMuxAfterDownload(opt, []outputFile{{Path: "/tmp/live.ts"}}) {
		t.Fatal("live pipe mux output is already muxed and should not enter final mux-after-done")
	}
}
