package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const version = "N_m3u8DL-GO-HLS 0.1.0"

func main() {
	if err := run(); err != nil {
		if err.Error() == "help" {
			fmt.Print(usage())
			return
		}
		if err.Error() == "version" {
			fmt.Println(version)
			return
		}
		if strings.HasPrefix(err.Error(), "morehelp:") {
			fmt.Print(moreHelp(strings.TrimPrefix(err.Error(), "morehelp:")))
			return
		}
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}

func run() error {
	opt, err := parseArgs(os.Args[1:])
	if err != nil {
		return err
	}
	consoleRedirected := applyConsoleRedirectDefaults(&opt, os.Stdout, os.Stderr)
	cleanupLog, _, err := setupLogging(opt, os.Args)
	if err != nil {
		return err
	}
	defer func() { _ = cleanupLog() }()
	if consoleRedirected {
		fmt.Println(tr(opt, "consoleRedirected"))
	}
	fmt.Println(version)
	maybeCheckUpdate(opt)
	if err := validateOptions(opt); err != nil {
		return err
	}
	for _, msg := range applyOptionImplicationsWithMessages(&opt) {
		fmt.Println(msg)
	}
	waitForTaskStart(opt, time.Now, time.Sleep, func(msg string) { fmt.Println(msg) })
	applyDerivedDefaults(&opt, time.Now())
	client, err := newHTTPClient(opt)
	if err != nil {
		return err
	}
	ctx := context.Background()
	streams, p, err := parseSource(ctx, client, opt)
	if err != nil {
		return err
	}
	fmt.Println(tr(opt, "streamsParsed", len(streams)))
	selected, err := chooseStreams(streams, opt)
	if err != nil {
		return err
	}
	for i := range selected {
		if selected[i].Playlist == nil {
			if err := p.fetchPlaylist(ctx, &selected[i]); err != nil {
				return err
			}
		}
	}
	for _, msg := range prepareSelectedStreams(selected, &opt) {
		fmt.Println(msg)
	}
	if opt.WriteMetaJSON {
		if err := writeMeta(opt, p, streams, selected); err != nil {
			return err
		}
	}
	if opt.SkipDownload {
		fmt.Println(tr(opt, "skipDownload"))
		return nil
	}
	outs, liveHandled, err := downloadLiveRealtimeIfNeeded(ctx, client, selected, p, opt)
	if err != nil {
		return err
	}
	if !liveHandled {
		if err := recordLiveIfNeeded(ctx, client, selected, p, opt); err != nil {
			return err
		}
		outs, err = downloadAll(ctx, client, selected, opt)
		if err != nil {
			return err
		}
	}
	if opt.LivePipeMux && !opt.SkipMerge && !liveHandled {
		outs, err = runLivePipeMuxOutputs(outs, opt)
		if err != nil {
			return err
		}
	}
	if err := cleanupRawMetaAfterDownload(opt, p); err != nil {
		return err
	}
	if shouldMuxAfterDownload(opt, outs) {
		muxed, err := muxOutputs(outs, opt)
		if err != nil {
			return err
		}
		if muxed != "" {
			fmt.Println(tr(opt, "muxOutput", muxed))
		}
	}
	for _, o := range outs {
		fmt.Println(tr(opt, "output", o.Path))
	}
	return nil
}

func waitForTaskStart(opt Options, now func() time.Time, sleep func(time.Duration), announce func(string)) {
	if opt.TaskStartAt == nil {
		return
	}
	current := now()
	if !opt.TaskStartAt.After(current) {
		return
	}
	if announce != nil {
		announce(tr(opt, "taskStartAt", opt.TaskStartAt.Format("2006-01-02 15:04:05")))
	}
	// long: 默认保存名包含时间戳；等待必须发生在派生 SaveName 之前，否则定时任务跨分钟/跨天时文件名会记录排队时间而不是实际开始时间。
	sleep(opt.TaskStartAt.Sub(current))
}

func shouldMuxAfterDownload(opt Options, outs []outputFile) bool {
	if opt.MuxAfterDone == nil || len(outs) == 0 {
		return false
	}
	if opt.LivePipeMux {
		// long: 直播 pipe mux 本身已经把非字幕轨道写成 TS；上游不会再把这个 pipe mux 结果塞进普通最终混流队列。
		return false
	}
	// long: 上游只有单轨合并产物真实存在时才加入最终混流；--skip-merge 留下的是分片目录，不能当媒体文件喂给 ffmpeg/mkvmerge。
	return !opt.SkipMerge
}

func downloadLiveRealtimeIfNeeded(ctx context.Context, client *http.Client, selected []StreamSpec, p *parser, opt Options) ([]outputFile, bool, error) {
	if opt.LivePerformAsVOD || opt.LiveRecordLimit == nil || !opt.LiveRealTimeMerge || opt.SkipMerge || !hasLiveStream(selected) {
		return nil, false, nil
	}
	for i := range selected {
		trimLiveInitial(&selected[i], opt.LiveTakeCount)
	}
	syncLiveStreams(selected, opt.LiveTakeCount)

	limiter := newRateLimiter(opt.MaxSpeed)
	states := make([]*liveRealtimeDownloadState, len(selected))
	outs := make([]outputFile, len(selected))
	ok := make([]bool, len(selected))
	var pipeStateIndexes []int
	for i := range selected {
		stream := selected[i]
		if stream.Playlist == nil {
			continue
		}
		if stream.Playlist.IsLive && (stream.MediaType == nil || *stream.MediaType != MediaSubtitles) {
			state, out, err := newLiveRealtimeDownloadState(client, streamForTask(stream, i), opt, limiter)
			if err != nil {
				return nil, true, err
			}
			states[i] = state
			outs[i] = out
			if opt.LivePipeMux {
				pipeStateIndexes = append(pipeStateIndexes, i)
			} else {
				ok[i] = true
			}
			continue
		}
		if stream.Playlist.IsLive {
			continue
		}
		out, err := downloadStream(ctx, client, streamForTask(stream, i), opt, limiter, nil)
		if err != nil {
			return nil, true, err
		}
		outs[i] = out
		ok[i] = true
	}
	var pipeSession *livePipeSession
	var pipeOutput *outputFile
	if len(pipeStateIndexes) > 0 {
		baseOutput := outs[pipeStateIndexes[0]].Path
		var err error
		pipeSession, err = prepareLivePipeMux(opt.FFmpegBinaryPath, len(pipeStateIndexes), baseOutput, currentLivePipeEnv())
		if err != nil {
			return nil, true, err
		}
		for pipeIndex, stateIndex := range pipeStateIndexes {
			states[stateIndex].pipe = pipeSession.Pipes[pipeIndex]
			states[stateIndex].output = pipeSession.OutputPath
		}
	}
	for _, stateIndex := range pipeStateIndexes {
		ok[stateIndex] = false
	}
	for i, state := range states {
		if state == nil {
			continue
		}
		if err := state.downloadAndAppend(ctx, liveFirstPartSegments(selected[i])); err != nil {
			if pipeSession != nil {
				_ = pipeSession.Close()
			}
			return nil, true, err
		}
	}

	refreshedDurations := liveInitialRefreshedDurations(selected)
	wait := liveRefreshWaitDuration(selected, opt)
	deadline := time.Now().Add(*opt.LiveRecordLimit)
	for !liveRecordLimitReached(selected, refreshedDurations, *opt.LiveRecordLimit) && time.Now().Before(deadline) {
		time.Sleep(wait)
		allDone := true
		for i := range selected {
			if selected[i].Playlist == nil || !selected[i].Playlist.IsLive {
				continue
			}
			allDone = false
			next := selected[i]
			if err := p.fetchPlaylist(ctx, &next); err != nil {
				if pipeSession != nil {
					_ = pipeSession.Close()
				}
				return nil, true, err
			}
			added, duration := appendNewLiveSegmentsDetailed(&selected[i], next)
			refreshedDurations[i] += duration
			if states[i] != nil {
				states[i].stream = selected[i]
				if err := states[i].downloadAndAppend(ctx, added); err != nil {
					if pipeSession != nil {
						_ = pipeSession.Close()
					}
					return nil, true, err
				}
			}
		}
		if allDone {
			break
		}
	}
	for i := range selected {
		if selected[i].Playlist != nil {
			selected[i].Playlist.IsLive = false
		}
		if states[i] != nil && opt.DelAfterDone {
			_ = os.RemoveAll(states[i].tmpDir)
		}
	}
	if pipeSession != nil {
		if err := finishOpenLivePipeMuxSession(pipeSession); err != nil {
			return nil, true, err
		}
		pipeOutput = &outputFile{Path: pipeSession.OutputPath}
	}
	for i := range selected {
		if pipeSession != nil && states[i] != nil {
			continue
		}
		if ok[i] || selected[i].Playlist == nil {
			continue
		}
		out, err := downloadStream(ctx, client, streamForTask(selected[i], i), opt, limiter, nil)
		if err != nil {
			return nil, true, err
		}
		outs[i] = out
		ok[i] = true
	}
	ordered := make([]outputFile, 0, len(outs)+1)
	if pipeOutput != nil {
		ordered = append(ordered, *pipeOutput)
	}
	for i, out := range outs {
		if ok[i] {
			ordered = append(ordered, out)
		}
	}
	return ordered, true, nil
}

func prepareSelectedStreams(selected []StreamSpec, opt *Options) []string {
	var messages []string
	living := hasLiveStream(selected) && !opt.LivePerformAsVOD
	if !opt.BinaryMerge && hasUnknownEncryption(selected) {
		// long: 未识别加密方式在裁剪前就要触发二进制合并；否则用户范围刚好裁掉未知片段时，会和上游的全局流判断不一致。
		opt.BinaryMerge = true
		messages = append(messages, tr(*opt, "autoBinaryMergeUnknown"))
	}
	if !opt.BinaryMerge && hasCENCEncryption(selected) {
		// long: CENC 分片通常需要保持 init 和媒体分片的原始盒结构，先用二进制合并能为后续外部解密保留完整上下文。
		opt.BinaryMerge = true
		messages = append(messages, tr(*opt, "autoBinaryMergeCENC"))
	}
	if !opt.BinaryMerge && hasFMP4Media(selected) {
		// long: fMP4 HLS 由 init 与 m4s 分片共同组成完整媒体，按上游策略优先顺序拼接，避免 ffmpeg 过早改写片段时间线。
		opt.BinaryMerge = true
		messages = append(messages, tr(*opt, "autoBinaryMerge"))
	}
	if !living {
		for i := range selected {
			applyCustomRange(&selected[i], opt.CustomRange)
		}
	}
	cleanAdSegments(selected, opt.AdKeywords)
	return messages
}

func hasLiveStream(selected []StreamSpec) bool {
	for _, stream := range selected {
		if stream.Playlist != nil && stream.Playlist.IsLive {
			return true
		}
	}
	return false
}

func cleanAdSegments(selected []StreamSpec, keywords []string) {
	if len(keywords) == 0 {
		return
	}
	var regs []*regexp.Regexp
	for _, keyword := range keywords {
		if keyword == "" {
			continue
		}
		regs = append(regs, regexp.MustCompile(keyword))
	}
	if len(regs) == 0 {
		return
	}
	for streamIndex := range selected {
		pl := selected[streamIndex].Playlist
		if pl == nil {
			continue
		}
		var parts []MediaPart
		for _, part := range pl.Parts {
			var kept []Segment
			for _, seg := range part.Segments {
				if !matchesAnyAdKeyword(seg.URL, regs) {
					kept = append(kept, seg)
				}
			}
			if len(kept) > 0 {
				// long: 用户广告关键字是对最终选中轨道做正则清理，空 part 要删除，否则后续分片计数和合并都会把已清空广告段当成有效媒体段。
				parts = append(parts, MediaPart{Segments: kept})
			}
		}
		pl.Parts = parts
	}
}

func matchesAnyAdKeyword(url string, regs []*regexp.Regexp) bool {
	for _, reg := range regs {
		if reg.MatchString(url) {
			return true
		}
	}
	return false
}

func writeMeta(opt Options, p *parser, all []StreamSpec, selected []StreamSpec) error {
	if !opt.WriteMetaJSON {
		return nil
	}
	dir := rawMetaDir(opt)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for name, content := range p.rawFiles {
		if err := writeFileIfAbsent(filepath.Join(dir, name), []byte(content)); err != nil {
			return err
		}
	}
	allJSON, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	if err := writeFileIfAbsent(filepath.Join(dir, "meta.json"), allJSON); err != nil {
		return err
	}
	selectedJSON, err := json.MarshalIndent(selected, "", "  ")
	if err != nil {
		return err
	}
	return writeFileIfAbsent(filepath.Join(dir, "meta_selected.json"), selectedJSON)
}

func rawMetaDir(opt Options) string {
	return taskTempDir(opt)
}

func taskTempDir(opt Options) string {
	root := opt.TmpDir
	if root == "" {
		root = "."
	}
	saveName := opt.SaveName
	if saveName == "" {
		saveName = deriveSaveNameFromInput(opt.Input, time.Now())
	}
	// long: 原版以 SaveName 建立任务临时根目录，raw/meta 和各轨道分片目录都挂在这里，避免多任务共用 --tmp-dir 时互相覆盖。
	return filepath.Join(root, saveName)
}

func cleanupRawMetaAfterDownload(opt Options, p *parser) error {
	if opt.SkipMerge || !opt.DelAfterDone {
		return nil
	}
	return cleanupRawMetaFiles(opt, p)
}

func cleanupRawMetaFiles(opt Options, p *parser) error {
	dir := rawMetaDir(opt)
	names := map[string]struct{}{}
	if opt.WriteMetaJSON {
		names["meta.json"] = struct{}{}
		names["meta_selected.json"] = struct{}{}
	}
	if p != nil {
		for name := range p.rawFiles {
			names[name] = struct{}{}
		}
	}
	for name := range names {
		err := os.Remove(filepath.Join(dir, name))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		// long: 上游 SafeDeleteDir 会沿父级继续清理空目录；非空目录会立刻停止，用户额外放入的排障文件不会被碰到。
		return safeDeleteEmptyParents(dir)
	}
	return nil
}

func safeDeleteEmptyParents(dir string) error {
	if dir == "" {
		return nil
	}
	current := filepath.Clean(dir)
	for {
		entries, err := os.ReadDir(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if len(entries) != 0 {
			return nil
		}
		if err := os.Remove(current); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

func writeFileIfAbsent(path string, data []byte) error {
	_, err := os.Stat(path)
	if err == nil {
		// long: 原版 WriteRawFilesAsync 只补齐缺失的解析结果，重跑任务时要保留用户已经留下的 raw/meta 文件。
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func recordLiveIfNeeded(ctx context.Context, client *http.Client, selected []StreamSpec, p *parser, opt Options) error {
	if opt.LivePerformAsVOD {
		for i := range selected {
			if selected[i].Playlist != nil {
				selected[i].Playlist.IsLive = false
				selected[i].Playlist.WasLive = false
			}
		}
		return nil
	}
	if opt.LiveRecordLimit == nil {
		return nil
	}
	for i := range selected {
		trimLiveInitial(&selected[i], opt.LiveTakeCount)
	}
	syncLiveStreams(selected, opt.LiveTakeCount)
	refreshedDurations := liveInitialRefreshedDurations(selected)
	if liveRecordLimitReached(selected, refreshedDurations, *opt.LiveRecordLimit) {
		for i := range selected {
			if selected[i].Playlist != nil {
				selected[i].Playlist.IsLive = false
			}
		}
		return nil
	}
	wait := liveRefreshWaitDuration(selected, opt)
	deadline := time.Now().Add(*opt.LiveRecordLimit)
	for time.Now().Before(deadline) {
		allDone := true
		for i := range selected {
			if selected[i].Playlist != nil && selected[i].Playlist.IsLive {
				allDone = false
				next := selected[i]
				if err := p.fetchPlaylist(ctx, &next); err != nil {
					return err
				}
				refreshedDurations[i] += appendNewLiveSegments(&selected[i], next)
			}
		}
		if allDone || liveRecordLimitReached(selected, refreshedDurations, *opt.LiveRecordLimit) {
			break
		}
		time.Sleep(wait)
	}
	for i := range selected {
		if selected[i].Playlist != nil {
			selected[i].Playlist.IsLive = false
		}
	}
	return nil
}

func liveInitialRefreshedDurations(selected []StreamSpec) []float64 {
	out := make([]float64, len(selected))
	for i, stream := range selected {
		if stream.Playlist == nil || !stream.Playlist.IsLive {
			continue
		}
		out[i] = liveFirstPartDuration(stream)
	}
	return out
}

func liveFirstPartDuration(stream StreamSpec) float64 {
	if stream.Playlist == nil || len(stream.Playlist.Parts) == 0 {
		return 0
	}
	var dur float64
	for _, seg := range stream.Playlist.Parts[0].Segments {
		dur += seg.Duration
	}
	return dur
}

func liveFirstPartSegments(stream StreamSpec) []Segment {
	if stream.Playlist == nil || len(stream.Playlist.Parts) == 0 {
		return nil
	}
	return append([]Segment(nil), stream.Playlist.Parts[0].Segments...)
}

func liveRecordLimitReached(selected []StreamSpec, refreshed []float64, limit time.Duration) bool {
	if limit <= 0 {
		return true
	}
	limitSec := limit.Seconds()
	hasLive := false
	for i, stream := range selected {
		if stream.Playlist == nil || !stream.Playlist.IsLive {
			continue
		}
		hasLive = true
		if i >= len(refreshed) || refreshed[i] < limitSec {
			return false
		}
	}
	return hasLive
}

func liveRefreshWaitDuration(selected []StreamSpec, opt Options) time.Duration {
	waitSec := 0
	if opt.LiveWaitTime != nil {
		waitSec = *opt.LiveWaitTime
	} else {
		minDur := shortestLiveFirstPartDuration(selected)
		if minDur > 0 {
			// long: 上游用直播窗口总时长的一半再提前两秒刷新，既减少空轮询，也尽量避免等到窗口滑走才抓新分片。
			waitSec = int(minDur/2) - 2
		}
	}
	if waitSec <= 0 {
		waitSec = 1
	}
	return time.Duration(waitSec) * time.Second
}

func shortestLiveFirstPartDuration(selected []StreamSpec) float64 {
	min := 0.0
	for _, stream := range selected {
		if stream.Playlist == nil || !stream.Playlist.IsLive || len(stream.Playlist.Parts) == 0 {
			continue
		}
		dur := 0.0
		for _, seg := range stream.Playlist.Parts[0].Segments {
			dur += seg.Duration
		}
		if dur <= 0 {
			continue
		}
		if min == 0 || dur < min {
			min = dur
		}
	}
	return min
}

func syncLiveStreams(selected []StreamSpec, takeLastCount int) {
	var candidates []*StreamSpec
	for i := range selected {
		if selected[i].Playlist == nil || !selected[i].Playlist.IsLive || !hasAnySegments(selected[i].Playlist) {
			continue
		}
		candidates = append(candidates, &selected[i])
	}
	if len(candidates) == 0 {
		return
	}
	if allLiveSegmentsHaveDate(candidates) {
		start := latestFirstLiveDate(candidates)
		for _, item := range candidates {
			for partIndex := range item.Playlist.Parts {
				item.Playlist.Parts[partIndex].Segments = filterSegmentsByDateSecond(item.Playlist.Parts[partIndex].Segments, start)
			}
		}
	} else {
		start := latestFirstLiveIndex(candidates)
		for _, item := range candidates {
			for partIndex := range item.Playlist.Parts {
				item.Playlist.Parts[partIndex].Segments = filterSegmentsByMinIndex(item.Playlist.Parts[partIndex].Segments, start)
			}
		}
	}
	if takeLastCount <= 0 {
		return
	}
	if !anyFirstPartLongerThan(candidates, takeLastCount) {
		return
	}
	skipCount := shortestFirstPartCount(candidates) - takeLastCount + 1
	if skipCount < 0 {
		skipCount = 0
	}
	for _, item := range candidates {
		for partIndex := range item.Playlist.Parts {
			segs := item.Playlist.Parts[partIndex].Segments
			if skipCount < len(segs) {
				// long: 直播起播时只保留各轨共同拥有的最新窗口，避免视频、音频、字幕从不同时间点开始导致后续混流错位。
				item.Playlist.Parts[partIndex].Segments = segs[skipCount:]
			} else {
				item.Playlist.Parts[partIndex].Segments = nil
			}
		}
	}
}

func hasAnySegments(pl *Playlist) bool {
	for _, part := range pl.Parts {
		if len(part.Segments) > 0 {
			return true
		}
	}
	return false
}

func allLiveSegmentsHaveDate(streams []*StreamSpec) bool {
	for _, s := range streams {
		if len(s.Playlist.Parts) == 0 {
			return false
		}
		for _, seg := range s.Playlist.Parts[0].Segments {
			if seg.DateTime == nil {
				return false
			}
		}
	}
	return true
}

func latestFirstLiveDate(streams []*StreamSpec) time.Time {
	var out time.Time
	for _, s := range streams {
		if len(s.Playlist.Parts) == 0 || len(s.Playlist.Parts[0].Segments) == 0 {
			continue
		}
		minDate := s.Playlist.Parts[0].Segments[0].DateTime
		for _, seg := range s.Playlist.Parts[0].Segments {
			if seg.DateTime != nil && minDate != nil && seg.DateTime.Before(*minDate) {
				minDate = seg.DateTime
			}
		}
		if minDate != nil && minDate.After(out) {
			out = *minDate
		}
	}
	return out
}

func latestFirstLiveIndex(streams []*StreamSpec) int64 {
	var out int64
	set := false
	for _, s := range streams {
		if len(s.Playlist.Parts) == 0 || len(s.Playlist.Parts[0].Segments) == 0 {
			continue
		}
		minIndex := s.Playlist.Parts[0].Segments[0].Index
		for _, seg := range s.Playlist.Parts[0].Segments {
			if seg.Index < minIndex {
				minIndex = seg.Index
			}
		}
		if !set || minIndex > out {
			out = minIndex
			set = true
		}
	}
	return out
}

func filterSegmentsByDateSecond(segs []Segment, start time.Time) []Segment {
	startSecond := start.Unix()
	var out []Segment
	for _, seg := range segs {
		if seg.DateTime != nil && seg.DateTime.Unix() >= startSecond {
			out = append(out, seg)
		}
	}
	return out
}

func filterSegmentsByMinIndex(segs []Segment, start int64) []Segment {
	var out []Segment
	for _, seg := range segs {
		if seg.Index >= start {
			out = append(out, seg)
		}
	}
	return out
}

func anyFirstPartLongerThan(streams []*StreamSpec, count int) bool {
	for _, s := range streams {
		if len(s.Playlist.Parts) > 0 && len(s.Playlist.Parts[0].Segments) > count {
			return true
		}
	}
	return false
}

func shortestFirstPartCount(streams []*StreamSpec) int {
	min := -1
	for _, s := range streams {
		if len(s.Playlist.Parts) == 0 {
			continue
		}
		count := len(s.Playlist.Parts[0].Segments)
		if min == -1 || count < min {
			min = count
		}
	}
	if min < 0 {
		return 0
	}
	return min
}

func trimLiveInitial(s *StreamSpec, takeLast int) {
	if s.Playlist == nil || !s.Playlist.IsLive || takeLast <= 0 || len(s.Playlist.Parts) == 0 {
		return
	}
	segs := s.Playlist.Parts[0].Segments
	if len(segs) > takeLast {
		s.Playlist.Parts[0].Segments = segs[len(segs)-takeLast:]
	}
}

func appendNewLiveSegments(current *StreamSpec, next StreamSpec) float64 {
	_, duration := appendNewLiveSegmentsDetailed(current, next)
	return duration
}

func appendNewLiveSegmentsDetailed(current *StreamSpec, next StreamSpec) ([]Segment, float64) {
	if current.Playlist == nil || next.Playlist == nil || len(next.Playlist.Parts) == 0 {
		return nil, 0
	}
	if len(current.Playlist.Parts) == 0 {
		current.Playlist.Parts = next.Playlist.Parts
		current.Playlist.WasLive = true
		added := liveFirstPartSegments(*current)
		return added, liveFirstPartDuration(*current)
	}
	seen := map[string]bool{}
	for _, seg := range current.Playlist.Parts[0].Segments {
		seen[liveSegmentKey(seg)] = true
	}
	maxIndex := int64(-1)
	for _, seg := range current.Playlist.Parts[0].Segments {
		if seg.Index > maxIndex {
			maxIndex = seg.Index
		}
	}
	var addedSegments []Segment
	var addedDuration float64
	for _, seg := range next.Playlist.Parts[0].Segments {
		key := liveSegmentKey(seg)
		if seen[key] {
			continue
		}
		if seg.Index <= maxIndex {
			maxIndex++
			seg.Index = maxIndex
		}
		current.Playlist.Parts[0].Segments = append(current.Playlist.Parts[0].Segments, seg)
		seen[key] = true
		addedSegments = append(addedSegments, seg)
		addedDuration += seg.Duration
	}
	current.Playlist.IsLive = true
	current.Playlist.WasLive = true
	current.Playlist.RefreshIntervalMS = next.Playlist.RefreshIntervalMS
	return addedSegments, addedDuration
}

func liveSegmentKey(seg Segment) string {
	if seg.DateTime != nil {
		return fmt.Sprintf("%d", seg.DateTime.Unix())
	}
	return fmt.Sprintf("%d", seg.Index)
}
