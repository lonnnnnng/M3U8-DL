package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func chooseStreams(streams []StreamSpec, opt Options) ([]StreamSpec, error) {
	streams = sortStreamsLikeUpstream(streams)
	streams = applyDrops(streams, opt)
	if opt.AutoSelect {
		if len(streams) == 0 {
			return nil, fmt.Errorf("没有可用轨道")
		}
		autoOpt := opt
		autoOpt.SubOnly = false
		return autoSelect(streams, autoOpt), nil
	}
	if opt.SubOnly {
		var subs []StreamSpec
		for _, s := range streams {
			if s.MediaType != nil && *s.MediaType == MediaSubtitles {
				subs = append(subs, s)
			}
		}
		if len(subs) == 0 {
			return nil, fmt.Errorf("没有可用轨道")
		}
		return subs, nil
	} else if hasKeepFilters(opt) {
		streams = applyFilters(streams, opt)
		if len(streams) == 0 {
			return nil, fmt.Errorf("没有可用轨道")
		}
		return streams, nil
	}
	if len(streams) == 1 {
		return streams, nil
	}
	for _, s := range streams {
		fmt.Printf("[%d] %s bw=%d codec=%s lang=%s name=%s url=%s\n", s.ID, s.Short(), s.Bandwidth, s.Codecs, s.Language, s.Name, s.URL)
	}
	fmt.Print("请选择轨道编号，多个用逗号分隔，直接回车选择默认轨道: ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultInteractiveSelection(streams), nil
	}
	ids := map[int]bool{}
	for _, p := range strings.Split(line, ",") {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err == nil {
			ids[v] = true
		}
	}
	var out []StreamSpec
	for _, s := range streams {
		if ids[s.ID] {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("没有匹配所选编号的轨道")
	}
	return out, nil
}

func defaultInteractiveSelection(streams []StreamSpec) []StreamSpec {
	if len(streams) == 0 {
		return nil
	}
	first := streams[0]
	selected := map[string]bool{streamKey(first): true}
	if first.AudioID != "" {
		for _, s := range streams {
			if s.MediaType != nil && *s.MediaType == MediaAudio && s.GroupID == first.AudioID {
				// long: 上游交互列表会默认勾选首个视频声明的 AUDIO 组，避免直接回车只下载视频而丢掉主音轨。
				selected[streamKey(s)] = true
				break
			}
		}
	}
	if first.SubtitleID != "" {
		for _, s := range streams {
			if s.MediaType != nil && *s.MediaType == MediaSubtitles && s.GroupID == first.SubtitleID {
				// long: HLS master 可通过 SUBTITLES 指出默认字幕组，交互模式直接回车时应复刻上游默认勾选。
				selected[streamKey(s)] = true
				break
			}
		}
	}
	var out []StreamSpec
	for _, s := range streams {
		if selected[streamKey(s)] {
			out = append(out, s)
		}
	}
	return out
}

func filterStreams(streams []StreamSpec, opt Options) []StreamSpec {
	streams = sortStreamsLikeUpstream(streams)
	// long: 上游先执行丢弃过滤，再执行选择过滤；两者同时存在时顺序会改变最终轨道集合。
	return applyFilters(applyDrops(streams, opt), opt)
}

func hasKeepFilters(opt Options) bool {
	return opt.VideoFilter != nil || opt.AudioFilter != nil || opt.SubtitleFilter != nil
}

func autoSelect(streams []StreamSpec, opt Options) []StreamSpec {
	streams = sortStreamsLikeUpstream(streams)
	if opt.SubOnly {
		return bestByType(streams, MediaSubtitles, true)
	}
	var selected []StreamSpec
	videos := bestByType(streams, MediaVideo, false)
	if len(videos) > 0 {
		selected = append(selected, videos[0])
	}
	selected = append(selected, bestAudioPerLanguage(streams)...)
	selected = append(selected, bestByType(streams, MediaSubtitles, true)...)
	if len(selected) > 0 {
		return selected
	}
	return []StreamSpec{bestByBandwidth(streams)}
}

func bestAudioPerLanguage(streams []StreamSpec) []StreamSpec {
	var out []StreamSpec
	seen := map[string]bool{}
	for _, s := range streams {
		if s.MediaType == nil || *s.MediaType != MediaAudio || seen[s.Language] {
			continue
		}
		seen[s.Language] = true
		var sameLang []StreamSpec
		for _, candidate := range streams {
			if candidate.MediaType != nil && *candidate.MediaType == MediaAudio && candidate.Language == s.Language {
				sameLang = append(sameLang, candidate)
			}
		}
		// long: 上游自动选择会按语言各取最高码率音轨，避免多语言音频在自动模式下被静默丢弃。
		out = append(out, bestByBandwidth(sameLang))
	}
	return out
}

func bestByType(streams []StreamSpec, mt MediaType, all bool) []StreamSpec {
	var items []StreamSpec
	for _, s := range streams {
		if s.MediaType == nil && mt == MediaVideo {
			items = append(items, s)
		}
		if s.MediaType != nil && *s.MediaType == mt {
			items = append(items, s)
		}
	}
	items = sortStreamsLikeUpstream(items)
	if all || len(items) <= 1 {
		return items
	}
	return items[:1]
}

func bestByBandwidth(streams []StreamSpec) StreamSpec {
	streams = sortStreamsByQuality(streams)
	return streams[0]
}

func filterGroup(streams []StreamSpec, mt MediaType, group string) []StreamSpec {
	var out []StreamSpec
	for _, s := range streams {
		if s.MediaType != nil && *s.MediaType == mt && s.GroupID == group {
			out = append(out, s)
		}
	}
	return out
}

func applyFilters(streams []StreamSpec, opt Options) []StreamSpec {
	streams = sortStreamsLikeUpstream(streams)
	if opt.VideoFilter == nil && opt.AudioFilter == nil && opt.SubtitleFilter == nil {
		return streams
	}
	var out []StreamSpec
	out = append(out, applyFilterKeepByType(streams, MediaVideo, opt.VideoFilter)...)
	out = append(out, applyFilterKeepByType(streams, MediaAudio, opt.AudioFilter)...)
	out = append(out, applyFilterKeepByType(streams, MediaSubtitles, opt.SubtitleFilter)...)
	return out
}

func applyFilterKeepByType(streams []StreamSpec, mt MediaType, f *Filter) []StreamSpec {
	if f == nil {
		return nil
	}
	var candidates []StreamSpec
	for _, s := range streams {
		if isMediaType(s, mt) {
			candidates = append(candidates, s)
		}
	}
	candidates = sortStreamsLikeUpstream(candidates)
	candidates = filterByPredicate(candidates, func(s StreamSpec) bool {
		return matchFilterRegexFields(s, *f)
	})
	if hasSegmentCountFilter(*f) && allHaveSegments(candidates) {
		candidates = filterByPredicate(candidates, func(s StreamSpec) bool {
			return checkSegmentsCount(s, *f)
		})
	}
	candidates = filterByPredicate(candidates, func(s StreamSpec) bool {
		return checkPlaylistDuration(s, *f) && checkBandwidth(s, *f) && checkRole(s, *f)
	})
	return applyFilterFor(candidates, f.For)
}

func applyFilterFor(streams []StreamSpec, forValue string) []StreamSpec {
	if len(streams) == 0 || forValue == "" || forValue == "all" {
		return streams
	}
	if forValue == "best" {
		return streams[:1]
	}
	if forValue == "worst" {
		return streams[len(streams)-1:]
	}
	if strings.HasPrefix(forValue, "best") {
		if n, err := strconv.Atoi(strings.TrimPrefix(forValue, "best")); err == nil && n < len(streams) {
			return streams[:n]
		}
	}
	if strings.HasPrefix(forValue, "worst") {
		if n, err := strconv.Atoi(strings.TrimPrefix(forValue, "worst")); err == nil && n < len(streams) {
			return streams[len(streams)-n:]
		}
	}
	return streams
}

func applyDrops(streams []StreamSpec, opt Options) []StreamSpec {
	streams = sortStreamsLikeUpstream(streams)
	if opt.DropVideoFilter == nil && opt.DropAudioFilter == nil && opt.DropSubtitleFilter == nil {
		return streams
	}
	drops := append([]StreamSpec{}, applyFilterKeepByType(streams, MediaVideo, opt.DropVideoFilter)...)
	drops = append(drops, applyFilterKeepByType(streams, MediaAudio, opt.DropAudioFilter)...)
	drops = append(drops, applyFilterKeepByType(streams, MediaSubtitles, opt.DropSubtitleFilter)...)
	dropSet := map[string]bool{}
	for _, s := range drops {
		dropSet[streamKey(s)] = true
	}
	var out []StreamSpec
	for _, s := range streams {
		if dropSet[streamKey(s)] {
			continue
		}
		out = append(out, s)
	}
	return out
}

func sortStreamsLikeUpstream(streams []StreamSpec) []StreamSpec {
	out := append([]StreamSpec(nil), streams...)
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := mediaSortRank(out[i]), mediaSortRank(out[j])
		if ri != rj {
			return ri < rj
		}
		return betterQuality(out[i], out[j])
	})
	return out
}

