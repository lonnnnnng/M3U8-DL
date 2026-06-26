package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
)

const largeSingleFileSplitSize = 10 * 1024 * 1024

type outputFile struct {
	Path        string
	MediaType   *MediaType
	Language    string
	Name        string
	StreamCount int
}

type liveAudioStartTracker struct {
	mu    sync.Mutex
	ready bool
	value time.Duration
}

type liveRealtimeDownloadState struct {
	stream            StreamSpec
	opt               Options
	client            *http.Client
	limiter           *rateLimiter
	tmpDir            string
	output            string
	pipe              *os.File
	pad               int
	ordinal           int
	liveDateTimeNames bool
	initDone          bool
	initPath          string
	currentKID        string
}

type realtimeInitSegmentResult struct {
	mergePath       string
	decryptInitPath string
	kid             string
}

func (t *liveAudioStartTracker) set(value time.Duration) {
	if t == nil || value <= 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.ready {
		t.ready = true
		t.value = value
	}
}

func (t *liveAudioStartTracker) wait(timeout time.Duration) (time.Duration, bool) {
	if t == nil {
		return 0, false
	}
	deadline := time.Now().Add(timeout)
	for {
		t.mu.Lock()
		value, ready := t.value, t.ready
		t.mu.Unlock()
		if ready {
			return value, true
		}
		if timeout <= 0 || time.Now().After(deadline) {
			return 0, false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func downloadAll(ctx context.Context, client *http.Client, streams []StreamSpec, opt Options) ([]outputFile, error) {
	limiter := newRateLimiter(opt.MaxSpeed)
	var audioStart *liveAudioStartTracker
	if opt.LiveFixVTTByAudio && hasSelectedAudio(streams) {
		audioStart = &liveAudioStartTracker{}
	}
	if opt.ConcurrentDownload {
		var mu sync.Mutex
		outs := make([]outputFile, len(streams))
		ok := make([]bool, len(streams))
		var firstErr error
		var wg sync.WaitGroup
		for i := range streams {
			i := i
			wg.Add(1)
			go func(s StreamSpec) {
				defer wg.Done()
				o, err := downloadStream(ctx, client, streamForTask(s, i), opt, limiter, audioStart)
				mu.Lock()
				defer mu.Unlock()
				if err != nil && firstErr == nil {
					firstErr = err
				}
				if err == nil {
					// long: 上游最终混流会按任务 Index 排序；这里按输入轨道下标落位，避免并发完成顺序改变视频、音频、字幕的 mux 顺序。
					outs[i] = o
					ok[i] = true
				}
			}(streams[i])
		}
		wg.Wait()
		ordered := make([]outputFile, 0, len(streams))
		for i, out := range outs {
			if ok[i] {
				ordered = append(ordered, out)
			}
		}
		return ordered, firstErr
	}
	var outs []outputFile
	for i, s := range streams {
		o, err := downloadStream(ctx, client, streamForTask(s, i), opt, limiter, audioStart)
		if err != nil {
			return outs, err
		}
		outs = append(outs, o)
	}
	return outs, nil
}

func streamForTask(s StreamSpec, taskID int) StreamSpec {
	// long: 原版保存模板里的 <Id> 和临时目录前缀来自 Spectre 的下载任务 ID，而不是 master 列表里的原始流编号；过滤后仍要按本次选择顺序重新编号。
	s.ID = taskID
	return s
}

func downloadStartMessage(opt Options, s StreamSpec) string {
	return tr(opt, "startDownloading") + s.Short()
}

func hasSelectedAudio(streams []StreamSpec) bool {
	for _, stream := range streams {
		if stream.MediaType != nil && *stream.MediaType == MediaAudio {
			return true
		}
	}
	return false
}

func newLiveRealtimeDownloadState(client *http.Client, s StreamSpec, opt Options, limiter *rateLimiter) (*liveRealtimeDownloadState, outputFile, error) {
	if s.Playlist == nil {
		return nil, outputFile{}, fmt.Errorf("轨道缺少 playlist: %s", s.URL)
	}
	tmpRoot := taskTempDir(opt)
	saveDir := opt.SaveDir
	if saveDir == "" {
		saveDir = "."
	}
	dirName := safeName(fmt.Sprintf("%d_%s_%s_%d_%s", s.ID, s.GroupID, s.Codecs, s.Bandwidth, s.Language))
	if dirName == "" || strings.HasPrefix(dirName, "_") {
		dirName = fmt.Sprintf("stream_%d", s.ID)
	}
	tmpDir := filepath.Join(tmpRoot, dirName)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, outputFile{}, err
	}
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return nil, outputFile{}, err
	}
	output := collisionPathForStream(filepath.Join(saveDir, outputBaseName(defaultName(opt, s, dirName), dirName)+liveRealtimeOutputExt(s, opt)), s)
	pad := len(fmt.Sprintf("%d", playlistSegmentCount(s.Playlist)+1000))
	state := &liveRealtimeDownloadState{
		stream:  s,
		opt:     opt,
		client:  client,
		limiter: limiter,
		tmpDir:  tmpDir,
		output:  output,
		pad:     pad,
	}
	return state, outputFile{Path: output, MediaType: s.MediaType, Language: s.Language, Name: s.Name}, nil
}

func playlistSegmentCount(pl *Playlist) int {
	if pl == nil {
		return 0
	}
	count := 0
	if pl.MediaInit != nil {
		count++
	}
	for _, part := range pl.Parts {
		count += len(part.Segments)
	}
	return count
}

func (s *liveRealtimeDownloadState) downloadAndAppend(ctx context.Context, batch []Segment) error {
	if s == nil || len(batch) == 0 && (s.stream.Playlist == nil || s.stream.Playlist.MediaInit == nil || s.initDone) {
		return nil
	}
	var segments []Segment
	var files []string
	s.liveDateTimeNames = allMediaSegmentsHaveProgramDateTime(batch)
	if s.stream.Playlist != nil && s.stream.Playlist.MediaInit != nil && !s.initDone {
		initSeg := *s.stream.Playlist.MediaInit
		initResult, err := downloadRealtimeInitSegment(ctx, s.client, initSeg, s.nextLiveSegmentPath(initSeg, "mp4"), s.opt, s.limiter)
		if err != nil {
			return err
		}
		s.initDone = true
		s.initPath = initResult.decryptInitPath
		s.currentKID = initResult.kid
		if shouldKeepRealtimeInitForMerge(s.opt, initResult.kid) {
			files = append(files, initResult.mergePath)
			segments = append(segments, initSeg)
		}
	}
	if len(batch) > 0 {
		batchFiles := make([]string, len(batch))
		errCh := make(chan error, len(batch))
		workers := s.opt.ThreadCount
		if workers <= 0 {
			workers = 1
		}
		sem := make(chan struct{}, workers)
		var wg sync.WaitGroup
		for i, seg := range batch {
			i, seg := i, seg
			ext := s.stream.Extension
			if ext == "" {
				ext = "ts"
			}
			path := s.nextLiveSegmentPath(seg, ext)
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				actual, err := s.downloadLiveSegment(ctx, seg, path)
				if err != nil {
					errCh <- err
					return
				}
				batchFiles[i] = actual
			}()
		}
		wg.Wait()
		close(errCh)
		var downloadErrs []error
		for err := range errCh {
			if err != nil {
				downloadErrs = append(downloadErrs, err)
			}
		}
		if len(downloadErrs) > 0 && s.opt.CheckSegmentsCount {
			return fmt.Errorf("%s %w", tr(s.opt, "segmentCountCheckNotPass", len(batch), countDownloadedFiles(batchFiles)), downloadErrs[0])
		}
		files = append(files, compactDownloadedFiles(batchFiles)...)
		segments = append(segments, compactDownloadedSegments(batchFiles, batch)...)
	}
	if len(files) == 0 {
		return nil
	}
	if s.pipe != nil {
		return copyFilesToOpenLivePipe(s.pipe, files, s.opt.LiveKeepSegments)
	}
	return liveRealtimeMergeFiles(files, segments, s.output, s.opt.LiveKeepSegments)
}

