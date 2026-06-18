# N_m3u8DL-GO-HLS

这是基于上游 `nilaoda/N_m3u8DL-RE` 源码行为拆出来的 Go 版 HLS 下载器复刻目录。

当前目标：只做 HLS，不做 DASH/MSS；CLI、解析、下载、解密、合并等能力尽量向原项目对齐。

## 快速运行

```bash
go run . "<m3u8-url-or-file>" --auto-select --save-dir ./downloads -M format=mp4
```

## 已实现主链路

- HLS Master playlist / Media playlist 解析
- `EXT-X-STREAM-INF`、`EXT-X-MEDIA`、`EXT-X-KEY`、`EXT-X-MAP`、`EXT-X-BYTERANGE`
- `EXT-X-PROGRAM-DATE-TIME` 支持严格 RFC3339、紧凑时区偏移和无时区时间，尽量贴近上游 `DateTime.Parse` 对真实 HLS 源的容错
- AES-128 CBC / AES-128 ECB 分片解密
- `--custom-hls-method` 兼容上游 `AES_128/SAMPLE_AES_CTR` 等下划线枚举名
- 重复 `EXT-X-KEY` 行复用已解析 key，避免同一 key URI 反复请求
- CHACHA20 分片解密
- 无法识别的 HLS 加密方式按 `UNKNOWN` 保留原始分片，并自动启用二进制合并
- 检测到 fMP4 或 CENC 加密方式时自动启用二进制合并
- CENC/SAMPLE-AES/SAMPLE-AES-CTR 下载后外部工具解密入口，含基础实时分片解密路径；实时 MP4 解密会先保留原始 init 读取 KID，本地读不到且使用 shaka 时会先从缺 key 错误探测 `key_id`；shaka/ffmpeg 实时解密会按上游跳过单独 `_init.mp4`，再把 init 与媒体分片交给外部工具并避免最终重复合并 init，实时解密开启后也不会在合并产物上重复整文件解密
- `mp4decrypt` 按上游在媒体目录临时改名并切换工作目录执行，同目录 init 信息使用相对文件名，兼容中文/特殊路径
- 传入 `--key`/`--key-text-file` 时提前校验外部解密工具路径
- HTTP 请求头、代理、超时、重试、playlist/key 默认 `Accept-Encoding: gzip, deflate` 与 `Cache-Control: no-cache`、gzip/deflate/br 响应解压、并发下载
- 源 m3u8 和子 playlist HTTP 文本加载按上游默认重试 10 次
- 重跑任务时复用已下载分片和已生成的 `_dec` 解密分片
- 单大分片在服务端支持 Range 时自动拆成多个字节区间并发下载
- 响应 `Content-Length` 与 `EXT-X-BYTERANGE` 期望长度校验
- 图片伪装分片头剥离、原始 gzip 分片内容解压；处理顺序按上游保持为 HLS 解密后再剥头/解压
- `base64://`、`hex://`、`file:` 分片 URL 读取
- `--log-file-path` 日志写出并按上游清理非法文件名、默认日志目录、`--no-log` 禁用日志、`--log-level` 日志文件等级过滤
- stdout/stderr 被重定向时按上游自动启用 `--force-ansi-console` 并清除 ANSI 颜色，避免管道/日志中混入颜色控制序列
- `--ui-language`、`--log-level`、`--sub-format`、`--decryption-engine`、`--custom-hls-method` 按上游白名单/枚举值提前校验，需要参数值的选项缺值时会立即报错，`--thread-count`、`--download-retry-count`、`--live-wait-time`、`--live-take-count` 等数值选项会在解析阶段拒绝非法整数，避免非法值静默按默认行为执行
- `--ui-language` 已开始接入核心运行输出、解析后四项轨道统计、自动派生选项提示、自动二进制合并提示、更新检查提示、下载进度文本和外部工具缺失提示，默认中文，支持 `zh-CN/zh-TW/en-US`
- GitHub latest release 更新检查、`--disable-update-check`
- 未传 `--save-name` 时从输入 URL/文件名推导默认保存名，`--save-pattern` 支持 `<FrameRate>` 等变量，并按上游清理空字段产生的多余分隔符
- 单请求限速、`--max-speed`，按上游规则只接受 `K`/`M` 单位，例如 `512K`、`1.5M`
- 分片范围下载、`--custom-range`，支持 `0-10`、`-300`、`300-`、`00:10-` 等上游开区间写法，并按上游语义计算字幕偏移时长
- `--ad-keyword` 按上游语义使用正则清理已选轨道分片，并移除清空后的媒体段
- `--check-segments-count` 默认严格校验；关闭时跳过失败分片继续合并
- 布尔选项支持显式 `true/false`，例如 `--check-segments-count false`
- 输入 URL 查询参数继承到分片 URL 和 HLS key URL、`--append-url-params`
- 二进制合并、ffmpeg concat 合并，支持 `mp4/mkv/flv/ts/m4a/aac/eac3/ac3` 等上游单轨输出格式；TS 使用 `h264_mp4toannexb`，AAC 音频只在上游会启用的格式中条件添加 `aac_adtstoasc`；超长分片列表会按上游策略先分批预合并
- 多轨最终 ffmpeg/mkvmerge 混流、`-M format=mp4|mkv|ts:muxer=ffmpeg|mkvmerge`，并按上游拒绝空 `bin_path` 等非法混流参数；`--morehelp mux-import` 提供与上游一致的外部轨道导入参数说明
- 最终混流会按上游完整语言表转换语言码，并在名称为空时填入默认语言描述
- 最终 ffmpeg 混流清理输入 metadata、复制未知流，并设置默认视频/首音频/字幕非默认轨道标记；`keep=false` 时按上游只清理实际参与混流的轨道和外部导入轨，保留被 `skip_sub` 跳过的字幕
- `--mux-import` 与 `-M` 互斥校验、外部导入文件存在性校验、开启最终混流时自动启用二进制合并
- 下载后会用 ffprobe 修正真实媒体类型，字幕 TS 可转入 VTT 修复路径，HDR/Dolby Vision 会按上游从媒体探测结果识别，Dolby Vision 会强制二进制合并
- 输出文件冲突时按上游优先使用分辨率、码率、语言、声道等流元数据生成唯一文件名
- 自动选择最佳视频、每语言最高码率音频和全部字幕，支持交互选择与选择/丢弃过滤器的 `for=bestN|worstN|all`、带宽、时长、分片数、Role 等条件，并按上游提前拒绝非法 `for`、正则和数值字段
- 基础直播录制轮询、新分片追加，刷新间隔未显式指定时按上游用直播窗口时长自动计算，并按上游逻辑用日期或序号对齐多轨起点；`--live-record-limit` 按已刷出的分片时长累计，初始直播窗口也计入限制；直播刷新去重按 HLS 的 `PROGRAM-DATE-TIME` 秒级时间戳或分片序号处理，并容忍紧凑时区偏移/无时区时间，避免相同 URL 的新分片被漏掉
- 直播录制分片临时文件按上游优先使用 `PROGRAM-DATE-TIME` 秒级时间戳命名，无日期时回退到分片 `Index`，fMP4 init 固定为 `_init.mp4`
- `--live-real-time-merge` 的非字幕输出会在直播刷新过程中按批次下载新增分片并顺序追加输出，`--live-keep-segments false` 时保留 init、删除普通媒体分片
- `--live-pipe-mux` 自动强制启用 `--live-real-time-merge`
- PipeMux 的 ffmpeg 参数构造支持 `RE_LIVE_PIPE_OPTIONS`、`RE_LIVE_PIPE_TMP_DIR` 和 Windows/Unix 管道路径差异；macOS/Linux 可创建 FIFO 管道，Windows 可创建 `\\.\pipe\...` 命名管道，启动 mux 进程后直播刷新过程中会把非字幕轨道按批次持续写入 pipe mux，字幕继续作为独立输出保留
- VTT 字幕合并、SRT 输出，自动字幕修复开启时按 `--sub-format` 决定扩展名，支持 `X-TIMESTAMP-MAP`/`MPEGTS` 时间轴修正；`Base64::` 图形字幕会按上游写出 PNG 并替换为文件名；`--live-fix-vtt-by-audio` 会读取音频 stream 或全局 format `start_time` 并按上游逻辑修正 VTT 字幕偏移
- 裸 TTML、MP4 stpp 字幕基础抽取；MP4 WebVTT 按 `mdhd/tfdt/tfhd/trun` 解析真实字幕时间
- CENC `tenc` / `schm` / Widevine PSSH / PlayReady PSSH XML 解析，Widevine 会暴露原始 PSSH data、支持 v1 KID 列表和 protobuf `key_id`，PlayReady 支持文本节点和 `VALUE` 属性两种 KID 写法；`tenc default_KID` 为全 0 时会按上游继续从 Widevine PSSH 回退真实 KID，并在外部解密时使用 track/label `1` 的 MultiDRM 参数形态；使用 `SHAKA_PACKAGER` 且本地解析不到 KID 时，会按上游从 shaka 的缺 key 错误里探测 `key_id`，用于匹配 `--key-text-file`
- 并发分片共享 `--max-speed` 速度预算
- `raw.m3u8`、全部轨道 `meta.json` 与选中轨道 `meta_selected.json` 输出；`--write-meta-json false` 可关闭，重跑时不会覆盖已有 raw/meta 文件
- GitHub Actions 已接入测试、macOS/Linux/Windows 构建和 `v*` tag Release 产物发布

## 仍需继续追平的细节

见 [docs/FUNCTIONS.md](docs/FUNCTIONS.md)。
