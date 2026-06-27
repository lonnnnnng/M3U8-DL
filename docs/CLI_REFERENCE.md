# m3u8dl-go 功能、用法与参数参考

更新时间：2026-06-27（北京时间）

`m3u8dl-go` 是基于 `nilaoda/N_m3u8DL-RE` 源码行为复刻的 Go 版 HLS 下载器。本项目只实现 HLS/m3u8；DASH、MSS 和 Live TS 只做输入类型识别并返回不支持。

## 快速开始

自动选择最佳轨道并用 ffmpeg 输出 MP4：

```zsh
go run . "https://example.com/index.m3u8" --auto-select true --save-dir ./downloads
```

只解析，不下载，写出 `raw.m3u8`、`meta.json`、`meta_selected.json`：

```zsh
go run . "https://example.com/index.m3u8" --skip-download true --save-name probe
```

只探测资源并输出机器可读轨道摘要，不写文件、不下载分片：

```zsh
go run . "https://example.com/index.m3u8" --probe-json --auto-select true
```

下载前 3 个分片并二进制直拼为 TS：

```zsh
go run . "https://example.com/index.m3u8" --auto-select true --custom-range 0-2 --binary-merge true
```

带 Cookie/Referer 下载：

```zsh
go run . "https://example.com/index.m3u8" -H "Cookie: xxx" -H "Referer: https://example.com"
```

下载完成后最终混流为 MP4：

```zsh
go run . "https://example.com/index.m3u8" --auto-select true -M format=mp4:muxer=ffmpeg
```

录制直播 5 分钟并实时追加输出：

```zsh
go run . "https://example.com/live.m3u8" --live-real-time-merge true --live-record-limit 00:05:00
```

## 已实现功能

- HLS master/media playlist 解析：`EXT-X-STREAM-INF`、`EXT-X-MEDIA`、`EXTINF`、`EXT-X-MAP`、`EXT-X-BYTERANGE`、`EXT-X-DISCONTINUITY`、`EXT-X-PROGRAM-DATE-TIME`、`EXT-X-ENDLIST`。
- 输入支持：HTTP/HTTPS URL、`file:` URL、本地 m3u8 文件。
- 非 HLS 类型识别：DASH/MSS/Live TS/Binary 会输出上游风格匹配提示后返回不支持。
- HTTP 能力：默认 User-Agent、自定义 Header、系统代理、自定义代理、超时、重试、gzip/deflate/br 解压、跳转后保留 Header/Range。
- 下载能力：分片并发、多轨并发、限速、断点复用、BYTERANGE、大文件 Range 切分、分片数量校验。
- HLS key：HTTP key、本地 key、`base64:`、`data:;base64,`、`data:text/plain;base64,`，失败后按上游降级。
- HLS 解密：AES-128 CBC、AES-128 ECB、CHACHA20、自定义 method/key/iv。
- MP4/CENC 解密入口：CENC、SAMPLE-AES、SAMPLE-AES-CTR 通过 `mp4decrypt`、`shaka-packager` 或 `ffmpeg` 处理。
- KID/PSSH：解析 `tenc`、`schm`、Widevine PSSH、PlayReady PSSH，支持 key file 按 KID 查找。
- 轨道选择：自动选择、交互选择、视频/音频/字幕选择过滤、丢弃过滤、只选字幕。
- 输出：保存目录、临时目录、保存名、保存模板、raw/meta JSON、日志文件。
- 资源探测：`--probe-json` 可解析 m3u8/master/子 playlist，并输出视频、音频、字幕轨道数量、直播/点播、分片数、时长、加密方式和自动选择结果。
- 合并：二进制合并、ffmpeg concat 协议、ffmpeg concat demuxer、最终 ffmpeg/mkvmerge 混流。
- 字幕：VTT 修复、VTT 转 SRT、TTML、MP4 WebVTT/TTML 基础抽取、图形字幕 PNG 落盘。
- 直播：刷新轮询、新分片追加、录制时长限制、实时合并、PipeMux、直播 VTT 音频时间轴修正。
- 多语言：`zh-CN`、`zh-TW`、`en-US`，未指定时默认简体中文，繁中系统默认繁中。
- 更新检查：GitHub latest release，可关闭。

