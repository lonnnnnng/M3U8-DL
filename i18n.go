package main

import "fmt"

type localizedText struct {
	ZhCN string
	ZhTW string
	EnUS string
}

var localizedTexts = map[string]localizedText{
	"newVersionFound": {
		ZhCN: "发现新版本: %s",
		ZhTW: "發現新版本: %s",
		EnUS: "New version found: %s",
	},
	"livePipeMuxForcesRealtime": {
		ZhCN: "检测到 LivePipeMux，已强制启用 LiveRealTimeMerge",
		ZhTW: "檢測到 LivePipeMux，已強制啟用 LiveRealTimeMerge",
		EnUS: "LivePipeMux detected, forced enable LiveRealTimeMerge",
	},
	"muxAfterDoneForcesBinaryMerge": {
		ZhCN: "检测到 MuxAfterDone，已强制启用 BinaryMerge",
		ZhTW: "檢測到 MuxAfterDone，已強制啟用 BinaryMerge",
		EnUS: "MuxAfterDone detected, forced enable BinaryMerge",
	},
	"taskStartAt": {
		ZhCN: "等待任务开始: %s",
		ZhTW: "等待任務開始: %s",
		EnUS: "Waiting until: %s",
	},
	"streamsParsed": {
		ZhCN: "解析到 %d 条轨道",
		ZhTW: "解析到 %d 條軌道",
		EnUS: "Parsed %d streams",
	},
	"skipDownload": {
		ZhCN: "已按 --skip-download 跳过下载",
		ZhTW: "已按 --skip-download 跳過下載",
		EnUS: "Skip download due to --skip-download",
	},
	"muxOutput": {
		ZhCN: "混流输出: %s",
		ZhTW: "混流輸出: %s",
		EnUS: "Mux output: %s",
	},
	"output": {
		ZhCN: "输出: %s",
		ZhTW: "輸出: %s",
		EnUS: "Output: %s",
	},
	"singleFileSplitFailed": {
		ZhCN: "单分片切片检测失败: %v",
		ZhTW: "單分片切片檢測失敗: %v",
		EnUS: "Single segment split check failed: %v",
	},
	"singleFileRealtimeDisabled": {
		ZhCN: "单分片切片已启用，自动关闭实时 MP4 解密",
		ZhTW: "單分片切片已啟用，自動關閉即時 MP4 解密",
		EnUS: "Single segment split enabled, MP4 real-time decryption has been disabled",
	},
	"singleFileSplitEnabled": {
		ZhCN: "检测到单分片大文件，已启用 Range 切片下载",
		ZhTW: "檢測到單分片大文件，已啟用 Range 切片下載",
		EnUS: "Single large segment detected, Range split download enabled",
	},
	"downloadProgress": {
		ZhCN: "%s 下载进度 %d/%d",
		ZhTW: "%s 下載進度 %d/%d",
		EnUS: "%s download progress %d/%d",
	},
}

func tr(opt Options, key string, args ...any) string {
	text, ok := localizedTexts[key]
	if !ok {
		return key
	}
	format := text.EnUS
	switch opt.UILanguage {
	case "zh-CN", "zh-SG", "zh-Hans", "":
		format = text.ZhCN
	case "zh-TW", "zh-HK", "zh-Hant":
		format = text.ZhTW
	}
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}
