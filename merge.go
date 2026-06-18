package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const partialMergeThreshold = 1800

func upstreamDateMetadata(t time.Time) string {
	// long: ffmpeg 的 date metadata 来自原版 DateTime.Now.ToString("o")；固定 7 位小数能保持 .NET tick 级 round-trip 形态。
	return t.Format("2006-01-02T15:04:05.0000000Z07:00")
}

func ffmpegMerge(files []string, outputBase, format string, opt Options, useAACFilter bool) (string, error) {
	if opt.FFmpegBinaryPath == "" {
		opt.FFmpegBinaryPath = "ffmpeg"
	}
	format = strings.ToLower(format)
	if format == "" {
		format = "mp4"
	}
	outputFormat := format
	if outputFormat == "aac" {
		outputFormat = "m4a"
	}
	output := outputBase + "." + outputFormat
	if outputFormat == "m4a" {
		output = outputBase + ".m4a"
	}
	output = collisionPath(output)
	workFiles := files
	if len(workFiles) >= partialMergeThreshold {
		var err error
		// long: 超长分片列表会让 ffmpeg concat 参数膨胀到系统限制，先按上游策略分批二进制合并成临时块再交给 ffmpeg。
		workFiles, err = partialCombineMultipleFiles(workFiles)
		if err != nil {
			return "", err
		}
	}
	args := []string{"-loglevel", "warning", "-nostdin", "-y"}
	workDir := filepath.Dir(workFiles[0])
	if opt.UseFFmpegConcatDemuxer {
		list := filepath.Join(workDir, "concat.txt")
		var b strings.Builder
		for _, f := range workFiles {
			b.WriteString("file '")
			b.WriteString(strings.ReplaceAll(f, "'", "'\\''"))
			b.WriteString("'\n")
		}
		if err := os.WriteFile(list, []byte(b.String()), 0644); err != nil {
			return "", err
		}
		args = append(args, "-f", "concat", "-safe", "0", "-i", list)
	} else {
		var rel []string
		for _, f := range workFiles {
			rel = append(rel, filepath.Base(f))
		}
		args = append(args, "-i", "concat:"+strings.Join(rel, "|"))
	}
	switch format {
	case "mp4":
		args = append(args, "-map", "0:v?", "-map", "0:a?", "-map", "0:s?", "-c", "copy")
		if !opt.NoDateInfo {
			// long: 原版单轨 ffmpeg 合并只在 MP4 分支写 date；其他容器不应被提前塞入 metadata，避免参数面和输出标签偏离上游。
			args = append(args, "-metadata", "date="+upstreamDateMetadata(time.Now()))
		}
		args = append(args,
			"-metadata", "encoding_tool=",
			"-metadata", "title=",
			"-metadata", "copyright=",
			"-metadata", "comment=",
			"-metadata:s:a:0", "title=",
			"-metadata:s:a:0", "handler=",
		)
		if useAACFilter {
			args = append(args, "-bsf:a", "aac_adtstoasc")
		}
		args = append(args, output)
	case "mkv":
		args = append(args, "-map", "0", "-c", "copy")
		if useAACFilter {
			args = append(args, "-bsf:a", "aac_adtstoasc")
		}
		args = append(args, output)
	case "flv":
		args = append(args, "-map", "0", "-c", "copy")
		if useAACFilter {
			args = append(args, "-bsf:a", "aac_adtstoasc")
		}
		args = append(args, output)
	case "ts":
		args = append(args, "-map", "0", "-c", "copy", "-f", "mpegts", "-bsf:v", "h264_mp4toannexb", output)
	case "m4a":
		args = append(args, "-map", "0", "-c", "copy", "-f", "mp4")
		if useAACFilter {
			args = append(args, "-bsf:a", "aac_adtstoasc")
		}
		args = append(args, output)
	case "aac":
		args = append(args, "-map", "0:a", "-c", "copy", output)
	case "eac3", "ac3":
		args = append(args, "-map", "0:a", "-c", "copy", output)
	default:
		return "", fmt.Errorf("暂不支持 ffmpeg 合并格式: %s", format)
	}
	cmd := exec.Command(opt.FFmpegBinaryPath, args...)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg 合并失败: %v\n%s", err, string(out))
	}
	return output, nil
}