## 合并方式选择

二进制合并：

- 参数：`--binary-merge true`
- 输出：通常保留原始扩展，例如 TS 分片输出 `.ts`。
- 特点：按已下载、已解密分片顺序直接拼接，不转码、不重封装，速度快，保留原始结构。
- 适合：TS 原始保真、fMP4/CENC/未知加密、排障、直播实时追加。

ffmpeg 单轨合并：

- 默认普通 TS 点播会走 ffmpeg，除音频外输出 `.mp4`。
- 特点：`-c copy` 重新封装，不转码；播放器兼容性通常更好。
- 适合：普通观看、需要 MP4/M4A 文件。

最终混流：

- 参数：`-M/--mux-after-done`
- 用途：把已下载的视频、音频、字幕和 `--mux-import` 外部轨道混成最终 MP4/MKV/TS。
- 注意：开启 `-M` 时会自动启用二进制合并，先保留单轨输出，再交给最终混流器。

真实样本对比见 [REAL_SAMPLE_VALIDATION.md](REAL_SAMPLE_VALIDATION.md)。

## 参数总览

### 基础

| 参数 | 说明 |
| --- | --- |
| `<input>` | 输入 URL、`file:` URL 或本地 m3u8 文件。 |
| `-h, --help, -?` | 显示帮助信息。 |
| `--version` | 显示版本信息。 |
| `--version-json` | 以 JSON 输出版本信息，供桌面端或脚本识别当前下载核心。 |
| `--capabilities-json` | 以 JSON 输出当前核心能力清单，包含 HLS 范围、输入类型、下载/解密/合并/字幕/直播/诊断能力和明确不支持项，供桌面端或脚本按当前 CLI 能力渲染界面。 |
| `--doctor` | 检测 `ffmpeg`、`ffprobe`、`mkvmerge`、`mp4decrypt`、`shaka-packager` 等外部工具；如果指定了 `--ffmpeg-binary-path`，会优先检测同目录 `ffprobe`。 |
| `--doctor-json` | 以 JSON 输出外部工具检测结果，适合脚本或客户端集成。 |
| `--print-effective-options` | 以 JSON 输出解析、校验和派生后的有效参数；会隐藏 Cookie、Authorization、代理密码和 key 原文。 |
| `--probe-json` | 解析资源并以 JSON 输出轨道摘要，不下载分片、不写 meta 文件；stdout 保持纯 JSON，适合桌面端预检查或脚本探测资源。 |
| `--ui-language <zh-CN\|zh-TW\|en-US>` | 设置 UI 语言；不传时默认简体中文，繁中系统默认繁中。 |

### 输入与网络

| 参数 | 说明 |
| --- | --- |
| `--base-url <url>` | 设置 BaseURL，用于修正相对 URL。 |
| `-H, --header <header>` | 设置 HTTP 请求头，可重复传入，例如 `-H "Cookie: xxx" -H "User-Agent: iOS"`。 |
| `--urlprocessor-args <args>` | 传给 URL Processor 的参数。HLS 复刻版仅保留参数入口。 |
| `--use-system-proxy [true\|false]` | 是否使用系统代理，默认启用。 |
| `--custom-proxy <url>` | 设置请求代理，例如 `http://127.0.0.1:8888`。 |
| `--append-url-params [true\|false]` | 把输入 URL 查询参数追加到分片、init 和 HLS key URL。 |
| `--http-request-timeout <seconds>` | HTTP 请求超时，支持小数秒。 |

### 下载控制

