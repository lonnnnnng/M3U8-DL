package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type vttCue struct {
	Start    time.Duration
	End      time.Duration
	ID       string
	Settings string
	Payload  string
}

type vttDocument struct {
	Cues   []vttCue
	MPEGTS *int64
}

func mergeVTTFiles(files []string, output string, skippedDuration float64, format string) error {
	return mergeVTTFilesWithSegments(files, nil, output, skippedDuration, format)
}

func mergeVTTFilesWithSegments(files []string, segments []Segment, output string, skippedDuration float64, format string) error {
	return mergeVTTFilesWithSegmentsAndOffset(files, segments, output, time.Duration(skippedDuration*float64(time.Second)), format)
}

func mergeVTTFilesWithSegmentsAndOffset(files []string, segments []Segment, output string, baseOffset time.Duration, format string) error {
	return mergeVTTFilesWithSegmentsAndOffsetOpt(files, segments, output, baseOffset, format, defaultOptions())
}

func mergeVTTFilesWithSegmentsAndOffsetOpt(files []string, segments []Segment, output string, baseOffset time.Duration, format string, opt Options) error {
	var cues []vttCue
	offsets := subtitleFileOffsets(segments, len(files))
	var baseMPEGTS *int64
	for i, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		doc, err := parseVTTDocument(string(b))
		if err != nil {
			return err
		}
		parsed := doc.Cues
		if doc.MPEGTS != nil {
			if baseMPEGTS == nil {
				baseMPEGTS = doc.MPEGTS
			} else if len(parsed) > 0 {
				offset := mpegtsOffsetDuration(*doc.MPEGTS, *baseMPEGTS)
				if shouldApplyVTTMPEGTSOffset(cues, parsed, offset) {
					// long: WebVTT 的 X-TIMESTAMP-MAP 使用 90kHz MPEGTS 时钟；上游以首段为基准修正后续片段，避免直播字幕每片都从 0 秒重叠。
					parsed = addCueOffset(parsed, offset)
				}
			}
		}
		if i < len(offsets) && offsets[i] > 0 {
			if doc.MPEGTS == nil {
				// long: 没有 X-TIMESTAMP-MAP 时，上游会用前序分片时长手动补一个 MPEGTS 时间轴，避免多段字幕互相重叠。
				parsed = addCueOffset(parsed, offsets[i])
			}
		}
		cues = append(cues, parsed...)
	}
	sort.SliceStable(cues, func(i, j int) bool {
		if cues[i].Start == cues[j].Start {
			return cues[i].End < cues[j].End
		}
		return cues[i].Start < cues[j].Start
	})
	var err error
	cues, err = writeImageSubtitleCuesOpt(cues, filepath.Dir(output), opt)
	if err != nil {
		return err
	}
	cues = compactCues(shiftCues(cues, baseOffset))
	var text string
	if strings.EqualFold(format, "SRT") {
		text = cuesToSRT(cues)
	} else {
		text = cuesToVTT(cues)
	}
	return os.WriteFile(output, []byte(text), 0644)
}

func writeImageSubtitleCues(cues []vttCue, dir string) ([]vttCue, error) {
	return writeImageSubtitleCuesOpt(cues, dir, defaultOptions())
}

func writeImageSubtitleCuesOpt(cues []vttCue, dir string, opt Options) ([]vttCue, error) {
	out := append([]vttCue(nil), cues...)
	nextIndex := 0
	printed := false
	for i := range out {
		if !strings.HasPrefix(out[i].Payload, "Base64::") {
			continue
		}
		if !printed {
			// long: 原版在首次发现 Base64 图形字幕时会给出资源化提示，便于用户知道接下来会额外生成 PNG 文件。
			fmt.Println(tr(opt, "processImageSub"))
			printed = true
		}
		img, err := base64.StdEncoding.DecodeString(strings.TrimSpace(strings.TrimPrefix(out[i].Payload, "Base64::")))
		if err != nil {
			return nil, err
		}
		name := ""
		path := ""
		for {
			name = fmt.Sprintf("%d.png", nextIndex)
			nextIndex++
			path = filepath.Join(dir, name)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				break
			}
		}
		// long: 上游会把图形字幕的 Base64 负载落成 PNG 文件，并让 VTT/SRT 引用文件名；这样图片字幕不会被当成普通文本写进字幕文件。
		if err := os.WriteFile(path, img, 0644); err != nil {
			return nil, err
		}
		out[i].Payload = name
	}
	return out, nil
}

func mpegtsOffsetDuration(mpegts, base int64) time.Duration {
	return time.Duration(float64(mpegts-base) / 90000 * float64(time.Second))
}

func shouldApplyVTTMPEGTSOffset(existing []vttCue, incoming []vttCue, offset time.Duration) bool {
	if offset <= 0 || len(incoming) == 0 {
		return false
	}
	if len(existing) == 0 {
		return incoming[0].Start < offset
	}
	last := existing[len(existing)-1]
	return incoming[0].Start < last.End && incoming[0].End != last.End && incoming[0].Start < offset
}

func subtitleFileOffsets(segments []Segment, fileCount int) []time.Duration {
	offsets := make([]time.Duration, fileCount)
	if len(segments) == 0 {
		return offsets
	}
	var elapsed time.Duration
	for i := 0; i < fileCount && i < len(segments); i++ {
		seg := segments[i]
		if seg.Index == -1 {
			offsets[i] = 0
			continue
		}
		offsets[i] = elapsed
		elapsed += time.Duration(seg.Duration * float64(time.Second))
	}
	return offsets
}