func (s *liveRealtimeDownloadState) nextLiveSegmentPath(seg Segment, ext string) string {
	path := segmentTempPath(
		s.tmpDir,
		seg,
		s.ordinal,
		s.pad,
		ext,
		true,
		s.liveDateTimeNames,
	)
	s.ordinal++
	return path
}

func (s *liveRealtimeDownloadState) downloadLiveSegment(ctx context.Context, seg Segment, path string) (string, error) {
	return downloadSegment(ctx, s.client, seg, path, s.opt, s.limiter, s.currentKID, s.initPath)
}

func downloadStream(ctx context.Context, client *http.Client, s StreamSpec, opt Options, limiter *rateLimiter, audioStart *liveAudioStartTracker) (outputFile, error) {
	if s.Playlist == nil {
		return outputFile{}, fmt.Errorf("轨道缺少 playlist: %s", s.URL)
	}
	tmpRoot := taskTempDir(opt)
	saveDir := opt.SaveDir
	if saveDir == "" {
		saveDir = "."
	}
	dirName := safeName(fmt.Sprintf("%d_%s_%s_%d_%s", s.ID, s.GroupID, s.Codecs, s.Bandwidth, s.Language))
	if dirName == "" || strings.HasPrefix(dirName, "_") {
		dirName = fmt.Sprintf("stream_%d", s.ID)
	}
	tmpDir := filepath.Join(tmpRoot, dirName)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return outputFile{}, err
	}
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return outputFile{}, err
	}

	if split, err := splitSingleMediaSegment(ctx, client, s, opt); err != nil {
		fmt.Println(tr(opt, "singleFileSplitFailed", err))
	} else if len(split) > 0 {
		// long: 单 URL 大文件支持 Range 时按上游拆成多个虚拟分片，让既有并发下载逻辑能并行拉取同一个媒体资源的不同字节窗口。
		s.Playlist.Parts = []MediaPart{{Segments: split}}
		if opt.MP4RealTimeDecryption {
			opt.MP4RealTimeDecryption = false
			fmt.Println(tr(opt, "singleFileRealtimeDecryptWarn"))
		}
		fmt.Println(tr(opt, "singleFileSplitWarn"))
	}

	var allSegs []Segment
	if s.Playlist.MediaInit != nil {
		allSegs = append(allSegs, *s.Playlist.MediaInit)
	}
	var mediaSegs []Segment
	for _, part := range s.Playlist.Parts {
		allSegs = append(allSegs, part.Segments...)
		mediaSegs = append(mediaSegs, part.Segments...)
	}
	if len(allSegs) == 0 {
		return outputFile{}, fmt.Errorf("没有分片可下载")
	}
	fmt.Println(downloadStartMessage(opt, s))

	pad := len(fmt.Sprintf("%d", len(allSegs)))
	liveSegmentNames := shouldUseLiveSegmentNames(s, opt)
	liveDateTimeNames := liveSegmentNames && allMediaSegmentsHaveProgramDateTime(mediaSegs)
	files := make([]string, len(allSegs))
	var done int64
	startAt := 0
	initPath := ""
	currentKID := ""
	if opt.MP4RealTimeDecryption && s.Playlist.MediaInit != nil && hasExternalMP4Encryption(s) {
		initSeg := *s.Playlist.MediaInit
		initPath = segmentTempPath(tmpDir, initSeg, 0, pad, "mp4", liveSegmentNames, liveDateTimeNames)
		initResult, err := downloadRealtimeInitSegment(ctx, client, initSeg, initPath, opt, limiter)
		if err != nil {
			return outputFile{}, err
		}
		files[0] = initResult.mergePath
		initPath = initResult.decryptInitPath
		currentKID = initResult.kid
		if !shouldKeepRealtimeInitForMerge(opt, initResult.kid) {
			// long: shaka/ffmpeg 实时解密会把 init 与每个媒体分片临时拼接后交给外部工具；最终再合并独立 init 会比上游多出一段重复初始化数据。
			files[0] = ""
		}
		startAt = 1
		atomic.AddInt64(&done, 1)
	}
	progress := newDownloadProgressReporter(opt, s.Short(), len(allSegs))
	sem := make(chan struct{}, opt.ThreadCount)
	errCh := make(chan error, len(allSegs))
	var wg sync.WaitGroup
	for i, seg := range allSegs[startAt:] {
		i := i + startAt
		i, seg := i, seg
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			ext := s.Extension
			if seg.Index == -1 {
				ext = "mp4"
			}
			path := segmentTempPath(tmpDir, seg, i, pad, ext, liveSegmentNames, liveDateTimeNames)
			actual, err := downloadSegment(ctx, client, seg, path, opt, limiter, currentKID, initPath)
			if err != nil {
				errCh <- err
				return
			}
			files[i] = actual
			progress.addFile(actual)
			v := atomic.AddInt64(&done, 1)
			progress.print(v)
		}()
	}
	wg.Wait()
	close(errCh)
	progress.finish()
	var downloadErrs []error
	for err := range errCh {
		if err != nil {
			downloadErrs = append(downloadErrs, err)
		}
	}
	if len(downloadErrs) > 0 && opt.CheckSegmentsCount {
		return outputFile{}, fmt.Errorf("%s %w", tr(opt, "segmentCountCheckNotPass", len(allSegs), countDownloadedFiles(files)), downloadErrs[0])
	}
	fileSegments := compactDownloadedSegments(files, allSegs)
	files = compactDownloadedFiles(files)
	if len(files) == 0 {
		if len(downloadErrs) > 0 {
			return outputFile{}, fmt.Errorf("没有成功下载的分片: %w", downloadErrs[0])
		}
		return outputFile{}, fmt.Errorf("没有成功下载的分片")
	}
	fmt.Println(tr(opt, "readingInfo"))
	mediaInfos := probeMediaInfo(firstMediaProbeFile(files), opt)
	for _, msg := range applyMediaInfoToStream(&s, &opt, mediaInfos) {
		fmt.Println(msg)
	}
	useAACFilter := mediaInfosUseAACFilter(mediaInfos)
	if audioStart != nil && s.MediaType != nil && *s.MediaType == MediaAudio {
		start, ok := mediaInfosAudioStart(mediaInfos)
		if ok {
			audioStart.set(start)
		}
	}
	liveSubtitleOffset := time.Duration(s.SkippedDuration * float64(time.Second))
	if opt.LiveFixVTTByAudio && audioStart != nil && s.MediaType != nil && *s.MediaType == MediaSubtitles {
		if start, ok := audioStart.wait(5 * time.Second); ok {
			// long: 直播 WebVTT 有时以音频轨的非零 start_time 为时间基准，上游会等待音频探测结果并扣掉这段偏移，避免字幕整体晚出。
			liveSubtitleOffset += start
		}
	}

	outputExt := outputExt(s, opt)
	if opt.LiveRealTimeMerge && s.Playlist != nil && s.Playlist.IsLive {
		outputExt = liveRealtimeOutputExt(s, opt)
	}
	outName := outputBaseName(defaultName(opt, s, dirName), dirName) + outputExt
	output := collisionPathForStream(filepath.Join(saveDir, outName), s)
	if opt.SkipMerge {
		return outputFile{Path: tmpDir, MediaType: s.MediaType, Language: s.Language, Name: s.Name}, nil
	}
	if opt.AutoSubtitleFix && s.MediaType != nil && *s.MediaType == MediaSubtitles && strings.EqualFold(s.Extension, "vtt") {
		fmt.Println(tr(opt, "fixingVTT"))
		fixedOutput := strings.TrimSuffix(output, filepath.Ext(output))
		if strings.EqualFold(opt.SubFormat, "SRT") {
			fixedOutput += ".srt"
		} else {
			fixedOutput += ".vtt"
		}
		fixedOutput = collisionPathForStream(fixedOutput, s)
		if err := mergeVTTFilesWithSegmentsAndOffsetOpt(files, fileSegments, fixedOutput, liveSubtitleOffset, opt.SubFormat, opt); err != nil {
			return outputFile{}, err
		}
		cleanupFixedSubtitleSourceFiles(files, false)
		output = fixedOutput
	} else if opt.AutoSubtitleFix && s.MediaType != nil && *s.MediaType == MediaSubtitles && strings.Contains(strings.ToLower(s.Extension), "ttml") {
		fmt.Println(tr(opt, "fixingTTML"))
		fixedOutput := strings.TrimSuffix(output, filepath.Ext(output))
		if strings.EqualFold(opt.SubFormat, "SRT") {
			fixedOutput += ".srt"
		} else {
			fixedOutput += ".vtt"
		}
		fixedOutput = collisionPathForStream(fixedOutput, s)
		if err := mergeTTMLFilesWithSegmentsOpt(files, fileSegments, fixedOutput, s.SkippedDuration, opt.SubFormat, opt); err != nil {
			return outputFile{}, err
		}
		cleanupFixedSubtitleSourceFiles(files, true)
		output = fixedOutput
	} else if opt.AutoSubtitleFix && s.MediaType != nil && *s.MediaType == MediaSubtitles && strings.Contains(strings.ToLower(s.Extension), "m4s") && (strings.Contains(strings.ToLower(s.Codecs), "stpp") || mp4FilesContainBox(files, "stpp")) {
		fmt.Println(tr(opt, "fixingTTMLmp4"))
		fixedOutput := strings.TrimSuffix(output, filepath.Ext(output))
		if strings.EqualFold(opt.SubFormat, "SRT") {
			fixedOutput += ".srt"
		} else {
			fixedOutput += ".vtt"
		}
		fixedOutput = collisionPathForStream(fixedOutput, s)
		ok, err := extractMP4TTMLFilesWithSegmentsOpt(files, fileSegments, fixedOutput, s.SkippedDuration, opt.SubFormat, opt)
		if err != nil {
			return outputFile{}, err
		}
		if ok {
			cleanupFixedSubtitleSourceFiles(files, true)
			output = fixedOutput
		} else if err := binaryMergeWithMessage(opt, files, output); err != nil {
			return outputFile{}, err
		}
	} else if opt.AutoSubtitleFix && s.MediaType != nil && *s.MediaType == MediaSubtitles && strings.Contains(strings.ToLower(s.Extension), "m4s") && !strings.Contains(strings.ToLower(s.Codecs), "stpp") {
		fixedOutput := strings.TrimSuffix(output, filepath.Ext(output))
		if strings.EqualFold(opt.SubFormat, "SRT") {
			fixedOutput += ".srt"
		} else {
			fixedOutput += ".vtt"
		}
		fixedOutput = collisionPathForStream(fixedOutput, s)
		ok, err := extractMP4WebVTTFilesWithSegmentsOpt(files, fileSegments, fixedOutput, s.SkippedDuration, opt.SubFormat, opt)
		if err != nil {
			return outputFile{}, err
		}
		if ok {
			fmt.Println(tr(opt, "fixingVTTmp4"))
			cleanupFixedSubtitleSourceFiles(files, false)
			output = fixedOutput
		} else if err := binaryMergeWithMessage(opt, files, output); err != nil {
			return outputFile{}, err
		}
	} else if opt.LiveRealTimeMerge && (s.MediaType == nil || *s.MediaType != MediaSubtitles) {
		// long: 直播实时合并不会等 ffmpeg 重新封装单轨，而是把已经完成的媒体分片顺序追加到输出；这一步先对齐上游的文件输出语义，后续 producer/consumer 主循环可复用同一批追加逻辑。
		if err := liveRealtimeMergeFiles(files, fileSegments, output, opt.LiveKeepSegments); err != nil {
			return outputFile{}, err
		}
	} else if opt.BinaryMerge || s.MediaType != nil && *s.MediaType == MediaSubtitles || s.Playlist.MediaInit != nil && opt.MuxAfterDone == nil {
		// long: fMP4 和字幕默认采用二进制顺序拼接，避免 ffmpeg 在没有完整轨道上下文时错误改写时间戳。
		if err := binaryMergeWithMessage(opt, files, output); err != nil {
			return outputFile{}, err
		}
	} else {
		format := singleTrackFFmpegFormat(s)
		merged, err := ffmpegMerge(files, strings.TrimSuffix(output, filepath.Ext(output)), format, opt, useAACFilter)
		if err != nil {
			return outputFile{}, err
		}
		output = merged
	}
	if !opt.MP4RealTimeDecryption && hasExternalMP4Encryption(s) {
		decrypted, err := decryptMP4Output(output, opt)
		if err != nil {
			return outputFile{}, err
		}
		output = decrypted
	}
	if opt.DelAfterDone {
		_ = cleanupDownloadedTempDir(tmpDir, files)
	}
	return outputFile{Path: output, MediaType: s.MediaType, Language: s.Language, Name: s.Name}, nil
}

