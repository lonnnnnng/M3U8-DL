# N_m3u8DL-RE 功能清单与 Go HLS 复刻进度对比

更新时间：2026-06-19 04:59:52（北京时间）

## 代码目录

- 原版代码：`../N_m3u8DL-RE`
- Go HLS 复刻版：`../N_m3u8DL-GO-HLS`

本次整理后，原版 C# 项目和 Go 复刻项目是两个独立目录；Go 复刻目录内不再依赖原版源码路径。

## 依据来源

- 原版说明：`../N_m3u8DL-RE/README.md`、`../N_m3u8DL-RE/README.en.md`
- 原版命令行：`../N_m3u8DL-RE/src/N_m3u8DL-RE/CommandLine/CommandInvoker.cs`
- 原版 HLS 解析：`../N_m3u8DL-RE/src/N_m3u8DL-RE.Parser/Extractor/HLSExtractor.cs`
- 原版 HLS 预处理与 key：`../N_m3u8DL-RE/src/N_m3u8DL-RE.Parser/Processor/HLS/DefaultHLSContentProcessor.cs`、`DefaultHLSKeyProcessor.cs`
- 原版下载、合并、直播：`SimpleDownloader.cs`、`SimpleDownloadManager.cs`、`SimpleLiveRecordManager2.cs`、`MergeUtil.cs`、`MP4DecryptUtil.cs`
- 复刻进度：`../N_m3u8DL-GO-HLS/README.md`、`docs/FUNCTIONS.md`、当前 Go 源码与测试

## 原版 N_m3u8DL-RE 功能清单

### 输入与协议识别

- 支持输入 URL、`file:` URL、本地播放列表文件。
- 自动识别 HLS/m3u8、DASH/mpd、MSS/ism、直播 TS、二进制异常内容。
- 支持 HLS/DASH/MSS 点播，支持 HLS/DASH 直播。
- 支持设置 `--base-url` 和 URL Processor 扩展参数。

### 命令行与配置

- 支持保存目录、临时目录、保存文件名、保存模板、日志路径。
- 支持 `zh-CN`、`zh-TW`、`en-US` UI 语言。
- 支持日志级别、ANSI 颜色控制、更新检查开关。
- 支持任务延迟开始 `--task-start-at`。
- 支持布尔选项默认值和命令行解析。

### HTTP 与下载

- 默认 User-Agent，可叠加/覆盖自定义 Header。
- 支持系统代理、自定义代理、请求超时、下载重试。
- 支持分片并发下载，多轨并发下载。
- 支持断点式跳过已存在分片和已解密分片。
- 支持 BYTERANGE 的 Range 请求和大文件 Range 切分。
- 支持跳转后保留 Header/Range。
- 支持下载限速 `--max-speed`。
- 支持 gzip/deflate/br HTTP 压缩响应自动解压。
- 支持图片伪装分片头剥离和 gzip 分片解压。

### HLS Master 解析

- 解析 `EXT-X-STREAM-INF` 视频基础流。
- 解析 `EXT-X-MEDIA` 音频、字幕、视频变体，跳过 CLOSED-CAPTIONS。
- 解析码率、平均码率、分辨率、帧率、编码、语言、名称、声道、HDR/DV、Group ID。
- 处理音视频/字幕引用关系，例如 `AUDIO`、`SUBTITLES`。

### HLS Media 解析

- 解析 `EXTINF`、`EXT-X-MEDIA-SEQUENCE`、`EXT-X-TARGETDURATION`。
- 解析 `EXT-X-BYTERANGE`、`EXT-X-MAP`、`EXT-X-DISCONTINUITY`。
- 解析 `EXT-X-PROGRAM-DATE-TIME` 并用于直播同步。
- 解析 `EXT-X-PLAYLIST-TYPE`、`EXT-X-ENDLIST`，区分直播和点播。
- 支持多 `EXT-X-MAP` 的默认截断与实验性继续解析。
- 支持上游 HLS 内容预处理：孤立 `\r` 换行、YSP 回放补 `ENDLIST`、Youku/Disney+/AppleTV 修正、`EXT-X-KEY` 顺序修正。
- 支持 Uplynk/Youku 广告分片处理。

### HLS 加密与解密

