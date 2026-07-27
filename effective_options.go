package main

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"
)

type effectiveOptionsReport struct {
	Version      versionInfo           `json:"version"`
	Input        string                `json:"input"`
	Paths        effectivePaths        `json:"paths"`
	Network      effectiveNetwork      `json:"network"`
	Download     effectiveDownload     `json:"download"`
	Subtitles    effectiveSubtitles    `json:"subtitles"`
	Decryption   effectiveDecryption   `json:"decryption"`
	Live         effectiveLive         `json:"live"`
	Mux          effectiveMux          `json:"mux"`
	Filters      effectiveFilters      `json:"filters"`
	Implications []string              `json:"implications,omitempty"`
	Diagnostics  effectiveDiagnostics  `json:"diagnostics"`
	Raw          effectiveRawFootprint `json:"raw"`
}

type effectivePaths struct {
	SaveDir      string `json:"saveDir"`
	SaveName     string `json:"saveName"`
	SavePattern  string `json:"savePattern,omitempty"`
	TmpDir       string `json:"tmpDir,omitempty"`
	LogFilePath  string `json:"logFilePath,omitempty"`
	FFmpegBinary string `json:"ffmpegBinary"`
}

type effectiveNetwork struct {
	BaseURL            string            `json:"baseUrl,omitempty"`
	Headers            map[string]string `json:"headers"`
	UseSystemProxy     bool              `json:"useSystemProxy"`
	CustomProxy        string            `json:"customProxy,omitempty"`
	AppendURLParams    bool              `json:"appendUrlParams"`
	HTTPRequestTimeout float64           `json:"httpRequestTimeout"`
}

type effectiveDownload struct {
	ThreadCount            int    `json:"threadCount"`
	DownloadRetryCount     int    `json:"downloadRetryCount"`
	MaxSpeedBytes          int64  `json:"maxSpeedBytes"`
	ConcurrentDownload     bool   `json:"concurrentDownload"`
	CheckSegmentsCount     bool   `json:"checkSegmentsCount"`
	AutoSelect             bool   `json:"autoSelect"`
	SkipDownload           bool   `json:"skipDownload"`
	SkipMerge              bool   `json:"skipMerge"`
	BinaryMerge            bool   `json:"binaryMerge"`
	UseFFmpegConcatDemuxer bool   `json:"useFFmpegConcatDemuxer"`
	DelAfterDone           bool   `json:"delAfterDone"`
	WriteMetaJSON          bool   `json:"writeMetaJson"`
	DisableUpdateCheck     bool   `json:"disableUpdateCheck"`
	LogLevel               string `json:"logLevel"`
	UILanguage             string `json:"uiLanguage"`
}

type effectiveSubtitles struct {
	SubOnly             bool   `json:"subOnly"`
	SubFormat           string `json:"subFormat"`
	AutoSubtitleFix     bool   `json:"autoSubtitleFix"`
	LiveFixVTTByAudio   bool   `json:"liveFixVttByAudio"`
	AllowHLSMultiExtMap bool   `json:"allowHlsMultiExtMap"`
}

type effectiveDecryption struct {
	Engine              string `json:"engine"`
	BinaryPath          string `json:"binaryPath,omitempty"`
	KeysCount           int    `json:"keysCount"`
	KeyTextFile         string `json:"keyTextFile,omitempty"`
	MP4RealTime         bool   `json:"mp4RealTime"`
	CustomHLSMethod     string `json:"customHlsMethod,omitempty"`
	CustomHLSKeyPresent bool   `json:"customHlsKeyPresent"`
	CustomHLSKeyBytes   int    `json:"customHlsKeyBytes,omitempty"`
	CustomHLSIVPresent  bool   `json:"customHlsIvPresent"`
	CustomHLSIVBytes    int    `json:"customHlsIvBytes,omitempty"`
}

type effectiveLive struct {
	PerformAsVOD  bool   `json:"performAsVod"`
	RealTimeMerge bool   `json:"realTimeMerge"`
	KeepSegments  bool   `json:"keepSegments"`
	PipeMux       bool   `json:"pipeMux"`
	RecordLimit   string `json:"recordLimit,omitempty"`
	WaitTime      *int   `json:"waitTime,omitempty"`
	TakeCount     int    `json:"takeCount"`
	TaskStartAt   string `json:"taskStartAt,omitempty"`
}

type effectiveMux struct {
	AfterDone  *MuxOptions `json:"afterDone,omitempty"`
	Imports    []string    `json:"imports,omitempty"`
	NoDateInfo bool        `json:"noDateInfo"`
}

type effectiveFilters struct {
	CustomRange  string   `json:"customRange,omitempty"`
	Video        *Filter  `json:"video,omitempty"`
	Audio        *Filter  `json:"audio,omitempty"`
	Subtitle     *Filter  `json:"subtitle,omitempty"`
	DropVideo    *Filter  `json:"dropVideo,omitempty"`
	DropAudio    *Filter  `json:"dropAudio,omitempty"`
	DropSubtitle *Filter  `json:"dropSubtitle,omitempty"`
	AdKeywords   []string `json:"adKeywords,omitempty"`
}

type effectiveDiagnostics struct {
	ForceANSIConsole bool `json:"forceAnsiConsole"`
	NoANSIColor      bool `json:"noAnsiColor"`
	ProgressJSON     bool `json:"progressJson"`
	ProbeJSON        bool `json:"probeJson"`
	NoLog            bool `json:"noLog"`
}

type effectiveRawFootprint struct {
	URLProcessorArgs string `json:"urlProcessorArgs,omitempty"`
}