func cleanupDownloadedTempDir(tmpDir string, files []string) error {
	for _, file := range files {
		if err := os.Remove(file); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	// long: concat.txt 是 Go 版为 ffmpeg concat demuxer 生成的任务内辅助文件；清掉它后再按上游 SafeDeleteDir 只删除空目录，用户额外放入的排障文件会阻止目录被删除。
	if err := os.Remove(filepath.Join(tmpDir, "concat.txt")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return safeDeleteEmptyParents(tmpDir)
}

func cleanupFixedSubtitleSourceFiles(files []string, honorImageKeepEnv bool) {
	if honorImageKeepEnv && os.Getenv("RE_KEEP_IMAGE_SEGMENTS") == "1" {
		return
	}
	for _, file := range files {
		// long: 字幕修复成功后，上游会把原始 VTT/TTML/m4s 分片从结果集合中移除；保留它们会让 --del-after-done=false 时多出非最终产物。
		_ = os.Remove(file)
	}
}

func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func shouldUseLiveSegmentNames(s StreamSpec, opt Options) bool {
	return !opt.LivePerformAsVOD && s.Playlist != nil && s.Playlist.WasLive
}

func allMediaSegmentsHaveProgramDateTime(segments []Segment) bool {
	hasMedia := false
	for _, seg := range segments {
		if seg.Index == -1 {
			continue
		}
		hasMedia = true
		if seg.DateTime == nil {
			return false
		}
	}
	return hasMedia
}

func segmentTempPath(tmpDir string, seg Segment, ordinal int, pad int, ext string, liveNames bool, liveDateTimeNames bool) string {
	stem := fmt.Sprintf("%0*d", pad, ordinal)
	if liveNames {
		// long: 直播窗口会持续滑动，上游用 HLS 的业务键命名分片，避免同 URL 的新分片被数组序号覆盖或错序复用。
		switch {
		case seg.Index == -1:
			stem = "_init"
		case liveDateTimeNames && seg.DateTime != nil:
			stem = fmt.Sprintf("%d", seg.DateTime.Unix())
		default:
			stem = fmt.Sprintf("%d", seg.Index)
		}
	}
	fileStem := safeName(stem)
	if stem == "_init" {
		fileStem = stem
	}
	return filepath.Join(tmpDir, fileStem+"."+ext)
}

func splitSingleMediaSegment(ctx context.Context, client *http.Client, s StreamSpec, opt Options) ([]Segment, error) {
	if s.Playlist == nil || len(s.Playlist.Parts) != 1 || len(s.Playlist.Parts[0].Segments) != 1 {
		return nil, nil
	}
	seg := s.Playlist.Parts[0].Segments[0]
	if seg.StartRange != nil {
		return nil, nil
	}
	size, ok, err := probeRangeSize(ctx, client, seg.URL, opt.Headers)
	if err != nil || !ok || size <= largeSingleFileSplitSize {
		return nil, err
	}
	var split []Segment
	for remaining, start, index := size, int64(0), int64(0); remaining > 0; index++ {
		end := start + largeSingleFileSplitSize
		if remaining-largeSingleFileSplitSize <= 0 {
			end = size
		}
		length := end - start + 1
		next := seg
		next.Index = index
		next.StartRange = cloneInt64(start)
		next.ExpectLength = cloneInt64(length)
		next.Duration = 0
		split = append(split, next)
		if remaining-largeSingleFileSplitSize > 0 {
			// long: 原版 LargeSingleFileSplitUtil 用闭区间生成 0-10MiB、下一段从 10MiB+1 开始；最后一段 end 还会等于 Content-Length，交给 HTTP Range 语义截到文件尾。
			remaining -= largeSingleFileSplitSize
			start = end + 1
			continue
		}
		break
	}
	return split, nil
}

func probeRangeSize(ctx context.Context, client *http.Client, rawURL string, headers map[string]string) (int64, bool, error) {
	rangeResp, err := doRequestWithRedirects(ctx, client, http.MethodHead, rawURL, nil, nil)
	if err != nil {
		return 0, false, err
	}
	defer rangeResp.Body.Close()
	if rangeResp.StatusCode < 200 || rangeResp.StatusCode >= 300 {
		return 0, false, fmt.Errorf("HEAD %s 返回 HTTP %d", rawURL, rangeResp.StatusCode)
	}
	// long: 原版 CanSplitAsync 的第一次 HEAD 不附带用户 headers，只检查是否声明 Accept-Ranges；带鉴权才出现的 Range 能力不能触发单文件拆分。
	if rangeResp.Header.Get("Accept-Ranges") == "" {
		return 0, false, nil
	}
	sizeResp, err := doRequestWithRedirects(ctx, client, http.MethodHead, rawURL, headers, nil)
	if err != nil {
		return 0, false, err
	}
	defer sizeResp.Body.Close()
	if sizeResp.StatusCode < 200 || sizeResp.StatusCode >= 300 {
		return 0, false, fmt.Errorf("HEAD %s 返回 HTTP %d", rawURL, sizeResp.StatusCode)
	}
	return sizeResp.ContentLength, sizeResp.ContentLength > 0, nil
}

func cloneInt64(v int64) *int64 {
	return &v
}

func countDownloadedFiles(files []string) int {
	return len(compactDownloadedFiles(files))
}

func compactDownloadedFiles(files []string) []string {
	var out []string
	for _, file := range files {
		if file == "" {
			continue
		}
		if _, err := os.Stat(file); err != nil {
			continue
		}
		out = append(out, file)
	}
	return out
}

func compactDownloadedSegments(files []string, segments []Segment) []Segment {
	var out []Segment
	for i, file := range files {
		if file == "" || i >= len(segments) {
			continue
		}
		if _, err := os.Stat(file); err != nil {
			continue
		}
		out = append(out, segments[i])
	}
	return out
}

func firstMediaProbeFile(files []string) string {
	for _, file := range files {
		base := strings.ToLower(filepath.Base(file))
		if strings.Contains(base, "init") || strings.HasSuffix(base, ".mp4") && strings.HasPrefix(base, "_") {
			continue
		}
		return file
	}
	if len(files) > 0 {
		return files[0]
	}
	return ""
}

type downloadProgressReporter struct {
	opt        Options
	streamName string
	total      int
	startedAt  time.Time
	bytes      int64
}

type downloadProgressEvent struct {
	Type      string  `json:"type"`
	Timestamp string  `json:"timestamp"`
	Stream    string  `json:"stream"`
	Current   int64   `json:"current"`
	Total     int     `json:"total"`
	Speed     string  `json:"speed"`
	Bytes     int64   `json:"bytes"`
	Percent   float64 `json:"percent"`
}

func newDownloadProgressReporter(opt Options, streamName string, total int) *downloadProgressReporter {
	return &downloadProgressReporter{
		opt:        opt,
		streamName: streamName,
		total:      total,
		startedAt:  time.Now(),
	}
}

func (p *downloadProgressReporter) addFile(path string) {
	if p == nil || strings.TrimSpace(path) == "" {
		return
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return
	}
	atomic.AddInt64(&p.bytes, info.Size())
}

func (p *downloadProgressReporter) print(current int64) {
	if p == nil {
		return
	}
	bytes := atomic.LoadInt64(&p.bytes)
	speed := formatByteRate(bytes, time.Since(p.startedAt))
	if p.opt.ProgressJSON {
		percent := 0.0
		if p.total > 0 {
			percent = float64(current) / float64(p.total)
		}
		// long: 桌面端需要稳定协议而不是解析本地化文案；JSON 直写原始 stdout，避免日志层时间戳破坏逐行 JSON 解析。
		event := downloadProgressEvent{
			Type:      "progress",
			Timestamp: time.Now().Format("2006-01-02 15:04:05"),
			Stream:    p.streamName,
			Current:   current,
			Total:     p.total,
			Speed:     speed,
			Bytes:     bytes,
			Percent:   percent,
		}
		emitRawJSONEvent(p.opt, event)
		return
	}
	message := tr(p.opt, "downloadProgressWithSpeed", p.streamName, current, p.total, speed)
	if p.opt.ForceANSIConsole {
		// long: 桌面端通过管道按换行读取 CLI 输出；重定向场景必须逐行输出，否则 UI 只能在下载结束后才收到进度。
		fmt.Println(message)
		return
	}
	fmt.Print("\r" + message)
}

func (p *downloadProgressReporter) finish() {
	if p == nil || p.opt.ForceANSIConsole || p.opt.ProgressJSON {
		return
	}
	fmt.Println()
}

func formatByteRate(bytes int64, elapsed time.Duration) string {
	if bytes <= 0 || elapsed <= 0 {
		return "0 B/s"
	}
	return formatByteSize(float64(bytes)/elapsed.Seconds()) + "/s"
}

func formatByteSize(value float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%.0f %s", value, units[unit])
	}
	if value >= 10 {
		return fmt.Sprintf("%.1f %s", value, units[unit])
	}
	return fmt.Sprintf("%.2f %s", value, units[unit])
}

func downloadSegment(ctx context.Context, client *http.Client, seg Segment, path string, opt Options, limiter *rateLimiter, kid string, initPath string) (string, error) {
	if shouldPreferExistingDecryptedSegment(seg, opt) {
		if dec := decryptedSegmentPath(path); dec != path {
			if _, err := os.Stat(dec); err == nil {
				// long: 实时 MP4 解密续跑时，原始加密分片和 _dec 可能同时存在；继续用 _dec 才不会把已解密片段退回加密态。
				return dec, nil
			}
		}
	}
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if dec := decryptedSegmentPath(path); dec != path {
		if _, err := os.Stat(dec); err == nil {
			// long: 实时 MP4 解密失败重跑时，上游会优先复用已生成的 _dec 分片，避免再次下载和再次调用外部解密工具。
			return dec, nil
		}
	}
	if data, ok, validateLength, err := readSpecialSegmentBytes(seg); ok || err != nil {
		if err != nil {
			return "", err
		}
		if validateLength {
			if err := validateDownloadedLength(seg, len(data), -1, false); err != nil {
				return "", err
			}
		}
		if seg.IsEncrypted() {
			data, err = decryptSegment(data, seg.Encrypt)
			if err != nil {
				return "", err
			}
		}
		data = processDownloadedPayload(data)
		if err := os.WriteFile(path, data, 0644); err != nil {
			return "", err
		}
		if opt.MP4RealTimeDecryption && isExternalEncryptedSegment(seg) && len(collectDecryptKeys(opt, kid)) > 0 {
			dec, err := decryptMP4File(path, opt, kid, initPath)
			if err != nil {
				return "", err
			}
			path = dec
		}
		return path, nil
	}
	var lastErr error
	for try := 0; try <= opt.DownloadRetryCount; try++ {
		resp, err := doRequestWithRedirects(ctx, client, http.MethodGet, seg.URL, opt.Headers, func(req *http.Request) {
			if seg.StartRange != nil && seg.ExpectLength != nil {
				end := *seg.StartRange + *seg.ExpectLength - 1
				// long: EXT-X-BYTERANGE 表示同一个资源里的字节窗口，必须转换成 HTTP Range 才不会把整段文件重复拉下来。
				req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", *seg.StartRange, end))
			}
		})
		if err != nil {
			lastErr = err
			time.Sleep(time.Second)
			continue
		}
		data, encoded, err := readResponseBytes(resp, opt, limiter)
		contentLength := resp.ContentLength
		resp.Body.Close()
		if err != nil {
			lastErr = err
			time.Sleep(time.Second)
			continue
		}
		if err := validateDownloadedLength(seg, len(data), contentLength, encoded); err != nil {
			lastErr = err
			time.Sleep(time.Second)
			continue
		}
		if seg.IsEncrypted() {
			data, err = decryptSegment(data, seg.Encrypt)
			if err != nil {
				return "", err
			}
		}
		data = processDownloadedPayload(data)
		if err := os.WriteFile(path, data, 0644); err != nil {
			return "", err
		}
		if opt.MP4RealTimeDecryption && isExternalEncryptedSegment(seg) && len(collectDecryptKeys(opt, kid)) > 0 {
			dec, err := decryptMP4File(path, opt, kid, initPath)
			if err != nil {
				return "", err
			}
			path = dec
		}
		return path, nil
	}
	return "", lastErr
}

func shouldPreferExistingDecryptedSegment(seg Segment, opt Options) bool {
	return opt.MP4RealTimeDecryption && isExternalEncryptedSegment(seg)
}

func downloadRealtimeInitSegment(ctx context.Context, client *http.Client, seg Segment, path string, opt Options, limiter *rateLimiter) (realtimeInitSegmentResult, error) {
	initOpt := opt
	initOpt.MP4RealTimeDecryption = false
	actual, err := downloadSegment(ctx, client, seg, path, initOpt, limiter, "", "")
	if err != nil {
		return realtimeInitSegmentResult{}, err
	}
	kid := extractDefaultKIDFromFile(actual)
	if kid == "" && strings.EqualFold(opt.DecryptionEngine, "SHAKA_PACKAGER") {
		bin := opt.DecryptionBinaryPath
		if bin == "" {
			bin = firstExecutable("shaka-packager", "packager-linux-x64", "packager-osx-x64", "packager-win-x64")
		}
		if detected, err := detectKIDWithShaka(actual, bin); err == nil && detected != "" {
			// long: 某些 fMP4/WebM init 不暴露 tenc/PSSH KID；上游会借 shaka 的缺 key 错误反查 key_id，再继续匹配 key-file 做实时解密。
			kid = detected
		}
	}
	result := realtimeInitSegmentResult{mergePath: actual, decryptInitPath: actual, kid: kid}
	if !opt.MP4RealTimeDecryption || kid == "" || len(collectDecryptKeys(opt, kid)) == 0 {
		return result, nil
	}
	if !canDecryptRealtimeInitFile(opt) {
		return result, nil
	}
	decPath := decryptedSegmentPath(actual)
	if err := copyFile(actual, decPath); err != nil {
		return realtimeInitSegmentResult{}, err
	}
	// long: 上游把原始 init 留给后续媒体分片的 --fragments-info，同时把解密后的 init 放入合并队列；两条路径不能混用。
	dec, err := decryptMP4File(decPath, opt, kid, "")
	if err != nil {
		_ = os.Remove(decPath)
		return realtimeInitSegmentResult{}, err
	}
	result.mergePath = dec
	result.decryptInitPath = actual
	return result, nil
}

func canDecryptRealtimeInitFile(opt Options) bool {
	engine := strings.ToUpper(opt.DecryptionEngine)
	return engine == "" || engine == "MP4DECRYPT"
}

func shouldKeepRealtimeInitForMerge(opt Options, kid string) bool {
	if !opt.MP4RealTimeDecryption || kid == "" || len(collectDecryptKeys(opt, kid)) == 0 {
		return true
	}
	return canDecryptRealtimeInitFile(opt)
}

func decryptedSegmentPath(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return path + "_dec"
	}
	return strings.TrimSuffix(path, ext) + "_dec" + ext
}

