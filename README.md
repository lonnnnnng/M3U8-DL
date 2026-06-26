# m3u8dl-go

`m3u8dl-go` 是基于上游 [`nilaoda/N_m3u8DL-RE`](https://github.com/nilaoda/N_m3u8DL-RE) 源码行为复刻的 Go 版 HLS 下载器。

当前目标：

- 只复刻 HLS/m3u8 主链路。
- 不实现 DASH/MSS 下载；遇到 DASH/MSS/Live TS/Binary 输入时只保留上游风格识别提示并返回不支持。
- CLI、解析、下载、解密、合并、字幕、直播等行为尽量向原项目对齐。

## 快速运行

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

## 桌面版

桌面版基于 Wails 构建，界面负责填写 m3u8、输出目录、文件名、请求头和常用合并选项，实际下载仍复用同版本 `m3u8dl-go` 命令行核心。

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

发布流水线会先运行 CLI 和桌面端测试，再生成 macOS amd64/arm64、Linux amd64、Windows amd64 桌面包。产物会写入 `dist/m3u8dl-go_<version>_desktop_<os>_<arch>.*`，桌面打包脚本会检查包内桌面程序、内置 CLI 和 `.sha256` 排除规则。

## 已实现主能力

- HLS master/media playlist 解析。
- HTTP headers、代理、超时、重试、gzip/deflate/br 解压。
- 分片并发下载、限速、断点复用、Range、分片数量校验。
- AES-128、AES-128-ECB、CHACHA20 内置解密。
- CENC、SAMPLE-AES、SAMPLE-AES-CTR 外部解密入口。
- 自动选轨、交互选轨、选择/丢弃过滤器、广告分片过滤、自定义范围。
- 二进制合并、ffmpeg 合并、最终 ffmpeg/mkvmerge 混流。
- VTT/SRT/TTML/MP4 字幕处理和直播字幕修正。
- 直播刷新、实时合并、PipeMux。
- `zh-CN`、`zh-TW`、`en-US` 多语言输出。
- raw/meta JSON、日志文件、更新检查。

## 合并方式速记

- 日常观看：默认 ffmpeg 合并，普通 TS 点播通常输出 `.mp4`。
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