func sortStreamsByQuality(streams []StreamSpec) []StreamSpec {
	out := append([]StreamSpec(nil), streams...)
	sort.SliceStable(out, func(i, j int) bool {
		return betterQuality(out[i], out[j])
	})
	return out
}

func betterQuality(a, b StreamSpec) bool {
	if a.Bandwidth != b.Bandwidth {
		return a.Bandwidth > b.Bandwidth
	}
	// long: 上游同码率音轨会继续按 CHANNELS 的首个数字排序，自动选择时应优先保留 5.1/Atmos 这类声道数更多的版本。
	return channelOrder(a.Channels) > channelOrder(b.Channels)
}

func mediaSortRank(s StreamSpec) int {
	if s.MediaType == nil {
		return 0
	}
	switch *s.MediaType {
	case MediaAudio:
		return 1
	case MediaVideo:
		// long: 原版按 nullable MediaType 排序，基础流(null)永远排在 EXT-X-MEDIA:TYPE=VIDEO 前面，不能和备用视频 rendition 按码率混排。
		return 2
	case MediaSubtitles:
		return 3
	default:
		return 4
	}
}

func channelOrder(channels string) int {
	if channels == "" {
		return 0
	}
	first := strings.Split(channels, "/")[0]
	first = strings.TrimSpace(first)
	order, err := strconv.Atoi(first)
	if err != nil {
		return 0
	}
	return order
}

