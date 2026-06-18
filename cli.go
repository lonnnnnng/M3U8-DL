package main

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func defaultOptions() Options {
	return Options{
		ThreadCount:        runtime.NumCPU(),
		DownloadRetryCount: 3,
		HTTPRequestTimeout: 100,
		CheckSegmentsCount: true,
		DelAfterDone:       true,
		WriteMetaJSON:      true,
		Headers: map[string]string{
			"user-agent": "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36",
		},
		SubFormat:          "SRT",
		AutoSubtitleFix:    true,
		LogLevel:           "INFO",
		UILanguage:         "zh-CN",
		DecryptionEngine:   "MP4DECRYPT",
		UseSystemProxy:     true,
		LiveKeepSegments:   true,
		LiveTakeCount:      16,
		CustomHLSMethod:    "",
		FFmpegBinaryPath:   "ffmpeg",
		DisableUpdateCheck: false,
	}
}

func parseArgs(args []string) (Options, error) {
	opt := defaultOptions()
	for i := 0; i < len(args); i++ {
		a := args[i]
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s 缺少参数值", a)
			}
			i++
			return args[i], nil
		}
		boolFlag := func(defaultValue bool) bool {
			if i+1 >= len(args) {
				return defaultValue
			}
			switch strings.ToLower(args[i+1]) {
			case "true":
				i++
				return true
			case "false":
				i++
				return false
			default:
				return defaultValue
			}
		}
		switch a {
		case "-h", "--help", "-?":
			return opt, errors.New("help")
		case "--version":
			return opt, errors.New("version")
		case "--morehelp":
			v, err := next()
			if err != nil {
				return opt, err
			}
			return opt, fmt.Errorf("morehelp:%s", v)
		case "--tmp-dir":
			v, err := next()
			if err != nil {
				return opt, err
			}
			opt.TmpDir = v
		case "--save-dir":
			v, err := next()
			if err != nil {
				return opt, err
			}
			opt.SaveDir = v
		case "--save-name":
			v, err := next()
			if err != nil {
				return opt, err
			}
			saveName := validSaveName(v)
			if saveName == "" {
				return opt, fmt.Errorf("Invalid save name!")
			}
			opt.SaveName = saveName
		case "--save-pattern":
			v, err := next()
			if err != nil {
				return opt, err
			}
			opt.SavePattern = v
		case "--log-file-path":
			v, err := next()
			if err != nil {
				return opt, err
			}
			logPath, err := parseLogFilePathOption(v)
			if err != nil {
				return opt, err
			}
			opt.LogFilePath = logPath
		case "--base-url":
			v, err := next()
			if err != nil {
				return opt, err
			}
			opt.BaseURL = v
		case "--thread-count":
			v, err := next()
			if err != nil {
				return opt, err
			}
			opt.ThreadCount, err = parseIntOption("ThreadCount", v)
			if err != nil {
				return opt, err
			}
		case "--download-retry-count":
			v, err := next()
			if err != nil {
				return opt, err
			}
			opt.DownloadRetryCount, err = parseIntOption("DownloadRetryCount", v)
			if err != nil {
				return opt, err
			}
		case "--http-request-timeout":
			v, err := next()
			if err != nil {
				return opt, err
			}
			timeout, err := strconv.ParseFloat(v, 64)
			if err != nil || timeout <= 0 {
				return opt, fmt.Errorf("error in parse HttpRequestTimeout: %s", v)
			}
			opt.HTTPRequestTimeout = timeout
		case "--force-ansi-console":
			opt.ForceANSIConsole = boolFlag(true)
		case "--no-ansi-color":
			opt.NoANSIColor = boolFlag(true)
		case "--auto-select":
			opt.AutoSelect = boolFlag(true)
		case "--skip-merge":
			opt.SkipMerge = boolFlag(true)
		case "--skip-download":
			opt.SkipDownload = boolFlag(true)
		case "--check-segments-count":
			opt.CheckSegmentsCount = boolFlag(true)
		case "--binary-merge":
			opt.BinaryMerge = boolFlag(true)
		case "--use-ffmpeg-concat-demuxer":
			opt.UseFFmpegConcatDemuxer = boolFlag(true)
		case "--del-after-done":
			opt.DelAfterDone = boolFlag(true)
		case "--no-date-info":
			opt.NoDateInfo = boolFlag(true)
		case "--no-log":
			opt.NoLog = boolFlag(true)
		case "--write-meta-json":
			opt.WriteMetaJSON = boolFlag(true)
		case "--append-url-params":
			opt.AppendURLParams = boolFlag(true)
		case "-mt", "--concurrent-download":
			opt.ConcurrentDownload = boolFlag(true)
		case "-H", "--header":
			v, err := next()
			if err != nil {
				return opt, err
			}
			addHeaderValue(opt.Headers, v)
			for i+1 < len(args) && !isOptionToken(args[i+1]) && !isLikelyInput(args[i+1]) {
				// long: 上游 --header 是 OneOrMore，并按第一个冒号拆分；连续 header 属于同一次 HTTP 配置，输入 URL 不能被吞掉。
				i++
				addHeaderValue(opt.Headers, args[i])
			}
		case "--sub-only":
			opt.SubOnly = boolFlag(true)
		case "--sub-format":
			v, err := next()
			if err != nil {
				return opt, err
			}
			opt.SubFormat = v
		case "--auto-subtitle-fix":
			opt.AutoSubtitleFix = boolFlag(true)
		case "--ffmpeg-binary-path":
			v, err := next()
			if err != nil {
				return opt, err
			}
			opt.FFmpegBinaryPath = v
		case "--log-level":
			v, err := next()
			if err != nil {
				return opt, err
			}
			opt.LogLevel = v
		case "--ui-language":
			v, err := next()
			if err != nil {
				return opt, err
			}
			opt.UILanguage = v
		case "--urlprocessor-args":
			v, err := next()
			if err != nil {
				return opt, err
			}
			opt.URLProcessorArgs = v
		case "--key":
			v, err := next()
			if err != nil {
				return opt, err
			}
			key, err := parseDecryptKey(v)
			if err != nil {
				return opt, err
			}
			opt.Keys = append(opt.Keys, key)
			for i+1 < len(args) && !isOptionToken(args[i+1]) {
				candidate := args[i+1]
				nextKey, keyErr := parseDecryptKey(candidate)
				if keyErr != nil {
					if isLikelyInput(candidate) {
						break
					}
					return opt, keyErr
				}
				// long: 上游 --key 是 OneOrMore，一次可接多条解密密钥；这里连续吸收合法 key，但遇到 m3u8 输入时保留给位置参数。
				i++
				opt.Keys = append(opt.Keys, nextKey)
			}
		case "--key-text-file":
			v, err := next()
			if err != nil {
				return opt, err
			}
			opt.KeyTextFile = v
		case "--decryption-engine":
			v, err := next()
			if err != nil {
				return opt, err
			}
			opt.DecryptionEngine = v
		case "--decryption-binary-path":
			v, err := next()
			if err != nil {
				return opt, err
			}
			opt.DecryptionBinaryPath = v
		case "--mp4-real-time-decryption":
			opt.MP4RealTimeDecryption = boolFlag(true)
		case "--use-shaka-packager":
			opt.DecryptionEngine = "SHAKA_PACKAGER"
		case "--custom-hls-method":
			v, err := next()
			if err != nil {
				return opt, err
			}
			opt.CustomHLSMethod = normalizeEncryptMethod(v)
		case "--custom-hls-key":
			v, err := next()
			if err != nil {
				return opt, err
			}
			b, err := parseKeyBytes(v)
			if err != nil {
				return opt, err
			}
			opt.CustomHLSKey = b
		case "--custom-hls-iv":
			v, err := next()
			if err != nil {
				return opt, err
			}
			b, err := parseKeyBytes(v)
			if err != nil {
				return opt, err
			}
			opt.CustomHLSIV = b
		case "--use-system-proxy":
			opt.UseSystemProxy = boolFlag(true)
		case "--custom-proxy":
			v, err := next()
			if err != nil {
				return opt, err
			}
			if err := validateProxyURL(v); err != nil {
				return opt, err
			}
			opt.CustomProxy = v
		case "--custom-range":
			v, err := next()
			if err != nil {
				return opt, err
			}
			cr, err := parseCustomRange(v)
			if err != nil {
				return opt, err
			}
			opt.CustomRange = cr
		case "--task-start-at":
			v, err := next()
			if err != nil {
				return opt, err
			}
			t, err := time.ParseInLocation("20060102150405", v, time.Local)
			if err != nil {
				return opt, err
			}
			opt.TaskStartAt = &t
		case "--live-perform-as-vod":
			opt.LivePerformAsVOD = boolFlag(true)
		case "--live-real-time-merge":
			opt.LiveRealTimeMerge = boolFlag(true)
		case "--live-keep-segments":
			opt.LiveKeepSegments = boolFlag(true)
		case "--live-pipe-mux":
			opt.LivePipeMux = boolFlag(true)
		case "--live-record-limit":
			v, err := next()
			if err != nil {
				return opt, err
			}
			d, err := parseDuration(v)
			if err != nil {
				return opt, err
			}
			opt.LiveRecordLimit = &d
		case "--live-wait-time":
			v, err := next()
			if err != nil {
				return opt, err
			}
			wait, err := parseIntOption("LiveWaitTime", v)
			if err != nil {
				return opt, err
			}
			opt.LiveWaitTime = &wait
		case "--live-take-count":
			v, err := next()
			if err != nil {
				return opt, err
			}
			opt.LiveTakeCount, err = parseIntOption("LiveTakeCount", v)
			if err != nil {
				return opt, err
			}
		case "--live-fix-vtt-by-audio":
			opt.LiveFixVTTByAudio = boolFlag(true)
		case "-M", "--mux-after-done":
			v, err := next()
			if err != nil {
				return opt, err
			}
			mux, err := parseMux(v)
			if err != nil {
				return opt, err
			}
			opt.MuxAfterDone = mux
		case "--mux-import":
			v, err := next()
			if err != nil {
				return opt, err
			}
			if err := validateMuxImportArg(v); err != nil {
				return opt, err
			}
			opt.MuxImports = append(opt.MuxImports, v)
			for i+1 < len(args) && !isOptionToken(args[i+1]) && !isLikelyInput(args[i+1]) {
				// long: 上游 --mux-import 是 OneOrMore，单个 flag 后可接多个外部轨道；遇到 m3u8 输入时停止，避免吞掉位置参数。
				i++
				if err := validateMuxImportArg(args[i]); err != nil {
					return opt, err
				}
				opt.MuxImports = append(opt.MuxImports, args[i])
			}
		case "-sv", "--select-video":
			v, err := next()
			if err != nil {
				return opt, err
			}
			filter, err := parseFilterArg(v)
			if err != nil {
				return opt, err
			}
			opt.VideoFilter = filter
		case "-sa", "--select-audio":
			v, err := next()
			if err != nil {
				return opt, err
			}
			filter, err := parseFilterArg(v)
			if err != nil {
				return opt, err
			}
			opt.AudioFilter = filter
		case "-ss", "--select-subtitle":
			v, err := next()
			if err != nil {
				return opt, err
			}
			filter, err := parseFilterArg(v)
			if err != nil {
				return opt, err
			}
			opt.SubtitleFilter = filter
		case "-dv", "--drop-video":
			v, err := next()
			if err != nil {
				return opt, err
			}
			filter, err := parseFilterArg(v)
			if err != nil {
				return opt, err
			}
			opt.DropVideoFilter = filter
		case "-da", "--drop-audio":
			v, err := next()
			if err != nil {
				return opt, err
			}
			filter, err := parseFilterArg(v)
			if err != nil {
				return opt, err
			}
			opt.DropAudioFilter = filter
		case "-ds", "--drop-subtitle":
			v, err := next()
			if err != nil {
				return opt, err
			}
			filter, err := parseFilterArg(v)
			if err != nil {
				return opt, err
			}
			opt.DropSubtitleFilter = filter
		case "--ad-keyword":
			v, err := next()
			if err != nil {
				return opt, err
			}
			opt.AdKeywords = append(opt.AdKeywords, v)
			for i+1 < len(args) && !isOptionToken(args[i+1]) && !isLikelyInput(args[i+1]) {
				// long: 广告清理关键字在上游是字符串数组；连续正则属于同一次清理配置，输入 URL 要留给位置参数。
				i++
				opt.AdKeywords = append(opt.AdKeywords, args[i])
			}
		case "--disable-update-check":
			opt.DisableUpdateCheck = boolFlag(true)
		case "--allow-hls-multi-ext-map":
			opt.AllowHLSMultiExtMap = boolFlag(true)
		case "-R", "--max-speed":
			v, err := next()
			if err != nil {
				return opt, err
			}
			speed, err := parseSpeed(v)
			if err != nil {
				return opt, err
			}
			opt.MaxSpeed = speed
		default:
			if strings.HasPrefix(a, "-") {
				return opt, fmt.Errorf("暂不认识的选项: %s", a)
			}
			if opt.Input != "" {
				return opt, fmt.Errorf("只能指定一个 input")
			}
			opt.Input = a
		}
	}
	if opt.Input == "" {
		return opt, errors.New("缺少 input")
	}
	if opt.ThreadCount < 1 {
		opt.ThreadCount = 1
	}
	return opt, nil
}

