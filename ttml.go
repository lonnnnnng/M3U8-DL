package main

import (
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

func mergeTTMLFiles(files []string, output string, skippedDuration float64, format string) error {
	return mergeTTMLFilesWithSegments(files, nil, output, skippedDuration, format)
}

func mergeTTMLFilesWithSegments(files []string, segments []Segment, output string, skippedDuration float64, format string) error {
	return mergeTTMLFilesWithSegmentsOpt(files, segments, output, skippedDuration, format, defaultOptions())
}

func mergeTTMLFilesWithSegmentsOpt(files []string, segments []Segment, output string, skippedDuration float64, format string, opt Options) error {
	var cues []vttCue
	offsets := subtitleFileOffsets(segments, len(files))
	for i, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		parsed := parseTTMLCues(string(b))
		if i < len(offsets) && offsets[i] > 0 {
			parsed = addCueOffset(parsed, offsets[i])
		}
		cues = append(cues, parsed...)
	}
	return writeSubtitleCuesOpt(cues, output, skippedDuration, format, opt)
}

func extractMP4TTMLFiles(files []string, output string, skippedDuration float64, format string) (bool, error) {
	return extractMP4TTMLFilesWithSegments(files, nil, output, skippedDuration, format)
}

func extractMP4TTMLFilesWithSegments(files []string, segments []Segment, output string, skippedDuration float64, format string) (bool, error) {
	return extractMP4TTMLFilesWithSegmentsOpt(files, segments, output, skippedDuration, format, defaultOptions())
}

func extractMP4TTMLFilesWithSegmentsOpt(files []string, segments []Segment, output string, skippedDuration float64, format string, opt Options) (bool, error) {
	var cues []vttCue
	offsets := subtitleFileOffsets(segments, len(files))
	for i, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			return false, err
		}
		var fileCues []vttCue
		for _, mdat := range findMP4Boxes(b, "mdat") {
			fileCues = append(fileCues, parseTTMLCues(string(mdat.Payload))...)
		}
		if i < len(offsets) && offsets[i] > 0 {
			fileCues = addCueOffset(fileCues, offsets[i])
		}
		cues = append(cues, fileCues...)
	}
	if len(cues) == 0 {
		return false, nil
	}
	return true, writeSubtitleCuesOpt(cues, output, skippedDuration, format, opt)
}

func parseTTMLCues(text string) []vttCue {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	rootRe := regexp.MustCompile(`(?s)<tt[\s\S]*?</tt>`)
	var roots []string
	if rootRe.MatchString(text) {
		for _, m := range rootRe.FindAllString(text, -1) {
			roots = append(roots, m)
		}
	} else {
		roots = []string{text}
	}
	pRe := regexp.MustCompile(`(?s)<p\b([^>]*)>(.*?)</p>`)
	attrRe := regexp.MustCompile(`\b(begin|end)="([^"]+)"`)
	tagRe := regexp.MustCompile(`(?s)<[^>]+>`)
	var cues []vttCue
	for _, root := range roots {
		for _, m := range pRe.FindAllStringSubmatch(root, -1) {
			attrs := map[string]string{}
			for _, a := range attrRe.FindAllStringSubmatch(m[1], -1) {
				attrs[a[1]] = a[2]
			}
			begin, bok := attrs["begin"]
			end, eok := attrs["end"]
			if !bok || !eok {
				continue
			}
			start, err := parseCueTime(begin)
			if err != nil {
				continue
			}
			finish, err := parseCueTime(end)
			if err != nil {
				continue
			}
			payload := strings.ReplaceAll(m[2], "<br/>", "\n")
			payload = strings.ReplaceAll(payload, "<br />", "\n")
			payload = strings.ReplaceAll(payload, "<br>", "\n")
			payload = tagRe.ReplaceAllString(payload, "")
			payload = strings.TrimSpace(html.UnescapeString(payload))
			if payload != "" {
				cues = append(cues, vttCue{Start: start, End: finish, Payload: payload})
			}
		}
	}
	sort.SliceStable(cues, func(i, j int) bool { return cues[i].Start < cues[j].Start })
	return cues
}

func writeSubtitleCues(cues []vttCue, output string, skippedDuration float64, format string) error {
	return writeSubtitleCuesOpt(cues, output, skippedDuration, format, defaultOptions())
}

func writeSubtitleCuesOpt(cues []vttCue, output string, skippedDuration float64, format string, opt Options) error {
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
	cues = compactCues(shiftCues(cues, time.Duration(skippedDuration*float64(time.Second))))
	var text string
	if strings.EqualFold(format, "SRT") {
		text = cuesToSRT(cues)
	} else {
		text = cuesToVTT(cues)
	}
	return os.WriteFile(output, []byte(text), 0644)
}
