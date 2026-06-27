package main

import (
	"context"
	"encoding/json"
	"sort"
	"time"
)

type probeReport struct {
	Version     versionInfo     `json:"version"`
	Input       string          `json:"input"`
	Master      bool            `json:"master"`
	Live        bool            `json:"live"`
	TrackCounts probeTrackCount `json:"trackCounts"`
	SelectedIDs []int           `json:"selectedIds,omitempty"`
	Tracks      []probeTrack    `json:"tracks"`
}

type probeTrackCount struct {
	Total     int `json:"total"`
	Video     int `json:"video"`
	Audio     int `json:"audio"`
	Subtitles int `json:"subtitles"`
}

type probeTrack struct {
	ID              int      `json:"id"`
	Type            string   `json:"type"`
	Selected        bool     `json:"selected,omitempty"`
	Display         string   `json:"display"`
	GroupID         string   `json:"groupId,omitempty"`
	Name            string   `json:"name,omitempty"`
	Language        string   `json:"language,omitempty"`
	Codecs          string   `json:"codecs,omitempty"`
	Resolution      string   `json:"resolution,omitempty"`
	Bandwidth       int      `json:"bandwidth,omitempty"`
	FrameRate       float64  `json:"frameRate,omitempty"`
	Channels        string   `json:"channels,omitempty"`
	Extension       string   `json:"extension,omitempty"`
	Live            bool     `json:"live"`
	SegmentCount    int      `json:"segmentCount"`
	DurationSeconds float64  `json:"durationSeconds"`
	Encrypted       bool     `json:"encrypted"`
	EncryptMethods  []string `json:"encryptMethods,omitempty"`
}

func runProbeJSON(ctx context.Context, opt Options, now time.Time) (string, error) {
	if err := validateOptions(opt); err != nil {
		return "", err
	}
	opt.ProbeJSON = true
	applyOptionImplicationsWithMessages(&opt)
	applyDerivedDefaults(&opt, now)
	client, err := newHTTPClient(opt)
	if err != nil {
		return "", err
	}
	streams, p, err := parseSource(ctx, client, opt)
	if err != nil {
		return "", err
	}
	streams, err = fetchProbePlaylists(ctx, p, streams)
	if err != nil {
		return "", err
	}
	selectedIDs, err := probeSelectedIDs(streams, opt)
	if err != nil {
		return "", err
	}
	report := buildProbeReport(opt, p, streams, selectedIDs)
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}

func fetchProbePlaylists(ctx context.Context, p *parser, streams []StreamSpec) ([]StreamSpec, error) {
	for i := range streams {
		if streams[i].Playlist != nil {
			continue
		}
		if err := p.fetchPlaylist(ctx, &streams[i]); err != nil {
			return nil, err
		}
	}
	return streams, nil
}

func probeSelectedIDs(streams []StreamSpec, opt Options) ([]int, error) {
	if !(opt.AutoSelect || opt.SubOnly || hasKeepFilters(opt) || len(streams) == 1) {
		return nil, nil
	}
	selected, err := chooseStreams(streams, opt)
	if err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(selected))
	for _, stream := range selected {
		ids = append(ids, stream.ID)
	}
	sort.Ints(ids)
	return ids, nil
}

func buildProbeReport(opt Options, p *parser, streams []StreamSpec, selectedIDs []int) probeReport {
	selected := map[int]bool{}
	for _, id := range selectedIDs {
		selected[id] = true
	}
	report := probeReport{
		Version:     currentVersionInfo(),
		Input:       opt.Input,
		Master:      p != nil && p.master,
		TrackCounts: probeCounts(streams),
		SelectedIDs: selectedIDs,
		Tracks:      make([]probeTrack, 0, len(streams)),
	}
	for _, stream := range streams {
		track := buildProbeTrack(stream, selected[stream.ID])
		if track.Live {
			report.Live = true
		}
		report.Tracks = append(report.Tracks, track)
	}
	return report
}

func probeCounts(streams []StreamSpec) probeTrackCount {
	video, audio, subtitles := countStreamGroups(streams)
	return probeTrackCount{
		Total:     len(streams),
		Video:     video,
		Audio:     audio,
		Subtitles: subtitles,
	}
}

func buildProbeTrack(stream StreamSpec, selected bool) probeTrack {
	methods := probeEncryptMethods(stream.Playlist)
	return probeTrack{
		ID:              stream.ID,
		Type:            probeTrackType(stream),
		Selected:        selected,
		Display:         stream.DisplayString(),
		GroupID:         stream.GroupID,
		Name:            stream.Name,
		Language:        stream.Language,
		Codecs:          stream.Codecs,
		Resolution:      stream.Resolution,
		Bandwidth:       stream.Bandwidth,
		FrameRate:       stream.FrameRate,
		Channels:        stream.Channels,
		Extension:       stream.Extension,
		Live:            stream.Playlist != nil && stream.Playlist.IsLive,
		SegmentCount:    playlistMediaSegmentCount(stream.Playlist),
		DurationSeconds: playlistDuration(stream.Playlist),
		Encrypted:       len(methods) > 0,
		EncryptMethods:  methods,
	}
}

func probeTrackType(stream StreamSpec) string {
	if stream.MediaType == nil {
		return string(MediaVideo)
	}
	return string(*stream.MediaType)
}

func probeEncryptMethods(pl *Playlist) []string {
	seen := map[string]bool{}
	if pl != nil && pl.MediaInit != nil && pl.MediaInit.IsEncrypted() {
		seen[string(pl.MediaInit.Encrypt.Method)] = true
	}
	if pl != nil {
		for _, part := range pl.Parts {
			for _, segment := range part.Segments {
				if segment.IsEncrypted() {
					seen[string(segment.Encrypt.Method)] = true
				}
			}
		}
	}
	methods := make([]string, 0, len(seen))
	for method := range seen {
		methods = append(methods, method)
	}
	sort.Strings(methods)
	return methods
}
