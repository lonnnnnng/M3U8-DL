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
		info.DolbyVision = isDolbyVisionMediaInfo(info)
		infos = append(infos, info)
	}
	return infos
}

func isDolbyVisionMediaInfo(info mediaInfo) bool {
	haystacks := []string{info.Type, info.CodecName, info.CodecTag}
	for _, text := range haystacks {
		if strings.Contains(text, "dvhe") ||
			strings.Contains(text, "dvh1") ||
			strings.Contains(text, "dovi") ||
			strings.Contains(text, "dvvideo") {
			// long: 原版从 ffmpeg stderr 的 BaseInfo/Type 中识别 dvhe、dvh1、DOVI 和 dvvideo；Go 版用 ffprobe JSON 时同样要覆盖这些别名。
			return true
		}
	}
	return false
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
	for _, info := range infos {
		if info.Type != "audio" {
			continue
		}
		if !strings.Contains(info.CodecName, "aac") {
			return false
		}
	}
	// long: 原版使用 LINQ All 判断“所有音频流都是 AAC”；没有音频流时 All 也为 true，所以纯视频或空探测结果也会沿用 aac_adtstoasc 参数。
	return true
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