- 支持 `NONE`、`AES-128`、`AES-128-ECB`、`CHACHA20`、`CENC`、`SAMPLE-AES`、`SAMPLE-AES-CTR`、`UNKNOWN`。
- 支持自定义 HLS method/key/iv。
- 支持 key URI、`base64:`、`data:;base64,`、`data:text/plain;base64,`、本地 key 文件。
- 支持 key 加载重试，失败时降级为 `UNKNOWN`。
- AES-128 CBC/ECB 和 CHACHA20 可内置解密。
- CENC/SAMPLE-AES/SAMPLE-AES-CTR 可调用 `mp4decrypt`、`shaka-packager`、`ffmpeg` 等外部工具。
- 支持 `--key`、`--key-text-file`、`--decryption-engine`、`--decryption-binary-path`。
- 支持 MP4 实时解密 `--mp4-real-time-decryption`。

### 轨道选择与过滤

- 支持交互选择、自动选择最佳轨道、只选字幕。
- 支持 `-sv/-sa/-ss` 选择视频/音频/字幕。
- 支持 `-dv/-da/-ds` 丢弃视频/音频/字幕。
- 过滤条件包括 ID、语言、名称、编码、分辨率、帧率、声道、HDR/DV、URL、分片数、播放列表时长、码率范围、Role。
- 支持 `for=best`、`bestN`、`worstN`、`all`。
- 支持按广告 URL 正则清理分片。

### 输出、元数据与清理

- 支持 `raw.m3u8`、`meta.json`、`meta_selected.json`。
- 支持保存模板变量：`SaveName`、`Id`、`Codecs`、`Language`、`Resolution`、`Bandwidth`、`MediaType`、`Channels`、`FrameRate`、`VideoRange`、`GroupId`、`Ext`。
- 支持输出文件冲突处理。
- 支持 `--skip-download`、`--skip-merge`、`--del-after-done`。
- 支持下载完成后清理临时分片。

### 合并与混流

- 支持二进制合并。
- 支持 ffmpeg concat 协议和 concat demuxer。
- 支持 ffmpeg/mkvmerge 最终混流。
- 支持 `-M/--mux-after-done` 指定容器、muxer、工具路径、是否保留源文件、是否跳过字幕。
- 支持 `--mux-import` 引入外部音轨/字幕。
- 支持写入语言、标题、默认轨道等元数据。
- 支持 `ffprobe`/mediainfo 读取媒体类型并修正输出。

### 字幕

- 支持 VTT 修复、VTT 转 SRT。
- 支持 TTML 处理。
- 支持 MP4 内嵌 WebVTT/TTML 抽取。
- 支持 `X-TIMESTAMP-MAP`、`MPEGTS` 时间轴修正。
- 支持图形字幕 Base64 PNG 落盘。
- 支持直播 VTT 通过音频起始时间修正。

### 直播

- 支持直播刷新轮询和新增分片追加。
- 支持录制时长限制。
- 支持初始直播窗口裁剪和多轨同步。
- 支持实时合并 `--live-real-time-merge`。
- 支持保留/删除直播分片 `--live-keep-segments`。
- 支持 pipe mux `--live-pipe-mux`，用 ffmpeg 管道实时混流 TS。
- 支持直播刷新去重、序号修正、按 `PROGRAM-DATE-TIME` 或文件名判断新增分片。

### 非 HLS 能力

- DASH 解析、下载、直播录制、SegmentTemplate/SegmentList/SegmentBase 处理。
- MSS 解析、下载、直播录制、moov 生成/修正。
- Live TS 直接录制。
- bitmovin/nowehoryzonty 等 DASH 专用处理器。

## Go HLS 复刻进度对比

状态说明：

- 已追平：当前 Go 版已有实现，并有源码或测试覆盖证明核心语义。
- 基本追平：核心行为已实现，但 UI、边缘平台或复杂场景仍弱于原版。
- 部分追平：只覆盖常见或基础路径，仍缺重要分支。
- 未追平：当前 Go 版没有等价实现。
- 不在范围：用户明确要求本复刻只做 HLS，不做 DASH/MSS。

