package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const latestReleaseURL = "https://github.com/nilaoda/N_m3u8DL-RE/releases/latest"
const releaseTagURLPrefix = "https://github.com/nilaoda/N_m3u8DL-RE/releases/tag/"

var versionNumberRE = regexp.MustCompile(`\d+(?:\.\d+){0,3}`)

func maybeCheckUpdate(opt Options) {
	if opt.DisableUpdateCheck {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client, err := newUpdateHTTPClient(opt)
	if err != nil {
		return
	}
	latest, newer, err := checkLatestRelease(ctx, client, latestReleaseURL, currentVersionTag(version))
	if err != nil || !newer {
		return
	}
	fmt.Println(updateFoundMessage(opt, latest))
}

func updateFoundMessage(opt Options, latest string) string {
	// long: 原版 ResString.newVersionFound 是固定资源文本，latest tag 作为后续红色片段单独追加；不要把版本号塞进资源模板。
	return strings.TrimSpace(tr(opt, "newVersionFound") + " " + latest)
}

func newUpdateHTTPClient(opt Options) (*http.Client, error) {
	client, err := newHTTPClient(opt)
	if err != nil {
		return nil, err
	}
	client.Timeout = 3 * time.Second
	// long: GitHub latest 通过 302 暴露真实 tag，上游也是读取 Location，而不是跟随跳转后解析 HTML。
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return client, nil
}

func checkLatestRelease(ctx context.Context, client *http.Client, url string, currentTag string) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", false, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()
	location := resp.Header.Get("Location")
	if location == "" {
		return "", false, nil
	}
	latest := latestReleaseTagFromLocation(location)
	if latest == "" || strings.HasPrefix(latest, "http") {
		return "", false, nil
	}
	return latest, isDifferentLatestTag(latest, currentTag), nil
}

func latestReleaseTagFromLocation(location string) string {
	// long: 上游只把官方 releases/tag 前缀替换为空；未知跳转地址会保留 https 前缀，随后被当成无效结果忽略。
	return strings.TrimSpace(strings.Replace(location, releaseTagURLPrefix, "", 1))
}

func isDifferentLatestTag(latest, currentTag string) bool {
	// long: 原版没有做语义化版本大小比较，只要 latest tag 不是当前 vMajor.Minor.Build 的前缀就提示。
	return !strings.HasPrefix(latest, currentTag) && !strings.HasPrefix(latest, "https")
}

func currentVersionTag(input string) string {
	matches := versionNumberRE.FindAllString(input, -1)
	if len(matches) == 0 {
		return "v0.0.0"
	}
	return "v" + matches[len(matches)-1]
}

func applyOptionImplications(opt *Options) bool {
	return len(applyOptionImplicationsWithMessages(opt)) > 0
}

func validateOptions(opt Options) error {
	if err := validateEnumOptions(opt); err != nil {
		return err
	}
	if opt.MuxAfterDone == nil && len(opt.MuxImports) > 0 {
		return errors.New("MuxAfterDone disabled, MuxImports not allowed!")
	}
	for _, raw := range opt.MuxImports {
		path := muxImportPath(raw)
		if path == "" {
			return errors.New("path empty or file not exists!")
		}
		if _, err := os.Stat(path); err != nil {
			return errors.New("path empty or file not exists!")
		}
	}
	if err := validateFFmpegTool(opt); err != nil {
		return err
	}
	if err := validateMuxTool(opt); err != nil {
		return err
	}
	if err := validateAdKeywords(opt); err != nil {
		return err
	}
	if err := validateFilters(opt); err != nil {
		return err
	}
	if err := validateDecryptionTool(opt); err != nil {
		return err
	}
	return nil
}

func validateEnumOptions(opt Options) error {
	if !isOneOfFold(opt.UILanguage, "", "en-US", "zh-CN", "zh-TW") {
		return fmt.Errorf("--ui-language %q not valid, allowed: en-US, zh-CN, zh-TW", opt.UILanguage)
	}
	if !isOneOfFold(opt.LogLevel, "", "OFF", "ERROR", "WARN", "INFO", "DEBUG") {
		return fmt.Errorf("--log-level %q not valid, allowed: OFF, ERROR, WARN, INFO, DEBUG", opt.LogLevel)
	}
	if !isOneOfFold(opt.SubFormat, "", "SRT", "VTT") {
		return fmt.Errorf("--sub-format %q not valid, allowed: SRT, VTT", opt.SubFormat)
	}
	if !isOneOfFold(opt.DecryptionEngine, "", "MP4DECRYPT", "SHAKA_PACKAGER", "FFMPEG") {
		return fmt.Errorf("--decryption-engine %q not valid, allowed: MP4DECRYPT, SHAKA_PACKAGER, FFMPEG", opt.DecryptionEngine)
	}
	if opt.CustomHLSMethod != "" && !isKnownHLSMethod(opt.CustomHLSMethod) {
		// long: 自定义 HLS 加密方式会覆盖 playlist 中的真实 METHOD，非法枚举不能静默降级，否则可能把分片按错误算法解密。
		return fmt.Errorf("--custom-hls-method %q not valid", opt.CustomHLSMethod)
	}
	return nil
}