func partialCombineMultipleFiles(files []string) ([]string, error) {
	if len(files) == 0 {
		return nil, nil
	}
	div := 100
	if len(files) > 90000 {
		div = 200
	}
	dir := filepath.Dir(files[0])
	var out []string
	for start, index := 0, 0; start < len(files); start, index = start+div, index+1 {
		end := start + div
		if end > len(files) {
			end = len(files)
		}
		output := filepath.Join(dir, fmt.Sprintf("T%04d.ts", index))
		if err := binaryMerge(files[start:end], output); err != nil {
			return nil, err
		}
		out = append(out, output)
		for _, file := range files[start:end] {
			_ = os.Remove(file)
		}
	}
	return out, nil
}

func muxOutputs(files []outputFile, opt Options) (string, error) {
	if opt.MuxAfterDone == nil {
		return "", nil
	}
	mux := opt.MuxAfterDone
	var inputs []outputFile
	for _, f := range files {
		if mux.SkipSubtitle && f.MediaType != nil && *f.MediaType == MediaSubtitles {
			continue
		}
		inputs = append(inputs, f)
	}
	for _, raw := range opt.MuxImports {
		p := splitComplex(raw)
		path := p["path"]
		if path == "" {
			path = raw
		}
		inputs = append(inputs, outputFile{Path: path, Language: p["lang"], Name: p["name"]})
	}
	for i := range inputs {
		inputs[i] = normalizeOutputFileLanguage(inputs[i])
		if inputs[i].Language == "" {
			// long: HLS 主列表常省略基础视频语言；上游最终混流会把缺失的 LangCode 写成 und，避免输出轨道没有语言 metadata。
			inputs[i].Language = "und"
		}
		if inputs[i].StreamCount <= 0 {
			inputs[i].StreamCount = probeMediaStreamCount(inputs[i].Path, opt)
		}
	}
	if len(inputs) == 0 {
		return "", nil
	}
	if mux.Muxer == "mkvmerge" {
		return muxOutputsByMkvmerge(inputs, opt)
	}
	saveDir := opt.SaveDir
	if saveDir == "" {
		saveDir = "."
	}
	outName := opt.SaveName
	if outName == "" {
		outName = time.Now().Format("2006-01-02_15-04-05")
	}
	format := mux.Format
	if format == "" {
		format = "mp4"
	}
	output := collisionPath(filepath.Join(saveDir, safeName(outName)+".MUX."+format))
	args := []string{"-loglevel", "warning", "-nostdin", "-y", "-dn"}
	for _, f := range inputs {
		args = append(args, "-i", f.Path)
	}
	for i := range inputs {
		args = append(args, "-map", fmt.Sprintf("%d", i))
	}
	switch strings.ToLower(format) {
	case "mp4":
		args = append(args, "-strict", "unofficial", "-c:a", "copy", "-c:v", "copy", "-c:s", "mov_text")
	case "mkv":
		args = append(args, "-strict", "unofficial", "-c:a", "copy", "-c:v", "copy", "-c:s", muxSubtitleCodec(inputs))
	case "ts":
		args = append(args, "-strict", "unofficial", "-c:a", "copy", "-c:v", "copy")
	default:
		return "", fmt.Errorf("未知混流格式: %s", format)
	}
	// long: 最终混流应以本次选择的轨道信息为准，上游会清理输入文件旧 metadata 并复制未知流，避免源文件遗留标签污染输出。
	args = append(args, "-map_metadata", "-1")
	if !opt.NoDateInfo {
		args = append(args, "-metadata", "date="+upstreamDateMetadata(time.Now()))
	}
	streamIndex := 0
	for _, f := range inputs {
		if f.Language != "" {
			args = append(args, fmt.Sprintf("-metadata:s:%d", streamIndex), "language="+f.Language)
		}
		if f.Name != "" {
			args = append(args, fmt.Sprintf("-metadata:s:%d", streamIndex), "title="+f.Name)
		}
		if f.StreamCount > 0 {
			streamIndex += f.StreamCount
		} else {
			streamIndex++
		}
	}
	args = append(args, muxDispositionArgs(inputs)...)
	args = append(args, "-ignore_unknown", "-copy_unknown")
	args = append(args, output)
	bin := muxFFmpegBinary(opt)
	if bin == "" {
		bin = "ffmpeg"
	}
	cmd := exec.Command(bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("最终混流失败: %v\n%s", err, string(out))
	}
	if !mux.Keep {
		for _, f := range inputs {
			// long: 上游在最终混流时会先应用 skip_sub、再追加 mux-import，清理阶段只删除实际参与混流的轨道；这样被 skip_sub 跳过的字幕会保留，而外部导入轨会按 keep=false 一并清理。
			_ = os.Remove(f.Path)
		}
	}
	return output, nil
}