func validateDownloadedLength(seg Segment, actual int, responseLength int64, encoded bool) error {
	actualLength := int64(actual)
	if !encoded && responseLength >= 0 && responseLength != actualLength {
		// long: 上游 DownloadResult.Success 只比较 HTTP Content-Length 和实际写入长度；HLS BYTERANGE 的 ExpectLength 不参与成功判定。
		return fmt.Errorf("响应长度校验失败: Content-Length %d, 实际 %d, url=%s", responseLength, actualLength, seg.URL)
	}
	return nil
}

func readSpecialSegmentBytes(seg Segment) ([]byte, bool, bool, error) {
	var data []byte
	var err error
	applyRange := false
	switch {
	case strings.HasPrefix(seg.URL, "base64://"):
		data, err = decodeInlineBase64Segment(seg.URL[len("base64://"):])
	case strings.HasPrefix(seg.URL, "hex://"):
		data, err = decodeInlineHexSegment(seg.URL[len("hex://"):])
	case strings.HasPrefix(seg.URL, "file:"):
		u, parseErr := url.Parse(seg.URL)
		if parseErr != nil {
			return nil, true, false, parseErr
		}
		data, err = os.ReadFile(fileURLPath(u))
		applyRange = true
	case !strings.HasPrefix(seg.URL, "http://") && !strings.HasPrefix(seg.URL, "https://"):
		data, err = os.ReadFile(seg.URL)
		applyRange = true
	default:
		return nil, false, false, nil
	}
	if err != nil {
		return nil, true, false, err
	}
	if applyRange {
		data, err = applySegmentRange(data, seg)
		if err != nil {
			return nil, true, true, err
		}
	}
	return data, true, applyRange, nil
}

