package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const version = "m3u8dl-go 1.0.5"

type versionInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	FullVersion string `json:"fullVersion"`
}

type downloadSummaryEvent struct {
	Type      string                  `json:"type"`
	Timestamp string                  `json:"timestamp"`
	Status    string                  `json:"status"`
	Outputs   []downloadSummaryOutput `json:"outputs"`
}

type downloadSummaryOutput struct {
	Path        string `json:"path"`
	MediaType   string `json:"mediaType,omitempty"`
	Language    string `json:"language,omitempty"`
	Name        string `json:"name,omitempty"`
	StreamCount int    `json:"streamCount,omitempty"`
	Size        int64  `json:"size,omitempty"`
}

type downloadErrorEvent struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

func main() {
	if err := run(); err != nil {
		var controlErr *cliControlError
		if errors.As(err, &controlErr) {
			switch controlErr.kind {
			case "help":
				fmt.Print(usageWithOptions(controlErr.opt))
				return
			case "version":
				fmt.Println(version)
				return
			case "version-json":
				b, _ := json.Marshal(currentVersionInfo())
				fmt.Println(string(b))
				return
			case "capabilities-json":
				fmt.Print(runCapabilitiesJSON())
				return
			case "doctor":
				fmt.Print(runDoctorText(controlErr.opt))
				return
			case "doctor-json":
				fmt.Print(runDoctorJSON(controlErr.opt))
				return
			case "morehelp":
				fmt.Print(moreHelpWithOptions(controlErr.value, controlErr.opt))
				return
			}
		}
		fmt.Fprintln(os.Stderr, timestampConsoleMessage("错误: "+err.Error(), time.Now()))
		os.Exit(1)
	}
}

func currentVersionInfo() versionInfo {
	return versionInfo{
		Name:        "m3u8dl-go",
		Version:     strings.TrimSpace(strings.TrimPrefix(version, "m3u8dl-go")),
		FullVersion: version,
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return runWithContext(ctx, os.Args[1:], os.Args)
}

func runWithContext(ctx context.Context, args []string, command []string) (err error) {
	opt, err := parseArgs(args)
	if err != nil {
		var controlErr *cliControlError
		if !errors.As(err, &controlErr) && progressJSONRequested(args) {
			emitDownloadErrorJSON(Options{ProgressJSON: true}, err)
		}
		return err
	}
	if opt.PrintEffectiveOptions {
		output, err := runPrintEffectiveOptions(opt, command, time.Now())
		if err != nil {
			return err
		}
		fmt.Print(output)
		return nil
	}
	if opt.ProbeJSON {
		output, err := runProbeJSON(ctx, opt, time.Now())
		if err != nil {
			return err
		}
		fmt.Print(output)
		return nil
	}
	consoleRedirected := applyConsoleRedirectDefaults(&opt, os.Stdout, os.Stderr)
	cleanupLog, _, err := setupLogging(opt, command)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			emitDownloadErrorJSON(opt, err)
		}
		_ = cleanupLog()
	}()
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
	streams, p, err := parseSource(ctx, client, opt)
	if err != nil {
		return err
	}
	basicCount, audioCount, subtitleCount := countStreamGroups(streams)
	fmt.Println(tr(opt, "streamsInfo", len(streams), basicCount, audioCount, subtitleCount))
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
	for _, msg := range selectedStreamMessages(opt, selected) {
		fmt.Println(msg)
	}
	if opt.WriteMetaJSON {
		fmt.Println(tr(opt, "writeJson"))
		if err := writeMeta(opt, p, streams, selected); err != nil {
			return err
		}
	}
	if opt.SkipDownload {
		fmt.Println(tr(opt, "skipDownload"))
		return nil
	}
	fmt.Println(saveNameMessage(opt))
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
	for _, msg := range disableMuxAfterDoneForDolbyVisionOutputs(&opt, outs) {
		fmt.Println(msg)
	}
	var muxed string
	if shouldMuxAfterDownload(opt, outs) {
		muxed, err = muxOutputs(outs, opt)
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
	emitDownloadSummaryJSON(opt, "completed", summaryOutputs(outs, muxed))
	return nil
}

