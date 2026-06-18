package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var widevineSystemID = []byte{0xED, 0xEF, 0x8B, 0xA9, 0x79, 0xD6, 0x4A, 0xCE, 0xA3, 0xC8, 0x27, 0xDC, 0xD5, 0x1D, 0x21, 0xED}
var playReadySystemID = []byte{0x9A, 0x04, 0xF0, 0x79, 0x98, 0x40, 0x42, 0x86, 0xAB, 0x92, 0xE6, 0x5B, 0xE0, 0x88, 0x5F, 0x95}

type mp4Box struct {
	Type    string
	Payload []byte
}

func readMP4Boxes(data []byte) []mp4Box {
	var boxes []mp4Box
	for off := 0; off+8 <= len(data); {
		size := uint64(binary.BigEndian.Uint32(data[off : off+4]))
		typ := string(data[off+4 : off+8])
		header := uint64(8)
		if size == 1 {
			if off+16 > len(data) {
				break
			}
			size = binary.BigEndian.Uint64(data[off+8 : off+16])
			header = 16
		} else if size == 0 {
			size = uint64(len(data) - off)
		}
		if size < header || off+int(size) > len(data) {
			break
		}
		boxes = append(boxes, mp4Box{Type: typ, Payload: data[off+int(header) : off+int(size)]})
		off += int(size)
	}
	return boxes
}

func findMP4Boxes(data []byte, target string) []mp4Box {
	var out []mp4Box
	for _, box := range readMP4Boxes(data) {
		if box.Type == target {
			out = append(out, box)
		}
		payload := box.Payload
		if box.Type == "stsd" && len(payload) >= 8 && binary.BigEndian.Uint32(payload[:4]) == 0 && binary.BigEndian.Uint32(payload[4:8]) > 0 {
			// long: 真实 MP4 的 stsd 是 FullBox，前 8 字节不是子 box；跳过后才能发现 stpp/wvtt/encv 等 sample entry。
			payload = payload[8:]
		}
		switch box.Type {
		case "moov", "trak", "mdia", "minf", "stbl", "moof", "traf", "stsd", "encv", "enca", "enct", "encs", "sinf", "schi", "vttc":
			out = append(out, findMP4Boxes(payload, target)...)
		}
	}
	return out
}

func mp4FilesContainBox(files []string, typ string) bool {
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		if len(findMP4Boxes(b, typ)) > 0 {
			return true
		}
	}
	return false
}

func extractDefaultKID(data []byte) string {
	kid, _ := extractDefaultKIDInfo(data)
	return kid
}

func extractDefaultKIDInfo(data []byte) (string, bool) {
	zeroTencKID := false
	for _, box := range findMP4Boxes(data, "tenc") {
		if len(box.Payload) >= 20 {
			// long: 上游从 tenc 的 full-box 负载中读取 default_KID；保留这个定位规则能兼容同一批 CENC 初始化段。
			kid := hex.EncodeToString(box.Payload[8:24])
			if !strings.EqualFold(kid, zeroKID) {
				return kid, false
			}
			zeroTencKID = true
			break
		}
	}
	if kid := extractWidevineKIDFromPSSH(data); kid != "" {
		return kid, zeroTencKID
	}
	if kid := extractPlayReadyKIDFromPSSH(data); kid != "" {
		return kid, false
	}
	if zeroTencKID {
		return zeroKID, false
	}
	idx := strings.Index(string(data), "tenc")
	if idx >= 0 && idx+28 <= len(data) {
		kid := hex.EncodeToString(data[idx+12 : idx+28])
		if !strings.EqualFold(kid, zeroKID) {
			return kid, false
		}
		if psshKid := extractWidevineKIDFromPSSH(data); psshKid != "" {
			return psshKid, true
		}
		if psshKid := extractPlayReadyKIDFromPSSH(data); psshKid != "" {
			return psshKid, false
		}
		return zeroKID, false
	}
	return "", false
}