| 参数 | 说明 |
| --- | --- |
| `--auto-select [true\|false]` | 自动选择最佳视频、每语言最高码率音频和全部字幕。 |
| `--thread-count <n>` | 分片下载线程数。 |
| `--download-retry-count <n>` | 单分片下载异常重试次数。 |
| `-R, --max-speed <15M\|100K>` | 总下载限速，只接受 `K`/`M` 单位。 |
| `-mt, --concurrent-download [true\|false]` | 并发下载已选择的音视频字幕轨。 |
| `--check-segments-count [true\|false]` | 校验实际下载分片数和预期是否一致，默认开启。 |
| `--skip-download [true\|false]` | 只解析和写 meta，不下载分片。 |
| `--skip-merge [true\|false]` | 下载分片但跳过合并，保留任务临时目录。 |
| `--binary-merge [true\|false]` | 二进制直拼分片。 |
| `--use-ffmpeg-concat-demuxer [true\|false]` | ffmpeg 合并时使用 concat demuxer，而不是 concat 协议。 |
| `--del-after-done [true\|false]` | 完成后删除临时文件，默认开启。 |
| `--allow-hls-multi-ext-map [true\|false]` | 允许 HLS 多个 `EXT-X-MAP`，实验性。 |
| `--disable-update-check [true\|false]` | 禁用版本更新检查。 |

### 输出与日志

| 参数 | 说明 |
| --- | --- |
| `--save-dir <dir>` | 输出目录。 |
| `--save-name <name>` | 保存文件名，按上游规则清理非法字符。 |
| `--save-pattern <pattern>` | 保存名模板。支持 `<SaveName>`、`<Id>`、`<Codecs>`、`<Language>`、`<Resolution>`、`<Bandwidth>`、`<MediaType>`、`<Channels>`、`<FrameRate>`、`<VideoRange>`、`<GroupId>`、`<Ext>`。 |
| `--tmp-dir <dir>` | 临时目录。raw/meta 和分片目录会放在 `<tmp-dir>/<save-name>/`。 |
| `--write-meta-json [true\|false]` | 是否写出 `raw.m3u8`、`meta.json`、`meta_selected.json`。 |
| `--log-level <level>` | 日志等级，支持上游枚举值。 |
| `--log-file-path <path>` | 日志文件路径。 |
| `--no-log [true\|false]` | 关闭日志文件输出。 |
| `--force-ansi-console [true\|false]` | 强制认为终端支持 ANSI 且可交互。 |
| `--no-ansi-color [true\|false]` | 去除 ANSI 颜色。 |
| `--progress-json [true\|false]` | 按行输出机器可读事件 JSON。`progress` 事件包含 `timestamp`、`stream`、`current`、`total`、`speed`、`bytes`、`percent`；成功完成时会追加 `summary` 事件和实际存在的输出文件列表；参数解析失败或运行期失败会追加 `error` 事件和错误信息；stdout 保持纯 JSON 行，默认日志文件会记录同一事件，桌面端默认开启。 |
| `--no-date-info [true\|false]` | 混流时不写入日期 metadata。 |

### 字幕与选流

| 参数 | 说明 |
| --- | --- |
| `--sub-only [true\|false]` | 只选择字幕轨。 |
| `--sub-format <SRT\|VTT>` | 字幕输出格式。 |
| `--auto-subtitle-fix [true\|false]` | 自动修复字幕时间轴并按 `--sub-format` 输出。 |
| `-sv, --select-video <filter>` | 选择视频轨，详情见 `--morehelp select-video`。 |
| `-sa, --select-audio <filter>` | 选择音频轨，详情见 `--morehelp select-audio`。 |
| `-ss, --select-subtitle <filter>` | 选择字幕轨，详情见 `--morehelp select-subtitle`。 |
| `-dv, --drop-video <filter>` | 丢弃匹配的视频轨。 |
| `-da, --drop-audio <filter>` | 丢弃匹配的音频轨。 |
| `-ds, --drop-subtitle <filter>` | 丢弃匹配的字幕轨。 |
| `--ad-keyword <regex>` | 按 URL 正则清理广告分片，可重复传入。 |
| `--custom-range <range>` | 只下载部分分片或时间范围，详情见 `--morehelp custom-range`。 |

过滤器常用字段包括 `id`、`lang`、`name`、`codec`、`res`、`frame`、`channels`、`url`、`segsMin`、`segsMax`、`plistDurMin`、`plistDurMax`、`bwMin`、`bwMax`、`role`、`for=best|bestN|worstN|all`。

