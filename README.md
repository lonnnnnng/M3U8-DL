# M3U8-DL

`M3U8-DL` 是基于上游 [`nilaoda/N_m3u8DL-RE`](https://github.com/nilaoda/N_m3u8DL-RE) 源码行为复刻的 Go 版 HLS 下载器，命令行入口继续使用 `m3u8dl-go`。

- 当前版本：[`v1.0.5`](https://github.com/lonnnnnng/M3U8-DL/releases/tag/v1.0.5)
- 项目仓库：[`lonnnnnng/M3U8-DL`](https://github.com/lonnnnnng/M3U8-DL)
- 品牌与兼容边界：应用、仓库和发布归档使用 `M3U8-DL`；CLI、桌面内置核心和日志兼容名称继续使用 `m3u8dl-go`。

当前目标：

- 只复刻 HLS/m3u8 主链路。
- 不实现 DASH/MSS 下载；遇到 DASH/MSS/Live TS/Binary 输入时只保留上游风格识别提示并返回不支持。
- CLI、解析、下载、解密、合并、字幕、直播等行为尽量向原项目对齐。

## 下载与运行

从 [GitHub Releases](https://github.com/lonnnnnng/M3U8-DL/releases/latest) 下载与系统架构对应的归档。CLI 包解压后运行 `m3u8dl-go`，Windows 对应 `m3u8dl-go.exe`；macOS 桌面包解压后得到 `M3U8-DL.app`，Windows 桌面包内包含 `m3u8dl-go-desktop.exe`。

macOS 桌面包当前使用自签名而非 Apple Developer ID 公证签名。如果首次打开被系统拦截，按[桌面版说明](docs/DESKTOP.md#macos-提示-app-已损坏)清除隔离标记后再打开。

从源码运行需要 Go 1.25 或更高版本：

自动选择最佳轨道并输出 MP4：

```zsh
go run . "https://example.com/index.m3u8" --auto-select true --save-dir ./downloads
```

带 Cookie/Referer：

```zsh
go run . "https://example.com/index.m3u8" -H "Cookie: xxx" -H "Referer: https://example.com"
```

只解析并写出 meta：

```zsh
go run . "https://example.com/index.m3u8" --skip-download true --save-name probe
```

下载前 3 个分片并二进制直拼为 TS：

```zsh
go run . "https://example.com/index.m3u8" --auto-select true --custom-range 0-2 --binary-merge true
```

查看程序内完整参数：

```zsh
go run . --help
go run . --morehelp mux-after-done
go run . --morehelp custom-range
```

默认帮助语言为简体中文；需要英文时显式传入 `--ui-language en-US --help`。

## 文档

- [功能、用法与参数参考](docs/CLI_REFERENCE.md)
- [桌面版说明](docs/DESKTOP.md)
- [原版功能清单与 Go HLS 复刻进度对比](docs/FEATURE_COMPARISON.md)
- [当前功能清单](docs/FUNCTIONS.md)
- [真实样本验证记录](docs/REAL_SAMPLE_VALIDATION.md)
- [当前项目状态与继续开发入口](docs/PAUSE_HANDOFF.md)

## 桌面版

桌面版基于 Wails 构建，界面按基础与输出、轨道与字幕、解密与密钥、直播与过滤、混流与任务控制组织参数，负责填写 m3u8、输出目录、临时目录、文件名、保存模板、BaseURL、请求头、HTTP 超时、追加 URL 参数、选轨/字幕过滤、解密参数、直播录制、广告过滤、定时开始、最终保存格式、是否使用 FFmpeg 合并、最终混流、外部轨道导入、解析/下载/合并控制和常用合并选项；输出目录、临时目录、Key 文件、解密工具、FFmpeg 路径和外部轨道文件可直接选择，外部轨道可用文件、语言、名称结构化追加，实际下载仍复用同版本 `m3u8dl-go` 命令行核心。右上角主题按钮支持浅色、深色、自动三态切换，选择会持久化，自动模式跟随系统外观变化。

桌面构建默认使用 Wails CLI v2.12.0。Linux 还需要 `build-essential`、`pkg-config`、GTK 3 和 WebKitGTK 4.0 开发库；Windows 需要 PowerShell 和 WebView2 构建环境。

本地构建 macOS `.app`：

```zsh
./scripts/build_desktop_macos.sh
```

Linux/Windows 桌面包在对应系统上构建：

```zsh
./scripts/build_desktop_linux.sh
```

```powershell
./scripts/build_desktop_windows.ps1
```

当前 `v1.0.5` 为本地构建后手工发布，包含 6 个 CLI 包、2 个 macOS 桌面包和 1 个 Windows amd64 桌面包，共 9 个资产；Linux 桌面包尚未进入该 Release。仓库发布流水线已配置 6 个 CLI 包以及 macOS amd64/arm64、Linux amd64、Windows amd64 桌面包，共 10 个预期资产。发布归档统一使用 `M3U8-DL_v<version>_...` 前缀，包内 CLI 仍名为 `m3u8dl-go`。

## 已实现主能力

- HLS master/media playlist 解析。
- HTTP headers、代理、超时、重试、gzip/deflate/br 解压。
- 分片并发下载、限速、断点复用、Range、分片数量校验。
- AES-128、AES-128-ECB、CHACHA20 内置解密。
- CENC、SAMPLE-AES、SAMPLE-AES-CTR 外部解密入口。
- 自动选轨、交互选轨、选择/丢弃过滤器、广告分片与广告 `EXT-X-MAP` 过滤、自定义范围。
- `--probe-json` 资源探测，输出轨道、分片、时长、直播和加密摘要。
- 二进制合并、ffmpeg 合并、最终 ffmpeg/mkvmerge 混流。
- VTT/SRT/TTML/MP4 字幕处理和直播字幕修正。
- 直播刷新、实时合并、PipeMux；直播轨道独立刷新，使用毫秒级 `PROGRAM-DATE-TIME` 或分片序号去重，并在每轮刷新后重新过滤广告内容。
- `zh-CN`、`zh-TW`、`en-US` 多语言输出。
- raw/meta JSON、日志文件、更新检查。

## 合并方式速记

- 日常观看：桌面端默认最终保存 MP4 并使用 ffmpeg 合并，普通 TS 点播通常输出 `.mp4`。
- 原始保真/排障：`--binary-merge true`，按分片顺序直接拼接。
- fMP4、CENC、未知加密：程序会自动倾向二进制合并，避免过早改写媒体结构。
- 最终多轨封装：使用 `-M format=mp4|mkv|ts:muxer=ffmpeg|mkvmerge`。

## 真实样本状态

已用真实 URL `https://play.jisuzyv.com/play/bYE7AEMb/index.m3u8` 验证：

- master playlist 解析。
- AES-128 key 加载。
- 前 3 个 TS 分片真实下载、解密。
- 二进制合并输出 MPEG-TS。
- ffmpeg 合并输出 MP4。
- `ffprobe` 可识别 H.264 1080p 视频和 AAC 音频。

详见 [docs/REAL_SAMPLE_VALIDATION.md](docs/REAL_SAMPLE_VALIDATION.md)。

## 常用验证

```zsh
git diff --check
go test -count=1 ./...
go build -o /tmp/m3u8dl-go-check .
GOOS=windows GOARCH=amd64 go test -c -o /tmp/m3u8dl-go-windows.test.exe .
```

这些验证只能证明当前测试覆盖的 HLS 行为通过，不能证明已与原版完整等价。