func runPrintEffectiveOptions(opt Options, command []string, now time.Time) (string, error) {
	if err := validateOptions(opt); err != nil {
		return "", err
	}
	implications := applyOptionImplicationsWithMessages(&opt)
	applyDerivedDefaults(&opt, now)
	report := buildEffectiveOptionsReport(opt, implications)
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}

func buildEffectiveOptionsReport(opt Options, implications []string) effectiveOptionsReport {
	return effectiveOptionsReport{
		Version: currentVersionInfo(),
		Input:   opt.Input,
		Paths: effectivePaths{
			SaveDir:      opt.SaveDir,
			SaveName:     opt.SaveName,
			SavePattern:  opt.SavePattern,
			TmpDir:       opt.TmpDir,
			LogFilePath:  opt.LogFilePath,
			FFmpegBinary: opt.FFmpegBinaryPath,
		},
		Network: effectiveNetwork{
			BaseURL:            opt.BaseURL,
			Headers:            redactHeaders(opt.Headers),
			UseSystemProxy:     opt.UseSystemProxy,
			CustomProxy:        redactProxy(opt.CustomProxy),
			AppendURLParams:    opt.AppendURLParams,
			HTTPRequestTimeout: opt.HTTPRequestTimeout,
		},
		Download: effectiveDownload{
			ThreadCount:            opt.ThreadCount,
			DownloadRetryCount:     opt.DownloadRetryCount,
			MaxSpeedBytes:          opt.MaxSpeed,
			ConcurrentDownload:     opt.ConcurrentDownload,
			CheckSegmentsCount:     opt.CheckSegmentsCount,
			AutoSelect:             opt.AutoSelect,
			SkipDownload:           opt.SkipDownload,
			SkipMerge:              opt.SkipMerge,
			BinaryMerge:            opt.BinaryMerge,
			UseFFmpegConcatDemuxer: opt.UseFFmpegConcatDemuxer,
			DelAfterDone:           opt.DelAfterDone,
			WriteMetaJSON:          opt.WriteMetaJSON,
			DisableUpdateCheck:     opt.DisableUpdateCheck,
			LogLevel:               opt.LogLevel,
			UILanguage:             opt.UILanguage,
		},
		Subtitles: effectiveSubtitles{
			SubOnly:             opt.SubOnly,
			SubFormat:           opt.SubFormat,
			AutoSubtitleFix:     opt.AutoSubtitleFix,
			LiveFixVTTByAudio:   opt.LiveFixVTTByAudio,
			AllowHLSMultiExtMap: opt.AllowHLSMultiExtMap,
		},
		Decryption: effectiveDecryption{
			Engine:              opt.DecryptionEngine,
			BinaryPath:          opt.DecryptionBinaryPath,
			KeysCount:           len(opt.Keys),
			KeyTextFile:         opt.KeyTextFile,
			MP4RealTime:         opt.MP4RealTimeDecryption,
			CustomHLSMethod:     string(opt.CustomHLSMethod),
			CustomHLSKeyPresent: len(opt.CustomHLSKey) > 0,
			CustomHLSKeyBytes:   len(opt.CustomHLSKey),
			CustomHLSIVPresent:  len(opt.CustomHLSIV) > 0,
			CustomHLSIVBytes:    len(opt.CustomHLSIV),
		},
		Live: effectiveLive{
			PerformAsVOD:  opt.LivePerformAsVOD,
			RealTimeMerge: opt.LiveRealTimeMerge,
			KeepSegments:  opt.LiveKeepSegments,
			PipeMux:       opt.LivePipeMux,
			RecordLimit:   durationString(opt.LiveRecordLimit),
			WaitTime:      opt.LiveWaitTime,
			TakeCount:     opt.LiveTakeCount,
			TaskStartAt:   timeString(opt.TaskStartAt),
		},
		Mux: effectiveMux{
			AfterDone:  opt.MuxAfterDone,
			Imports:    append([]string(nil), opt.MuxImports...),
			NoDateInfo: opt.NoDateInfo,
		},
		Filters: effectiveFilters{
			CustomRange:  customRangeString(opt.CustomRange),
			Video:        opt.VideoFilter,
			Audio:        opt.AudioFilter,
			Subtitle:     opt.SubtitleFilter,
			DropVideo:    opt.DropVideoFilter,
			DropAudio:    opt.DropAudioFilter,
			DropSubtitle: opt.DropSubtitleFilter,
			AdKeywords:   append([]string(nil), opt.AdKeywords...),
		},
		Implications: implications,
		Diagnostics: effectiveDiagnostics{
			ForceANSIConsole: opt.ForceANSIConsole,
			NoANSIColor:      opt.NoANSIColor,
			ProgressJSON:     opt.ProgressJSON,
			ProbeJSON:        opt.ProbeJSON,
			NoLog:            opt.NoLog,
		},
		Raw: effectiveRawFootprint{
			URLProcessorArgs: opt.URLProcessorArgs,
		},
	}
}

func redactHeaders(headers map[string]string) map[string]string {
	out := make(map[string]string, len(headers))
	for key, value := range headers {
		if sensitiveHeaderName(key) {
			out[key] = "<redacted>"
			continue
		}
		out[key] = value
	}
	return out
}

func sensitiveHeaderName(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	return key == "cookie" ||
		key == "authorization" ||
		key == "proxy-authorization" ||
		key == "x-api-key" ||
		key == "x-auth-token"
}

func redactProxy(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	username := u.User.Username()
	if _, ok := u.User.Password(); ok {
		u.User = url.UserPassword(username, "redacted")
		return u.String()
	}
	return raw
}

func durationString(value *time.Duration) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func timeString(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02 15:04:05")
}

func customRangeString(value *CustomRange) string {
	if value == nil {
		return ""
	}
	return value.Raw
}
