package main

import "time"

type MediaType string

const (
	MediaVideo     MediaType = "VIDEO"
	MediaAudio     MediaType = "AUDIO"
	MediaSubtitles MediaType = "SUBTITLES"
)

type EncryptMethod string

const (
	EncryptNone      EncryptMethod = "NONE"
	EncryptAES128    EncryptMethod = "AES-128"
	EncryptAES128ECB EncryptMethod = "AES-128-ECB"
	EncryptCENC      EncryptMethod = "CENC"
	EncryptSampleAES EncryptMethod = "SAMPLE-AES"
	EncryptSampleCTR EncryptMethod = "SAMPLE-AES-CTR"
	EncryptChaCha20  EncryptMethod = "CHACHA20"
	EncryptUnknown   EncryptMethod = "UNKNOWN"
)

type EncryptInfo struct {
	Method EncryptMethod `json:"method"`
	Key    []byte        `json:"-"`
	IV     []byte        `json:"-"`
}

type Segment struct {
	Index        int64       `json:"index"`
	URL          string      `json:"url"`
	Duration     float64     `json:"duration"`
	DateTime     *time.Time  `json:"dateTime,omitempty"`
	StartRange   *int64      `json:"startRange,omitempty"`
	ExpectLength *int64      `json:"expectLength,omitempty"`
	Encrypt      EncryptInfo `json:"encrypt"`
}

func (s Segment) IsEncrypted() bool {
	return s.Encrypt.Method != "" && s.Encrypt.Method != EncryptNone
}

type MediaPart struct {
	Segments []Segment `json:"segments"`
}

type Playlist struct {
	MediaInit         *Segment    `json:"mediaInit,omitempty"`
	Parts             []MediaPart `json:"parts"`
	TargetDuration    float64     `json:"targetDuration,omitempty"`
	IsLive            bool        `json:"isLive"`
	WasLive           bool        `json:"wasLive,omitempty"`
	RefreshIntervalMS int         `json:"refreshIntervalMs,omitempty"`
}

type StreamSpec struct {
	ID              int        `json:"id"`
	URL             string     `json:"url"`
	OriginalURL     string     `json:"originalUrl,omitempty"`
	MediaType       *MediaType `json:"mediaType,omitempty"`
	GroupID         string     `json:"groupId,omitempty"`
	Name            string     `json:"name,omitempty"`
	Language        string     `json:"language,omitempty"`
	Codecs          string     `json:"codecs,omitempty"`
	Resolution      string     `json:"resolution,omitempty"`
	Bandwidth       int        `json:"bandwidth,omitempty"`
	FrameRate       float64    `json:"frameRate,omitempty"`
	AudioID         string     `json:"audioId,omitempty"`
	VideoID         string     `json:"videoId,omitempty"`
	SubtitleID      string     `json:"subtitleId,omitempty"`
	VideoRange      string     `json:"videoRange,omitempty"`
	Channels        string     `json:"channels,omitempty"`
	Characteristics string     `json:"characteristics,omitempty"`
	Default         bool       `json:"default"`
	Extension       string     `json:"extension,omitempty"`
	Playlist        *Playlist  `json:"playlist,omitempty"`
	SkippedDuration float64    `json:"skippedDuration,omitempty"`
}

func (s StreamSpec) Short() string {
	mt := "VIDEO"
	if s.MediaType != nil {
		mt = string(*s.MediaType)
	}
	label := s.Resolution
	if label == "" {
		label = s.Name
	}
	if label == "" {
		label = s.URL
	}
	return mt + " " + label
}

type CustomRange struct {
	Raw      string
	StartSeg *int64
	EndSeg   *int64
	StartSec *float64
	EndSec   *float64
}

type Filter struct {
	For          string
	GroupID      string
	Language     string
	Name         string
	Codecs       string
	Resolution   string
	FrameRate    string
	Channels     string
	VideoRange   string
	URL          string
	SegmentsMin  *int64
	SegmentsMax  *int64
	PlaylistMin  *float64
	PlaylistMax  *float64
	BandwidthMin *int
	BandwidthMax *int
}

type MuxOptions struct {
	Format       string
	Muxer        string
	BinPath      string
	Keep         bool
	SkipSubtitle bool
}

type Options struct {
	Input                  string
	TmpDir                 string
	SaveDir                string
	SaveName               string
	SavePattern            string
	LogFilePath            string
	BaseURL                string
	ThreadCount            int
	DownloadRetryCount     int
	HTTPRequestTimeout     float64
	ForceANSIConsole       bool
	NoANSIColor            bool
	AutoSelect             bool
	SkipMerge              bool
	SkipDownload           bool
	CheckSegmentsCount     bool
	BinaryMerge            bool
	UseFFmpegConcatDemuxer bool
	DelAfterDone           bool
	NoDateInfo             bool
	NoLog                  bool
	WriteMetaJSON          bool
	AppendURLParams        bool
	ConcurrentDownload     bool
	Headers                map[string]string
	SubOnly                bool
	SubFormat              string
	AutoSubtitleFix        bool
	FFmpegBinaryPath       string
	LogLevel               string
	UILanguage             string
	URLProcessorArgs       string
	Keys                   []string
	KeyTextFile            string
	DecryptionEngine       string
	DecryptionBinaryPath   string
	MP4RealTimeDecryption  bool
	CustomHLSMethod        EncryptMethod
	CustomHLSKey           []byte
	CustomHLSIV            []byte
	UseSystemProxy         bool
	CustomProxy            string
	CustomRange            *CustomRange
	TaskStartAt            *time.Time
	LivePerformAsVOD       bool
	LiveRealTimeMerge      bool
	LiveKeepSegments       bool
	LivePipeMux            bool
	LiveRecordLimit        *time.Duration
	LiveWaitTime           *int
	LiveTakeCount          int
	LiveFixVTTByAudio      bool
	MuxAfterDone           *MuxOptions
	MuxImports             []string
	VideoFilter            *Filter
	AudioFilter            *Filter
	SubtitleFilter         *Filter
	DropVideoFilter        *Filter
	DropAudioFilter        *Filter
	DropSubtitleFilter     *Filter
	AdKeywords             []string
	DisableUpdateCheck     bool
	AllowHLSMultiExtMap    bool
	MaxSpeed               int64
}