func decodeInlineBase64Segment(raw string) ([]byte, error) {
	clean := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			// long: 上游 Convert.FromBase64String 会忽略 Base64 文本中的空白字符；内联分片常来自清单文本，保留该宽松解析可避免格式化换行导致下载失败。
			return -1
		}
		return r
	}, raw)
	return base64.StdEncoding.DecodeString(clean)
}

func decodeInlineHexSegment(raw string) ([]byte, error) {
	clean := strings.TrimSpace(raw)
	if strings.HasPrefix(clean, "0x") || strings.HasPrefix(clean, "0X") {
		// long: 上游 HexUtil.HexToBytes 会先裁掉 0x/0X 前缀；HLS 内联 hex 分片也走这条工具函数，带前缀的片段不能被误判为坏 URL。
		clean = clean[2:]
	}
	return hex.DecodeString(clean)
}

func applySegmentRange(data []byte, seg Segment) ([]byte, error) {
	if seg.StartRange == nil {
		return data, nil
	}
	start := *seg.StartRange
	if start < 0 {
		return nil, fmt.Errorf("分片 Range 起点无效: %d", start)
	}
	if seg.ExpectLength == nil {
		if start == 0 {
			return data, nil
		}
		if start > int64(len(data)) {
			return nil, fmt.Errorf("分片 Range 起点无效: %d", start)
		}
		out := make([]byte, int64(len(data))-start+1)
		// long: 原版本地文件 fromPosition 有值但 toPosition 为空时会按 Length-Position+1 分配缓冲区，因此起点非 0 的开放范围会多写一个尾部 0。
		copy(out, data[start:])
		return out, nil
	}
	length := *seg.ExpectLength
	if length < 0 {
		return nil, fmt.Errorf("分片 Range 长度无效: %d", length)
	}
	out := make([]byte, length)
	if start >= int64(len(data)) {
		// long: 原版 FileStream.ReadAsync 读不到完整本地 BYTERANGE 时仍会写出预分配缓冲区，未读到的尾部保持 0。
		return out, nil
	}
	copy(out, data[start:])
	return out, nil
}