func extractPlayReadyKIDFromPSSH(data []byte) string {
	for _, box := range findMP4Boxes(data, "pssh") {
		if len(box.Payload) < 24 {
			continue
		}
		systemID := box.Payload[4:20]
		if !bytes.Equal(systemID, playReadySystemID) {
			continue
		}
		pos := 20
		if box.Payload[0] == 1 {
			if len(box.Payload) < pos+4 {
				continue
			}
			count := int(binary.BigEndian.Uint32(box.Payload[pos : pos+4]))
			pos += 4 + count*16
			if len(box.Payload) < pos {
				continue
			}
		}
		if len(box.Payload) < pos+4 {
			continue
		}
		size := int(binary.BigEndian.Uint32(box.Payload[pos : pos+4]))
		pos += 4
		if size <= 0 || len(box.Payload) < pos+size {
			continue
		}
		if kid := extractPlayReadyKIDFromData(box.Payload[pos : pos+size]); kid != "" {
			return kid
		}
	}
	return ""
}

func extractPlayReadyKIDFromData(data []byte) string {
	text := stripZeroBytes(data)
	rawText := extractPlayReadyKIDText(text)
	if rawText == "" {
		return ""
	}
	rawKID, err := base64.StdEncoding.DecodeString(strings.TrimSpace(rawText))
	if err != nil || len(rawKID) != 16 {
		return ""
	}
	// long: PlayReady XML 中的 KID 使用 GUID 小端字节序，上游会翻转前三段后再作为 CENC KID 匹配密钥文件。
	kid := append([]byte{}, rawKID...)
	kid[0], kid[1], kid[2], kid[3] = rawKID[3], rawKID[2], rawKID[1], rawKID[0]
	kid[4], kid[5], kid[6], kid[7] = rawKID[5], rawKID[4], rawKID[7], rawKID[6]
	return hex.EncodeToString(kid)
}

func extractPlayReadyKIDText(text string) string {
	if start := strings.Index(text, "<KID>"); start >= 0 {
		start += len("<KID>")
		end := strings.Index(text[start:], "<")
		if end >= 0 {
			return strings.TrimSpace(text[start : start+end])
		}
	}
	upper := strings.ToUpper(text)
	tagStart := strings.Index(upper, "<KID")
	if tagStart < 0 {
		return ""
	}
	tagEnd := strings.Index(text[tagStart:], ">")
	if tagEnd < 0 {
		return ""
	}
	tag := text[tagStart : tagStart+tagEnd]
	if value := xmlAttributeValue(tag, "VALUE"); value != "" {
		// long: PlayReady WRMHEADER 新版本常把 KID 放在 VALUE 属性里；这里抽出属性值后复用同一套 GUID 字节序修正逻辑。
		return strings.TrimSpace(value)
	}
	return ""
}

func xmlAttributeValue(tag string, name string) string {
	upperTag := strings.ToUpper(tag)
	needle := strings.ToUpper(name) + "="
	pos := strings.Index(upperTag, needle)
	if pos < 0 {
		return ""
	}
	pos += len(needle)
	if pos >= len(tag) {
		return ""
	}
	quote := tag[pos]
	if quote != '\'' && quote != '"' {
		return ""
	}
	start := pos + 1
	end := strings.IndexByte(tag[start:], quote)
	if end < 0 {
		return ""
	}
	return tag[start : start+end]
}

func stripZeroBytes(data []byte) string {
	out := make([]byte, 0, len(data))
	for _, b := range data {
		if b != 0 {
			out = append(out, b)
		}
	}
	return string(out)
}

func extractWidevineKIDFromPSSH(data []byte) string {
	for _, box := range findMP4Boxes(data, "pssh") {
		if len(box.Payload) < 24 {
			continue
		}
		version := box.Payload[0]
		systemID := box.Payload[4:20]
		if !bytes.Equal(systemID, widevineSystemID) {
			continue
		}
		pos := 20
		if version == 1 {
			if len(box.Payload) < pos+4 {
				continue
			}
			count := int(binary.BigEndian.Uint32(box.Payload[pos : pos+4]))
			pos += 4
			if len(box.Payload) < pos+count*16 {
				continue
			}
			if count > 0 {
				return hex.EncodeToString(box.Payload[pos : pos+16])
			}
			pos += count * 16
		}
		if len(box.Payload) < pos+4 {
			continue
		}
		size := int(binary.BigEndian.Uint32(box.Payload[pos : pos+4]))
		pos += 4
		if size <= 0 || len(box.Payload) < pos+size {
			continue
		}
		psshData := box.Payload[pos : pos+size]
		for i := 0; i+18 <= len(psshData); i++ {
			if psshData[i] == 0x12 && psshData[i+1] == 0x10 {
				return hex.EncodeToString(psshData[i+2 : i+18])
			}
		}
		if len(psshData) >= 18 {
			// long: 部分老数据缺少标准 protobuf key_id 标记，保留上游 psshData[2:18] 的兜底兼容策略。
			return hex.EncodeToString(psshData[2:18])
		}
	}
	return ""
}

