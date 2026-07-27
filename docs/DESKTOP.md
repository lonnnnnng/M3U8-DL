# M3U8-DL 桌面版

桌面版使用 Wails v2 构建，目标是复用同一套 Go 下载核心，在 macOS、Windows、Linux 上保持一致的桌面操作入口。当前发布版为 [`M3U8-DL v1.0.5`](https://github.com/lonnnnnng/M3U8-DL/releases/tag/v1.0.5)。仓库发布流水线已配置 macOS amd64/arm64、Linux amd64、Windows amd64 桌面构建；当前 `v1.0.5` 手工 Release 实际提供 macOS amd64/arm64 和 Windows amd64 桌面包。

## 功能

- 创建下载任务，支持创建后立即下载或先加入等待队列。
- 支持在 m3u8 地址框粘贴多行地址批量创建任务，地址框 placeholder 会直接展示单任务、批量任务和带保存名的填写格式。
- 支持 `资源名称|资源地址` 这类带名称的批量行；名称分隔符可在界面中配置并保存为默认值，地址框示例会随当前分隔符同步更新，匹配到的资源名称会作为该任务保存名称。
- 新建任务弹窗可直接复制当前表单对应的 CLI 命令；批量地址会生成多行命令，便于在终端预演。
- 新建任务弹窗支持预检查，会在创建任务前检查下载地址数量、输出目录写入权限、下载核心可用性、核心参数解析结果、Key 文本文件、解密工具路径、外部导入轨道、单任务资源探测结果，以及 FFmpeg 合并所需工具。
- 新建任务弹窗只保留地址、输出目录、保存名称、请求头和本次任务覆盖项；本次任务可单独覆盖下载范围、HTTP 超时、失败重试次数、定时开始和临时密钥。`BaseURL`、临时目录、保存模板、名称分隔符、轨道字幕、解密工具、直播过滤和混流任务控制等默认参数移动到设置页维护。设置页按 `基础`、`核心工具`、`轨道字幕`、`解密`、`直播过滤`、`混流任务` 分类展示，`名称分隔符` 位于 `设置 -> 基础`。
- 输出目录、默认输出目录、临时目录、Key 文本文件、解密工具路径和外部轨道文件可直接从系统对话框选择。
- 外部轨道导入支持通过文件、语言码和轨道名结构化追加到 `--mux-import` 参数列表，仍保留原始文本框方便编辑复杂参数。
- 设置页支持更贴近 CLI 的常用默认参数：`BaseURL`、临时目录、保存模板、HTTP 请求超时、追加 URL 参数、限速、自动选轨、视频/音频/字幕选择过滤、丢弃过滤、只下载字幕、字幕格式、自动修复字幕、key 文本文件、外部解密引擎、解密工具路径、MP4 实时解密、自定义 HLS method、广告关键字、直播当点播、直播实时合并、直播分片保留策略、直播 PipeMux、直播录制时长、直播刷新间隔、首取分片数、直播字幕对齐、最终混流参数、最终保存格式、是否使用 FFmpeg 合并、外部轨道导入、多轨并发、只解析资源、跳过合并、校验分片数、写 meta JSON、保留临时文件和关闭日志文件。
- 右上角主题按钮按浅色、深色、自动循环切换并持久化选择；自动模式会实时跟随系统外观变化。
- 设置页可把当前默认参数保存到用户配置目录；保存后再次启动桌面端会自动填入这些默认下载参数。
- 新建任务弹窗可一键重置为已保存默认参数，并清空地址、请求头、密钥、定时开始、外部轨道构建器和上一次预检查结果。
- 新建任务弹窗把预检查、复制命令和重置表单放在独立工具区，把加入队列、创建并下载放在固定底部操作区；批量地址输入时会实时显示有效地址数量。
- 新建任务表单有未创建内容时，点击关闭、取消、遮罩或按 `Esc` 会先确认是否放弃，取消确认后保留表单内容和输入焦点。
- 任务列表展示等待、下载中、完成、失败、停止状态。
- 任务队列支持按名称、地址、输出目录和状态文本搜索，可按等待、下载中、完成、失败、已停止状态筛选，并支持按最新、最旧、状态、进度和名称排序。
- 任务进度监控，优先解析命令行核心的 `--progress-json` 进度事件并更新进度条、速度、已耗时和预计剩余时间；下载完成时会读取 `summary` 事件中的输出文件列表，失败时会保留 `error` 事件中的真实错误信息，目录扫描仍作为兼容回退；旧版 `下载进度 n/m` 文本仍可解析。
- 任务详情展示输出目录、保存名称、下载参数和任务日志。
- 任务详情展示实际 CLI 命令，可一键复制源地址、命令或日志用于排障；日志面板可导出脱敏后的 `.log` 文件，也可清空当前任务已持久化的日志；Cookie、Authorization、代理密码不会原样写入导出日志或持久化状态文件。
- 任务详情可把当前任务复制回新建任务表单，便于调整参数后重新创建；复制时会保留 FFmpeg 路径等普通参数，不会回填敏感请求头、密钥、自定义 HLS key/iv、代理密码和定时开始时间。
- 任务队列支持右键菜单，可直接开始、停止、重试、复制地址、复制命令、复制日志、复制为新任务或移除当前任务；键盘可用上下方向键、`Home`、`End` 切换任务，通过菜单键或 `Shift + F10` 打开同一操作菜单。
- 解密 key、自定义 HLS key 和 IV 只保留在当前运行会话中用于启动任务和复制当前命令，不会写入状态文件；导出日志和预检查错误会用 `<redacted-key>`、`<redacted-iv>` 替换密钥原文。
- 完成文件管理，支持刷新文件列表、打开输出目录、打开文件、复制文件路径、删除任务输出文件。
- 支持单个任务开始、停止、重试、移除，也支持批量开始等待任务、停止运行中任务、重试失败任务和清理已完成/失败/停止任务。
- 任务详情操作会按当前状态自动启用或禁用：运行中任务只能停止，排队任务可取消排队，失败任务优先走重试，未选中任务时禁用详情操作。
- 任务详情只显示当前状态可执行的开始、停止或重试主动作，复制日志、复制为新任务和移除收纳在“更多”菜单中；未选择任务时自动收起整组操作。
- 可设置最大同时运行任务数；批量启动超过上限时会进入排队状态，前一个任务结束或停止后自动调度下一个排队任务。
- 设置默认输出目录、临时目录、保存模板、`BaseURL`、HTTP 请求超时、限速、失败重试次数和名称分隔符；单次任务的保存名称、请求头、下载范围、HTTP 超时覆盖、失败重试覆盖、密钥和定时开始仍在新建任务弹窗里填写。
- 每行填写一个请求头，例如 `Cookie: xxx`、`Referer: https://example.com`。
- 开关控制自动选轨、只下载字幕、自动修复字幕、MP4 实时解密、直播当点播、直播实时合并、直播丢弃分片、直播 PipeMux、直播字幕对齐、使用 FFmpeg 合并、不写日期 metadata、追加 URL 参数、多轨并发、只解析资源、跳过合并、校验分片数、写 meta JSON、保留临时文件和关闭日志文件。
- 参数设置：设置页使用分类 Tab 管理默认值，`基础` 包含默认输出目录、`BaseURL`、临时目录、保存模板、名称分隔符、限速、HTTP 超时、同时运行任务数、线程数、失败重试次数和代理；`核心工具` 包含下载核心、FFmpeg 路径和外部工具检测；其余分类分别管理轨道字幕、解密工具、直播过滤和混流任务控制默认值。
- 设置页支持一键检测 `ffmpeg`、`ffprobe`、`mkvmerge`、`mp4decrypt` 和 Shaka Packager；`ffprobe` 会优先按 FFmpeg 同目录派生路径查找，Shaka Packager 会识别常见平台别名。
- 设置页会标记“已保存”或“有未保存更改”，未保存时主导航的设置项同步显示橙色标记；没有参数变化时保存按钮保持禁用，避免重复写入配置。
- 设置页会显示当前下载核心状态、版本、路径、能力摘要和不支持项，便于确认桌面包实际调用的 `m3u8dl-go-cli` 以及同版本 CLI 暴露的真实能力边界。
- 任务和全局参数会持久化到用户配置目录；Cookie 等请求头不会写入磁盘。

桌面版不会重新实现下载协议，所有下载、解密、合并和字幕逻辑仍由内置 `m3u8dl-go-cli` 执行。

桌面任务会固定向内置核心传入 `--disable-update-check true`，避免每个队列任务重复访问 GitHub；应用版本以桌面包内置核心和 Release 版本为准。

常用键盘操作：`Command/Ctrl + N` 打开新建任务，`Command/Ctrl + Enter` 在新建弹窗中创建并下载，`Command/Ctrl + S` 保存设置，`Esc` 关闭菜单或弹窗；新建表单已有内容时会先确认是否放弃。弹窗和任务菜单关闭后会把焦点恢复到打开前的控件。

## 本地构建

源码要求 Go 1.25 或更高版本；桌面模块依赖 Wails v2.12.0，三个平台脚本默认安装该版本的 Wails CLI。Linux 构建还需要 `build-essential`、`pkg-config`、`libgtk-3-dev` 和 `libwebkit2gtk-4.0-dev`。

macOS `.app`：

```zsh
./scripts/build_desktop_macos.sh
```

脚本会执行：

- 构建当前仓库的 CLI helper：`m3u8dl-go-cli`。
- 使用 Wails v2 构建 macOS `.app`。
- 将 `m3u8dl-go-cli` 放入 `.app/Contents/Resources/`。
- 对内置 CLI 和 `.app` 重新做本机自签名，并执行 `codesign --verify --deep --strict` 校验，避免注入 CLI 后破坏 macOS 应用密封。
- 生成 `dist/M3U8-DL_v<version>_desktop_macos_<arch>.zip`。
- 打包后调用 `scripts/verify_desktop_archive.sh` 校验 `.app`、桌面可执行和内置 `m3u8dl-go-cli` 都在包内，且没有 `.sha256` 文件。

Linux 桌面包需要在 Linux 上构建：

```zsh
./scripts/build_desktop_linux.sh
```

脚本会构建桌面程序和同版本 `m3u8dl-go-cli`，并打包为 `dist/M3U8-DL_v<version>_desktop_linux_<arch>.tar.gz`。构建环境需要 GTK/WebKitGTK 开发库；发布流水线使用 Ubuntu 22.04 和 `libgtk-3-dev`、`libwebkit2gtk-4.0-dev`。
打包后同样会调用 `scripts/verify_desktop_archive.sh` 校验包内桌面程序、内置 CLI 和 `.sha256` 排除规则。

Windows 桌面包需要在 Windows PowerShell 上构建：

```powershell
./scripts/build_desktop_windows.ps1
```

脚本会构建桌面程序和同版本 `m3u8dl-go-cli.exe`，并打包为 `dist/M3U8-DL_v<version>_desktop_windows_<arch>.zip`。
打包后会在 PowerShell 脚本内校验包内桌面程序、内置 CLI 和 `.sha256` 排除规则。

## 运行要求

- macOS 需要允许运行自签名应用。
- Linux 运行桌面版需要系统安装 GTK/WebKitGTK 运行库；不同发行版包名可能不同，Ubuntu/Debian 通常来自 `libgtk-3-0` 和 `libwebkit2gtk-4.0-37`。
- Windows 运行桌面版依赖系统 WebView2 Runtime；Windows 10/11 通常已经内置或可由系统自动安装。
- MP4/TS 的 FFmpeg 合并依赖 `ffmpeg`，媒体探测依赖 `ffprobe`。通过 Homebrew 安装时通常在 `/opt/homebrew/bin/ffmpeg`、`/opt/homebrew/bin/ffprobe` 或 `/usr/local/bin/` 下。
- 如果系统找不到 FFmpeg，可以在桌面版里填写 FFmpeg 路径。
- 预检查输出目录时会创建目录并写入一个临时探针文件，探针文件会立即删除。
- 预检查本地文件时会校验 Key 文本文件、解密工具路径和 `--mux-import` 外部轨道是否存在且不是目录；批量任务中的重复路径只检查一次。
- 预检查参数时会调用内置 CLI 的 `--print-effective-options`，只解析参数，不联网解析 m3u8、不下载分片。
- 单任务预检查会调用内置 CLI 的 `--probe-json` 联网解析 m3u8/master/子 playlist，展示点播/直播、视频/音频/字幕数量、分片数、时长和加密方式；批量任务会跳过这一步，避免一次预检查请求过多地址。
- 预检查和复制命令会带上当前表单中的 `--base-url`、`--tmp-dir`、`--save-pattern`、`--http-request-timeout`、`--append-url-params`、`--sub-only`、`--sub-format`、`--auto-subtitle-fix`、`--select-video`、`--select-audio`、`--select-subtitle`、`--drop-video`、`--drop-audio`、`--drop-subtitle`、`--key`、`--key-text-file`、`--decryption-engine`、`--decryption-binary-path`、`--mp4-real-time-decryption`、`--custom-hls-method`、`--custom-hls-key`、`--custom-hls-iv`、`--ad-keyword`、`--task-start-at`、`--live-perform-as-vod`、`--live-real-time-merge`、`--live-keep-segments`、`--live-pipe-mux`、`--live-record-limit`、`--live-wait-time`、`--live-take-count`、`--live-fix-vtt-by-audio`、`--mux-after-done`、`--mux-import`、`--no-date-info`、`--skip-download`、`--skip-merge`、`--check-segments-count`、`--write-meta-json`、`--del-after-done` 和 `--no-log`，创建任务后实际执行也复用同一套命令参数。
- 填写“最终混流参数”时会优先使用完整 `-M/--mux-after-done` 参数；留空时才根据“最终保存格式”和“使用 FFmpeg 合并”生成 `format=mp4:muxer=ffmpeg` 或 `format=ts:muxer=ffmpeg`。默认最终保存格式是 MP4，默认使用 FFmpeg 合并；关闭 FFmpeg 合并时会使用 TS 二进制直拼。
- 直播 PipeMux 会触发 FFmpeg 预检查；定时开始是单次任务参数，不会保存为默认值。
- 只解析资源或跳过合并时，预检查不会因为默认 FFmpeg 合并而强制要求 FFmpeg；如果仍手动填写了 FFmpeg 路径，预检查会继续校验该路径。
- 下载核心能力摘要来自内置 CLI 的 `--capabilities-json`，旧版核心不支持该参数时仍可显示版本和路径，并给出能力读取提示。
- 如果代理地址包含密码，桌面端只在当前运行会话中保留明文；写入状态文件时会改成 `redacted`，下次启动会清空该代理字段，需要重新输入密码。

## macOS 提示 App 已损坏

当前 macOS 桌面包是自签名应用，还没有接入 Apple Developer ID 公证。通过浏览器下载 zip 后，系统可能给 `.app` 加上隔离标记，启动时显示：

```text
“M3U8-DL.app”已损坏，无法打开。你应该将它移到废纸篓。
```

这通常不是文件真的损坏。解压后在终端执行下面命令清除隔离标记，再重新打开：

```zsh
xattr -dr com.apple.quarantine ~/Downloads/M3U8-DL.app
```

如果已经拖到“应用程序”，路径改成：

```zsh
xattr -dr com.apple.quarantine /Applications/M3U8-DL.app
```

执行后建议第一次用右键菜单打开一次。后续正式消除这个提示，需要用 Apple Developer ID 签名并完成 notarization。

## 发布产物

发布流水线预期同时生成 CLI 和桌面端：

- CLI：`M3U8-DL_v<version>_<os>_<arch>`，覆盖 darwin/linux/windows 的 amd64/arm64。
- 桌面端：`M3U8-DL_v<version>_desktop_macos_amd64.zip`、`M3U8-DL_v<version>_desktop_macos_arm64.zip`、`M3U8-DL_v<version>_desktop_linux_amd64.tar.gz`、`M3U8-DL_v<version>_desktop_windows_amd64.zip`。
- 构建产物不包含 `.sha256` 文件。

当前 `v1.0.5` Release 实际包含 6 个 CLI 包、2 个 macOS 桌面包和 1 个 Windows amd64 桌面包，共 9 个资产；Linux desktop 尚未发布。

Release workflow 的桌面包 job 依赖两个门禁：

- 根目录 `go test -count=1 ./...`。
- `desktop` 模块 `go test -count=1 ./...` 和 `node --check desktop/frontend/dist/app.js`。

## 已验证样本

真实地址 `https://hd.ijycnd.com/play/dL9Zywje/index.m3u8` 已用于验证：

- CLI parse-only：识别为 HLS master playlist，1 条 1080p AES-128 视频流，38 个分片，约 2 分 17 秒。
- CLI 小范围下载：`--custom-range 0-2 -M format=mp4:muxer=ffmpeg` 成功输出 MP4。
- 桌面后端任务流：使用同一 URL、同一范围，通过 `StartDownload` 创建任务、监控进度、扫描完成文件，成功找到输出 MP4。