func isExternalEncryptedSegment(seg Segment) bool {
	switch seg.Encrypt.Method {
	case EncryptCENC, EncryptSampleAES, EncryptSampleCTR:
		return true
	default:
		return false
	}
}

func readResponseBytes(resp *http.Response, opt Options, limiter *rateLimiter) ([]byte, bool, error) {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Request.URL)
	}
	body, cleanup, encoded, err := decodedResponseBody(resp)
	if err != nil {
		return nil, encoded, err
	}
	defer cleanup()
	data, err := readAllWithLimiter(body, limiter)
	return data, encoded, err
}

func processDownloadedPayload(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	if isImageHeader(data) {
		// long: 上游在 HLS 解密完成后再剥图片伪装头；这样加密分片解出真实媒体数据后，仍能进入同一套清理路径。
		data = stripImageHeader(data)
	}
	if len(data) > 2 && data[0] == 0x1f && data[1] == 0x8b {
		if out, err := gunzipBytes(data); err == nil {
			data = out
		}
	}
	return data
}

func isImageHeader(data []byte) bool {
	if len(data) > 3 && data[0] == 137 && data[1] == 80 && data[2] == 78 && data[3] == 71 {
		return true
	}
	if len(data) > 3 && data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return true
	}
	if len(data) > 10 && data[0] == 0x42 && data[1] == 0x4d && data[5] == 0 && data[6] == 0 && data[7] == 0 && data[8] == 0 {
		return true
	}
	return len(data) > 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff
}