func muxSubtitleCodec(inputs []outputFile) string {
	for _, input := range inputs {
		if strings.EqualFold(filepath.Ext(input.Path), ".srt") {
			// long: MKV 最终混流时上游会把 SRT 和 WebVTT 明确写成对应字幕编码，避免 ffmpeg 对外挂字幕 copy 时产生容器兼容性差异。
			return "srt"
		}
	}
	return "webvtt"
}

func muxFFmpegBinary(opt Options) string {
	if opt.MuxAfterDone != nil && opt.MuxAfterDone.BinPath != "" && opt.MuxAfterDone.BinPath != "auto" {
		// long: -M 的 bin_path 是本次最终混流的专用工具路径，应覆盖全局 ffmpeg，方便用户为混流单独指定封装器版本。
		return opt.MuxAfterDone.BinPath
	}
	return opt.FFmpegBinaryPath
}

func probeMediaStreamCount(path string, opt Options) int {
	bin := ffprobeBinary(opt)
	if bin == "" {
		return 1
	}
	cmd := exec.Command(bin, "-v", "error", "-show_entries", "stream=index", "-of", "csv=p=0", path)
	out, err := cmd.Output()
	if err != nil {
		return 1
	}
	count := 0
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	if count < 1 {
		return 1
	}
	return count
}

func ffprobeBinary(opt Options) string {
	candidates := []string{}
	if opt.FFmpegBinaryPath != "" {
		dir := filepath.Dir(opt.FFmpegBinaryPath)
		base := filepath.Base(opt.FFmpegBinaryPath)
		if strings.HasPrefix(base, "ffmpeg") && dir != "." {
			candidates = append(candidates, filepath.Join(dir, strings.Replace(base, "ffmpeg", "ffprobe", 1)))
		}
	}
	candidates = append(candidates, "ffprobe")
	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	return ""
}

func muxDispositionArgs(inputs []outputFile) []string {
	var args []string
	hasVideo := false
	hasAudio := false
	audioIndex := 0
	for _, f := range inputs {
		if f.MediaType == nil || *f.MediaType == MediaVideo {
			hasVideo = true
			continue
		}
		if *f.MediaType == MediaSubtitles {
			continue
		}
		if *f.MediaType == MediaAudio {
			hasAudio = true
			if audioIndex == 0 {
				args = append(args, "-disposition:a:0", "default")
			} else {
				args = append(args, fmt.Sprintf("-disposition:a:%d", audioIndex), "0")
			}
			audioIndex++
		}
	}
	if hasVideo {
		args = append(args, "-disposition:v:0", "default")
	}
	if hasAudio {
		// long: 原版 MuxInputsByFFmpeg 的 subTracks 误用 AUDIO 过滤条件；这会在有音频时追加字幕 disposition，即使当前没有字幕轨。
		args = append(args, "-disposition:s", "0")
	}
	return args
}

func muxOutputsByMkvmerge(inputs []outputFile, opt Options) (string, error) {
	saveDir := opt.SaveDir
	if saveDir == "" {
		saveDir = "."
	}
	outName := opt.SaveName
	if outName == "" {
		outName = time.Now().Format("2006-01-02_15-04-05")
	}
	output := collisionPath(filepath.Join(saveDir, safeName(outName)+".MUX.mkv"))
	bin := opt.MuxAfterDone.BinPath
	if bin == "" {
		bin = firstExecutable("mkvmerge")
	}
	if bin == "" {
		return "", fmt.Errorf("找不到 mkvmerge，请设置 bin_path")
	}
	args := []string{"-q", "--output", output, "--no-chapters"}
	audioDefaultSet := false
	for _, f := range inputs {
		f = normalizeOutputFileLanguage(f)
		lang := f.Language
		if lang == "" {
			lang = "und"
		}
		args = append(args, "--language", "0:"+lang)
		if f.MediaType != nil && *f.MediaType == MediaSubtitles {
			args = append(args, "--default-track", "0:no")
		}
		if f.MediaType != nil && *f.MediaType == MediaAudio {
			if audioDefaultSet {
				args = append(args, "--default-track", "0:no")
			}
			audioDefaultSet = true
		}
		if f.Name != "" {
			args = append(args, "--track-name", "0:"+f.Name)
		}
		args = append(args, f.Path)
	}
	cmd := exec.Command(bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("mkvmerge 混流失败: %v\n%s", err, string(out))
	}
	if opt.MuxAfterDone != nil && !opt.MuxAfterDone.Keep {
		for _, f := range inputs {
			_ = os.Remove(f.Path)
		}
	}
	return output, nil
}