func extractDefaultKIDFromFile(path string) string {
	kid, _ := extractDefaultKIDInfoFromFile(path)
	return kid
}

func extractDefaultKIDInfoFromFile(path string) (string, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	if len(b) > 1024*1024 {
		b = b[:1024*1024]
	}
	return extractDefaultKIDInfo(b)
}

func extractMP4WebVTTFiles(files []string, output string, format string) (bool, error) {
	return extractMP4WebVTTFilesWithSegments(files, nil, output, 0, format)
}

func extractMP4WebVTTFilesWithSegments(files []string, segments []Segment, output string, skippedDuration float64, format string) (bool, error) {
	var cues []vttCue
	timescale := mp4WebVTTTimescale(files)
	offsets := subtitleFileOffsets(segments, len(files))
	for i, file := range files {
		if filepath.Base(file) == "_init.mp4" || strings.Contains(filepath.Base(file), "init") {
			continue
		}
		b, err := os.ReadFile(file)
		if err != nil {
			return false, err
		}
		extracted, timed, err := extractMP4WebVTTCuesWithTiming(b, timescale)
		if err != nil {
			return false, err
		}
		if !timed {
			extracted = extractMP4WebVTTCues(b, len(cues))
			if i < len(offsets) && offsets[i] > 0 {
				// long: 没有 MP4 时间盒时保留旧的宽松抽取，同时借用 HLS 分片时长尽量恢复跨片字幕顺序。
				extracted = addCueOffset(extracted, offsets[i])
			}
		}
		cues = append(cues, extracted...)
	}
	if len(cues) == 0 {
		return false, nil
	}
	var err error
	cues, err = writeImageSubtitleCues(cues, filepath.Dir(output))
	if err != nil {
		return false, err
	}
	cues = compactCues(shiftCues(cues, time.Duration(skippedDuration*float64(time.Second))))
	var text string
	if strings.EqualFold(format, "SRT") {
		text = cuesToSRT(cues)
	} else {
		text = cuesToVTT(cues)
	}
	return true, os.WriteFile(output, []byte(text), 0644)
}

func mp4WebVTTTimescale(files []string) uint32 {
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		sawWVTT, timescale := parseMP4WebVTTInit(b)
		if sawWVTT && timescale != 0 {
			return timescale
		}
	}
	return 0
}

func parseMP4WebVTTInit(data []byte) (bool, uint32) {
	sawWVTT := len(findMP4Boxes(data, "wvtt")) > 0
	var timescale uint32
	for _, box := range findMP4Boxes(data, "mdhd") {
		if len(box.Payload) < 16 {
			continue
		}
		version := box.Payload[0]
		switch version {
		case 0:
			if len(box.Payload) >= 16 {
				timescale = binary.BigEndian.Uint32(box.Payload[12:16])
			}
		case 1:
			if len(box.Payload) >= 24 {
				timescale = binary.BigEndian.Uint32(box.Payload[20:24])
			}
		}
	}
	return sawWVTT, timescale
}

type mp4WebVTTSample struct {
	Duration              uint32
	Size                  uint32
	CompositionTimeOffset int64
}

func extractMP4WebVTTCuesWithTiming(data []byte, timescale uint32) ([]vttCue, bool, error) {
	if timescale == 0 {
		return nil, false, nil
	}
	baseTime, sawTFDT, err := parseMP4TFDT(data)
	if err != nil {
		return nil, false, err
	}
	defaultDuration, err := parseMP4TFHDDefaultDuration(data)
	if err != nil {
		return nil, false, err
	}
	samples, sawTRUN, err := parseMP4TRUNSamples(data)
	if err != nil {
		return nil, false, err
	}
	mdats := findMP4Boxes(data, "mdat")
	if !sawTFDT || !sawTRUN || len(mdats) == 0 {
		return nil, false, nil
	}
	if len(mdats) > 1 {
		return nil, true, fmt.Errorf("VTT cues in mp4 with multiple MDAT are not currently supported")
	}
	cues, err := parseMP4WebVTTSamples(mdats[0].Payload, samples, baseTime, defaultDuration, timescale)
	return cues, true, err
}

