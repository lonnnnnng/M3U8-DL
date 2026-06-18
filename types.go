package main

import (
	"fmt"
	"strings"
	"time"
)

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
	Role            string     `json:"role,omitempty"`
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

func (s StreamSpec) DisplayString() string {
	prefix, fields := s.displayPrefixAndFields()
	prefix = strings.TrimSpace(prefix)
	body := strings.Join(compactDisplayFields(fields), " | ")
	if body == "" {
		if s.Playlist != nil {
			body = "~" + formatUpstreamSeconds(int(playlistDuration(s.Playlist)))
		}
		if body == "" {
			return prefix
		}
		return strings.TrimSpace(prefix + " " + body)
	}
	if s.Playlist != nil {
		body += " | ~" + formatUpstreamSeconds(int(playlistDuration(s.Playlist)))
	}
	return strings.TrimSpace(prefix + " " + body)
}

func (s StreamSpec) displayPrefixAndFields() (string, []string) {
	enc := encryptedDisplayPrefix(s.Playlist)
	if s.MediaType != nil && *s.MediaType == MediaAudio {
		channels := ""
		if s.Channels != "" {
			channels = s.Channels + "CH"
		}
		return "Aud " + enc, []string{
			s.GroupID,
			bandwidthKbps(s.Bandwidth, false),
			s.Name,
			s.Codecs,
			s.Language,
			channels,
			segmentsCountText(s.Playlist),
			s.Role,
		}
	}
	if s.MediaType != nil && *s.MediaType == MediaSubtitles {
		return "Sub " + enc, []string{
			s.GroupID,
			s.Language,
			s.Name,
			s.Codecs,
			s.Characteristics,
			segmentsCountText(s.Playlist),
			s.Role,
		}
	}
	return "Vid " + enc, []string{
		s.Resolution,
		bandwidthKbps(s.Bandwidth, true),
		s.GroupID,
		floatDisplay(s.FrameRate),
		s.Codecs,
		s.VideoRange,
		segmentsCountText(s.Playlist),
		s.Role,
	}
}

func encryptedDisplayPrefix(pl *Playlist) string {
	if pl == nil {
		return ""
	}
	seen := map[EncryptMethod]bool{}
	var methods []string
	for _, part := range pl.Parts {
		for _, seg := range part.Segments {
			method := seg.Encrypt.Method
			if method == "" || method == EncryptNone || seen[method] {
				continue
			}
			seen[method] = true
			methods = append(methods, string(method))
		}
	}
	if len(methods) == 0 {
		return ""
	}
	return "*" + strings.Join(methods, ",") + " "
}

func compactDisplayFields(fields []string) []string {
	var out []string
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}

func bandwidthKbps(value int, keepZero bool) string {
	if value == 0 && !keepZero {
		return ""
	}
	return fmt.Sprintf("%d Kbps", value/1000)
}

func floatDisplay(value float64) string {
	if value == 0 {
		return ""
	}
	return fmt.Sprintf("%g", value)
}

func segmentsCountText(pl *Playlist) string {
	count := playlistMediaSegmentCount(pl)
	if count == 0 {
		return ""
	}
	if count == 1 {
		return "1 Segment"
	}
	return fmt.Sprintf("%d Segments", count)
}

func playlistMediaSegmentCount(pl *Playlist) int {
	if pl == nil {
		return 0
	}
	total := 0
	for _, part := range pl.Parts {
		total += len(part.Segments)
	}
	return total
}

func formatUpstreamSeconds(total int) string {
	d := time.Duration(total) * time.Second
	hours := int(d / time.Hour)
	minutes := int((d % time.Hour) / time.Minute)
	seconds := int((d % time.Minute) / time.Second)
	if hours == 0 {
		return fmt.Sprintf("%02dm%02ds", minutes, seconds)
	}
	return fmt.Sprintf("%02dh%02dm%02ds", hours, minutes, seconds)
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
	Role         string
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
