# 功能清单

来源：已读取上游 `README.md`、`Program.cs`、`CommandInvoker.cs`、`StreamExtractor.cs`、`HLSExtractor.cs`、`SimpleDownloadManager.cs`、`SimpleDownloader.cs`、`AESUtil.cs`、`MergeUtil.cs`。

## 原项目总体能力

- 输入：URL、`file:` URL、本地播放列表文件。
- 支持协议：HLS/m3u8、DASH/mpd、MSS/ism；本 Go 复刻版只实现 HLS。
- 播放列表识别：根据正文识别 HLS、DASH、MSS、直播 TS、二进制异常。
- 多语言 CLI：`zh-CN`、`zh-TW`、`en-US`。
- 更新检查、日志文件、ANSI 进度条。
- HTTP：默认 User-Agent、自定义 Header、playlist/key 默认 `Accept-Encoding: gzip, deflate` 与 `Cache-Control: no-cache`、系统代理、自定义代理、超时、重试、gzip/deflate/br 响应解压。
- Master playlist：解析视频、音频、字幕、分辨率、码率、语言、名称、频道、HDR/DV 标记。
- Media playlist：解析分片、起始序号、时长、`EXT-X-MAP`、`EXT-X-BYTERANGE`、`EXT-X-DISCONTINUITY`、`EXT-X-PROGRAM-DATE-TIME`，并兼容上游 `DateTime.Parse` 可接受的紧凑时区偏移/无时区时间、直播刷新间隔。
- HLS 加密：`AES-128`、`AES-128-ECB`、`CHACHA20`、`CENC`、`SAMPLE-AES`、`SAMPLE-AES-CTR`、自定义 key/iv/method、key 加载重试与失败降级。
- MP4/CENC 解密：调用 `mp4decrypt`、`shaka-packager` 或 `ffmpeg`。
- 下载：并发分片、重试、限速、断点式跳过已下载分片、单大文件 range 切分、30x 跳转后保留自定义 header/range。
- 选择：自动选择最佳轨道；通过 `-sv/-sa/-ss` 和 `-dv/-da/-ds` 正则筛选/丢弃。
- 输出：保存目录、临时目录、保存名、保存模板、meta JSON、日志。
- 合并：二进制合并、ffmpeg concat 协议、ffmpeg concat demuxer、mkvmerge/ffmpeg 混流。
- 字幕：VTT/TTML 修复、VTT 转 SRT、MP4 内嵌字幕抽取。
- 直播：按刷新间隔录制、实时合并、pipe mux、录制时长限制。
- 清理：下载完成后清理临时文件。

## Go 版当前实现范围

