package main

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type mediaInfo struct {
	Type        string
	CodecName   string
	CodecTag    string
	StartTime   time.Duration
	DolbyVision bool
}

type ffprobeStreams struct {
	Streams []struct {
		CodecType      string `json:"codec_type"`
		CodecName      string `json:"codec_name"`
		CodecTagString string `json:"codec_tag_string"`
		StartTime      string `json:"start_time"`
	} `json:"streams"`
}

func probeMediaInfo(path string, opt Options) []mediaInfo {
	bin := ffprobeBinary(opt)
	if bin == "" || path == "" {
		return nil
	}
	cmd := exec.Command(bin, "-v", "error", "-show_entries", "stream=codec_type,codec_name,codec_tag_string,start_time", "-of", "json", path)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var parsed ffprobeStreams
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil
	}
	var infos []mediaInfo
	for _, stream := range parsed.Streams {
		info := mediaInfo{
			Type:      strings.ToLower(stream.CodecType),
			CodecName: strings.ToLower(stream.CodecName),
			CodecTag:  strings.ToLower(stream.CodecTagString),
			StartTime: parseMediaStartTime(stream.StartTime),
		}
		info.DolbyVision = strings.Contains(info.CodecName, "dvhe") ||
			strings.Contains(info.CodecName, "dvh1") ||
			strings.Contains(info.CodecTag, "dvhe") ||
			strings.Contains(info.CodecTag, "dvh1")
		infos = append(infos, info)
	}
	return infos
}

func parseMediaStartTime(raw string) time.Duration {
	if raw == "" || raw == "N/A" {
		return 0
	}
	seconds, err := strconv.ParseFloat(raw, 64)
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds * float64(time.Second))
}

func mediaInfosAudioStart(infos []mediaInfo) (time.Duration, bool) {
	for _, info := range infos {
		if info.Type == "audio" && info.StartTime > 0 {
			return info.StartTime, true
		}
	}
	return 0, false
}

func applyMediaInfoToStream(s *StreamSpec, opt *Options, infos []mediaInfo) {
	if len(infos) == 0 {
		return
	}
	if anyDolbyVision(infos) {
		// long: Dolby Vision 片段在上游会强制二进制合并，避免普通 ffmpeg copy 破坏 DV 元数据或产生不可用输出。
		opt.BinaryMerge = true
		opt.MuxAfterDone = nil
	}
	if allMediaInfoType(infos, "audio") {
		audio := MediaAudio
		s.MediaType = &audio
		return
	}
	if allMediaInfoType(infos, "subtitle") {
		sub := MediaSubtitles
		s.MediaType = &sub
		if s.Extension == "" || strings.EqualFold(s.Extension, "ts") {
			// long: 有些 HLS 字幕 playlist 仍写 ts 扩展；真实媒体信息为字幕时，上游会改成 vtt 走字幕修复输出。
			s.Extension = "vtt"
		}
	}
}

func mediaInfosUseAACFilter(infos []mediaInfo) bool {
	hasAudio := false
	for _, info := range infos {
		if info.Type != "audio" {
			continue
		}
		hasAudio = true
		if !strings.Contains(info.CodecName, "aac") {
			return false
		}
	}
	// long: 上游只有在 ffmpeg 探测出的音频流都是 AAC 时才添加 aac_adtstoasc，避免 EAC3/AC3 等音轨被错误套用 AAC bitstream filter。
	return hasAudio
}

func anyDolbyVision(infos []mediaInfo) bool {
	for _, info := range infos {
		if info.DolbyVision {
			return true
		}
	}
	return false
}

func allMediaInfoType(infos []mediaInfo, typ string) bool {
	if len(infos) == 0 {
		return false
	}
	for _, info := range infos {
		if info.Type != typ {
			return false
		}
	}
	return true
}
