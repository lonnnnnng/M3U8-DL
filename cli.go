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
	"unicode"
)

var filterDurationRegex = regexp.MustCompile(`^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?$`)

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
		UILanguage:         defaultUILanguage(),
		DecryptionEngine:   "MP4DECRYPT",
		UseSystemProxy:     true,
		LiveKeepSegments:   true,
		LiveTakeCount:      16,
		CustomHLSMethod:    "",
		FFmpegBinaryPath:   "ffmpeg",
		DisableUpdateCheck: false,
	}
}

func defaultUILanguage() string {
	loc := "en-US"
	curr := os.Getenv("LC_ALL")
	if curr == "" {
		curr = os.Getenv("LANG")
	}
	curr = strings.ReplaceAll(strings.Split(curr, ".")[0], "_", "-")
	// long: 原版默认资源语言是英文，只把系统中文区域自动归到简中或繁中；其他语言环境不应被误映射成中文。
	switch {
	case curr == "zh-CN" || curr == "zh-SG":
		return "zh-CN"
	case strings.HasPrefix(curr, "zh-"):
		return "zh-TW"
	default:
		return loc
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
			if boolFlag(true) {
				// long: 上游保留了隐藏的 Shaka 快捷开关，但它仍是普通 bool 选项；显式传 false 时不能覆盖用户已经指定的解密引擎。
				opt.DecryptionEngine = "SHAKA_PACKAGER"
			}
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
			b, err := parseHLSCustomKeyOption(v)
			if err != nil {
				return opt, err
			}
			opt.CustomHLSKey = b
		case "--custom-hls-iv":
			v, err := next()
			if err != nil {
				return opt, err
			}
			b, err := parseHLSCustomKeyOption(v)
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
			if v == "" {
				opt.CustomProxy = ""
				continue
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
			cr, err := parseCustomRangeOption(v)
			if err != nil {
				return opt, err
			}
			opt.CustomRange = cr
		case "--task-start-at":
			v, err := next()
			if err != nil {
				return opt, err
			}
			t, err := parseTaskStartAt(v)
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
			d, err := parseLiveRecordLimit(v)
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
	return decodeBase64LikeDotNet(input)
}

func decodeBase64LikeDotNet(input string) ([]byte, error) {
	// long: 上游 Convert.FromBase64String 会忽略复制 key 时常见的换行和空格，所有 key 的 base64 分支都要保留这个宽容度。
	clean := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, input)
	return base64.StdEncoding.DecodeString(clean)
}

func parseHLSCustomKeyOption(input string) ([]byte, error) {
	if input == "" {
		return nil, nil
	}
	b, err := parseKeyBytes(input)
	if err != nil {
		// long: 自定义 HLS key/iv 来自同一个上游 parser，失败时只回显用户输入，避免把 Go 的 base64/hex 错误文案暴露成兼容性差异。
		return nil, fmt.Errorf("error in parse hls custom key: %s", input)
	}
	return b, nil
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
	if !ok || strings.Count(input, "-") != 1 {
		return nil, errors.New("Bad format!")
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
	match := segmentRangeRE.FindStringSubmatch(input)
	if match == nil {
		return nil, errors.New("Bad format!")
	}
	left, right = match[1], match[2]
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

func parseCustomRangeOption(input string) (*CustomRange, error) {
	if input == "" {
		return nil, nil
	}
	cr, err := parseCustomRange(input)
	if err != nil {
		// long: 原版 ParseCustomRange 会把所有格式和时长错误包成统一前缀，避免 CLI 暴露底层数字/时间解析实现差异。
		return nil, fmt.Errorf("error in parse CustomRange: %s", err.Error())
	}
	return cr, nil
}

func parseDuration(input string) (time.Duration, error) {
	input = strings.ReplaceAll(input, "：", ":")
	if strings.Count(input, ":") > 0 {
		parts := strings.Split(input, ":")
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
		limit := len(values)
		if limit > len(units) {
			limit = len(units)
		}
		for i := 0; i < limit; i++ {
			value := values[len(values)-1-i]
			// long: 上游从右往左把冒号时间解释为秒、分、时、天；超过四段时仍会校验全部数字，但最左侧多余字段不参与计算。
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

func parseLiveRecordLimit(input string) (time.Duration, error) {
	d, err := parseDuration(input)
	if err != nil {
		// long: 原版直播录制上限解析失败只回显用户输入，不拼接底层时长错误，便于保持三语言 CLI 行为稳定。
		return 0, fmt.Errorf("error in parse LiveRecordLimit: %s", input)
	}
	return d, nil
}

func parseTaskStartAt(input string) (time.Time, error) {
	t, err := time.ParseInLocation("20060102150405", input, time.Local)
	if err != nil {
		// long: 原版 ParseStartTime 把格式错误统一映射成这个固定文案，CLI 用户和测试都不需要感知底层时间库的细节。
		return time.Time{}, fmt.Errorf("error in parse TaskStartTime: %s", input)
	}
	return t, nil
}

func parseMux(input string) (*MuxOptions, error) {
	p := splitComplex(input)
	rawFormat := p["format"]
	if rawFormat == "" {
		rawFormat = strings.Split(input, ":")[0]
	}
	format := strings.ToLower(rawFormat)
	switch format {
	case "mp4", "mkv", "ts":
	default:
		return nil, fmt.Errorf("format=%s not valid", rawFormat)
	}
	muxer := p["muxer"]
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
	if muxer == "mkvmerge" && rawFormat == "mp4" {
		// long: 原版冲突判断使用用户输入的原始 format 字符串；这里保持同样的大小写敏感语义，避免 CLI 解析层行为漂移。
		return nil, errors.New("mkvmerge can not do mp4")
	}
	keep, err := parseStrictMuxBool(input, p, "keep", "keep")
	if err != nil {
		return nil, err
	}
	skipSub, err := parseStrictMuxBool(input, p, "skip_sub", "keep")
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

func parseStrictMuxBool(input string, params map[string]string, key string, errorValueKey string) (bool, error) {
	hasExplicitValue := strings.Contains(input, key+"=")
	value := params[key]
	if !hasExplicitValue {
		if strings.Contains(input, key) && strings.HasSuffix(input, key) {
			// long: 上游 ComplexParamParser 只有在整串参数以 keep/skip_sub 结尾且没有 key= 时才把裸布尔解释为 true。
			return true, nil
		}
		return false, nil
	}
	if value != "true" && value != "false" {
		displayValue := value
		if errorValueKey != key {
			// long: 上游 skip_sub 非法时会误回显 keep 的值；CLI 兼容测试依赖这个文案形态，避免脚本按原版错误文本判断时失配。
			displayValue = params[errorValueKey]
			if displayValue == "" {
				displayValue = "false"
			}
		}
		// long: 混流选项直接影响临时文件保留和字幕是否参与封装，非法布尔值应像上游一样立即拒绝，避免误删或漏封轨道。
		return false, fmt.Errorf("%s=%s not valid", key, displayValue)
	}
	return value == "true", nil
}

func parseFilter(input string) *Filter {
	f := &Filter{For: "best"}
	if input == filterForRE.FindString(input) {
		// long: 原版裸选轨值只在整串正好匹配 best/worst/all 时生效；bestx 这类值会被当作复杂参数并回退默认 best。
		f.For = input
		return f
	}
	if v, ok := complexParamValue(input, "for"); ok && v != "" {
		f.For = v
	}
	f.GroupID = complexParamValueOrEmpty(input, "id")
	f.Language = complexParamValueOrEmpty(input, "lang")
	f.Name = complexParamValueOrEmpty(input, "name")
	f.Codecs = complexParamValueOrEmpty(input, "codecs")
	f.Resolution = complexParamValueOrEmpty(input, "res")
	f.FrameRate = complexParamValueOrEmpty(input, "frame")
	f.Channels = complexParamValueOrEmpty(input, "channel")
	if f.Channels == "" {
		f.Channels = complexParamValueOrEmpty(input, "ch")
	}
	f.VideoRange = complexParamValueOrEmpty(input, "range")
	f.URL = complexParamValueOrEmpty(input, "url")
	if v := parseOptionalInt64(complexParamValueOrEmpty(input, "segsMin")); v != nil {
		f.SegmentsMin = v
	}
	if v := parseOptionalInt64(complexParamValueOrEmpty(input, "segsMax")); v != nil {
		f.SegmentsMax = v
	}
	if v := parseOptionalFilterDurationSeconds(complexParamValueOrEmpty(input, "plistDurMin")); v != nil {
		f.PlaylistMin = v
	}
	if v := parseOptionalFilterDurationSeconds(complexParamValueOrEmpty(input, "plistDurMax")); v != nil {
		f.PlaylistMax = v
	}
	if v := parseOptionalBandwidthKbps(complexParamValueOrEmpty(input, "bwMin")); v != nil {
		f.BandwidthMin = v
	}
	if v := parseOptionalBandwidthKbps(complexParamValueOrEmpty(input, "bwMax")); v != nil {
		f.BandwidthMax = v
	}
	if role := normalizeRoleFilter(complexParamValueOrEmpty(input, "role")); role != "" {
		f.Role = role
	}
	return f
}

func complexParamValueOrEmpty(input, key string) string {
	value, _ := complexParamValue(input, key)
	return value
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
	for _, key := range []string{"segsMin", "segsMax"} {
		if value := complexParamValueOrEmpty(input, key); value != "" {
			if _, err := strconv.ParseInt(value, 10, 64); err != nil {
				return fmt.Errorf("%s=%s not valid", key, value)
			}
		}
	}
	for _, key := range []string{"plistDurMin", "plistDurMax"} {
		if value := complexParamValueOrEmpty(input, key); value != "" {
			if _, err := parseFilterDurationSeconds(value); err != nil {
				return fmt.Errorf("%s=%s not valid", key, value)
			}
		}
	}
	for _, key := range []string{"bwMin", "bwMax"} {
		if value := complexParamValueOrEmpty(input, key); value != "" {
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

func parseOptionalFilterDurationSeconds(input string) *float64 {
	if input == "" {
		return nil
	}
	v, err := parseFilterDurationSeconds(input)
	if err != nil {
		return nil
	}
	return &v
}

func parseFilterDurationSeconds(input string) (float64, error) {
	match := filterDurationRegex.FindStringSubmatch(input)
	if match == nil {
		return 0, fmt.Errorf("invalid filter duration")
	}
	var values [3]int
	for i := 1; i <= 3; i++ {
		if match[i] == "" {
			continue
		}
		value, err := strconv.Atoi(match[i])
		if err != nil {
			return 0, err
		}
		values[i-1] = value
	}
	// long: 选择器的播放列表时长走原版 ParseSeconds，只认 h/m/s 后缀；冒号时长属于 custom-range/live-record-limit，不能在这里放宽。
	return float64(values[0]*3600 + values[1]*60 + values[2]), nil
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

var speedLimitRE = regexp.MustCompile(`([\d.]+)(M|K)`)
var segmentRangeRE = regexp.MustCompile(`(\d*)-(\d*)`)
var pairKeyRE = regexp.MustCompile(`^[0-9a-fA-F]{32}:[0-9a-fA-F]{32}$`)
var idHexKeyRE = regexp.MustCompile(`^[0-9]+:[0-9a-fA-F]{32}$`)
var singleHexKeyRE = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)
var filterForRE = regexp.MustCompile(`((best|worst)\d*|all)`)

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
	if pairKeyRE.MatchString(input) || idHexKeyRE.MatchString(input) || singleHexKeyRE.MatchString(input) {
		// long: 上游对已匹配标准 HEX / KID:KEY / trackId:KEY 的输入会直接加入列表，不会归一化大小写；这会影响后续运行时 StartsWith(kid) 的精确匹配。
		return input, nil
	}
	rawParts := strings.Split(input, ":")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		part = strings.TrimSpace(part)
		if part != "" {
			// long: 上游 Split 使用 RemoveEmptyEntries 和 TrimEntries，空 KID/KEY 段会被丢弃后再判断是一段 key 还是 KID:KEY。
			parts = append(parts, part)
		}
	}
	if len(parts) < 1 || len(parts) > 2 {
		return "", fmt.Errorf("error in parse custom key: Input must be KEY or KID:KEY format. All Inputs=[%s]", input)
	}
	parsePart := func(part, label string) (string, error) {
		if singleHexKeyRE.MatchString(part) {
			return strings.ToLower(part), nil
		}
		raw, err := decodeBase64LikeDotNet(part)
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
	u, err := url.Parse(input)
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
	return usageWithOptions(Options{UILanguage: "zh-CN"})
}

func usageWithOptions(opt Options) string {
	lines := []struct {
		option string
		key    string
	}{
		{"--auto-select", "cmd_autoSelect"},
		{"--save-dir <dir>", "cmd_saveDir"},
		{"--save-name <name>", "cmd_saveName"},
		{"-H \"Cookie: xxx\" -H \"User-Agent: xxx\"", "cmd_header"},
		{"--thread-count <n>", "cmd_threadCount"},
		{"--custom-hls-key <file|hex|base64>", "cmd_customHLSKey"},
		{"--custom-hls-iv <file|hex|base64>", "cmd_customHLSIv"},
		{"--mp4-real-time-decryption", "cmd_MP4RealTimeDecryption"},
		{"--custom-range <0-10|01:00-02:00>", "cmd_customRange"},
		{"--skip-download", "cmd_skipDownload"},
		{"--skip-merge", "cmd_skipMerge"},
		{"--binary-merge", "cmd_binaryMerge"},
		{"-M format=mp4", "cmd_muxAfterDone"},
		{"--morehelp <mux-after-done|mux-import|custom-range|select-video|select-audio|select-subtitle>", "cmd_moreHelp"},
	}
	var b strings.Builder
	b.WriteString("N_m3u8DL-GO-HLS <input> [options]\n\n")
	b.WriteString("常用:\n")
	for _, line := range lines {
		// long: usage 属于用户直接看到的 CLI 表面；用原版 ResString key 生成说明，后续补齐语言资源时不会再遗留硬编码文案。
		fmt.Fprintf(&b, "  %-92s %s\n", line.option, tr(opt, line.key))
	}
	return b.String()
}

func moreHelp(topic string) string {
	// long: --morehelp 是原版公开 CLI 体验的一部分；这里保留原版标题结构和 zh-CN 详细文本，避免用户查复杂参数时看到简化版说明。
	msg, ok := map[string]string{
		"mux-after-done": `所有工作完成时尝试混流分离的音视频. 你能够以:分隔形式指定如下参数:

* format=FORMAT: 指定混流容器 mkv, mp4, ts
* muxer=MUXER: 指定混流程序 ffmpeg, mkvmerge (默认: ffmpeg)
* bin_path=PATH: 指定程序路径 (默认: 自动寻找)
* skip_sub=BOOL: 是否忽略字幕文件 (默认: false)
* keep=BOOL: 混流完成是否保留文件 true, false (默认: false)

例如:
# 混流为mp4容器
-M format=mp4
# 使用mkvmerge, 自动寻找程序
-M format=mkv:muxer=mkvmerge
# 使用mkvmerge, 自定义程序路径
-M format=mkv:muxer=mkvmerge:bin_path="C\:\Program Files\MKVToolNix\mkvmerge.exe"
`,
		"mux-import": `混流时引入外部媒体文件. 你能够以:分隔形式指定如下参数:

* path=PATH: 指定媒体文件路径
* lang=CODE: 指定媒体文件语言代码 (非必须)
* name=NAME: 指定媒体文件描述信息 (非必须)

例如:
# 引入外部字幕
--mux-import path=zh-Hans.srt:lang=chi:name="中文 (简体)"
# 引入外部音轨+字幕
--mux-import path="D\:\media\atmos.m4a":lang=eng:name="English Description Audio" --mux-import path="D\:\media\eng.vtt":lang=eng:name="English (Description)"
`,
		"custom-range": `下载点播内容时, 仅下载部分分片.

例如:
# 下载[0,10]共11个分片
--custom-range 0-10
# 下载从序号10开始的后续分片
--custom-range 10-
# 下载前100个分片
--custom-range -99
# 下载第5分钟到20分钟的内容
--custom-range 05:00-20:00
`,
		"select-video": `通过正则表达式选择符合要求的视频流. 你能够以:分隔形式指定如下参数:

id=REGEX:lang=REGEX:name=REGEX:codecs=REGEX:res=REGEX:frame=REGEX
segsMin=number:segsMax=number:ch=REGEX:range=REGEX:url=REGEX
plistDurMin=hms:plistDurMax=hms:bwMin=int:bwMax=int:role=string:for=FOR

* for=FOR: 选择方式. best[number], worst[number], all (默认: best)

例如:
# 选择最佳视频
-sv best
# 选择4K+HEVC视频
-sv res="3840*":codecs=hvc1:for=best
# 选择长度大于1小时20分钟30秒的视频
-sv plistDurMin="1h20m30s":for=best
-sv role="main":for=best
# 选择码率在800Kbps至1Mbps之间的视频
-sv bwMin=800:bwMax=1000
`,
		"select-audio": `通过正则表达式选择符合要求的音频流. 参考 --select-video

例如:
# 选择所有音频
-sa all
# 选择最佳英语音轨
-sa lang=en:for=best
# 选择最佳的2条英语(或日语)音轨
-sa lang="ja|en":for=best2
-sa role="main":for=best
`,
		"select-subtitle": `通过正则表达式选择符合要求的字幕流. 参考 --select-video

例如:
# 选择所有字幕
-ss all
# 选择所有带有"中文"的字幕
-ss name="中文":for=all
`,
	}[topic]
	if !ok {
		msg = "not found"
	}
	return fmt.Sprintf("More Help:\n\n  --%s\n\n%s", topic, msg)
}