func progressJSONRequested(args []string) bool {
	enabled := false
	for i := 0; i < len(args); i++ {
		if args[i] != "--progress-json" {
			continue
		}
		// long: 参数解析失败时还没有完整 Options；这里复用布尔选项的显式 true/false 语义，只用于决定是否输出机器可读错误。
		enabled = true
		if i+1 >= len(args) {
			continue
		}
		switch strings.ToLower(args[i+1]) {
		case "true":
			enabled = true
			i++
		case "false":
			enabled = false
			i++
		}
	}
	return enabled
}

func summaryOutputs(outs []outputFile, muxed string) []outputFile {
	if strings.TrimSpace(muxed) == "" {
		return outs
	}
	result := make([]outputFile, 0, len(outs)+1)
	result = append(result, outputFile{Path: muxed})
	result = append(result, outs...)
	return result
}

func emitDownloadSummaryJSON(opt Options, status string, outs []outputFile) {
	event := downloadSummaryEvent{
		Type:      "summary",
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Status:    status,
		Outputs:   downloadSummaryOutputs(outs),
	}
	emitRawJSONEvent(opt, event)
}

func emitDownloadErrorJSON(opt Options, err error) {
	if err == nil {
		return
	}
	event := downloadErrorEvent{
		Type:      "error",
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Status:    "failed",
		Message:   err.Error(),
	}
	emitRawJSONEvent(opt, event)
}

func downloadSummaryOutputs(outs []outputFile) []downloadSummaryOutput {
	result := make([]downloadSummaryOutput, 0, len(outs))
	seen := map[string]bool{}
	for _, out := range outs {
		path := strings.TrimSpace(out.Path)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		item := downloadSummaryOutput{
			Path:        path,
			Language:    out.Language,
			Name:        out.Name,
			StreamCount: out.StreamCount,
			Size:        info.Size(),
		}
		if out.MediaType != nil {
			item.MediaType = string(*out.MediaType)
		}
		result = append(result, item)
	}
	return result
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
		announce(tr(opt, "taskStartAt") + opt.TaskStartAt.Format("2006-01-02 15:04:05"))
	}
	// long: 默认保存名包含时间戳；等待必须发生在派生 SaveName 之前，否则定时任务跨分钟/跨天时文件名会记录排队时间而不是实际开始时间。
	sleep(opt.TaskStartAt.Sub(current))
}

func countStreamGroups(streams []StreamSpec) (basic int, audio int, subtitle int) {
	for _, stream := range streams {
		if stream.MediaType == nil {
			basic++
			continue
		}
		switch *stream.MediaType {
		case MediaAudio:
			audio++
		case MediaSubtitles:
			subtitle++
		}
	}
	return basic, audio, subtitle
}

func selectedStreamMessages(opt Options, selected []StreamSpec) []string {
	messages := []string{tr(opt, "selectedStream")}
	for _, stream := range selected {
		messages = append(messages, stream.DisplayString())
	}
	return messages
}

func saveNameMessage(opt Options) string {
	return tr(opt, "saveName") + opt.SaveName
}

