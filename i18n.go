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
	"namedPipeCreated": {
		ZhCN: "已创建命名管道：",
		ZhTW: "已創建命名管道：",
		EnUS: "Named pipe created: ",
	},
	"namedPipeMux": {
		ZhCN: "通过命名管道混流到",
		ZhTW: "通過命名管道混流到",
		EnUS: "Mux with named pipe, to",
	},
	"consoleRedirected": {
		ZhCN: "输出被重定向, 将清除ANSI颜色",
		ZhTW: "輸出被重定向, 將清除ANSI顏色",
		EnUS: "Output is redirected, ANSI colors are cleared.",
	},
	"processImageSub": {
		ZhCN: "正在处理图形字幕",
		ZhTW: "正在處理圖形字幕",
		EnUS: "Processing Image Sub",
	},
	"customRangeWarn": {
		ZhCN: "请注意，自定义下载范围有时会导致音画不同步",
		ZhTW: "請注意，自定義下載範圍有時會導致音畫不同步",
		EnUS: "Please note that custom range may sometimes result in audio and video being out of sync",
	},
	"customRangeInvalid": {
		ZhCN: "自定义下载范围无效",
		ZhTW: "自定義下載範圍無效",
		EnUS: "User customed range invalid",
	},
	"customAdKeywordsFound": {
		ZhCN: "用户自定义广告分片URL关键字：",
		ZhTW: "用戶自定義廣告分片URL關鍵字：",
		EnUS: "User customed Ad keyword: ",
	},
	"customRangeFound": {
		ZhCN: "用户自定义下载范围：",
		ZhTW: "用戶自定義下載範圍：",
		EnUS: "User customed range: ",
	},
	"livePipeMuxForcesRealtime": {
		ZhCN: "检测到 LivePipeMux，已强制启用 LiveRealTimeMerge",
		ZhTW: "檢測到 LivePipeMux，已強制啟用 LiveRealTimeMerge",
		EnUS: "LivePipeMux detected, forced enable LiveRealTimeMerge",
	},
	"autoBinaryMerge": {
		ZhCN: "检测到fMP4，自动开启二进制合并",
		ZhTW: "檢測到fMP4，自動開啟二進位制合併",
		EnUS: "fMP4 is detected, binary merging is automatically enabled",
	},
	"autoBinaryMerge2": {
		ZhCN: "检测到杜比视界内容，自动开启二进制合并",
		ZhTW: "檢測到杜比視界內容，自動開啟二進位制合併",
		EnUS: "Dolby Vision content is detected, binary merging is automatically enabled",
	},
	"autoBinaryMerge3": {
		ZhCN: "检测到无法识别的加密方式，自动开启二进制合并",
		ZhTW: "檢測到無法識別的加密方式，自動開啟二進位制合併",
		EnUS: "An unrecognized encryption method is detected, binary merging is automatically enabled",
	},
	"autoBinaryMerge4": {
		ZhCN: "检测到CENC加密方式，自动开启二进制合并",
		ZhTW: "檢測到CENC加密方式，自動開啟二進位制合併",
		EnUS: "When CENC encryption is detected, binary merging is automatically enabled",
	},
	"autoBinaryMerge5": {
		ZhCN: "检测到杜比视界内容，混流功能已禁用",
		ZhTW: "檢測到杜比視界內容，混流功能已禁用",
		EnUS: "Dolby Vision content is detected, mux after done is automatically disabled",
	},
	"autoBinaryMerge6": {
		ZhCN: "你已开启下载完成后混流，自动开启二进制合并",
		ZhTW: "你已開啟下載完成後混流，自動開啟二進制合併",
		EnUS: "MuxAfterDone is detected, binary merging is automatically enabled",
	},
	"realTimeDecMessage": {
		ZhCN: "启用实时解密时，建议用shaka-packager而非mp4decrypt/ffmpeg",
		ZhTW: "啟用即時解密時，建議用shaka-packager而非mp4decrypt/ffmpeg",
		EnUS: "When enabling real-time decryption, it is recommended to use shaka-packager instead of mp4decrypt/ffmpeg",
	},
	"taskStartAt": {
		ZhCN: "程序将等待，直到：",
		ZhTW: "程序將等待，直到：",
		EnUS: "The program will wait until: ",
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
	"selectedStream": {
		ZhCN: "已选择的流:",
		ZhTW: "已選擇的流:",
		EnUS: "Selected streams:",
	},
	"promptChoiceText": {
		ZhCN: "[grey](按键盘上下键以浏览更多内容)[/]",
		ZhTW: "[grey](按鍵盤上下鍵以瀏覽更多內容)[/]",
		EnUS: "[grey](Move up and down to reveal more streams)[/]",
	},
	"promptInfo": {
		ZhCN: "(按 [blue]空格键[/] 选择流, [green]回车键[/] 完成选择)",
		ZhTW: "(按 [blue]空格鍵[/] 選擇流, [green]確認鍵[/] 完成選擇)",
		EnUS: "(Press [blue]<space>[/] to toggle a stream, [green]<enter>[/] to accept)",
	},
	"promptTitle": {
		ZhCN: "请选择 [green]你要下载的内容[/]:",
		ZhTW: "請選擇 [green]你要下載的內容[/]:",
		EnUS: "Please select [green]what you want to download[/]:",
	},
	"noStreamsToDownload": {
		ZhCN: "没有找到需要下载的流",
		ZhTW: "沒有找到需要下載的流",
		EnUS: "No stream found to download",
	},
	"saveName": {
		ZhCN: "保存文件名: ",
		ZhTW: "保存檔案名: ",
		EnUS: "Save Name: ",
	},
	"writeJson": {
		ZhCN: "写出meta json",
		ZhTW: "寫出meta json",
		EnUS: "Writing meta json",
	},
	"startDownloading": {
		ZhCN: "开始下载...",
		ZhTW: "開始下載...",
		EnUS: "Start downloading...",
	},
	"readingInfo": {
		ZhCN: "读取媒体信息...",
		ZhTW: "讀取媒體訊息...",
		EnUS: "Reading media info...",
	},
	"searchKey": {
		ZhCN: "正在尝试从文本文件搜索KEY...",
		ZhTW: "正在嘗試從文本文件搜尋KEY...",
		EnUS: "Trying to search for KEY from text file...",
	},
	"cmd_loadKeyFailed": {
		ZhCN: "获取KEY失败，忽略读取.",
		ZhTW: "獲取KEY失敗，忽略讀取.",
		EnUS: "Failed to get KEY, ignore.",
	},
	"fixingTTML": {
		ZhCN: "正在提取TTML(raw)字幕...",
		ZhTW: "正在提取TTML(raw)字幕...",
		EnUS: "Extracting TTML(raw) subtitle...",
	},
	"fixingTTMLmp4": {
		ZhCN: "正在提取TTML(mp4)字幕...",
		ZhTW: "正在提取TTML(mp4)字幕...",
		EnUS: "Extracting TTML(mp4) subtitle...",
	},
	"fixingVTT": {
		ZhCN: "正在提取VTT(raw)字幕...",
		ZhTW: "正在提取VTT(raw)字幕...",
		EnUS: "Extracting VTT(raw) subtitle...",
	},
	"fixingVTTmp4": {
		ZhCN: "正在提取VTT(mp4)字幕...",
		ZhTW: "正在提取VTT(mp4)字幕...",
		EnUS: "Extracting VTT(mp4) subtitle...",
	},
	"decryptionFailed": {
		ZhCN: "解密失败",
		ZhTW: "解密失敗",
		EnUS: "Decryption failed",
	},
	"segmentCountCheckNotPass": {
		ZhCN: "分片数量校验不通过, 共%d个,已下载%d.",
		ZhTW: "分片數量校驗不通過, 共%d個,已下載%d.",
		EnUS: "Segment count check not pass, total: %d, downloaded: %d.",
	},
	"binaryMerge": {
		ZhCN: "二进制合并中...",
		ZhTW: "二進位制合併中...",
		EnUS: "Binary merging...",
	},
	"ffmpegMerge": {
		ZhCN: "调用ffmpeg合并中...",
		ZhTW: "調用ffmpeg合併中...",
		EnUS: "ffmpeg merging...",
	},
	"partMerge": {
		ZhCN: "分片数量大于1800个，开始分块合并...",
		ZhTW: "分片數量大於1800個，開始分塊合併...",
		EnUS: "Segments more than 1800, start partial merge...",
	},
	"loadingUrl": {
		ZhCN: "加载URL: ",
		ZhTW: "載入URL: ",
		EnUS: "Loading URL: ",
	},
	"loadUrlFailed": {
		ZhCN: "加载URL失败",
		ZhTW: "載入URL失敗",
		EnUS: "Failed to load URL",
	},
	"allowHlsMultiExtMap": {
		ZhCN: "已经允许识别多个#EXT-X-MAP标签, 本软件可能无法正确处理, 请手动确认内容完整性",
		ZhTW: "已經允許識別多個#EXT-X-MAP標籤, 本軟件可能無法正確處理, 請手動確認內容完整性",
		EnUS: "Multiple #EXT-X-MAP tags are now allowed for detection. However, this software may not handle them correctly. Please manually verify the content's integrity",
	},
	"badM3u8": {
		ZhCN: "错误的m3u8",
		ZhTW: "錯誤的m3u8",
		EnUS: "Bad m3u8",
	},
	"matchHLS": {
		ZhCN: "内容匹配: [white on deepskyblue1]HTTP Live Streaming[/]",
		ZhTW: "內容匹配: [white on deepskyblue1]HTTP Live Streaming[/]",
		EnUS: "Content Matched: [white on deepskyblue1]HTTP Live Streaming[/]",
	},
	"parsingStream": {
		ZhCN: "正在解析媒体信息...",
		ZhTW: "正在解析媒體信息...",
		EnUS: "Parsing streams...",
	},
	"masterM3u8Found": {
		ZhCN: "检测到Master列表，开始解析全部流信息",
		ZhTW: "檢測到Master列表，開始解析全部流訊息",
		EnUS: "Master List detected, try parse all streams",
	},
	"liveFound": {
		ZhCN: "检测到直播流",
		ZhTW: "檢測到直播流",
		EnUS: "Live stream found",
	},
	"liveLimit": {
		ZhCN: "本次直播录制时长上限: ",
		ZhTW: "本次直播錄製時長上限: ",
		EnUS: "Live recording duration limit: ",
	},
	"liveLimitReached": {
		ZhCN: "到达直播录制上限，即将停止录制",
		ZhTW: "到達直播錄製上限，即將停止錄製",
		EnUS: "Live recording limit reached, will stop recording soon",
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
	"singleFileRealtimeDecryptWarn": {
		ZhCN: "实时解密已被强制关闭",
		ZhTW: "即時解密已被強制關閉",
		EnUS: "Real-time decryption has been disabled",
	},
	"singleFileSplitWarn": {
		ZhCN: "整段文件已被自动切割为小分片以加速下载",
		ZhTW: "整段文件已被自動切割為小分片以加速下載",
		EnUS: "The entire file has been cut into small segments to accelerate",
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