func streamKey(s StreamSpec) string {
	mt := "VIDEO"
	if s.MediaType != nil {
		mt = string(*s.MediaType)
	}
	return fmt.Sprintf("%s|%d|%s|%s|%s|%d", mt, s.ID, s.URL, s.Language, s.Name, s.Bandwidth)
}

func matchFilter(s StreamSpec, f Filter) bool {
	return matchFilterRegexFields(s, f) &&
		checkBandwidth(s, f) &&
		checkSegmentsCount(s, f) &&
		checkPlaylistDuration(s, f) &&
		checkRole(s, f)
}

func matchFilterRegexFields(s StreamSpec, f Filter) bool {
	check := func(pattern, value string) bool {
		if pattern == "" {
			return true
		}
		ok, err := regexp.MatchString(pattern, value)
		return err == nil && ok
	}
	return check(f.GroupID, s.GroupID) &&
		check(f.Language, s.Language) &&
		check(f.Name, s.Name) &&
		check(f.Codecs, s.Codecs) &&
		check(f.Resolution, s.Resolution) &&
		check(f.FrameRate, formatFloat(s.FrameRate)) &&
		check(f.Channels, s.Channels) &&
		check(f.VideoRange, s.VideoRange) &&
		check(f.URL, s.URL)
}

func filterByPredicate(streams []StreamSpec, keep func(StreamSpec) bool) []StreamSpec {
	var out []StreamSpec
	for _, s := range streams {
		if keep(s) {
			out = append(out, s)
		}
	}
	return out
}