func stripImageHeader(data []byte) []byte {
	switch {
	case len(data) > 3 && data[0] == 137 && data[1] == 80 && data[2] == 78 && data[3] == 71:
		for _, skip := range []int{120, 6102, 69, 771} {
			if len(data) > skip && data[skip-2] == 96 && data[skip-1] == 130 {
				return data[skip:]
			}
		}
		return stripToTSSync(data, 4)
	case len(data) >= 42 && data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38:
		return data[42:]
	case len(data) >= 0x3e && data[0] == 0x42 && data[1] == 0x4d && data[5] == 0 && data[6] == 0 && data[7] == 0 && data[8] == 0:
		return data[0x3e:]
	case len(data) > 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		return stripToTSSync(data, 4)
	default:
		return data
	}
}

func stripToTSSync(data []byte, start int) []byte {
	for i := start; i < len(data)-188*2-4; i++ {
		if data[i] == 0x47 && data[i+188] == 0x47 && data[i+188+188] == 0x47 {
			return data[i:]
		}
	}
	return data
}

func gunzipBytes(data []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	return io.ReadAll(gz)
}

func readAllWithLimit(r io.Reader, maxSpeed int64) ([]byte, error) {
	return readAllWithLimiter(r, newRateLimiter(maxSpeed))
}

func readAllWithLimiter(r io.Reader, limiter *rateLimiter) ([]byte, error) {
	if limiter == nil || limiter.maxBytesPerSec <= 0 {
		return io.ReadAll(r)
	}
	var out []byte
	buf := make([]byte, 32*1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			out = append(out, buf[:n]...)
			limiter.Wait(int64(n))
		}
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return out, err
		}
	}
}

type rateLimiter struct {
	maxBytesPerSec int64
	start          time.Time
	total          int64
	mu             sync.Mutex
}

func newRateLimiter(maxBytesPerSec int64) *rateLimiter {
	return &rateLimiter{maxBytesPerSec: maxBytesPerSec, start: time.Now()}
}

func (r *rateLimiter) Wait(n int64) {
	if r == nil || r.maxBytesPerSec <= 0 || n <= 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.total += n
	want := time.Duration(float64(r.total) / float64(r.maxBytesPerSec) * float64(time.Second))
	if sleep := want - time.Since(r.start); sleep > 0 {
		time.Sleep(sleep)
	}
}

func binaryMerge(files []string, output string) error {
	out, err := os.Create(output)
	if err != nil {
		return err
	}
	defer out.Close()
	for _, f := range files {
		in, err := os.Open(f)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			in.Close()
			return err
		}
		in.Close()
	}
	return nil
}

func binaryMergeWithMessage(opt Options, files []string, output string) error {
	fmt.Println(tr(opt, "binaryMerge"))
	return binaryMerge(files, output)
}

func liveRealtimeMergeFiles(files []string, segments []Segment, output string, keepSegments bool) error {
	out, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	for i, f := range files {
		in, err := os.Open(f)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := in.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if !keepSegments && !isLiveInitSegmentFile(f, segments, i) {
			_ = os.Remove(f)
		}
	}
	return nil
}

func isLiveInitSegmentFile(path string, segments []Segment, index int) bool {
	if index >= 0 && index < len(segments) && segments[index].Index == -1 {
		return true
	}
	return strings.HasPrefix(filepath.Base(path), "_init")
}

func outputExt(s StreamSpec, opt Options) string {
	if s.MediaType != nil && *s.MediaType == MediaSubtitles {
		if opt.AutoSubtitleFix {
			if strings.EqualFold(opt.SubFormat, "SRT") {
				return ".srt"
			}
			return ".vtt"
		}
		if strings.EqualFold(s.Extension, "vtt") || s.Extension == "" {
			return ".vtt"
		}
		return "." + s.Extension
	}
	if s.MediaType != nil && *s.MediaType == MediaAudio && (s.Extension == "m4s" || s.Extension == "mp4") {
		return ".m4a"
	}
	if s.Extension == "m4s" || s.Extension == "mp4" {
		return ".mp4"
	}
	if s.Extension != "" {
		return "." + s.Extension
	}
	return ".ts"
}