func parseMP4TFDT(data []byte) (uint64, bool, error) {
	for _, box := range findMP4Boxes(data, "tfdt") {
		if len(box.Payload) < 8 {
			return 0, true, fmt.Errorf("invalid tfdt box")
		}
		version := box.Payload[0]
		if version == 1 {
			if len(box.Payload) < 12 {
				return 0, true, fmt.Errorf("invalid tfdt version 1 box")
			}
			return binary.BigEndian.Uint64(box.Payload[4:12]), true, nil
		}
		if version != 0 {
			return 0, true, fmt.Errorf("TFDT version can only be 0 or 1")
		}
		return uint64(binary.BigEndian.Uint32(box.Payload[4:8])), true, nil
	}
	return 0, false, nil
}

func parseMP4TFHDDefaultDuration(data []byte) (uint32, error) {
	var defaultDuration uint32
	for _, box := range findMP4Boxes(data, "tfhd") {
		if len(box.Payload) < 8 {
			return 0, fmt.Errorf("invalid tfhd box")
		}
		flags := fullBoxFlags(box.Payload)
		pos := 8
		if flags&0x000001 != 0 {
			pos += 8
		}
		if flags&0x000002 != 0 {
			pos += 4
		}
		if flags&0x000008 != 0 {
			if len(box.Payload) < pos+4 {
				return 0, fmt.Errorf("invalid tfhd default sample duration")
			}
			defaultDuration = binary.BigEndian.Uint32(box.Payload[pos : pos+4])
		}
	}
	return defaultDuration, nil
}

func parseMP4TRUNSamples(data []byte) ([]mp4WebVTTSample, bool, error) {
	var samples []mp4WebVTTSample
	sawTRUN := false
	for _, box := range findMP4Boxes(data, "trun") {
		sawTRUN = true
		if len(box.Payload) < 8 {
			return nil, true, fmt.Errorf("invalid trun box")
		}
		version := box.Payload[0]
		if version != 0 && version != 1 {
			return nil, true, fmt.Errorf("TRUN version can only be 0 or 1")
		}
		flags := fullBoxFlags(box.Payload)
		count := int(binary.BigEndian.Uint32(box.Payload[4:8]))
		pos := 8
		if flags&0x000001 != 0 {
			pos += 4
		}
		if flags&0x000004 != 0 {
			pos += 4
		}
		for i := 0; i < count; i++ {
			var sample mp4WebVTTSample
			if flags&0x000100 != 0 {
				if len(box.Payload) < pos+4 {
					return nil, true, fmt.Errorf("invalid trun sample duration")
				}
				sample.Duration = binary.BigEndian.Uint32(box.Payload[pos : pos+4])
				pos += 4
			}
			if flags&0x000200 != 0 {
				if len(box.Payload) < pos+4 {
					return nil, true, fmt.Errorf("invalid trun sample size")
				}
				sample.Size = binary.BigEndian.Uint32(box.Payload[pos : pos+4])
				pos += 4
			}
			if flags&0x000400 != 0 {
				pos += 4
			}
			if flags&0x000800 != 0 {
				if len(box.Payload) < pos+4 {
					return nil, true, fmt.Errorf("invalid trun composition time offset")
				}
				if version == 0 {
					sample.CompositionTimeOffset = int64(binary.BigEndian.Uint32(box.Payload[pos : pos+4]))
				} else {
					sample.CompositionTimeOffset = int64(int32(binary.BigEndian.Uint32(box.Payload[pos : pos+4])))
				}
				pos += 4
			}
			if len(box.Payload) < pos {
				return nil, true, fmt.Errorf("invalid trun sample table")
			}
			samples = append(samples, sample)
		}
	}
	return samples, sawTRUN, nil
}