### 解密

| 参数 | 说明 |
| --- | --- |
| `--key <KID:KEY\|KEY>` | MP4 外部解密 key。支持 raw key、`KID:KEY`、`trackId:KEY`，可重复传入。 |
| `--key-text-file <file>` | KID/key 文本文件；已知 KID 时按前缀匹配第一条。 |
| `--decryption-engine <engine>` | 外部解密引擎：`MP4DECRYPT`、`SHAKA_PACKAGER`、`FFMPEG`。 |
| `--decryption-binary-path <path>` | 外部解密工具路径。 |
| `--mp4-real-time-decryption [true\|false]` | 下载 MP4/fMP4 分片时实时外部解密。 |
| `--use-shaka-packager [true\|false]` | 使用 shaka-packager 替代 mp4decrypt。 |
| `--custom-hls-method <method>` | 覆盖 HLS 加密方式：`AES_128`、`AES_128_ECB`、`CENC`、`CHACHA20`、`NONE`、`SAMPLE_AES`、`SAMPLE_AES_CTR`、`UNKNOWN`。 |
| `--custom-hls-key <file\|hex\|base64>` | 覆盖 HLS key。 |
| `--custom-hls-iv <file\|hex\|base64>` | 覆盖 HLS IV。 |

### 直播

| 参数 | 说明 |
| --- | --- |
| `--live-perform-as-vod [true\|false]` | 把直播当点播下载当前窗口。 |
| `--live-real-time-merge [true\|false]` | 录制直播时边下载边追加输出。 |
| `--live-keep-segments [true\|false]` | 实时合并时是否保留分片。 |
| `--live-pipe-mux [true\|false]` | 通过管道和 ffmpeg 实时混流到 TS。 |
| `--live-record-limit <duration>` | 直播录制时长，例如 `00:05:00`。 |
| `--live-wait-time <seconds>` | 手动设置直播 playlist 刷新间隔。 |
| `--live-take-count <n>` | 首次取直播分片数量。 |
| `--live-fix-vtt-by-audio [true\|false]` | 读取音频起始时间修正直播 VTT 字幕。 |
| `--task-start-at <yyyyMMddHHmmss>` | 到指定时间才开始执行。 |

### 混流与更多帮助

| 参数 | 说明 |
| --- | --- |
| `-M, --mux-after-done <options>` | 下载完成后最终混流。详情见 `--morehelp mux-after-done`。 |
| `--mux-import <options>` | 最终混流时导入外部音轨/字幕。详情见 `--morehelp mux-import`。 |
| `--ffmpeg-binary-path <path>` | ffmpeg 可执行文件路径。 |
| `--morehelp <topic>` | 查看复杂参数详细帮助。 |

`--morehelp` 支持：

- `mux-after-done`
- `mux-import`
- `custom-range`
- `select-video`
- `select-audio`
- `select-subtitle`

## 复杂参数速查

最终混流：

```zsh
-M format=mp4:muxer=ffmpeg
-M format=mkv:muxer=mkvmerge:bin_path=/opt/homebrew/bin/mkvmerge:keep=true:skip_sub=false
```

导入外部字幕：

```zsh
--mux-import path=zh-Hans.srt:lang=chi:name="中文 (简体)"
```

范围下载：

```zsh
--custom-range 0-10
--custom-range -99
--custom-range 05:00-20:00
```

轨道选择：

```zsh
-sv res="1920x1080":for=best
-sa lang="ja|en":for=best2
-ss all
-dv codec="h265|hev1"
```

## 已知边界

- 不实现 DASH/MSS 下载。
- 不提供 Bilibili 页面解析；需要用户自行提供真实 m3u8 URL、headers、cookies 和 key。
- CENC/SAMPLE-AES 真实样本仍需继续补充验证。
- Windows PipeMux 已实现并交叉编译，但未在真实 Windows 环境运行验证。
- 没有完整复刻原版 Spectre Console 动态进度 UI。