func liveRecordLimitMessage(opt Options) string {
	if opt.LiveRecordLimit == nil {
		return ""
	}
	// long: 上游把 TimeSpan.TotalSeconds 转成 int 再格式化，保留这个截断语义可避免小数秒录制上限显示漂移。
	return tr(opt, "liveLimit") + formatUpstreamSeconds(int(opt.LiveRecordLimit.Seconds()))
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
	if opt.LivePerformAsVOD || !opt.LiveRealTimeMerge || opt.SkipMerge || !hasLiveStream(selected) {
		return nil, false, nil
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
		pipeSession, err = prepareLivePipeMux(opt.FFmpegBinaryPath, len(pipeStateIndexes), baseOutput, currentLivePipeEnv(), opt)
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
	limit := liveRecordLimitOrForever(opt.LiveRecordLimit)
	type liveTrackResult struct {
		limitReached bool
		err          error
	}
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	resultCh := make(chan liveTrackResult, len(selected))
	workerCount := 0
	for i := range selected {
		if selected[i].Playlist == nil || !selected[i].Playlist.IsLive {
			continue
		}
		workerCount++
		i := i
		trackParser := p.cloneForLiveTrack()
		wait := liveRefreshWaitDuration([]StreamSpec{selected[i]}, opt)
		go func() {
			// long: 每条直播轨道独立完成首批下载、刷新和追加；慢视频轨不能阻塞音频或字幕继续追赶直播窗口。
			limitReached, err := runLiveRealtimeTrack(workerCtx, &selected[i], states[i], trackParser, opt, wait, limit)
			resultCh <- liveTrackResult{limitReached: limitReached, err: err}
			if err != nil {
				cancelWorkers()
			}
		}()
	}
	var firstErr error
	limitReached := false
	for range workerCount {
		result := <-resultCh
		limitReached = limitReached || result.limitReached
		if result.err != nil && firstErr == nil {
			firstErr = result.err
		}
	}
	cancelWorkers()
	if firstErr != nil {
		if pipeSession != nil {
			_ = pipeSession.Close()
		}
		return nil, true, firstErr
	}
	if limitReached {
		fmt.Println(tr(opt, "liveLimitReached"))
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
	audioStart := liveRealtimeAudioStart(opt, selected, outs)
	for i := range selected {
		if pipeSession != nil && states[i] != nil {
			continue
		}
		if ok[i] || selected[i].Playlist == nil {
			continue
		}
		out, err := downloadStream(ctx, client, streamForTask(selected[i], i), opt, limiter, audioStart)
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

func runLiveRealtimeTrack(ctx context.Context, stream *StreamSpec, state *liveRealtimeDownloadState, p *parser, opt Options, wait, limit time.Duration) (bool, error) {
	refreshedDuration := int(liveFirstPartDuration(*stream))
	if state != nil {
		if err := state.downloadAndAppend(ctx, liveFirstPartSegments(*stream)); err != nil {
			if isContextCanceled(ctx, err) {
				return false, nil
			}
			return false, err
		}
	}
	for stream.Playlist != nil && stream.Playlist.IsLive {
		if liveTrackRecordLimitReached(refreshedDuration, limit) {
			return true, nil
		}
		if err := waitLiveRefresh(ctx, wait); err != nil {
			return false, nil
		}
		next := *stream
		if err := p.fetchPlaylist(ctx, &next); err != nil {
			if isContextCanceled(ctx, err) {
				return false, nil
			}
			return false, err
		}
		// long: 直播清单每轮都可能引入新的广告分片；必须在去重和时长累计前清理，避免广告污染录制内容与停止条件。
		cleanAdSegments([]StreamSpec{next}, opt.AdKeywords)
		added, duration := appendNewLiveSegmentsDetailed(stream, next)
		refreshedDuration += int(duration)
		if state != nil {
			state.stream = *stream
			if err := state.downloadAndAppend(ctx, added); err != nil {
				if isContextCanceled(ctx, err) {
					return false, nil
				}
				return false, err
			}
		}
	}
	return false, nil
}

func liveTrackRecordLimitReached(refreshedDuration int, limit time.Duration) bool {
	if limit <= 0 {
		return true
	}
	return float64(refreshedDuration) >= limit.Seconds()
}

func liveRealtimeAudioStart(opt Options, selected []StreamSpec, outs []outputFile) *liveAudioStartTracker {
	if !opt.LiveFixVTTByAudio || !hasSelectedAudio(selected) {
		return nil
	}
	tracker := &liveAudioStartTracker{}
	for i, stream := range selected {
		if stream.MediaType == nil || *stream.MediaType != MediaAudio || i >= len(outs) || outs[i].Path == "" {
			continue
		}
		if start, ok := mediaInfosAudioStart(probeMediaInfo(outs[i].Path, opt)); ok {
			tracker.set(start)
			break
		}
	}
	return tracker
}

func prepareSelectedStreams(selected []StreamSpec, opt *Options) []string {
	var messages []string
	living := hasLiveStream(selected) && !opt.LivePerformAsVOD
	if living {
		messages = append(messages, tr(*opt, "liveFound"))
		if msg := liveRecordLimitMessage(*opt); msg != "" {
			messages = append(messages, msg)
		}
	}
	if !opt.BinaryMerge && hasUnknownEncryption(selected) {
		// long: 未识别加密方式在裁剪前就要触发二进制合并；否则用户范围刚好裁掉未知片段时，会和上游的全局流判断不一致。
		opt.BinaryMerge = true
		messages = append(messages, tr(*opt, "autoBinaryMerge3"))
	}
	if !opt.BinaryMerge && hasCENCEncryption(selected) {
		// long: CENC 分片通常需要保持 init 和媒体分片的原始盒结构，先用二进制合并能为后续外部解密保留完整上下文。
		opt.BinaryMerge = true
		messages = append(messages, tr(*opt, "autoBinaryMerge4"))
	}
	if !opt.BinaryMerge && hasFMP4Media(selected) {
		// long: fMP4 HLS 由 init 与 m4s 分片共同组成完整媒体，按上游策略优先顺序拼接，避免 ffmpeg 过早改写片段时间线。
		opt.BinaryMerge = true
		messages = append(messages, tr(*opt, "autoBinaryMerge"))
	}
	if living {
		// long: 原版直播录制会在任务启动时强制多轨并发和 MP4 实时解密，避免直播 fMP4/CENC 等到整轨结束后才补救解密。
		opt.ConcurrentDownload = true
		opt.MP4RealTimeDecryption = true
		if opt.LiveFixVTTByAudio && !hasSelectedAudio(selected) {
			// long: 没有音频轨时不存在可用于修正 WebVTT 的音频 start_time；原版会直接关闭该直播字幕修正开关。
			opt.LiveFixVTTByAudio = false
		}
	}
	if shouldWarnRealtimeDecryption(opt) {
		// long: 上游在 MP4 实时解密配合 mp4decrypt/ffmpeg 和明文 key 时会提示更推荐 Shaka，避免用户误以为所有实时分片解密引擎稳定性相同。
		messages = append(messages, tr(*opt, "realTimeDecMessage"))
	}
	if opt.CustomRange != nil {
		messages = append(messages, tr(*opt, "customRangeFound")+opt.CustomRange.Raw)
		if !living {
			messages = append(messages, tr(*opt, "customRangeWarn"))
		}
	}
	if !living {
		for i := range selected {
			applyCustomRange(&selected[i], opt.CustomRange)
		}
	}
	for _, keyword := range opt.AdKeywords {
		messages = append(messages, tr(*opt, "customAdKeywordsFound")+keyword)
	}
	messages = append(messages, cleanAdSegments(selected, opt.AdKeywords)...)
	return messages
}

func shouldWarnRealtimeDecryption(opt *Options) bool {
	return opt.MP4RealTimeDecryption && !strings.EqualFold(opt.DecryptionEngine, "SHAKA_PACKAGER") && len(opt.Keys) > 0
}

func hasLiveStream(selected []StreamSpec) bool {
	for _, stream := range selected {
		if stream.Playlist != nil && stream.Playlist.IsLive {
			return true
		}
	}
	return false
}

func cleanAdSegments(selected []StreamSpec, keywords []string) []string {
	if len(keywords) == 0 {
		return nil
	}
	var regs []*regexp.Regexp
	for _, keyword := range keywords {
		if keyword == "" {
			continue
		}
		regs = append(regs, regexp.MustCompile(keyword))
	}
	if len(regs) == 0 {
		return nil
	}
	var messages []string
	for streamIndex := range selected {
		pl := selected[streamIndex].Playlist
		if pl == nil {
			continue
		}
		before := countPlaylistSegments(pl)
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
		if pl.MediaInit != nil && matchesAnyAdKeyword(pl.MediaInit.URL, regs) {
			// long: 广告时段的 EXT-X-MAP 不能继续作为主内容 init 下载，否则即使媒体广告已剔除，输出仍会混入广告初始化数据。
			pl.MediaInit = nil
		}
		after := countPlaylistSegments(pl)
		if before != after {
			messages = append(messages, fmt.Sprintf("%d segments => %d segments", before, after))
		}
	}
	return messages
}

func countPlaylistSegments(pl *Playlist) int {
	if pl == nil {
		return 0
	}
	count := 0
	for _, part := range pl.Parts {
		count += len(part.Segments)
	}
	return count
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
	if !hasLiveStream(selected) {
		return nil
	}
	syncLiveStreams(selected, opt.LiveTakeCount)
	refreshedDurations := liveInitialRefreshedDurations(selected)
	limit := liveRecordLimitOrForever(opt.LiveRecordLimit)
	if liveRecordLimitReached(selected, refreshedDurations, limit) {
		fmt.Println(tr(opt, "liveLimitReached"))
		for i := range selected {
			if selected[i].Playlist != nil {
				selected[i].Playlist.IsLive = false
			}
		}
		return nil
	}
	type liveRecordingTrackResult struct {
		limitReached bool
		err          error
	}
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	resultCh := make(chan liveRecordingTrackResult, len(selected))
	workerCount := 0
	for i := range selected {
		if selected[i].Playlist == nil || !selected[i].Playlist.IsLive {
			continue
		}
		workerCount++
		i := i
		trackParser := p.cloneForLiveTrack()
		trackWait := liveRefreshWaitDuration([]StreamSpec{selected[i]}, opt)
		go func() {
			// long: 录制阶段按轨道独立刷新完整清单；某一路源站迟缓时，其他轨道仍可在滑动窗口淘汰前收集新分片。
			limitReached, err := runLiveRecordingTrack(workerCtx, &selected[i], trackParser, opt, refreshedDurations[i], trackWait, limit)
			resultCh <- liveRecordingTrackResult{limitReached: limitReached, err: err}
			if err != nil {
				cancelWorkers()
			}
		}()
	}
	var firstErr error
	limitReached := false
	for range workerCount {
		result := <-resultCh
		limitReached = limitReached || result.limitReached
		if result.err != nil && firstErr == nil {
			firstErr = result.err
		}
	}
	cancelWorkers()
	if firstErr != nil {
		return firstErr
	}
	if limitReached {
		fmt.Println(tr(opt, "liveLimitReached"))
	}
	for i := range selected {
		if selected[i].Playlist != nil {
			selected[i].Playlist.IsLive = false
		}
	}
	return nil
}

func runLiveRecordingTrack(ctx context.Context, stream *StreamSpec, p *parser, opt Options, refreshedDuration int, wait, limit time.Duration) (bool, error) {
	for stream.Playlist != nil && stream.Playlist.IsLive {
		if liveTrackRecordLimitReached(refreshedDuration, limit) {
			return true, nil
		}
		next := *stream
		if err := p.fetchPlaylist(ctx, &next); err != nil {
			if isContextCanceled(ctx, err) {
				return false, nil
			}
			return false, err
		}
		// long: 非实时录制同样依赖刷新后的完整分片集合，广告必须在追加到最终下载列表之前剔除。
		cleanAdSegments([]StreamSpec{next}, opt.AdKeywords)
		refreshedDuration += int(appendNewLiveSegments(stream, next))
		if stream.Playlist == nil || !stream.Playlist.IsLive {
			return false, nil
		}
		if liveTrackRecordLimitReached(refreshedDuration, limit) {
			return true, nil
		}
		if err := waitLiveRefresh(ctx, wait); err != nil {
			return false, nil
		}
	}
	return false, nil
}

func isContextCanceled(ctx context.Context, err error) bool {
	if ctx != nil && ctx.Err() != nil {
		return true
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func liveRecordLimitOrForever(limit *time.Duration) time.Duration {
	if limit == nil {
		// long: 原版直播录制未设置 --live-record-limit 时会把限制改成 TimeSpan.MaxValue，让录制持续到直播结束或用户中断。
		return time.Duration(1<<63 - 1)
	}
	return *limit
}

func waitLiveRefresh(ctx context.Context, wait time.Duration) error {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func liveInitialRefreshedDurations(selected []StreamSpec) []int {
	out := make([]int, len(selected))
	for i, stream := range selected {
		if stream.Playlist == nil || !stream.Playlist.IsLive {
			continue
		}
		out[i] = int(liveFirstPartDuration(stream))
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

func liveRecordLimitReached(selected []StreamSpec, refreshed []int, limit time.Duration) bool {
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
		if i >= len(refreshed) || float64(refreshed[i]) < limitSec {
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
	startMillis := start.UnixMilli()
	var out []Segment
	for _, seg := range segs {
		if seg.DateTime != nil && seg.DateTime.UnixMilli() >= startMillis {
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
	currentSegments := current.Playlist.Parts[0].Segments
	nextSegments := append([]Segment(nil), next.Playlist.Parts[0].Segments...)
	maxIndex := int64(-1)
	for _, seg := range currentSegments {
		if seg.Index > maxIndex {
			maxIndex = seg.Index
		}
	}
	var addedSegments []Segment
	var addedDuration float64
	addedWindow := filterLiveRefreshWindow(currentSegments, nextSegments)
	adjustLiveRefreshIndexes(addedWindow, maxIndex)
	for _, seg := range addedWindow {
		current.Playlist.Parts[0].Segments = append(current.Playlist.Parts[0].Segments, seg)
		addedSegments = append(addedSegments, seg)
		addedDuration += seg.Duration
	}
	current.Playlist.IsLive = next.Playlist.IsLive
	current.Playlist.WasLive = true
	current.Playlist.RefreshIntervalMS = next.Playlist.RefreshIntervalMS
	return addedSegments, addedDuration
}

func filterLiveRefreshWindow(currentSegments []Segment, nextSegments []Segment) []Segment {
	if len(currentSegments) == 0 || len(nextSegments) == 0 {
		return nextSegments
	}
	last := currentSegments[len(currentSegments)-1]
	allNextDateTime := allMediaSegmentsHaveProgramDateTime(nextSegments)
	index := -1
	if last.DateTime != nil && allNextDateTime {
		lastUnixMillis := last.DateTime.UnixMilli()
		for i, seg := range nextSegments {
			if seg.DateTime != nil && seg.DateTime.UnixMilli() == lastUnixMillis {
				index = i
				break
			}
		}
	} else {
		lastName := liveRefreshSegmentName(last, allMediaSegmentsHaveProgramDateTime(currentSegments))
		for i, seg := range nextSegments {
			if liveRefreshSegmentName(seg, allNextDateTime) == lastName {
				index = i
				break
			}
		}
	}
	if index < 0 {
		return nextSegments
	}
	return nextSegments[index+1:]
}

func liveRefreshSegmentName(seg Segment, allHasDateTime bool) string {
	if allHasDateTime && seg.DateTime != nil {
		return fmt.Sprintf("%d", seg.DateTime.UnixMilli())
	}
	return fmt.Sprintf("%d", seg.Index)
}

func adjustLiveRefreshIndexes(segments []Segment, oldMax int64) {
	if len(segments) == 0 || oldMax < 0 {
		return
	}
	newMin := segments[0].Index
	for _, seg := range segments[1:] {
		if seg.Index < newMin {
			newMin = seg.Index
		}
	}
	if newMin >= oldMax {
		return
	}
	offset := oldMax - newMin + 1
	for i := range segments {
		segments[i].Index += offset
	}
}

func liveSegmentKey(seg Segment) string {
	if seg.DateTime != nil {
		return fmt.Sprintf("%d", seg.DateTime.UnixMilli())
	}
	return fmt.Sprintf("%d", seg.Index)
}