func hasSegmentCountFilter(f Filter) bool {
	return f.SegmentsMin != nil || f.SegmentsMax != nil
}

func allHaveSegments(streams []StreamSpec) bool {
	if len(streams) == 0 {
		return false
	}
	for _, s := range streams {
		if len(sortedSegments(s.Playlist)) == 0 {
			return false
		}
	}
	return true
}

func isMediaType(s StreamSpec, mt MediaType) bool {
	if s.MediaType == nil {
		return mt == MediaVideo
	}
	return *s.MediaType == mt
}

func checkBandwidth(s StreamSpec, f Filter) bool {
	if f.BandwidthMin != nil && s.Bandwidth < *f.BandwidthMin {
		return false
	}
	if f.BandwidthMax != nil && s.Bandwidth > *f.BandwidthMax {
		return false
	}
	return true
}

func checkSegmentsCount(s StreamSpec, f Filter) bool {
	count := int64(len(sortedSegments(s.Playlist)))
	if f.SegmentsMin != nil && count <= *f.SegmentsMin {
		return false
	}
	if f.SegmentsMax != nil && count >= *f.SegmentsMax {
		return false
	}
	return true
}

func checkPlaylistDuration(s StreamSpec, f Filter) bool {
	duration := playlistDuration(s.Playlist)
	if f.PlaylistMin != nil && duration <= *f.PlaylistMin {
		return false
	}
	if f.PlaylistMax != nil && duration >= *f.PlaylistMax {
		return false
	}
	return true
}

func checkRole(s StreamSpec, f Filter) bool {
	if f.Role == "" {
		return true
	}
	return strings.EqualFold(s.Role, f.Role)
}

func playlistDuration(pl *Playlist) float64 {
	var total float64
	if pl == nil {
		return 0
	}
	for _, seg := range sortedSegments(pl) {
		total += seg.Duration
	}
	return total
}

func applyCustomRange(s *StreamSpec, cr *CustomRange) {
	if cr == nil || s.Playlist == nil {
		return
	}
	var skipped float64
	var elapsed float64
	for pi := range s.Playlist.Parts {
		var kept []Segment
		var partSkippedBeforeFirstKept float64
		foundFirstKept := false
		for _, seg := range s.Playlist.Parts[pi].Segments {
			keep := true
			if cr.StartSeg != nil && seg.Index < *cr.StartSeg {
				keep = false
			}
			if cr.EndSeg != nil && seg.Index > *cr.EndSeg {
				keep = false
			}
			if cr.StartSec != nil || cr.EndSec != nil {
				start := elapsed
				if cr.StartSec != nil && start < *cr.StartSec {
					keep = false
				}
				if cr.EndSec != nil && start > *cr.EndSec {
					keep = false
				}
			}
			if keep {
				foundFirstKept = true
				kept = append(kept, seg)
			} else if !foundFirstKept {
				// long: 字幕修正只需要扣除范围起点前的时长，上游不会把范围结束后的丢弃分片计入 SkippedDuration。
				partSkippedBeforeFirstKept += seg.Duration
			}
			elapsed += seg.Duration
		}
		if len(kept) > 0 {
			skipped += partSkippedBeforeFirstKept
		}
		s.Playlist.Parts[pi].Segments = kept
	}
	s.SkippedDuration = skipped
}