func parseMP4WebVTTSamples(payload []byte, samples []mp4WebVTTSample, baseTime uint64, defaultDuration uint32, timescale uint32) ([]vttCue, error) {
	var cues []vttCue
	currentTime := baseTime
	reader := payload
	for _, sample := range samples {
		duration := sample.Duration
		if duration == 0 {
			duration = defaultDuration
		}
		if duration == 0 {
			return nil, fmt.Errorf("WVTT sample duration unknown, and no default found")
		}
		startTime := currentTime
		if sample.CompositionTimeOffset != 0 {
			if sample.CompositionTimeOffset < 0 && uint64(-sample.CompositionTimeOffset) > baseTime {
				startTime = 0
			} else if sample.CompositionTimeOffset < 0 {
				startTime = baseTime - uint64(-sample.CompositionTimeOffset)
			} else {
				startTime = baseTime + uint64(sample.CompositionTimeOffset)
			}
		}
		endTime := startTime + uint64(duration)
		currentTime = endTime

		totalSize := uint32(0)
		for {
			if len(reader) < 8 {
				return nil, fmt.Errorf("invalid mdat VTT payload")
			}
			size := binary.BigEndian.Uint32(reader[:4])
			if size < 8 || int(size) > len(reader) {
				return nil, fmt.Errorf("invalid mdat VTT sample size")
			}
			boxType := string(reader[4:8])
			boxPayload := reader[8:size]
			reader = reader[size:]
			totalSize += size
			if boxType == "vttc" {
				if cue := parseMP4VTTCBox(boxPayload, startTime, endTime, timescale); cue != nil {
					cues = appendOrExtendCue(cues, *cue)
				}
			}
			if sample.Size == 0 || totalSize >= sample.Size {
				break
			}
		}
		if sample.Size != 0 && totalSize > sample.Size {
			return nil, fmt.Errorf("the samples do not fit evenly into the sample sizes given in the TRUN box")
		}
	}
	return cues, nil
}

func parseMP4VTTCBox(payload []byte, startTime uint64, endTime uint64, timescale uint32) *vttCue {
	cue := vttCue{Start: mp4TicksToDuration(startTime, timescale), End: mp4TicksToDuration(endTime, timescale)}
	for _, child := range readMP4Boxes(payload) {
		switch child.Type {
		case "iden":
			cue.ID = string(child.Payload)
		case "payl":
			cue.Payload = string(child.Payload)
		case "sttg":
			cue.Settings = string(child.Payload)
		}
	}
	if cue.Payload == "" {
		return nil
	}
	return &cue
}

func appendOrExtendCue(cues []vttCue, cue vttCue) []vttCue {
	for i := len(cues) - 1; i >= 0; i-- {
		if cues[i].End == cue.Start && cues[i].Settings == cue.Settings && cues[i].Payload == cue.Payload {
			cues[i].End = cue.End
			return cues
		}
	}
	return append(cues, cue)
}

func fullBoxFlags(payload []byte) uint32 {
	if len(payload) < 4 {
		return 0
	}
	return uint32(payload[1])<<16 | uint32(payload[2])<<8 | uint32(payload[3])
}

func mp4TicksToDuration(ticks uint64, timescale uint32) time.Duration {
	if timescale == 0 {
		return 0
	}
	return time.Duration(float64(ticks) / float64(timescale) * float64(time.Second))
}

func extractMP4WebVTTCues(data []byte, offsetIndex int) []vttCue {
	var cues []vttCue
	mdats := findMP4Boxes(data, "mdat")
	if len(mdats) == 0 {
		mdats = []mp4Box{{Type: "mdat", Payload: data}}
	}
	for _, mdat := range mdats {
		for _, vttc := range findMP4Boxes(mdat.Payload, "vttc") {
			id := ""
			payload := ""
			settings := ""
			for _, child := range readMP4Boxes(vttc.Payload) {
				switch child.Type {
				case "iden":
					id = string(child.Payload)
				case "payl":
					payload = string(child.Payload)
				case "sttg":
					settings = string(child.Payload)
				}
			}
			if payload == "" {
				continue
			}
			start := time.Duration(offsetIndex+len(cues)) * time.Second
			cues = append(cues, vttCue{Start: start, End: start + time.Second, ID: id, Settings: settings, Payload: payload})
		}
	}
	return cues
}

func makeMP4Box(typ string, payload []byte) []byte {
	size := 8 + len(payload)
	out := make([]byte, size)
	binary.BigEndian.PutUint32(out[:4], uint32(size))
	copy(out[4:8], typ)
	copy(out[8:], payload)
	return out
}

func mustMP4Box(typ string, payload []byte) []byte {
	if len(typ) != 4 {
		panic(fmt.Sprintf("invalid mp4 box type %q", typ))
	}
	return makeMP4Box(typ, payload)
}