| 功能域 | 原版能力 | Go HLS 复刻进度 | 状态 |
| --- | --- | --- | --- |
| 目录隔离 | 原项目独立源码树 | 已整理为 `N_m3u8DL-RE/` 与 `N_m3u8DL-GO-HLS/` 两个目录 | 已追平 |
| HLS 输入 | URL、`file:`、本地 m3u8 | 支持 HTTP、本地文件、`file:` 及本地相对路径规范化 | 已追平 |
| DASH/MSS | 完整支持 DASH/MSS | 不实现 | 不在范围 |
| Live TS | 原版支持 Live TS extractor | Go 版只围绕 HLS | 不在范围 |
| HLS Master | 视频、音频、字幕、多属性解析 | 已解析基础流、音频、字幕、码率、分辨率、帧率、语言、声道、HDR/DV 等，并兼容长签名 URL | 已追平 |
| HLS Media | EXTINF、MAP、BYTERANGE、DISCONTINUITY、PDT、ENDLIST | 已实现并覆盖空直播窗口、多 MAP、fMP4 init 保留、PDT 容错、长签名分片 URL | 已追平 |
| HLS 内容预处理 | 孤立 `\r` 换行、YSP、Youku、Disney+、AppleTV、KEY 顺序修正 | 已实现，并新增站点级预处理回归测试；普通内容首尾空白也按上游保留 | 已追平 |
| URL 合并 | 相对 URL、BaseURL、URL Processor | HLS URL 合并和 `--base-url` 已支持；DASH 专用 URL Processor 不在范围 | 已追平 |
| Append URL Params | 分片 URL 继承输入 URL 参数 | 已覆盖媒体分片、init、HLS key URL | 已追平 |
| HTTP Header/文本编码/解压 | 默认 UA、自定义 Header、多 Header，playlist/key 默认 `Accept-Encoding: gzip, deflate` 与 `Cache-Control: no-cache`，播放列表文本按响应 charset 解码，gzip/deflate/br 响应自动解压 | 已支持；用户自定义 `Accept-Encoding` 不会被覆盖，跳转后保留 Header/Range，非 UTF-8 playlist 文本按 `Content-Type` charset 解码，playlist/key/分片 HTTP deflate 和 Brotli 响应可解压 | 已追平 |
| 代理 | 系统代理、自定义代理 | 已支持系统代理和 `--custom-proxy` 校验 | 已追平 |
| 重试/超时 | 请求超时、下载重试、key 重试 | 已支持，`--http-request-timeout` 支持小数秒 | 已追平 |
| CLI 参数解析 | 缺值、int/double 参数非法值在 CLI 阶段报错 | 已支持必填选项参数缺值拒绝、整数参数和超时/限速等自定义 parser 的非法值拒绝；`--task-start-at` 会先等待再派生默认保存名 | 已追平 |
| 并发下载 | 分片并发、多轨并发 | 已支持，且多轨输出顺序稳定 | 已追平 |
| 限速 | `--max-speed` | 已支持共享限速预算，参数规则按上游 `K/M` | 已追平 |
| Range 下载 | BYTERANGE 和大文件切片 | 已支持 BYTERANGE 校验、大文件 Range 并发切分 | 已追平 |
| 断点跳过 | 跳过已存在分片和 `_dec` 文件 | 已支持 | 已追平 |
| 图片伪装头 | PNG/GIF/BMP/JPEG 伪装头剥离 | 已支持固定偏移和 TS sync fallback，并有 GIF/PNG/JPEG 测试 | 已追平 |
| gzip 分片 | 解压原始 gzip payload | 已支持，且在 HLS 解密后处理 | 已追平 |
| HLS key 来源 | URI、base64、data URI、本地文件、HTTP | 已支持三类 inline key、本地文件、HTTP key 重试 | 已追平 |
| AES-128 | CBC 解密 | 已内置实现 | 已追平 |
| AES-128-ECB | ECB 解密 | 已内置实现 | 已追平 |
| CHACHA20 | 每 1024 字节解密 | 已实现 | 已追平 |
| UNKNOWN method | 保留原始分片并二进制合并 | 已实现 | 已追平 |
| CENC 外部解密 | mp4decrypt/shaka/ffmpeg | 已实现外部解密入口、结构化 MP4 info 解析、KID/key-file 匹配、shaka 缺 key 探测 | 基本追平 |
| SAMPLE-AES | 原版保留分片并通过 MP4 外部工具链处理 | Go 版按上游不做伪原生分片解密，保留分片并走外部解密入口 | 基本追平 |
| SAMPLE-AES-CTR | 外部工具路径 | Go 版支持外部解密入口 | 基本追平 |
| 复杂 PSSH/KID | 多 DRM、复杂 box 场景 | 已支持 `schm`、`tenc`、Widevine PSSH data、PSSH v1 KID 列表、PlayReady 文本/`VALUE`，复杂 DRM 样本仍未系统验证 | 部分追平 |
| MP4 实时解密 | `--mp4-real-time-decryption` | 已覆盖 init+fragment 基础实时外部解密；init 会先保留原始盒读取 KID，本地读不到且使用 shaka 时会从缺 key 错误探测 `key_id`；shaka/ffmpeg 会按上游跳过单独 `_init.mp4`，再用 init+fragment 构造外部工具输入并避免最终重复合并 init，实时解密开启时也会跳过最终整文件二次解密 | 部分追平 |
| 自定义 HLS method/key/iv | 文件、HEX、Base64、枚举 method | 已支持，并按上游枚举校验 | 已追平 |
| 轨道自动选择 | 最佳视频、音频、字幕 | 已支持最佳视频、每语言最高码率音频、全部字幕等路径 | 基本追平 |
| 交互选择 | Spectre 交互多选 | Go 版实现了默认回车选择语义，但没有完整 Spectre UI | 部分追平 |
| 选择/丢弃过滤 | `-sv/-sa/-ss`、`-dv/-da/-ds` | 已支持主要过滤条件、Role、`bestN/worstN/all`、先 drop 后 keep | 已追平 |
| 广告清理 | Uplynk/Youku 内置和 `--ad-keyword` | 已支持上游内置广告条件和自定义正则 | 已追平 |
| 自定义范围 | 分片序号和时间范围 | 已支持开区间、冒号时长、字幕偏移；直播按上游跳过 | 已追平 |
| meta 输出 | `raw.m3u8`、`meta.json`、`meta_selected.json`，`--write-meta-json false` 关闭写出，已有文件不覆盖 | 已支持 | 已追平 |
| 保存名/模板 | 文件名清理、模板变量、冲突处理 | 已支持上游清理语义、模板变量和冲突命名；`<Id>` 按本次下载任务序号而非原始流编号生成 | 已追平 |
| 二进制合并 | 普通分片合并、分批预合并 | 已支持，超长列表可分批预合并 | 已追平 |
| ffmpeg 单轨合并 | concat protocol/demuxer、多输出格式 | 已支持 `mp4/mkv/flv/ts/m4a/aac/eac3/ac3` 等常见路径，MP4 date metadata 按上游 `.NET` round-trip `"o"` 格式写入，`aac_adtstoasc` 按上游 `Where(Audio).All(...)` 空集合为真的语义启用 | 基本追平 |
| 最终混流 | ffmpeg/mkvmerge、metadata、disposition | 已支持多轨混流、语言/标题元数据、date metadata、默认轨道标记、外部导入；无语言轨道会按上游写入 `und`，并按上游处理 keep=false 清理范围；媒体探测识别 HDR bt2020 信号、Dolby Vision 的 `dvhe`、`dvh1`、`DOVI`、`dvvideo` 别名和 `DOVI configuration record` side data 后，DV 会按上游禁用最终混流；空探测结果会按上游保留 `Unknown` 占位，音频起点可从 stream 或全局 format `start_time` 读取 | 基本追平 |
| `--mux-import` | 多外部轨道导入 | 已支持多条导入和存在性校验 | 已追平 |
| `--skip-download` | 只解析/写 meta | 已支持 | 已追平 |
| `--skip-merge` | 保留分片目录 | 已支持，且不触发最终 `-M` | 已追平 |
| 临时文件清理 | `--del-after-done` | 点播和部分直播实时合并路径已支持；普通下载会删除已知分片和 Go 版 concat 辅助文件，并按上游 `SafeDeleteDir` 递归清理空父目录，非空目录会保留 | 基本追平 |
| VTT/SRT | VTT 修复、SRT 输出 | 已支持多分片时间轴、空 SRT 占位、`X-TIMESTAMP-MAP` | 已追平 |
| TTML | TTML 转 VTT/SRT | 已支持裸 TTML 和部分 MP4 TTML | 基本追平 |
| MP4 WebVTT | wvtt/vttc/payl 抽取 | 已支持并保留 `iden` cue id | 基本追平 |
| 图形字幕 | Base64 PNG 落盘 | 已支持 | 已追平 |
| 字幕修复后清理 | 删除原始字幕分片 | 已支持，TTML/MP4-TTML 可用环境变量保留 | 已追平 |
| 直播刷新 | 按窗口刷新追加新增分片 | 已支持基础刷新、去重、追加；未设置 `--live-record-limit` 时会按上游持续刷新到直播结束或外部中断；直播录制会按上游强制多轨并发和 MP4 实时解密 | 基本追平 |
| 直播起点同步 | 多轨按日期或序号对齐 | 已支持 PDT/序号对齐和 `--live-take-count` | 基本追平 |
| 直播录制限制 | `--live-record-limit` | 已支持，初始窗口计入限制 | 基本追平 |
| 直播实时合并 | 刷新过程中追加输出 | 非字幕输出已按批次实时追加；系统信号取消会进入已下载内容收尾；字幕收尾会按上游在无音频时关闭 VTT 音频时间轴修正，并在可探测音频输出时复用 start_time | 部分追平 |
| PipeMux | Unix FIFO、Windows named pipe、ffmpeg 参数 | 已实现 Unix FIFO/Windows 命名管道、参数构造和上游 date metadata 格式，Windows 仅交叉编译验证 | 部分追平 |
| ANSI 进度 UI | Spectre Console 动态进度列 | Go 版已按上游在 stdout/stderr 重定向时清除 ANSI 颜色并强制 console 状态，但没有完整动态进度 UI | 部分追平 |
| 多语言资源 | zh-CN/zh-TW/en-US 完整资源 | Go 版已接入默认环境语言映射、核心运行输出、加载 URL、HLS 内容匹配、解析媒体信息、Master 列表检出、直播流检出、直播录制上限和达到上限提示、PipeMux 命名管道创建/混流提示、字幕修复/抽取提示、解析后四项轨道统计、已选流列表、无流错误、保存文件名、meta json 写出、开始下载、读取媒体信息、二进制合并、ffmpeg 合并、分块合并、自动派生选项提示、按上游固定资源文本追加版本号的更新检查提示、实时解密引擎建议、下载进度文本、外部工具缺失提示和 `--morehelp` 详细帮助；完整 `ResString` 资源表仍未复刻 | 部分追平 |
| 更新检查 | GitHub latest release，按 redirect tag 与当前 `vMajor.Minor.Build` 前缀判断是否提示 | 已支持，可 `--disable-update-check` 禁用；提示文案与 tag 判断语义已按上游对齐 | 已追平 |
| 日志文件 | 日志路径、文件名校验、等级、禁用 | 已支持 `--log-file-path` 文件名清理和非法名拒绝、`--log-level`、`--no-log` | 已追平 |
| 跨平台构建 | 原版 .NET 多平台发布 | Go 版可本地构建，Windows 测试二进制可交叉编译；已新增 GitHub Actions 测试、跨平台构建和 `v*` tag Release 产物流水线 | 基本追平 |