func liveRealtimeOutputExt(s StreamSpec, opt Options) string {
	if s.Extension == "" {
		return ".ts"
	}
	if s.MediaType != nil && *s.MediaType == MediaAudio && s.Extension == "m4s" {
		return ".m4a"
	}
	if (s.MediaType == nil || *s.MediaType != MediaSubtitles) && s.Extension == "m4s" {
		return ".mp4"
	}
	if s.MediaType != nil && *s.MediaType == MediaSubtitles {
		if strings.EqualFold(opt.SubFormat, "SRT") {
			return ".srt"
		}
		return ".vtt"
	}
	// long: 直播实时合并的后缀判断来自原版 SimpleLiveRecordManager2；音频 mp4 不像点播那样改成 m4a，而是保留原始 mp4。
	return "." + s.Extension
}

func singleTrackFFmpegFormat(s StreamSpec) string {
	if s.MediaType != nil && *s.MediaType == MediaAudio {
		return "m4a"
	}
	// long: 原版单轨 ffmpeg 合并不会沿用 TS/FLV 等分片扩展名；除音频外统一封装成 MP4，最终混流格式另由 -M 控制。
	return "mp4"
}

func defaultName(opt Options, s StreamSpec, fallback string) string {
	if opt.SavePattern != "" {
		return formatSavePattern(opt.SavePattern, s, opt.SaveName)
	}
	if opt.SaveName != "" {
		if s.Language != "" {
			return opt.SaveName + "." + s.Language
		}
		return opt.SaveName
	}
	return fallback
}

func formatSavePattern(pattern string, s StreamSpec, saveName string) string {
	name := pattern
	repl := map[string]string{
		"<SaveName>":   saveName,
		"<Id>":         strconv.Itoa(s.ID),
		"<Codecs>":     s.Codecs,
		"<Language>":   s.Language,
		"<Resolution>": s.Resolution,
		"<FrameRate>":  formatFloat(s.FrameRate),
		"<Bandwidth>":  strconv.Itoa(s.Bandwidth),
		"<MediaType>":  mediaTypePatternValue(s),
		"<Channels>":   s.Channels,
		"<VideoRange>": s.VideoRange,
		"<GroupId>":    s.GroupID,
		"<Ext>":        s.Extension,
	}
	for k, v := range repl {
		name = strings.ReplaceAll(name, k, v)
	}
	// long: 上游模板允许可选字段为空；替换后清掉相邻分隔符，避免生成 "__"、".."
	// 这样的尴尬文件名片段。
	name = strings.ReplaceAll(name, "__", "_")
	name = strings.ReplaceAll(name, "..", ".")
	name = strings.Trim(name, "_")
	name = strings.Trim(name, ".")
	return validSaveName(name)
}

func mediaTypePatternValue(s StreamSpec) string {
	if s.MediaType == nil {
		return ""
	}
	return string(*s.MediaType)
}

func applyDerivedDefaults(opt *Options, now time.Time) {
	if opt.SaveName != "" {
		return
	}
	opt.SaveName = deriveSaveNameFromInput(opt.Input, now)
}

func deriveSaveNameFromInput(input string, now time.Time) string {
	stamp := now.Format("2006-01-02_15-04-05")
	name := ""
	if _, err := os.Stat(input); err == nil {
		name = strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	} else if u, err := url.Parse(strings.Split(input, "?")[0]); err == nil {
		name = strings.TrimSuffix(filepath.Base(u.Path), filepath.Ext(u.Path))
	}
	name = validSaveName(name)
	if name == "" {
		return stamp
	}
	return name + "_" + stamp
}

func outputBaseName(name string, fallback string) string {
	out := validSaveName(name)
	if out == "" {
		return safeName(fallback)
	}
	return out
}

func formatFloat(v float64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func safeName(input string) string {
	r := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_", " ", "_")
	return strings.Trim(r.Replace(input), "._")
}

func collisionPath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; ; i++ {
		c := fmt.Sprintf("%s_%d%s", base, i, ext)
		if _, err := os.Stat(c); os.IsNotExist(err) {
			return c
		}
	}
}

func collisionPathForStream(path string, s StreamSpec) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)
	for _, name := range streamCollisionNames(base, ext, s) {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	output := path
	for {
		output = filepath.Join(filepath.Dir(output), strings.TrimSuffix(filepath.Base(output), filepath.Ext(output))+".copy"+filepath.Ext(output))
		if _, err := os.Stat(output); os.IsNotExist(err) {
			return output
		}
	}
}

func streamCollisionNames(base, ext string, s StreamSpec) []string {
	var names []string
	if s.MediaType == nil || *s.MediaType == MediaVideo {
		if s.Resolution != "" {
			names = append(names, base+"."+safeName(s.Resolution)+ext)
		}
		if s.Bandwidth > 0 {
			mbps := float64(s.Bandwidth) / 1000000
			names = append(names, base+"."+strconv.FormatFloat(mbps, 'f', 1, 64)+"Mbps"+ext)
		}
		if s.Resolution != "" && s.Bandwidth > 0 {
			mbps := float64(s.Bandwidth) / 1000000
			names = append(names, base+"."+safeName(s.Resolution)+"."+strconv.FormatFloat(mbps, 'f', 1, 64)+"Mbps"+ext)
		}
		return names
	}
	if *s.MediaType == MediaAudio {
		if s.Language != "" {
			names = append(names, base+"."+safeName(s.Language)+ext)
		}
		if s.Channels != "" {
			names = append(names, base+"."+safeName(s.Channels)+"ch"+ext)
		}
		if s.Language != "" && s.Channels != "" {
			names = append(names, base+"."+safeName(s.Language)+"."+safeName(s.Channels)+"ch"+ext)
		}
		if s.Bandwidth > 0 {
			names = append(names, base+"."+strconv.Itoa(s.Bandwidth/1000)+"kbps"+ext)
		}
		return names
	}
	if *s.MediaType == MediaSubtitles && s.Language != "" {
		names = append(names, base+"."+safeName(s.Language)+ext)
	}
	return names
}

func sortedSegments(pl *Playlist) []Segment {
	var segs []Segment
	if pl == nil {
		return segs
	}
	for _, p := range pl.Parts {
		segs = append(segs, p.Segments...)
	}
	sort.SliceStable(segs, func(i, j int) bool { return segs[i].Index < segs[j].Index })
	return segs
}