func parseIntOption(name, input string) (int, error) {
	value, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("error in parse %s: %s", name, input)
	}
	return value, nil
}

func parseKeyBytes(input string) ([]byte, error) {
	if b, err := os.ReadFile(input); err == nil {
		return b, nil
	}
	clean := strings.TrimPrefix(strings.ToLower(input), "0x")
	if b, err := hex.DecodeString(clean); err == nil {
		return b, nil
	}
	return base64.StdEncoding.DecodeString(input)
}

func validSaveName(input string) string {
	var b strings.Builder
	for _, r := range input {
		switch {
		case r <= 31:
			b.WriteByte('_')
		case strings.ContainsRune("\"<>|:*?\\/", r):
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	// long: 上游 GetValidFileName 只裁掉首尾句点，空格和下划线都属于用户显式文件名的一部分，不能复用更激进的输出路径 safeName。
	return strings.Trim(b.String(), ".")
}

func parseLogFilePathOption(input string) (string, error) {
	path, err := filepath.Abs(input)
	if err != nil {
		return "", errors.New("Invalid log path!")
	}
	filename := filepath.Base(path)
	filename = validSaveName(filename)
	if filename == "" {
		return "", errors.New("Invalid log file name!")
	}
	return filepath.Join(filepath.Dir(path), filename), nil
}

func parseCustomRange(input string) (*CustomRange, error) {
	cr := &CustomRange{Raw: input}
	left, right, ok := strings.Cut(input, "-")
	if !ok {
		return nil, fmt.Errorf("custom-range 格式应为 a-b")
	}
	if strings.Contains(input, ":") {
		if left != "" {
			v, err := parseDuration(left)
			if err != nil {
				return nil, err
			}
			f := v.Seconds()
			cr.StartSec = &f
		} else {
			f := 0.0
			cr.StartSec = &f
		}
		if right != "" {
			v, err := parseDuration(right)
			if err != nil {
				return nil, err
			}
			f := v.Seconds()
			cr.EndSec = &f
		} else {
			f := math.MaxFloat64
			cr.EndSec = &f
		}
		return cr, nil
	}
	if left != "" {
		v, err := strconv.ParseInt(left, 10, 64)
		if err != nil {
			return nil, err
		}
		cr.StartSeg = &v
	} else {
		v := int64(0)
		cr.StartSeg = &v
	}
	if right != "" {
		v, err := strconv.ParseInt(right, 10, 64)
		if err != nil {
			return nil, err
		}
		cr.EndSeg = &v
	} else {
		v := int64(math.MaxInt64)
		cr.EndSeg = &v
	}
	return cr, nil
}

func parseDuration(input string) (time.Duration, error) {
	input = strings.ReplaceAll(input, "：", ":")
	if strings.Count(input, ":") > 0 {
		parts := strings.Split(input, ":")
		if len(parts) > 4 {
			return 0, fmt.Errorf("duration 格式应为 [days:]hours:minutes:seconds")
		}
		values := make([]int, len(parts))
		for i, p := range parts {
			v, err := strconv.Atoi(p)
			if err != nil {
				return 0, err
			}
			values[i] = v
		}
		units := []time.Duration{time.Second, time.Minute, time.Hour, 24 * time.Hour}
		var total time.Duration
		for i := 0; i < len(values); i++ {
			value := values[len(values)-1-i]
			// long: 上游从右往左把冒号时间解释为秒、分、时、天；不能简单当作无限 60 进制，否则四段时间会比真实值大很多。
			total += time.Duration(value) * units[i]
		}
		return total, nil
	}
	if _, err := strconv.Atoi(strings.TrimSpace(input)); err == nil {
		// long: 上游 ParseDur 对单段数字按“秒”解释；直播录制上限这类参数常被写成 30，不能强制用户补 30s。
		seconds, _ := strconv.Atoi(strings.TrimSpace(input))
		return time.Duration(seconds) * time.Second, nil
	}
	return time.ParseDuration(input)
}

func parseMux(input string) (*MuxOptions, error) {
	p := splitComplex(input)
	format := strings.ToLower(p["format"])
	if format == "" {
		format = strings.ToLower(strings.Split(input, ":")[0])
	}
	switch format {
	case "mp4", "mkv", "ts":
	default:
		return nil, fmt.Errorf("format=%s not valid", format)
	}
	muxer := strings.ToLower(p["muxer"])
	if muxer == "" {
		muxer = "ffmpeg"
	}
	if muxer != "ffmpeg" && muxer != "mkvmerge" {
		return nil, fmt.Errorf("muxer=%s not valid", muxer)
	}
	binPath, hasBinPath := p["bin_path"]
	if hasBinPath && binPath == "" {
		return nil, errors.New("bin_path= not valid")
	}
	if binPath == "auto" {
		binPath = ""
	}
	if muxer == "mkvmerge" && format == "mp4" {
		return nil, errors.New("mkvmerge can not do mp4")
	}
	keep, err := parseStrictMuxBool(p, "keep")
	if err != nil {
		return nil, err
	}
	skipSub, err := parseStrictMuxBool(p, "skip_sub")
	if err != nil {
		return nil, err
	}
	return &MuxOptions{
		Format:       format,
		Muxer:        muxer,
		BinPath:      binPath,
		Keep:         keep,
		SkipSubtitle: skipSub,
	}, nil
}

func parseStrictMuxBool(params map[string]string, key string) (bool, error) {
	value := params[key]
	if value == "" {
		return false, nil
	}
	if value != "true" && value != "false" {
		// long: 混流选项直接影响临时文件保留和字幕是否参与封装，非法布尔值应像上游一样立即拒绝，避免误删或漏封轨道。
		return false, fmt.Errorf("%s=%s not valid", key, value)
	}
	return value == "true", nil
}

func parseFilter(input string) *Filter {
	f := &Filter{For: "best"}
	if input == "best" || input == "worst" || input == "all" || strings.HasPrefix(input, "best") || strings.HasPrefix(input, "worst") {
		f.For = input
		return f
	}
	p := splitComplex(input)
	if v := p["for"]; v != "" {
		f.For = v
	}
	f.GroupID, f.Language, f.Name = p["id"], p["lang"], p["name"]
	f.Codecs, f.Resolution, f.FrameRate = p["codecs"], p["res"], p["frame"]
	f.Channels = p["channel"]
	if f.Channels == "" {
		f.Channels = p["ch"]
	}
	f.VideoRange, f.URL = p["range"], p["url"]
	if v := parseOptionalInt64(p["segsMin"]); v != nil {
		f.SegmentsMin = v
	}
	if v := parseOptionalInt64(p["segsMax"]); v != nil {
		f.SegmentsMax = v
	}
	if v := parseOptionalDurationSeconds(p["plistDurMin"]); v != nil {
		f.PlaylistMin = v
	}
	if v := parseOptionalDurationSeconds(p["plistDurMax"]); v != nil {
		f.PlaylistMax = v
	}
	if v := parseOptionalBandwidthKbps(p["bwMin"]); v != nil {
		f.BandwidthMin = v
	}
	if v := parseOptionalBandwidthKbps(p["bwMax"]); v != nil {
		f.BandwidthMax = v
	}
	if role := normalizeRoleFilter(p["role"]); role != "" {
		f.Role = role
	}
	return f
}

func parseFilterArg(input string) (*Filter, error) {
	filter := parseFilter(input)
	if !validFilterFor(filter.For) {
		return nil, fmt.Errorf("for=%s not valid", filter.For)
	}
	if err := validateFilterValueFields(input); err != nil {
		return nil, err
	}
	if err := validateFilters(Options{VideoFilter: filter}); err != nil {
		return nil, err
	}
	return filter, nil
}

func validateFilterValueFields(input string) error {
	p := splitComplex(input)
	for _, key := range []string{"segsMin", "segsMax"} {
		if value := p[key]; value != "" {
			if _, err := strconv.ParseInt(value, 10, 64); err != nil {
				return fmt.Errorf("%s=%s not valid", key, value)
			}
		}
	}
	for _, key := range []string{"plistDurMin", "plistDurMax"} {
		if value := p[key]; value != "" {
			if _, err := parseDuration(value); err != nil {
				return fmt.Errorf("%s=%s not valid", key, value)
			}
		}
	}
	for _, key := range []string{"bwMin", "bwMax"} {
		if value := p[key]; value != "" {
			if _, err := strconv.Atoi(value); err != nil {
				return fmt.Errorf("%s=%s not valid", key, value)
			}
		}
	}
	return nil
}

func normalizeRoleFilter(input string) string {
	if input == "" {
		return ""
	}
	for _, role := range []string{
		"Subtitle",
		"Main",
		"Alternate",
		"Supplementary",
		"Commentary",
		"Dub",
		"Description",
		"Sign",
		"Metadata",
		"ForcedSubtitle",
	} {
		if strings.EqualFold(input, role) {
			// long: role 过滤来自上游通用轨道筛选器；非法值在原版 Enum.TryParse 失败后会被忽略，不能把它当作正则或普通字符串误过滤。
			return role
		}
	}
	return ""
}

func parseOptionalInt64(input string) *int64 {
	if input == "" {
		return nil
	}
	v, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}

func parseOptionalDurationSeconds(input string) *float64 {
	if input == "" {
		return nil
	}
	d, err := parseDuration(input)
	if err != nil {
		return nil
	}
	v := d.Seconds()
	return &v
}

func parseOptionalBandwidthKbps(input string) *int {
	if input == "" {
		return nil
	}
	v, err := strconv.Atoi(input)
	if err != nil {
		return nil
	}
	v *= 1000
	return &v
}

var speedLimitRE = regexp.MustCompile(`^([\d.]+)(M|K)$`)
var pairKeyRE = regexp.MustCompile(`^[0-9a-fA-F]{32}:[0-9a-fA-F]{32}$`)
var idHexKeyRE = regexp.MustCompile(`^[0-9]+:[0-9a-fA-F]{32}$`)
var singleHexKeyRE = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)

func parseSpeed(input string) (int64, error) {
	input = strings.ToUpper(strings.TrimSpace(input))
	match := speedLimitRE.FindStringSubmatch(input)
	if match == nil {
		return 0, fmt.Errorf("error in parse SpeedLimit: %s", input)
	}
	mult := int64(1024)
	if match[2] == "M" {
		mult = 1024 * 1024
	}
	v, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0, fmt.Errorf("error in parse SpeedLimit: %s", input)
	}
	return int64(v * float64(mult)), nil
}