## 当前最需要继续补齐的 HLS 差距

1. CENC/PSSH/KID 已覆盖 `schm`、`tenc`、Widevine PSSH data、PSSH v1 KID 列表和 PlayReady 文本/`VALUE` 场景，但复杂 DRM 封装还缺系统性样本验证。
2. MP4 实时解密已补齐 init 先读 KID、shaka 缺 key 探测 KID、shaka/ffmpeg 跳过单独 `_init.mp4`、不重复合并 init、合并后不二次整文件解密的路径，但还没有达到上游直播状态机里所有边缘分支的等价程度。
3. SAMPLE-AES/SAMPLE-AES-CTR 已按上游走外部 MP4 工具链或保留分片，但还缺真实 SAMPLE-AES 样本覆盖。
4. 直播 producer/consumer 多轨状态机仍是简化实现；未设置 `--live-record-limit` 的普通和实时合并路径已补齐持续刷新到 `ENDLIST` 的行为，系统信号中断已能触发基础收尾，但更完整的多轨收尾仍弱于原版，PipeMux 在 Windows 真实环境未实际运行验证。
   已核对原版源码，未发现键盘 `q` 停止机制；原版全局 `Console.CancelKeyPress` 是 Ctrl+C 强制退出。
5. ANSI/Spectre 风格动态进度 UI 未复刻；当前只对齐了重定向时清除 ANSI 颜色的控制台初始化行为。
6. 多语言资源系统已覆盖默认环境语言映射、核心运行输出、加载 URL、HLS 内容匹配、解析媒体信息、Master 列表检出、直播流检出、直播录制上限和达到上限提示、PipeMux 命名管道创建/混流提示、字幕修复/抽取提示、解析统计、已选流列表、无流错误、保存文件名、meta json 写出、开始下载、读取媒体信息、二进制合并、ffmpeg 合并、分块合并、外部工具缺失提示和 `--morehelp` 详细帮助，但完整 `ResString` 资源表和全部错误提示仍未复刻。
7. ffmpeg/mkvmerge 已覆盖 metadata、disposition、字幕编码、工具路径、清理范围、AAC bitstream filter、HDR bt2020、Dolby Vision 别名/side data 探测和空媒体探测 `Unknown` fallback 的多个上游边缘语义；剩余仍需要更多真实媒体样本补充媒体探测和封装器组合证据。

## 最近验证口径

当前 Go 复刻版常用验证命令：

```zsh
go test -count=1 ./...
go build -o /tmp/n-m3u8dl-go-hls-check .
GOOS=windows GOARCH=amd64 go test -c -o /tmp/n-m3u8dl-go-hls-windows.test.exe .
```

这些验证只能证明当前测试覆盖的 HLS 行为通过，不能证明已与原版完整等价。