- 已实现：HLS Master/Media 解析、源/子 playlist HTTP 文本加载按上游默认重试 10 次，并会按响应 `Content-Type` 的 `charset` 解码非 UTF-8 播放列表文本，本地普通路径按上游规范化为绝对 `file://` URL 以正确解析同目录相对分片和 HLS key，Master 子 playlist 失效时重新抓取 master 刷新子 URL，HLS 逐行解析支持长签名 URL/data URI 这类超过 64KB 的长行，HLS URL 合并、请求头且 `--header` 支持一次传入多个 header、playlist/key 请求默认带上游 `Accept-Encoding: gzip, deflate` 与 `Cache-Control: no-cache`、HTTP 30x 跳转后保留自定义 header 与 BYTERANGE 的 `Range` 请求头、gzip/deflate/br 压缩响应解码、代理、`--custom-proxy` URL 提前校验、重试、并发下载、重跑时复用已下载分片和已生成的 `_dec` 解密分片、并发多轨下载返回顺序按输入轨道稳定保留、单大分片在服务端支持 Range 时自动拆成多个字节区间并发下载、响应 `Content-Length` 与 `EXT-X-BYTERANGE` 期望长度校验、图片伪装分片头剥离、原始 gzip 分片内容解压，且按上游保持为 HLS 解密后再剥头/解压、`base64://`、`hex://`、`file:` 分片 URL 读取、AES-128 CBC/ECB、CHACHA20、HLS key 加载按上游重试 3 次且失败后降级为 `UNKNOWN`、无法识别的 HLS 加密方式按 `UNKNOWN` 保留原始分片并自动启用二进制合并、检测到 fMP4 或 CENC 时自动启用二进制合并、`EXT-X-MAP`、多 `EXT-X-MAP` 默认截断及 `--allow-hls-multi-ext-map` 继续解析并按新 MAP 切分 `MediaPart`、空直播窗口按上游保留一个空 `MediaPart`、fMP4 直播刷新按上游保留首次解析到的 `EXT-X-MAP` init、`EXT-X-BYTERANGE`、重复 `EXT-X-KEY` 复用、自定义 HLS key/iv/method，且 `--custom-hls-method` 兼容上游下划线枚举名并按上游枚举提前拒绝非法值、`--append-url-params` 会按上游作用到媒体分片 URL、init URL 和 HLS key URL、轨道选择/过滤按上游质量顺序处理 `Bandwidth desc -> Channels desc`，同码率音轨自动选择更高声道版本、`--custom-range` 开区间和字幕偏移时长按上游语义处理，冒号时长按上游从右到左解释为秒/分/时/天并兼容中文冒号，直播流按上游跳过 `--custom-range`、`--live-wait-time` 未显式指定时按上游用当前直播窗口时长自动计算刷新间隔、`--ad-keyword` 正则广告分片过滤且支持一次传入多个正则、布尔选项显式 `true/false` 解析、`--ui-language`/`--log-level`/`--sub-format`/`--decryption-engine` 按上游白名单或枚举值校验，需要参数值的 CLI 选项缺值时按上游在解析阶段报错，`--thread-count`、`--download-retry-count`、`--live-wait-time`、`--live-take-count` 等数值选项按上游在解析阶段拒绝非法整数，核心运行输出、自动派生选项提示、自动二进制合并提示、实时解密引擎建议、更新检查提示和下载进度已按 `zh-CN/zh-TW/en-US` 本地化、`--check-segments-count` 严格校验及关闭后跳过失败分片、二进制/ffmpeg 合并，单轨 ffmpeg 输出支持 `mp4/mkv/flv/ts/m4a/aac/eac3/ac3`，TS 使用 `h264_mp4toannexb`，AAC 音频只在上游会启用的格式中条件添加 `aac_adtstoasc`，VTT/SRT、TTML、MP4 WebVTT/TTML 字幕抽取与时间轴处理，且 MP4 WebVTT 会按上游保留 `iden` cue id、超长分片列表会先分批二进制预合并、直播刷新追加和多轨起点同步、`raw.m3u8`、全部轨道 `meta.json` 与选中轨道 `meta_selected.json`，`--write-meta-json false` 可关闭写出且重跑时不覆盖已有 raw/meta 文件、日志文件写出且 `--log-file-path` 按上游清理非法文件名、`--no-log` 禁用、`--log-level` 日志文件等级过滤、GitHub latest release 更新检查与 `--disable-update-check`。
- 已实现：`CENC`、`SAMPLE-AES`、`SAMPLE-AES-CTR` 下载后外部解密入口，支持 `mp4decrypt`、`shaka-packager`、`ffmpeg` 形式；`SAMPLE-AES`/`SAMPLE-AES-CTR` 按上游不做伪原生分片解密，保留分片给 MP4 外部工具链处理；多 key 会按当前 KID 选择匹配项，单 key 作为兜底；CENC MP4 info 会集中解析 `schm` scheme、`tenc` default_KID、Widevine PSSH data、Widevine PSSH v1 KID 列表、protobuf `key_id` 和 PlayReady PSSH XML，PlayReady 同时支持 `<KID>...</KID>` 文本节点和 `<KID VALUE="...">` 属性写法，且 `tenc default_KID` 全 0 时按上游继续用 Widevine PSSH 真实 KID 匹配 key-file，并给 `mp4decrypt/shaka-packager` 使用 track/label `1` 的 MultiDRM 参数形态；非法 PSSH version 会像上游一样被拒绝；使用 `SHAKA_PACKAGER` 且本地解析不到 KID 时，会按上游从 shaka 的缺 key 错误里探测 `key_id`；Apple/Zero KID 场景会按上游给 `mp4decrypt` 使用 track id `1:key`，给 shaka-packager 使用 `label=1:key_id=000...`；`mp4decrypt` 按上游先在媒体目录临时改名并切换工作目录执行，同目录 `--fragments-info` 使用相对 init 文件名以规避中文/特殊路径问题；支持 `--key`、`--key-text-file`、`--decryption-engine`、`--decryption-binary-path`；`--key` 按上游接受 16 字节 hex/base64、`KID:KEY` 与 `trackId:KEY`，支持一次传入多条 key，并在命令行阶段校验格式；传入 key/key-file 时会提前校验外部解密工具；`--mp4-real-time-decryption` 已覆盖 init+fragment 的基础实时分片外部解密路径，且会先用原始 init 读取 KID；本地读不到 KID 且使用 shaka 时，会先从 shaka 缺 key 错误里探测 `key_id`；shaka/ffmpeg 会按上游用 init+fragment 构造外部工具输入，并避免最终合并时重复写入独立 init；实时解密开启后不会在最终合并产物上重复跑整文件解密。
- 已实现：最终 ffmpeg 多轨混流和 mkvmerge 混流，支持 `--mux-import`、`skip_sub`、`keep`、语言/名称元数据；`-M/--mux-after-done` 会按上游拒绝非法格式、非法 muxer、空 `bin_path`、非法布尔值和 `mkvmerge+mp4` 组合；`--mux-import` 支持一次传入多个外部轨道，复杂参数按上游支持单/双引号裁剪和 `\:` 冒号转义，`--morehelp mux-import` 提供 `path/lang/name` 参数说明；混流会按上游完整语言表转换语言码，并在名称为空时填入默认语言描述；ffmpeg 混流会清理输入 metadata、按输出流序号写入语言/标题 metadata、复制未知流，date metadata 按原版 `.NET` round-trip `"o"` 格式写入，MKV 字幕按 `.srt`/WebVTT 输入选择 `srt` 或 `webvtt` 编码，并设置默认视频/首音频/字幕非默认轨道标记；`keep=false` 时会按上游清理实际参与最终混流的下载轨和外部导入轨，并保留被 `skip_sub` 跳过的字幕；下载后会用 `ffprobe` 修正真实媒体类型，字幕 TS 可转入 VTT 修复路径，Dolby Vision 会按上游强制二进制合并并关闭最终混流；输出文件冲突时按上游优先使用分辨率、码率、语言、声道等流元数据生成唯一文件名；`--mux-import` 未搭配 `-M` 时按上游报错，并提前校验外部导入文件存在；开启最终混流时按上游自动启用二进制合并；`--skip-merge` 只保留分片目录，不再触发最终 `-M` 混流。
- 已实现但简化：交互选择直接回车会按上游默认勾选首个基础流及其 `AUDIO`/`SUBTITLES` 引用组，`--sub-only` 和显式 keep filter 会按上游直接返回匹配轨道而不再进入交互，自动选择最佳视频/每语言最高码率音频/全部字幕、VTT 字幕合并和 SRT 输出，空 SRT 按上游写出 1 秒占位字幕；自动字幕修复开启时按 `--sub-format` 决定 `.srt/.vtt` 输出扩展，支持 `X-TIMESTAMP-MAP`/`MPEGTS` 按上游修正多分片 VTT 时间轴；`Base64::` 图形字幕会按上游写出 PNG 并将字幕 payload 替换为图片文件名；字幕修复成功后按上游删除原始字幕分片，TTML/MP4-TTML 分支支持 `RE_KEEP_IMAGE_SEGMENTS=1` 保留分片；`--live-fix-vtt-by-audio` 会读取音频 `start_time` 并等待最多 5 秒用于修正 VTT 字幕整体偏移；裸 TTML 转 VTT/SRT、MP4 WebVTT (`wvtt/vttc/payl`) 抽取并按 init `mdhd` timescale、media `tfdt/tfhd/trun` sample 表计算真实字幕时间、MP4 TTML (`stpp`/`mdat`) 基础抽取；VTT/TTML/MP4 TTML 多分片字幕会按上游用前序 HLS 分片累计时长补齐全局时间轴；可从真实 `stsd` init box 识别未写入 codecs 的 `stpp` 字幕轨、默认保存名推导、`--save-name` 按上游仅替换非法文件名字符并裁掉首尾句点，保留空格/下划线，且拒绝清理后为空的名称、保存模板含 `<FrameRate>` 且按上游清理空字段产生的多余分隔符，`<MediaType>` 使用枚举值、选择/丢弃轨道过滤含 `for=bestN|worstN|all`、`bwMin/bwMax`、`segsMin/segsMax`、`plistDurMin/plistDurMax`、`role`，过滤器会按上游提前校验 `for`、正则和数值参数，且 `role` 非法值会像上游 `Enum.TryParse` 失败一样忽略，`segsMin/segsMax` 只在候选轨道都有分片数时应用，并按上游先 drop 后 keep、`--auto-select` 优先于 `--sub-only` 和 keep filter、`--morehelp`、直播刷新追加按 HLS 上游规则用 `PROGRAM-DATE-TIME` 秒级时间戳或分片序号去重，`PROGRAM-DATE-TIME` 支持紧凑时区偏移和无时区时间以贴近上游容错，避免相同 URL 的新分片被漏录；直播录制分片临时文件按上游优先使用 `PROGRAM-DATE-TIME` 秒级时间戳命名，无日期时回退分片 `Index`，fMP4 init 固定为 `_init.mp4`；直播录制会按上游强制多轨并发和 MP4 实时解密；`--live-record-limit` 按上游用已刷出的分片时长累计，初始直播窗口也计入限制；`--live-real-time-merge` 的非字幕文件输出会在直播刷新过程中按批次下载新增分片并顺序追加，`SIGINT/SIGTERM` 会取消直播等待/刷新并进入已下载内容的收尾阶段，`--live-keep-segments false` 时保留 init、删除普通媒体分片，直播字幕仍在累计刷新后走字幕修复/合并收尾；`--live-pipe-mux` 强制启用 `--live-real-time-merge`；PipeMux 的 ffmpeg 参数构造已按上游覆盖 `RE_LIVE_PIPE_OPTIONS`、`RE_LIVE_PIPE_TMP_DIR`、date metadata 和 Windows/Unix 管道路径差异，macOS/Linux 可创建 FIFO 管道，Windows 可创建 `\\.\pipe\...` 命名管道，启动 mux 进程后直播刷新过程中会把非字幕轨道按批次持续写入 pipe mux，字幕继续作为独立输出保留，pipe mux 产物不会再进入普通 `-M` 最终混流队列。
- 已实现：`--max-speed` 共享限速器，所有并发分片共用同一速度预算；参数解析按上游规则只接受 `K`/`M` 单位，例如 `512K`、`1.5M`，裸数字或 `G` 会直接报错；`--http-request-timeout` 按上游支持小数秒并拒绝非法值；stdout/stderr 被重定向时会按上游自动启用 `ForceANSIConsole` 并清除 ANSI 颜色。
- 已补齐：`--live-fix-vtt-by-audio` 无音频轨时会按上游关闭，直播实时合并收尾下载字幕时会复用已生成音频输出的 `start_time` 修正 VTT 时间轴。
- 暂未完全追平：CENC KID 自动读取已支持结构化 MP4 info、`schm`、`tenc`、Widevine PSSH data、PSSH v1 KID 列表、PlayReady PSSH XML 文本/`VALUE` 属性 KID 解析、`tenc default_KID` 全 0 时按上游回退 Widevine PSSH 真实 KID 和 MultiDRM track/label `1` 参数，以及 shaka 缺 key 错误探测 `key_id`，但仍缺更复杂 DRM/SAMPLE-AES 样本的系统验证；实时解密仍是基础外部工具流程而非完整上游等价实现、ANSI 已对齐重定向无颜色行为但未复刻完整动态进度 UI、多语言资源已覆盖核心运行输出但尚未复刻完整 `ResString` 资源表、直播 producer/consumer 主循环已覆盖非字幕实时合并、macOS/Linux pipe mux 批次写入和系统信号触发的基础收尾，Windows 命名管道已实现并通过交叉编译，仍未达到上游完整多轨状态机。

## HLS 复刻验收点

- 本地 m3u8 文件能解析并下载。
- Master m3u8 能列出轨道并自动选最佳视频。
- AES-128 HLS 能下载并解密。
- fMP4 HLS 能下载 init + m4s 并二进制合并。
- `--header`、`--custom-proxy`、`--base-url`、`--custom-range` 可生效。
- `--skip-download`、`--skip-merge`、`--write-meta-json` 可生效。
- `ffmpeg` 存在时 `-M format=mp4` 可合并输出 mp4。