func parseDecryptKey(input string) (string, error) {
	input = strings.TrimSpace(input)
	if pairKeyRE.MatchString(input) || idHexKeyRE.MatchString(input) || singleHexKeyRE.MatchString(input) {
		return strings.ToLower(input), nil
	}
	parts := strings.Split(input, ":")
	if len(parts) < 1 || len(parts) > 2 {
		return "", fmt.Errorf("error in parse custom key: Input must be KEY or KID:KEY format. All Inputs=[%s]", input)
	}
	parsePart := func(part, label string) (string, error) {
		part = strings.TrimSpace(part)
		if singleHexKeyRE.MatchString(part) {
			return strings.ToLower(part), nil
		}
		raw, err := base64.StdEncoding.DecodeString(part)
		if err == nil && len(raw) == 16 {
			return hex.EncodeToString(raw), nil
		}
		return "", fmt.Errorf("%s must be valid 16-byte HEX or Base64. Input string: %s", label, part)
	}
	key, err := parsePart(parts[len(parts)-1], "KEY")
	if err != nil {
		return "", fmt.Errorf("error in parse custom key: %s. All Inputs=[%s]", err, input)
	}
	if len(parts) == 1 {
		return key, nil
	}
	kid, err := parsePart(parts[0], "KID")
	if err != nil {
		return "", fmt.Errorf("error in parse custom key: %s. All Inputs=[%s]", err, input)
	}
	return kid + ":" + key, nil
}