func isOneOfFold(value string, allowed ...string) bool {
	for _, item := range allowed {
		if strings.EqualFold(value, item) {
			return true
		}
	}
	return false
}

func muxImportPath(raw string) string {
	p := splitComplex(raw)
	if path := p["path"]; path != "" {
		return path
	}
	return raw
}

func validateDecryptionTool(opt Options) error {
	if len(opt.Keys) == 0 && opt.KeyTextFile == "" {
		return nil
	}
	if opt.DecryptionBinaryPath != "" {
		if _, err := os.Stat(opt.DecryptionBinaryPath); err != nil {
			return fmt.Errorf("%s: %w", opt.DecryptionBinaryPath, err)
		}
		return nil
	}
	switch strings.ToUpper(opt.DecryptionEngine) {
	case "SHAKA_PACKAGER":
		if firstExecutable("shaka-packager", "packager-linux-x64", "packager-osx-x64", "packager-win-x64") == "" {
			return errors.New(tr(opt, "shakaPackagerNotFound"))
		}
	case "FFMPEG":
		bin := opt.FFmpegBinaryPath
		if bin == "" {
			bin = "ffmpeg"
		}
		if firstExecutable(bin) == "" {
			return errors.New(tr(opt, "ffmpegNotFound"))
		}
	default:
		if firstExecutable("mp4decrypt") == "" {
			return errors.New(tr(opt, "mp4decryptNotFound"))
		}
	}
	return nil
}

func validateFFmpegTool(opt Options) error {
	if opt.FFmpegBinaryPath == "" {
		return nil
	}
	if resolveToolPath(opt.FFmpegBinaryPath, "ffmpeg") == "" {
		return errors.New(tr(opt, "ffmpegNotFound"))
	}
	return nil
}

func validateMuxTool(opt Options) error {
	if opt.MuxAfterDone == nil {
		return nil
	}
	mux := opt.MuxAfterDone
	switch strings.ToLower(mux.Muxer) {
	case "", "ffmpeg":
		bin := muxFFmpegBinary(opt)
		if bin != "" && resolveToolPath(bin, "ffmpeg") == "" {
			return errors.New(tr(opt, "ffmpegNotFound"))
		}
	case "mkvmerge":
		if resolveToolPath(mux.BinPath, "mkvmerge") == "" {
			return errors.New(tr(opt, "mkvmergeNotFound"))
		}
	}
	return nil
}

func validateAdKeywords(opt Options) error {
	for _, keyword := range opt.AdKeywords {
		if keyword == "" {
			continue
		}
		if _, err := regexp.Compile(keyword); err != nil {
			return fmt.Errorf("ad-keyword 正则无效 %q: %w", keyword, err)
		}
	}
	return nil
}

func validateFilters(opt Options) error {
	filters := []*Filter{
		opt.VideoFilter,
		opt.AudioFilter,
		opt.SubtitleFilter,
		opt.DropVideoFilter,
		opt.DropAudioFilter,
		opt.DropSubtitleFilter,
	}
	for _, filter := range filters {
		if filter == nil {
			continue
		}
		if !validFilterFor(filter.For) {
			return fmt.Errorf("for=%s not valid", filter.For)
		}
		for name, pattern := range map[string]string{
			"id":     filter.GroupID,
			"lang":   filter.Language,
			"name":   filter.Name,
			"codecs": filter.Codecs,
			"res":    filter.Resolution,
			"frame":  filter.FrameRate,
			"ch":     filter.Channels,
			"range":  filter.VideoRange,
			"url":    filter.URL,
		} {
			if pattern == "" {
				continue
			}
			if _, err := regexp.Compile(pattern); err != nil {
				return fmt.Errorf("filter %s 正则无效 %q: %w", name, pattern, err)
			}
		}
	}
	return nil
}

func validFilterFor(value string) bool {
	if value == "" || value == "best" || value == "worst" || value == "all" {
		return true
	}
	if strings.HasPrefix(value, "best") {
		_, err := strconv.Atoi(strings.TrimPrefix(value, "best"))
		return err == nil
	}
	if strings.HasPrefix(value, "worst") {
		_, err := strconv.Atoi(strings.TrimPrefix(value, "worst"))
		return err == nil
	}
	return false
}

func resolveToolPath(candidate string, fallback string) string {
	if candidate == "" || candidate == "auto" {
		return firstExecutable(fallback)
	}
	return firstExecutable(candidate)
}

func applyOptionImplicationsWithMessages(opt *Options) []string {
	var messages []string
	if opt.LivePipeMux && !opt.LiveRealTimeMerge {
		opt.LiveRealTimeMerge = true
		messages = append(messages, tr(*opt, "livePipeMuxForcesRealtime"))
	}
	if opt.MuxAfterDone != nil && !opt.BinaryMerge {
		// long: 最终混流要拿到每条轨道的原始输出，上游会自动切到二进制合并，避免单轨先被 ffmpeg 改写后再混流。
		opt.BinaryMerge = true
		messages = append(messages, tr(*opt, "muxAfterDoneForcesBinaryMerge"))
	}
	return messages
}