func addCueOffset(cues []vttCue, offset time.Duration) []vttCue {
	for i := range cues {
		cues[i].Start += offset
		cues[i].End += offset
	}
	return cues
}

func parseVTT(text string) ([]vttCue, error) {
	doc, err := parseVTTDocument(text)
	if err != nil {
		return nil, err
	}
	return doc.Cues, nil
}

func parseVTTDocument(text string) (vttDocument, error) {
	if !strings.HasPrefix(strings.TrimSpace(text), "WEBVTT") {
		return vttDocument{}, fmt.Errorf("Bad vtt")
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	doc := vttDocument{}
	var cues []vttCue
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "X-TIMESTAMP-MAP") {
			if mpegts := parseVTTMPEGTS(line); mpegts != nil {
				doc.MPEGTS = mpegts
			}
			continue
		}
		if !strings.Contains(line, " --> ") {
			continue
		}
		parts := strings.Fields(strings.Replace(line, "-->", " ", 1))
		if len(parts) < 2 {
			continue
		}
		start, err := parseCueTime(parts[0])
		if err != nil {
			return vttDocument{}, err
		}
		end, err := parseCueTime(parts[1])
		if err != nil {
			return vttDocument{}, err
		}
		settings := ""
		if len(parts) > 2 {
			settings = strings.Join(parts[2:], " ")
		}
		var payload []string
		for i++; i < len(lines); i++ {
			p := strings.TrimRight(lines[i], " \t")
			if strings.TrimSpace(p) == "" {
				break
			}
			payload = append(payload, strings.ReplaceAll(p, "\u200b", ""))
		}
		if len(payload) > 0 {
			cues = append(cues, vttCue{Start: start, End: end, Settings: settings, Payload: removeVTTClassTags(strings.Join(payload, "\n"))})
		}
	}
	doc.Cues = cues
	return doc, nil
}

func parseVTTMPEGTS(line string) *int64 {
	re := regexp.MustCompile(`MPEGTS:(\d+)`)
	m := re.FindStringSubmatch(line)
	if len(m) < 2 {
		return nil
	}
	v, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return nil
	}
	return &v
}

func parseCueTime(input string) (time.Duration, error) {
	input = strings.TrimSuffix(strings.ReplaceAll(input, ",", "."), "s")
	main, frac, _ := strings.Cut(input, ".")
	fields := strings.Split(main, ":")
	var total int64
	for _, f := range fields {
		v, err := strconv.ParseInt(f, 10, 64)
		if err != nil {
			return 0, err
		}
		total = total*60 + v
	}
	for len(frac) < 3 {
		frac += "0"
	}
	if len(frac) > 3 {
		frac = frac[:3]
	}
	ms, _ := strconv.ParseInt(frac, 10, 64)
	return time.Duration(total)*time.Second + time.Duration(ms)*time.Millisecond, nil
}

func shiftCues(cues []vttCue, shift time.Duration) []vttCue {
	for i := range cues {
		if cues[i].Start > shift {
			cues[i].Start -= shift
		} else {
			cues[i].Start = 0
		}
		if cues[i].End > shift {
			cues[i].End -= shift
		} else {
			cues[i].End = 0
		}
	}
	return cues
}

func compactCues(cues []vttCue) []vttCue {
	var out []vttCue
	for _, cue := range cues {
		if cue.Payload == "" {
			continue
		}
		if len(out) > 0 {
			last := &out[len(out)-1]
			if cue.Payload == last.Payload && cue.Start-last.End <= time.Millisecond {
				last.End = cue.End
				continue
			}
		}
		out = append(out, cue)
	}
	return out
}

func cuesToVTT(cues []vttCue) string {
	var b strings.Builder
	b.WriteString("WEBVTT\n\n")
	for _, cue := range cues {
		if cue.ID != "" {
			b.WriteString(cue.ID)
			b.WriteByte('\n')
		}
		b.WriteString(formatVTTTime(cue.Start, "."))
		b.WriteString(" --> ")
		b.WriteString(formatVTTTime(cue.End, "."))
		if cue.Settings != "" {
			b.WriteByte(' ')
			b.WriteString(cue.Settings)
		}
		b.WriteByte('\n')
		b.WriteString(cue.Payload)
		b.WriteString("\n\n")
	}
	return b.String()
}

func cuesToSRT(cues []vttCue) string {
	if len(cues) == 0 {
		// long: 上游会为完全空的字幕写一个 1 秒占位，避免某些播放器或后续混流工具把空 SRT 当作无效文件。
		return "1\r\n00:00:00,000 --> 00:00:01,000"
	}
	var b strings.Builder
	for i, cue := range cues {
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteByte('\n')
		b.WriteString(formatVTTTime(cue.Start, ","))
		b.WriteString(" --> ")
		b.WriteString(formatVTTTime(cue.End, ","))
		b.WriteByte('\n')
		b.WriteString(cue.Payload)
		b.WriteString("\n\n")
	}
	return b.String()
}

func formatVTTTime(d time.Duration, sep string) string {
	if d < 0 {
		d = 0
	}
	totalMS := d.Milliseconds()
	ms := totalMS % 1000
	totalSec := totalMS / 1000
	sec := totalSec % 60
	min := (totalSec / 60) % 60
	hour := totalSec / 3600
	return fmt.Sprintf("%02d:%02d:%02d%s%03d", hour, min, sec, sep, ms)
}

func removeVTTClassTags(text string) string {
	re := regexp.MustCompile(`<c\.[^>]*>([\s\S]*?)</c>`)
	return re.ReplaceAllString(text, "$1")
}