func validateProxyURL(input string) error {
	u, err := url.Parse(strings.TrimSpace(input))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("error in parse proxy: %s", input)
	}
	return nil
}

func addHeaderValue(headers map[string]string, raw string) {
	k, val, ok := strings.Cut(raw, ":")
	if !ok {
		return
	}
	headers[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(val)
}

func isOptionToken(input string) bool {
	return strings.HasPrefix(input, "-") && input != "-"
}

func isLikelyInput(input string) bool {
	lower := strings.ToLower(strings.TrimSpace(input))
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "file:") ||
		strings.HasSuffix(lower, ".m3u8")
}

func validateMuxImportArg(raw string) error {
	path := muxImportPath(raw)
	// long: 上游 ParseImports 会在命令行解析阶段拒绝空路径或不存在的外部轨道，提前失败可避免后续把无效字幕/音轨混入任务状态。
	if path == "" {
		return errors.New("path empty or file not exists!")
	}
	if _, err := os.Stat(path); err != nil {
		return errors.New("path empty or file not exists!")
	}
	return nil
}

func usage() string {
	return `N_m3u8DL-GO-HLS <input> [options]

常用:
  --auto-select
  --save-dir <dir>
  --save-name <name>
  -H "Cookie: xxx" -H "User-Agent: xxx"
  --thread-count <n>
  --custom-hls-key <file|hex|base64>
  --custom-hls-iv <file|hex|base64>
  --mp4-real-time-decryption
  --custom-range <0-10|01:00-02:00>
  --skip-download
  --skip-merge
  --binary-merge
  -M format=mp4
  --morehelp <mux-after-done|mux-import|custom-range|select-video|select-audio|select-subtitle>
`
}

func moreHelp(topic string) string {
	switch topic {
	case "mux-after-done":
		return `--mux-after-done / -M
  format=mp4|mkv|ts:muxer=ffmpeg:keep=true|false:skip_sub=true|false
`
	case "mux-import":
		return `--mux-import
  path=PATH:lang=CODE:name=NAME

  path 指定外部媒体文件路径
  lang 指定语言代码，可省略
  name 指定轨道描述，可省略

示例:
  --mux-import path=zh-Hans.srt:lang=chi:name="中文 (简体)"
  --mux-import path="D:\media\atmos.m4a":lang=eng:name="English Description Audio"
`
	case "custom-range":
		return `--custom-range
  0-10          下载第 0 到 10 个分片
  10-           从第 10 个分片下载到末尾
  00:01-00:02   按时间范围下载
`
	case "select-video", "select-audio", "select-subtitle":
		return `选择器:
  best / worst / all
  best2 / worst3 / for=all
  res=1920x*:codecs=avc:for=best
  lang=zh|en:name=中文:for=all
  bwMin=800:bwMax=5000:segsMin=10:plistDurMin=00:10:for=best2
`
	default:
		return "暂无该选项的详细帮助\n"
	}
}
