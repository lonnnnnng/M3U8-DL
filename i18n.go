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
	"usage_section_basic": {
		ZhCN: "基础:",
		ZhTW: "基礎:",
		EnUS: "Basics:",
	},
	"usage_section_inputNetwork": {
		ZhCN: "输入与网络:",
		ZhTW: "輸入與網路:",
		EnUS: "Input and network:",
	},
	"usage_section_downloadControl": {
		ZhCN: "下载控制:",
		ZhTW: "下載控制:",
		EnUS: "Download control:",
	},
	"usage_section_outputLogs": {
		ZhCN: "输出与日志:",
		ZhTW: "輸出與日誌:",
		EnUS: "Output and logs:",
	},
	"usage_section_subtitleStreams": {
		ZhCN: "字幕与选流:",
		ZhTW: "字幕與選流:",
		EnUS: "Subtitle and stream selection:",
	},
	"usage_section_decryption": {
		ZhCN: "解密:",
		ZhTW: "解密:",
		EnUS: "Decryption:",
	},
	"usage_section_live": {
		ZhCN: "直播:",
		ZhTW: "直播:",
		EnUS: "Live:",
	},
	"usage_section_muxingHelp": {
		ZhCN: "混流与更多帮助:",
		ZhTW: "混流與更多幫助:",
		EnUS: "Muxing and more help:",
	},
	"usage_section_examples": {
		ZhCN: "示例:",
		ZhTW: "示例:",
		EnUS: "Examples:",
	},
	"usage_examples": {
		ZhCN: "  # 自动选择并用 ffmpeg 输出 mp4\n" +
			"  m3u8dl-go \"https://example.com/index.m3u8\" --auto-select true --save-dir ./downloads\n" +
			"  # 只解析并写出 raw/meta json\n" +
			"  m3u8dl-go \"https://example.com/index.m3u8\" --skip-download true --save-name probe\n" +
			"  # 下载前 3 个分片并二进制直拼为 TS\n" +
			"  m3u8dl-go \"https://example.com/index.m3u8\" --auto-select true --custom-range 0-2 --binary-merge true\n" +
			"  # 带 Cookie/Referer 下载并限制速度\n" +
			"  m3u8dl-go \"https://example.com/index.m3u8\" -H \"Cookie: xxx\" -H \"Referer: https://example.com\" -R 2M\n" +
			"  # 下载完成后最终混流为 mp4\n" +
			"  m3u8dl-go \"https://example.com/index.m3u8\" --auto-select true -M format=mp4:muxer=ffmpeg\n" +
			"  # 录制直播 5 分钟并实时追加输出\n" +
			"  m3u8dl-go \"https://example.com/live.m3u8\" --live-real-time-merge true --live-record-limit 00:05:00",
		ZhTW: "  # 自動選擇並用 ffmpeg 輸出 mp4\n" +
			"  m3u8dl-go \"https://example.com/index.m3u8\" --auto-select true --save-dir ./downloads\n" +
			"  # 只解析並寫出 raw/meta json\n" +
			"  m3u8dl-go \"https://example.com/index.m3u8\" --skip-download true --save-name probe\n" +
			"  # 下載前 3 個分片並二進位直拼為 TS\n" +
			"  m3u8dl-go \"https://example.com/index.m3u8\" --auto-select true --custom-range 0-2 --binary-merge true\n" +
			"  # 帶 Cookie/Referer 下載並限制速度\n" +
			"  m3u8dl-go \"https://example.com/index.m3u8\" -H \"Cookie: xxx\" -H \"Referer: https://example.com\" -R 2M\n" +
			"  # 下載完成後最終混流為 mp4\n" +
			"  m3u8dl-go \"https://example.com/index.m3u8\" --auto-select true -M format=mp4:muxer=ffmpeg\n" +
			"  # 錄製直播 5 分鐘並即時追加輸出\n" +
			"  m3u8dl-go \"https://example.com/live.m3u8\" --live-real-time-merge true --live-record-limit 00:05:00",
		EnUS: "  # Auto select and output mp4 through ffmpeg\n" +
			"  m3u8dl-go \"https://example.com/index.m3u8\" --auto-select true --save-dir ./downloads\n" +
			"  # Parse only and write raw/meta json\n" +
			"  m3u8dl-go \"https://example.com/index.m3u8\" --skip-download true --save-name probe\n" +
			"  # Download the first 3 segments and binary merge to TS\n" +
			"  m3u8dl-go \"https://example.com/index.m3u8\" --auto-select true --custom-range 0-2 --binary-merge true\n" +
			"  # Download with Cookie/Referer and speed limit\n" +
			"  m3u8dl-go \"https://example.com/index.m3u8\" -H \"Cookie: xxx\" -H \"Referer: https://example.com\" -R 2M\n" +
			"  # Final mux to mp4 after download\n" +
			"  m3u8dl-go \"https://example.com/index.m3u8\" --auto-select true -M format=mp4:muxer=ffmpeg\n" +
			"  # Record live for 5 minutes with real-time append output\n" +
			"  m3u8dl-go \"https://example.com/live.m3u8\" --live-real-time-merge true --live-record-limit 00:05:00",
	},
	"usage_section_moreHelpTopics": {
		ZhCN: "更多帮助主题:",
		ZhTW: "更多幫助主題:",
		EnUS: "More help topics:",
	},
	"usage_moreHelpTopics": {
		ZhCN: "  --morehelp mux-after-done    查看最终混流参数\n" +
			"  --morehelp mux-import        查看外部音轨/字幕导入参数\n" +
			"  --morehelp custom-range      查看分片/时间范围语法\n" +
			"  --morehelp select-video      查看视频选择过滤器\n" +
			"  --morehelp select-audio      查看音频选择过滤器\n" +
			"  --morehelp select-subtitle   查看字幕选择过滤器",
		ZhTW: "  --morehelp mux-after-done    查看最終混流參數\n" +
			"  --morehelp mux-import        查看外部音軌/字幕導入參數\n" +
			"  --morehelp custom-range      查看分片/時間範圍語法\n" +
			"  --morehelp select-video      查看影片選擇過濾器\n" +
			"  --morehelp select-audio      查看音訊選擇過濾器\n" +
			"  --morehelp select-subtitle   查看字幕選擇過濾器",
		EnUS: "  --morehelp mux-after-done    Final muxing options\n" +
			"  --morehelp mux-import        External audio/subtitle import options\n" +
			"  --morehelp custom-range      Segment/time range syntax\n" +
			"  --morehelp select-video      Video selection filter\n" +
			"  --morehelp select-audio      Audio selection filter\n" +
			"  --morehelp select-subtitle   Subtitle selection filter",
	},
	"cmd_help": {
		ZhCN: "显示帮助信息",
		ZhTW: "顯示幫助訊息",
		EnUS: "Show help information",
	},
	"cmd_version": {
		ZhCN: "显示版本信息",
		ZhTW: "顯示版本訊息",
		EnUS: "Show version information",
	},
	"cmd_versionJSON": {
		ZhCN: "以 JSON 输出版本信息",
		ZhTW: "以 JSON 輸出版本訊息",
		EnUS: "Show machine-readable version information",
	},
	"cmd_doctor": {
		ZhCN: "检测 ffmpeg、ffprobe 等外部工具",
		ZhTW: "檢測 ffmpeg、ffprobe 等外部工具",
		EnUS: "Check external tools such as ffmpeg and ffprobe",
	},
	"cmd_doctorJSON": {
		ZhCN: "以 JSON 输出外部工具检测结果",
		ZhTW: "以 JSON 輸出外部工具檢測結果",
		EnUS: "Show external tool diagnostics as JSON",
	},
	"cmd_printEffectiveOptions": {
		ZhCN: "以 JSON 输出解析后的有效参数",
		ZhTW: "以 JSON 輸出解析後的有效參數",
		EnUS: "Show parsed effective options as JSON",
	},
	"cmd_forceAnsiConsole": {
		ZhCN: "强制认定终端为支持ANSI且可交互的终端",
		ZhTW: "強制認定終端為支援ANSI且可交往的終端",
		EnUS: "Force assuming the terminal is ANSI-compatible and interactive",
	},
	"cmd_noAnsiColor": {
		ZhCN: "去除ANSI颜色",
		ZhTW: "關閉ANSI顏色",
		EnUS: "Remove ANSI colors",
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
	"checkingLast": {
		ZhCN: "验证最后一个分片有效性",
		ZhTW: "驗證最後一個分片有效性",
		EnUS: "Verifying the validity of the last segment",
	},
	"cmd_noDateInfo": {
		ZhCN: "混流时不写入日期信息",
		ZhTW: "混流時不寫入日期訊息",
		EnUS: "Date information is not written during muxing",
	},
	"cmd_appendUrlParams": {
		ZhCN: "将输入Url的Params添加至分片, 对某些网站很有用, 例如 kakao.com",
		ZhTW: "將輸入Url的Params添加至分片, 對某些網站很有用, 例如 kakao.com",
		EnUS: "Add Params of input Url to segments, useful for some websites, such as kakao.com",
	},
	"cmd_autoSelect": {
		ZhCN: "自动选择所有类型的最佳轨道",
		ZhTW: "自動選擇所有類型的最佳軌道",
		EnUS: "Automatically selects the best tracks of all types",
	},
	"cmd_binaryMerge": {
		ZhCN: "二进制合并",
		ZhTW: "二進位制合併",
		EnUS: "Binary merge",
	},
	"cmd_header": {
		ZhCN: "为HTTP请求设置特定的请求头, 例如:\n-H \"Cookie: mycookie\" -H \"User-Agent: iOS\"",
		ZhTW: "為HTTP請求設置特定的請求頭, 例如:\n-H \"Cookie: mycookie\" -H \"User-Agent: iOS\"",
		EnUS: "Pass custom header(s) to server, Example:\n-H \"Cookie: mycookie\" -H \"User-Agent: iOS\"",
	},
	"cmd_saveDir": {
		ZhCN: "设置输出目录",
		ZhTW: "設置輸出目錄",
		EnUS: "Set output directory",
	},
	"cmd_saveName": {
		ZhCN: "设置保存文件名",
		ZhTW: "設置保存檔案名",
		EnUS: "Set output filename",
	},
	"cmd_threadCount": {
		ZhCN: "设置下载线程数",
		ZhTW: "設置下載執行緒數",
		EnUS: "Set download thread count",
	},
	"cmd_MP4RealTimeDecryption": {
		ZhCN: "实时解密MP4分片",
		ZhTW: "即時解密MP4分片",
		EnUS: "Decrypt MP4 segments in real time",
	},
	"cmd_customRange": {
		ZhCN: "仅下载部分分片. 输入 \"--morehelp custom-range\" 以查看详细信息",
		ZhTW: "僅下載部分分片. 輸入 \"--morehelp custom-range\" 以查看詳細訊息",
		EnUS: "Download only part of the segments. Use \"--morehelp custom-range\" for more details",
	},
	"cmd_customHLSKey": {
		ZhCN: "指定HLS解密KEY. 可以是文件, HEX或Base64",
		ZhTW: "指定HLS解密KEY. 可以是文件, HEX或Base64",
		EnUS: "Set the HLS decryption key. Can be file, HEX or Base64",
	},
	"cmd_customHLSIv": {
		ZhCN: "指定HLS解密IV. 可以是文件, HEX或Base64",
		ZhTW: "指定HLS解密IV. 可以是文件, HEX或Base64",
		EnUS: "Set the HLS decryption iv. Can be file, HEX or Base64",
	},
	"cmd_moreHelp": {
		ZhCN: "查看某个选项的详细帮助信息",
		ZhTW: "查看某個選項的詳細幫助訊息",
		EnUS: "Set more help info about one option",
	},
	"cmd_muxAfterDone": {
		ZhCN: "所有工作完成时尝试混流分离的音视频. 输入 \"--morehelp mux-after-done\" 以查看详细信息",
		ZhTW: "所有工作完成時嘗試混流分離的影音. 輸入 \"--morehelp mux-after-done\" 以查看詳細訊息",
		EnUS: "When all works is done, try to mux the downloaded streams. Use \"--morehelp mux-after-done\" for more details",
	},
	"cmd_baseUrl": {
		ZhCN: "设置BaseURL",
		ZhTW: "設置BaseURL",
		EnUS: "Set BaseURL",
	},
	"cmd_maxSpeed": {
		ZhCN: "设置限速，单位支持 Mbps 或 Kbps，如：15M 100K",
		ZhTW: "設置限速，單位支持 Mbps 或 Kbps，如：15M 100K",
		EnUS: "Set speed limit, Mbps or Kbps, for example: 15M 100K.",
	},
	"cmd_noLog": {
		ZhCN: "关闭日志文件输出",
		ZhTW: "關閉日誌文件輸出",
		EnUS: "Disable log file output",
	},
	"cmd_allowHlsMultiExtMap": {
		ZhCN: "允许HLS中的多个#EXT-X-MAP(实验性)",
		ZhTW: "允許HLS中的多個#EXT-X-MAP(實驗性)",
		EnUS: "Allow multiple #EXT-X-MAP in HLS (experimental)",
	},
	"cmd_disableUpdateCheck": {
		ZhCN: "禁用版本更新检测",
		ZhTW: "禁用版本更新檢測",
		EnUS: "Disable version update check",
	},
	"cmd_useFFmpegConcatDemuxer": {
		ZhCN: "使用 ffmpeg 合并时，使用 concat 分离器而非 concat 协议",
		ZhTW: "使用 ffmpeg 合併時，使用 concat 分離器而非 concat 協議",
		EnUS: "When merging with ffmpeg, use the concat demuxer instead of the concat protocol",
	},
	"cmd_checkSegmentsCount": {
		ZhCN: "检测实际下载的分片数量和预期数量是否匹配",
		ZhTW: "檢測實際下載的分片數量和預期數量是否匹配",
		EnUS: "Check if the actual number of segments downloaded matches the expected number",
	},
	"cmd_downloadRetryCount": {
		ZhCN: "每个分片下载异常时的重试次数",
		ZhTW: "每個分片下載異常時的重試次數",
		EnUS: "The number of retries when download segment error",
	},
	"cmd_httpRequestTimeout": {
		ZhCN: "HTTP请求的超时时间(秒)",
		ZhTW: "HTTP請求的超時時間(秒)",
		EnUS: "Timeout duration for HTTP requests (in seconds)",
	},
	"cmd_decryptionBinaryPath": {
		ZhCN: "MP4解密所用工具的全路径, 例如 C:\\Tools\\mp4decrypt.exe",
		ZhTW: "MP4解密所用工具的全路徑, 例如 C:\\Tools\\mp4decrypt.exe",
		EnUS: "Full path to the tool used for MP4 decryption, like C:\\Tools\\mp4decrypt.exe",
	},
	"cmd_delAfterDone": {
		ZhCN: "完成后删除临时文件",
		ZhTW: "完成後刪除臨時文件",
		EnUS: "Delete temporary files when done",
	},
	"cmd_ffmpegBinaryPath": {
		ZhCN: "ffmpeg可执行程序全路径, 例如 C:\\Tools\\ffmpeg.exe",
		ZhTW: "ffmpeg可執行程序全路徑, 例如 C:\\Tools\\ffmpeg.exe",
		EnUS: "Full path to the ffmpeg binary, like C:\\Tools\\ffmpeg.exe",
	},
	"cmd_mkvmergeBinaryPath": {
		ZhCN: "mkvmerge可执行程序全路径, 例如 C:\\Tools\\mkvmerge.exe",
		ZhTW: "mkvmerge可執行程序全路徑, 例如 C:\\Tools\\mkvmerge.exe",
		EnUS: "Full path to the mkvmerge binary, like C:\\Tools\\mkvmerge.exe",
	},
	"cmd_liveFixVttByAudio": {
		ZhCN: "通过读取音频文件的起始时间修正VTT字幕",
		ZhTW: "透過讀取音訊檔案的起始時間修正VTT字幕",
		EnUS: "Correct VTT sub by reading the start time of the audio file",
	},
	"cmd_Input": {
		ZhCN: "链接或文件",
		ZhTW: "連結或文件",
		EnUS: "Input Url or File",
	},
	"cmd_keys": {
		ZhCN: "设置解密密钥, 程序调用mp4decrpyt/shaka-packager/ffmpeg进行解密. 格式:\r\n--key KID1:KEY1 --key KID2:KEY2\r\n对于KEY相同的情况可以直接输入 --key KEY",
		ZhTW: "設置解密密鑰, 程序調用mp4decrpyt/shaka-packager/ffmpeg進行解密. 格式:\r\n--key KID1:KEY1 --key KID2:KEY2\r\n對於KEY相同的情況可以直接輸入 --key KEY",
		EnUS: "Set decryption key(s) to mp4decrypt/shaka-packager/ffmpeg. format:\r\n--key KID1:KEY1 --key KID2:KEY2\r\nor use --key KEY if all tracks share the same key.",
	},
	"cmd_keyText": {
		ZhCN: "设置密钥文件,程序将从文件中按KID搜寻KEY以解密.(不建议使用特大文件)",
		ZhTW: "設置密鑰文件,程序將從文件中按KID搜尋KEY以解密.(不建議使用特大文件)",
		EnUS: "Set the kid-key file, the program will search the KEY with KID from the file.(Very large file are not recommended)",
	},
	"cmd_logLevel": {
		ZhCN: "设置日志级别",
		ZhTW: "設置日誌級別",
		EnUS: "Set log level",
	},
	"cmd_savePattern": {
		ZhCN: "设置保存文件命名模板, 支持使用变量: \n<SaveName>, <Id>, <Codecs>, <Language>, <Resolution>, \n<Bandwidth>, <MediaType>, <Channels>, <FrameRate>, \n<VideoRange>, <GroupId>, <Ext>\n示例: --save-pattern \"<SaveName>_<Resolution>_<Bandwidth>\"",
		ZhTW: "設置保存檔案命名模板, 支持使用變數: \n<SaveName>, <Id>, <Codecs>, <Language>, <Resolution>, \n<Bandwidth>, <MediaType>, <Channels>, <FrameRate>, \n<VideoRange>, <GroupId>, <Ext>\n示例: --save-pattern \"<SaveName>_<Resolution>_<Bandwidth>\"",
		EnUS: "Set output filename pattern with variables: \n<SaveName>, <Id>, <Codecs>, <Language>, <Resolution>, \n<Bandwidth>, <MediaType>, <Channels>, <FrameRate>, \n<VideoRange>, <GroupId>, <Ext>\nExample: --save-pattern \"<SaveName>_<Resolution>_<Bandwidth>\"",
	},
	"cmd_logFilePath": {
		ZhCN: "设置日志文件路径, 例如 C:\\Logs\\log.txt",
		ZhTW: "設定日誌檔案路徑, 例如 C:\\Logs\\log.txt",
		EnUS: "Set log file path, Example: C:\\Logs\\log.txt",
	},
	"cmd_subFormat": {
		ZhCN: "字幕输出类型",
		ZhTW: "字幕輸出類型",
		EnUS: "Subtitle output format",
	},
	"cmd_subOnly": {
		ZhCN: "只选取字幕轨道",
		ZhTW: "只選取字幕軌道",
		EnUS: "Select only subtitle tracks",
	},
	"cmd_tmpDir": {
		ZhCN: "设置临时文件存储目录",
		ZhTW: "設置臨時文件儲存目錄",
		EnUS: "Set temporary file directory",
	},
	"cmd_uiLanguage": {
		ZhCN: "设置UI语言",
		ZhTW: "設置UI語言",
		EnUS: "Set UI language",
	},
	"cmd_urlProcessorArgs": {
		ZhCN: "此字符串将直接传递给URL Processor",
		ZhTW: "此字符串將直接傳遞給URL Processor",
		EnUS: "Give these arguments to the URL Processors.",
	},
	"cmd_liveRealTimeMerge": {
		ZhCN: "录制直播时实时合并",
		ZhTW: "錄製直播時即時合併",
		EnUS: "Real-time merge into file when recording live",
	},
	"cmd_customProxy": {
		ZhCN: "设置请求代理, 如 http://127.0.0.1:8888",
		ZhTW: "設置請求代理, 如 http://127.0.0.1:8888",
		EnUS: "Set web request proxy, like http://127.0.0.1:8888",
	},
	"cmd_useSystemProxy": {
		ZhCN: "使用系统默认代理",
		ZhTW: "使用系統默認代理",
		EnUS: "Use system default proxy",
	},
	"cmd_livePerformAsVod": {
		ZhCN: "以点播方式下载直播流",
		ZhTW: "以點播方式下載直播流",
		EnUS: "Download live streams as vod",
	},
	"cmd_liveWaitTime": {
		ZhCN: "手动设置直播列表刷新间隔",
		ZhTW: "手動設置直播列表刷新間隔",
		EnUS: "Manually set the live playlist refresh interval",
	},
	"cmd_adKeyword": {
		ZhCN: "设置广告分片的URL关键字(正则表达式)",
		ZhTW: "設置廣告分片的URL關鍵字(正則表達式)",
		EnUS: "Set URL keywords (regular expressions) for AD segments",
	},
	"cmd_liveTakeCount": {
		ZhCN: "手动设置录制直播时首次获取分片的数量",
		ZhTW: "手動設置錄製直播時首次獲取分片的數量",
		EnUS: "Manually set the number of segments downloaded for the first time when recording live",
	},
	"cmd_customHLSMethod": {
		ZhCN: "指定HLS加密方式 (AES_128|AES_128_ECB|CENC|CHACHA20|NONE|SAMPLE_AES|SAMPLE_AES_CTR|UNKNOWN)",
		ZhTW: "指定HLS加密方式 (AES_128|AES_128_ECB|CENC|CHACHA20|NONE|SAMPLE_AES|SAMPLE_AES_CTR|UNKNOWN)",
		EnUS: "Set HLS encryption method (AES_128|AES_128_ECB|CENC|CHACHA20|NONE|SAMPLE_AES|SAMPLE_AES_CTR|UNKNOWN)",
	},
	"cmd_livePipeMux": {
		ZhCN: "录制直播并开启实时合并时通过管道+ffmpeg实时混流到TS文件",
		ZhTW: "錄製直播並開啟即時合併時通過管道+ffmpeg即時混流到TS文件",
		EnUS: "Real-time muxing to TS file through pipeline + ffmpeg (liveRealTimeMerge enabled)",
	},
	"cmd_liveKeepSegments": {
		ZhCN: "录制直播并开启实时合并时依然保留分片",
		ZhTW: "錄製直播並開啟即時合併時依然保留分片",
		EnUS: "Keep segments when recording a live (liveRealTimeMerge enabled)",
	},
	"cmd_liveRecordLimit": {
		ZhCN: "录制直播时的录制时长限制",
		ZhTW: "錄製直播時的錄製時長限制",
		EnUS: "Recording time limit when recording live",
	},
	"cmd_taskStartAt": {
		ZhCN: "在此时间之前不会开始执行任务",
		ZhTW: "在此時間之前不會開始執行任務",
		EnUS: "Task execution will not start before this time",
	},
	"cmd_useShakaPackager": {
		ZhCN: "解密时使用shaka-packager替代mp4decrypt",
		ZhTW: "解密時使用shaka-packager替代mp4decrypt",
		EnUS: "Use shaka-packager instead of mp4decrypt to decrypt",
	},
	"cmd_decryptionEngine": {
		ZhCN: "设置解密时使用的第三方程序",
		ZhTW: "設置解密時使用的第三方程序",
		EnUS: "Set the third-party program used for decryption",
	},
	"cmd_concurrentDownload": {
		ZhCN: "并发下载已选择的音频、视频和字幕",
		ZhTW: "並發下載已選擇的音訊、影片和字幕",
		EnUS: "Concurrently download the selected audio, video and subtitles",
	},
	"cmd_selectVideo": {
		ZhCN: "通过正则表达式选择符合要求的视频流. 输入 \"--morehelp select-video\" 以查看详细信息",
		ZhTW: "通過正則表達式選擇符合要求的影片軌. 輸入 \"--morehelp select-video\" 以查看詳細訊息",
		EnUS: "Select video streams by regular expressions. Use \"--morehelp select-video\" for more details",
	},
	"cmd_dropVideo": {
		ZhCN: "通过正则表达式去除符合要求的视频流.",
		ZhTW: "通過正則表達式去除符合要求的影片串流.",
		EnUS: "Drop video streams by regular expressions.",
	},
	"cmd_selectAudio": {
		ZhCN: "通过正则表达式选择符合要求的音频流. 输入 \"--morehelp select-audio\" 以查看详细信息",
		ZhTW: "通過正則表達式選擇符合要求的音軌. 輸入 \"--morehelp select-audio\" 以查看詳細訊息",
		EnUS: "Select audio streams by regular expressions. Use \"--morehelp select-audio\" for more details",
	},
	"cmd_dropAudio": {
		ZhCN: "通过正则表达式去除符合要求的音频流.",
		ZhTW: "通過正則表達式去除符合要求的音軌.",
		EnUS: "Drop audio streams by regular expressions.",
	},
	"cmd_selectSubtitle": {
		ZhCN: "通过正则表达式选择符合要求的字幕流. 输入 \"--morehelp select-subtitle\" 以查看详细信息",
		ZhTW: "通過正則表達式選擇符合要求的字幕流. 輸入 \"--morehelp select-subtitle\" 以查看詳細訊息",
		EnUS: "Select subtitle streams by regular expressions. Use \"--morehelp select-subtitle\" for more details",
	},
	"cmd_dropSubtitle": {
		ZhCN: "通过正则表达式去除符合要求的字幕流.",
		ZhTW: "通過正則表達式去除符合要求的字幕流.",
		EnUS: "Drop subtitle streams by regular expressions.",
	},
	"cmd_muxImport": {
		ZhCN: "混流时引入外部媒体文件. 输入 \"--morehelp mux-import\" 以查看详细信息",
		ZhTW: "混流時引入外部媒體檔案. 輸入 \"--morehelp mux-import\" 以查看詳細訊息",
		EnUS: "When MuxAfterDone enabled, allow to import local media files. Use \"--morehelp mux-import\" for more details",
	},
	"cmd_muxAfterDone_more": {
		ZhCN: "所有工作完成时尝试混流分离的音视频. 你能够以:分隔形式指定如下参数:\n\n" +
			"* format=FORMAT: 指定混流容器 mkv, mp4, ts\n" +
			"* muxer=MUXER: 指定混流程序 ffmpeg, mkvmerge (默认: ffmpeg)\n" +
			"* bin_path=PATH: 指定程序路径 (默认: 自动寻找)\n" +
			"* skip_sub=BOOL: 是否忽略字幕文件 (默认: false)\n" +
			"* keep=BOOL: 混流完成是否保留文件 true, false (默认: false)\n\n" +
			"例如: \n" +
			"# 混流为mp4容器\n" +
			"-M format=mp4\n" +
			"# 使用mkvmerge, 自动寻找程序\n" +
			"-M format=mkv:muxer=mkvmerge\n" +
			"# 使用mkvmerge, 自定义程序路径\n" +
			"-M format=mkv:muxer=mkvmerge:bin_path=\"C\\:\\Program Files\\MKVToolNix\\mkvmerge.exe\"\n",
		ZhTW: "所有工作完成時嘗試混流分離的影音. 你能夠以:分隔形式指定如下參數:\n\n" +
			"* format=FORMAT: 指定混流容器 mkv, mp4, ts\n" +
			"* muxer=MUXER: 指定混流程序 ffmpeg, mkvmerge (默認: ffmpeg)\n" +
			"* bin_path=PATH: 指定程序路徑 (默認: 自動尋找)\n" +
			"* skip_sub=BOOL: 是否忽略字幕文件 (默認: false)\n" +
			"* keep=BOOL: 混流完成是否保留文件 true, false (默認: false)\n\n" +
			"例如: \n" +
			"# 混流為mp4容器\n" +
			"-M format=mp4\n" +
			"# 使用mkvmerge, 自動尋找程序\n" +
			"-M format=mkv:muxer=mkvmerge\n" +
			"# 使用mkvmerge, 自訂程序路徑\n" +
			"-M format=mkv:muxer=mkvmerge:bin_path=\"C\\:\\Program Files\\MKVToolNix\\mkvmerge.exe\"\n",
		EnUS: "When all works is done, try to mux the downloaded streams. OPTIONS is a colon separated list of:\n\n" +
			"* format=FORMAT: set container. mkv, mp4, ts\n" +
			"* muxer=MUXER: set muxer. ffmpeg, mkvmerge (Default: ffmpeg)\n" +
			"* bin_path=PATH: set binary file path. (Default: auto)\n" +
			"* skip_sub=BOOL: set whether or not skip subtitle files (Default: false)\n" +
			"* keep=BOOL: set whether or not keep files. true, false (Default: false)\n\n" +
			"Examples: \n" +
			"# mux to mp4\n" +
			"-M format=mp4\n" +
			"# use mkvmerge, auto detect bin path\n" +
			"-M format=mkv:muxer=mkvmerge\n" +
			"# use mkvmerge, set bin path\n" +
			"-M format=mkv:muxer=mkvmerge:bin_path=\"C\\:\\Program Files\\MKVToolNix\\mkvmerge.exe\"\n",
	},
	"cmd_muxImport_more": {
		ZhCN: "混流时引入外部媒体文件. 你能够以:分隔形式指定如下参数:\n\n" +
			"* path=PATH: 指定媒体文件路径\n" +
			"* lang=CODE: 指定媒体文件语言代码 (非必须)\n" +
			"* name=NAME: 指定媒体文件描述信息 (非必须)\n\n" +
			"例如: \n" +
			"# 引入外部字幕\n" +
			"--mux-import path=zh-Hans.srt:lang=chi:name=\"中文 (简体)\"\n" +
			"# 引入外部音轨+字幕\n" +
			"--mux-import path=\"D\\:\\media\\atmos.m4a\":lang=eng:name=\"English Description Audio\" --mux-import path=\"D\\:\\media\\eng.vtt\":lang=eng:name=\"English (Description)\"",
		ZhTW: "混流時引入外部媒體檔案. 你能夠以:分隔形式指定如下參數:\n\n" +
			"* path=PATH: 指定媒體檔案路徑\n" +
			"* lang=CODE: 指定媒體檔案語言代碼 (非必須)\n" +
			"* name=NAME: 指定媒體檔案描述訊息 (非必須)\n\n" +
			"例如: \n" +
			"# 引入外部字幕\n" +
			"--mux-import path=zh-Hant.srt:lang=chi:name=\"中文 (繁體)\"\n" +
			"# 引入外部音軌+字幕\n" +
			"--mux-import path=\"D\\:\\media\\atmos.m4a\":lang=eng:name=\"English Description Audio\" --mux-import path=\"D\\:\\media\\eng.vtt\":lang=eng:name=\"English (Description)\"",
		EnUS: "When MuxAfterDone enabled, allow to import local media files. OPTIONS is a colon separated list of:\n\n" +
			"* path=PATH: set file path\n" +
			"* lang=CODE: set media language code (not required)\n" +
			"* name=NAME: set description (not required)\n\n" +
			"Examples: \n" +
			"# import subtitle\n" +
			"--mux-import path=en-US.srt:lang=eng:name=\"English (Original)\"\n" +
			"# import audio and subtitle\n" +
			"--mux-import path=\"D\\:\\media\\atmos.m4a\":lang=eng:name=\"English Description Audio\" --mux-import path=\"D\\:\\media\\eng.vtt\":lang=eng:name=\"English (Description)\"",
	},
	"cmd_custom_range": {
		ZhCN: "下载点播内容时, 仅下载部分分片.\n\n" +
			"例如: \n" +
			"# 下载[0,10]共11个分片\n" +
			"--custom-range 0-10\n" +
			"# 下载从序号10开始的后续分片\n" +
			"--custom-range 10-\n" +
			"# 下载前100个分片\n" +
			"--custom-range -99\n" +
			"# 下载第5分钟到20分钟的内容\n" +
			"--custom-range 05:00-20:00\n",
		ZhTW: "下載點播內容時, 僅下載部分分片.\n\n" +
			"例如: \n" +
			"# 下載[0,10]共11個分片\n" +
			"--custom-range 0-10\n" +
			"# 下載從序號10開始的後續分片\n" +
			"--custom-range 10-\n" +
			"# 下載前100個分片\n" +
			"--custom-range -99\n" +
			"# 下載第5分鐘到20分鐘的內容\n" +
			"--custom-range 05:00-20:00\n",
		EnUS: "Download only part of the segments when downloading vod content.\n\n" +
			"Examples: \n" +
			"# Download [0,10], a total of 11 segments\n" +
			"--custom-range 0-10\n" +
			"# Download subsequent segments starting from index 10\n" +
			"--custom-range 10-\n" +
			"# Download the first 100 segments\n" +
			"--custom-range -99\n" +
			"# Download content from the 05:00 to 20:00\n" +
			"--custom-range 05:00-20:00\n",
	},
	"cmd_selectVideo_more": {
		ZhCN: "通过正则表达式选择符合要求的视频流. 你能够以:分隔形式指定如下参数:\n\n" +
			"id=REGEX:lang=REGEX:name=REGEX:codecs=REGEX:res=REGEX:frame=REGEX\n" +
			"segsMin=number:segsMax=number:ch=REGEX:range=REGEX:url=REGEX\n" +
			"plistDurMin=hms:plistDurMax=hms:bwMin=int:bwMax=int:role=string:for=FOR\n\n" +
			"* for=FOR: 选择方式. best[number], worst[number], all (默认: best)\n\n" +
			"例如: \n" +
			"# 选择最佳视频\n" +
			"-sv best\n" +
			"# 选择4K+HEVC视频\n" +
			"-sv res=\"3840*\":codecs=hvc1:for=best\n" +
			"# 选择长度大于1小时20分钟30秒的视频\n" +
			"-sv plistDurMin=\"1h20m30s\":for=best\n" +
			"-sv role=\"main\":for=best\n" +
			"# 选择码率在800Kbps至1Mbps之间的视频\n" +
			"-sv bwMin=800:bwMax=1000\n",
		ZhTW: "通過正則表達式選擇符合要求的影片軌. 你能夠以:分隔形式指定如下參數:\n\n" +
			"id=REGEX:lang=REGEX:name=REGEX:codecs=REGEX:res=REGEX:frame=REGEX\n" +
			"segsMin=number:segsMax=number:ch=REGEX:range=REGEX:url=REGEX\n" +
			"plistDurMin=hms:plistDurMax=hms:bwMin=int:bwMax=int:role=string:for=FOR\n\n" +
			"* for=FOR: 選擇方式. best[number], worst[number], all (默認: best)\n\n" +
			"例如: \n" +
			"# 選擇最佳影片\n" +
			"-sv best\n" +
			"# 選擇4K+HEVC影片\n" +
			"-sv res=\"3840*\":codecs=hvc1:for=best\n" +
			"# 選擇長度大於1小時20分鐘30秒的影片\n" +
			"-sv plistDurMin=\"1h20m30s\":for=best\n" +
			"-sv role=\"main\":for=best\n" +
			"# 選擇碼率在800Kbps至1Mbps之間的影片\n" +
			"-sv bwMin=800:bwMax=1000\n",
		EnUS: "Select video streams by regular expressions. OPTIONS is a colon separated list of:\n\n" +
			"id=REGEX:lang=REGEX:name=REGEX:codecs=REGEX:res=REGEX:frame=REGEX\n" +
			"segsMin=number:segsMax=number:ch=REGEX:range=REGEX:url=REGEX\n" +
			"plistDurMin=hms:plistDurMax=hms:bwMin=int:bwMax=int:role=string:for=FOR\n\n" +
			"* for=FOR: Select type. best[number], worst[number], all (Default: best)\n\n" +
			"Examples: \n" +
			"# select best video\n" +
			"-sv best\n" +
			"# select 4K+HEVC video\n" +
			"-sv res=\"3840*\":codecs=hvc1:for=best\n" +
			"# Select best video with duration longer than 1 hour 20 minutes 30 seconds\n" +
			"-sv plistDurMin=\"1h20m30s\":for=best\n" +
			"-sv role=\"main\":for=best\n" +
			"# Select video with bandwidth between 800Kbps and 1Mbps\n" +
			"-sv bwMin=800:bwMax=1000\n",
	},
	"cmd_selectAudio_more": {
		ZhCN: "通过正则表达式选择符合要求的音频流. 参考 --select-video\n\n" +
			"例如: \n" +
			"# 选择所有音频\n" +
			"-sa all\n" +
			"# 选择最佳英语音轨\n" +
			"-sa lang=en:for=best\n" +
			"# 选择最佳的2条英语(或日语)音轨\n" +
			"-sa lang=\"ja|en\":for=best2\n" +
			"-sa role=\"main\":for=best\n",
		ZhTW: "通過正則表達式選擇符合要求的音軌. 參考 --select-video\n\n" +
			"例如: \n" +
			"# 選擇所有音訊\n" +
			"-sa all\n" +
			"# 選擇最佳英語音軌\n" +
			"-sa lang=en:for=best\n" +
			"# 選擇最佳的2條英語(或日語)音軌\n" +
			"-sa lang=\"ja|en\":for=best2\n" +
			"-sa role=\"main\":for=best\n",
		EnUS: "Select audio streams by regular expressions. ref --select-video\n\n" +
			"Examples: \n" +
			"# select all\n" +
			"-sa all\n" +
			"# select best eng audio\n" +
			"-sa lang=en:for=best\n" +
			"# select best 2, and language is ja or en\n" +
			"-sa lang=\"ja|en\":for=best2\n" +
			"-sa role=\"main\":for=best\n",
	},
	"cmd_selectSubtitle_more": {
		ZhCN: "通过正则表达式选择符合要求的字幕流. 参考 --select-video\n\n" +
			"例如: \n" +
			"# 选择所有字幕\n" +
			"-ss all\n" +
			"# 选择所有带有\"中文\"的字幕\n" +
			"-ss name=\"中文\":for=all\n",
		ZhTW: "通過正則表達式選擇符合要求的字幕流. 參考 --select-video\n\n" +
			"例如: \n" +
			"# 選擇所有字幕\n" +
			"-ss all\n" +
			"# 選擇所有帶有\"中文\"的字幕\n" +
			"-ss name=\"中文\":for=all\n",
		EnUS: "Select subtitle streams by regular expressions. ref --select-video\n\n" +
			"Examples: \n" +
			"# select all subs\n" +
			"-ss all\n" +
			"# select all subs containing \"English\"\n" +
			"-ss name=\"English\":for=all\n",
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
	"cmd_skipDownload": {
		ZhCN: "跳过下载",
		ZhTW: "跳過下載",
		EnUS: "Skip download",
	},
	"cmd_skipMerge": {
		ZhCN: "跳过合并分片",
		ZhTW: "跳過合併分片",
		EnUS: "Skip segments merge",
	},
	"cmd_subtitleFix": {
		ZhCN: "自动修正字幕",
		ZhTW: "自動修正字幕",
		EnUS: "Automatically fix subtitles",
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
	"cmd_writeMetaJson": {
		ZhCN: "解析后的信息是否输出json文件",
		ZhTW: "解析後的訊息是否輸出json文件",
		EnUS: "Write meta json after parsed",
	},
	"startDownloading": {
		ZhCN: "开始下载...",
		ZhTW: "開始下載...",
		EnUS: "Start downloading...",
	},
	"fetch": {
		ZhCN: "获取: ",
		ZhTW: "獲取: ",
		EnUS: "Fetch: ",
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
	"keyProcessorNotFound": {
		ZhCN: "找不到支持的Processor",
		ZhTW: "找不到支持的Processor",
		EnUS: "No Processor matched",
	},
	"matchTS": {
		ZhCN: "内容匹配: [white on green3]HTTP Live MPEG2-TS[/]",
		ZhTW: "內容匹配: [white on green3]HTTP Live MPEG2-TS[/]",
		EnUS: "Content Matched: [white on green3]HTTP Live MPEG2-TS[/]",
	},
	"matchDASH": {
		ZhCN: "内容匹配: [white on mediumorchid1]Dynamic Adaptive Streaming over HTTP[/]",
		ZhTW: "內容匹配: [white on mediumorchid1]Dynamic Adaptive Streaming over HTTP[/]",
		EnUS: "Content Matched: [white on mediumorchid1]Dynamic Adaptive Streaming over HTTP[/]",
	},
	"matchMSS": {
		ZhCN: "内容匹配: [white on steelblue1]Microsoft Smooth Streaming[/]",
		ZhTW: "內容匹配: [white on steelblue1]Microsoft Smooth Streaming[/]",
		EnUS: "Content Matched: [white on steelblue1]Microsoft Smooth Streaming[/]",
	},
	"matchBinaryData": {
		ZhCN: "内容匹配: [white on deepskyblue1]Binary Data[/]",
		ZhTW: "內容匹配: [white on deepskyblue1]Binary Data[/]",
		EnUS: "Content Matched: [white on deepskyblue1]Binary Data[/]",
	},
	"notSupported": {
		ZhCN: "当前输入不受支持 ",
		ZhTW: "當前輸入不受支援 ",
		EnUS: "Input not supported ",
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
	"downloadProgressWithSpeed": {
		ZhCN: "%s 下载进度 %d/%d，速度 %s",
		ZhTW: "%s 下載進度 %d/%d，速度 %s",
		EnUS: "%s download progress %d/%d, speed %s",
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
