package main

import "fmt"

type localizedText struct {
	ZhCN string
	ZhTW string
	EnUS string
}

var localizedTexts = map[string]localizedText{
	"newVersionFound": {
		ZhCN: "检测到新版本，请尽快升级！",
		ZhTW: "檢測到新版本，請盡快升級！",
		EnUS: "New version detected!",
	},
	"consoleRedirected": {
		ZhCN: "输出被重定向, 将清除ANSI颜色",
		ZhTW: "輸出被重定向, 將清除ANSI顏色",
		EnUS: "Output is redirected, ANSI colors are cleared.",
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
	"autoBinaryMerge": {
		ZhCN: "检测到fMP4，自动开启二进制合并",
		ZhTW: "檢測到fMP4，自動開啟二進位制合併",
		EnUS: "fMP4 is detected, binary merging is automatically enabled",
	},
	"autoBinaryMergeUnknown": {
		ZhCN: "检测到无法识别的加密方式，自动开启二进制合并",
		ZhTW: "檢測到無法識別的加密方式，自動開啟二進位制合併",
		EnUS: "An unrecognized encryption method is detected, binary merging is automatically enabled",
	},
	"autoBinaryMergeCENC": {
		ZhCN: "检测到CENC加密方式，自动开启二进制合并",
		ZhTW: "檢測到CENC加密方式，自動開啟二進位制合併",
		EnUS: "When CENC encryption is detected, binary merging is automatically enabled",
	},
	"realTimeDecMessage": {
		ZhCN: "启用实时解密时，建议用shaka-packager而非mp4decrypt/ffmpeg",
		ZhTW: "啟用即時解密時，建議用shaka-packager而非mp4decrypt/ffmpeg",
		EnUS: "When enabling real-time decryption, it is recommended to use shaka-packager instead of mp4decrypt/ffmpeg",
	},
	"taskStartAt": {
		ZhCN: "等待任务开始: %s",
		ZhTW: "等待任務開始: %s",
		EnUS: "Waiting until: %s",
	},
	"streamsInfo": {
		ZhCN: "已解析, 共计 %d 条媒体流, 基本流 %d 条, 可选音频流 %d 条, 可选字幕流 %d 条",
		ZhTW: "已解析, 共計 %d 條媒體流, 基本流 %d 條, 可選音頻流 %d 條, 可選字幕流 %d 條",
		EnUS: "Extracted, there are %d streams, with %d basic streams, %d audio streams, %d subtitle streams",
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
	"ffmpegNotFound": {
		ZhCN: "找不到ffmpeg，请自行下载：https://ffmpeg.org/download.html",
		ZhTW: "找不到ffmpeg，請自行下載：https://ffmpeg.org/download.html",
		EnUS: "ffmpeg not found, please download at: https://ffmpeg.org/download.html",
	},
	"mkvmergeNotFound": {
		ZhCN: "找不到mkvmerge，请自行下载：https://mkvtoolnix.download/downloads.html",
		ZhTW: "找不到mkvmerge，請自行下載：https://mkvtoolnix.download/downloads.html",
		EnUS: "mkvmerge not found, please download at: https://mkvtoolnix.download/downloads.html",
	},
	"shakaPackagerNotFound": {
		ZhCN: "找不到shaka-packager，请自行下载：https://github.com/shaka-project/shaka-packager/releases",
		ZhTW: "找不到shaka-packager，請自行下載：https://github.com/shaka-project/shaka-packager/releases",
		EnUS: "shaka-packager not found, please download at: https://github.com/shaka-project/shaka-packager/releases",
	},
	"mp4decryptNotFound": {
		ZhCN: "找不到mp4decrypt，请自行下载：https://www.bento4.com/downloads/",
		ZhTW: "找不到mp4decrypt，請自行下載：https://www.bento4.com/downloads/",
		EnUS: "mp4decrypt not found, please download at: https://www.bento4.com/downloads/",
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
