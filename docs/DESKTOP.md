# m3u8dl-go 桌面版

桌面版使用 Wails v2 构建，目标是复用同一套 Go 下载核心，在 macOS、Windows、Linux 上保持一致的桌面操作入口。发布流水线会生成 macOS amd64/arm64、Linux amd64、Windows amd64 桌面包。

## 功能

- 创建下载任务，支持创建后立即下载或先加入等待队列。
- 支持在 m3u8 地址框粘贴多行地址批量创建任务。
- 支持 `资源名称|资源地址` 这类带名称的批量行；名称分隔符可在界面中指定，匹配到的资源名称会作为该任务保存名称。
- 新建任务页可直接复制当前表单对应的 CLI 命令；批量地址会生成多行命令，便于在终端预演。
- 新建任务页支持预检查，会在创建任务前检查下载地址数量、输出目录写入权限、下载核心可用性、核心参数解析结果，以及 MP4 混流所需 FFmpeg。
- 任务列表展示等待、下载中、完成、失败、停止状态。
- 任务队列支持按名称、地址、输出目录和状态文本搜索，可按等待、下载中、完成、失败、已停止状态筛选，并支持按最新、最旧、状态、进度和名称排序。
- 任务进度监控，优先解析命令行核心的 `--progress-json` 进度事件并更新进度条、速度、已耗时和预计剩余时间；下载完成时会读取 `summary` 事件中的输出文件列表，失败时会保留 `error` 事件中的真实错误信息，目录扫描仍作为兼容回退；旧版 `下载进度 n/m` 文本仍可解析。
- 任务详情展示输出目录、保存名称、下载参数和任务日志。
- 任务详情展示实际 CLI 命令，可一键复制命令或日志用于排障；日志面板可导出脱敏后的 `.log` 文件，也可清空当前任务已持久化的日志；Cookie、Authorization、代理密码不会原样写入导出日志或持久化状态文件。
- 完成文件管理，支持刷新文件列表、打开输出目录、打开文件、删除任务输出文件。
- 支持单个任务开始、停止、重试、移除，也支持批量开始等待任务、停止运行中任务、重试失败任务和清理已完成/失败/停止任务。
- 任务详情操作会按当前状态自动启用或禁用：运行中任务只能停止，排队任务可取消排队，失败任务优先走重试，未选中任务时禁用详情操作。
- 可设置最大同时运行任务数；批量启动超过上限时会进入排队状态，前一个任务结束或停止后自动调度下一个排队任务。
- 设置输出目录、保存名称和下载范围。
- 每行填写一个请求头，例如 `Cookie: xxx`、`Referer: https://example.com`。
- 开关控制自动选轨、MP4 混流、二进制直拼。
- 参数设置：FFmpeg 路径、最大同时运行任务数、线程数、重试次数、限速、代理、系统代理、多轨并发；FFmpeg 路径旁可以直接检测当前工具是否可用。
- 设置页支持一键检测 `ffmpeg`、`ffprobe`、`mkvmerge`、`mp4decrypt` 和 Shaka Packager；`ffprobe` 会优先按 FFmpeg 同目录派生路径查找，Shaka Packager 会识别常见平台别名。
- 设置页会显示当前下载核心状态、版本和路径，便于确认桌面包实际调用的 `m3u8dl-go-cli`。
- 任务和全局参数会持久化到用户配置目录；Cookie 等请求头不会写入磁盘。

桌面版不会重新实现下载协议，所有下载、解密、合并和字幕逻辑仍由内置 `m3u8dl-go-cli` 执行。

## 本地构建

macOS `.app`：

```zsh
./scripts/build_desktop_macos.sh
```

脚本会执行：

- 构建当前仓库的 CLI helper：`m3u8dl-go-cli`。
- 使用 Wails v2 构建 macOS `.app`。
- 将 `m3u8dl-go-cli` 放入 `.app/Contents/Resources/`。
- 对内置 CLI 和 `.app` 重新做本机自签名，并执行 `codesign --verify --deep --strict` 校验，避免注入 CLI 后破坏 macOS 应用密封。
- 生成 `dist/m3u8dl-go_<version>_desktop_macos_<arch>.zip`。
- 打包后调用 `scripts/verify_desktop_archive.sh` 校验 `.app`、桌面可执行和内置 `m3u8dl-go-cli` 都在包内，且没有 `.sha256` 文件。

Linux 桌面包需要在 Linux 上构建：

```zsh
./scripts/build_desktop_linux.sh
```

脚本会构建桌面程序和同版本 `m3u8dl-go-cli`，并打包为 `dist/m3u8dl-go_<version>_desktop_linux_<arch>.tar.gz`。构建环境需要 GTK/WebKitGTK 开发库；发布流水线使用 Ubuntu 22.04 和 `libgtk-3-dev`、`libwebkit2gtk-4.0-dev`。
打包后同样会调用 `scripts/verify_desktop_archive.sh` 校验包内桌面程序、内置 CLI 和 `.sha256` 排除规则。

Windows 桌面包需要在 Windows PowerShell 上构建：

```powershell
./scripts/build_desktop_windows.ps1
```

脚本会构建桌面程序和同版本 `m3u8dl-go-cli.exe`，并打包为 `dist/m3u8dl-go_<version>_desktop_windows_<arch>.zip`。
打包后会在 PowerShell 脚本内校验包内桌面程序、内置 CLI 和 `.sha256` 排除规则。

## 运行要求

- macOS 需要允许运行自签名应用。
- Linux 运行桌面版需要系统安装 GTK/WebKitGTK 运行库；不同发行版包名可能不同，Ubuntu/Debian 通常来自 `libgtk-3-0` 和 `libwebkit2gtk-4.0-37`。
- Windows 运行桌面版依赖系统 WebView2 Runtime；Windows 10/11 通常已经内置或可由系统自动安装。
- MP4 混流依赖 `ffmpeg`，媒体探测依赖 `ffprobe`。通过 Homebrew 安装时通常在 `/opt/homebrew/bin/ffmpeg`、`/opt/homebrew/bin/ffprobe` 或 `/usr/local/bin/` 下。
- 如果系统找不到 FFmpeg，可以在桌面版里填写 FFmpeg 路径。
- 预检查输出目录时会创建目录并写入一个临时探针文件，探针文件会立即删除。
- 预检查参数时会调用内置 CLI 的 `--print-effective-options`，只解析参数，不联网解析 m3u8、不下载分片。
- 如果代理地址包含密码，桌面端只在当前运行会话中保留明文；写入状态文件时会改成 `redacted`，下次启动会清空该代理字段，需要重新输入密码。

## macOS 提示 App 已损坏

当前 macOS 桌面包是自签名应用，还没有接入 Apple Developer ID 公证。通过浏览器下载 zip 后，系统可能给 `.app` 加上隔离标记，启动时显示：

```text
“m3u8dl-go.app”已损坏，无法打开。你应该将它移到废纸篓。
```

这通常不是文件真的损坏。解压后在终端执行下面命令清除隔离标记，再重新打开：

```zsh
xattr -dr com.apple.quarantine ~/Downloads/m3u8dl-go.app
```

如果已经拖到“应用程序”，路径改成：

```zsh
xattr -dr com.apple.quarantine /Applications/m3u8dl-go.app
```

执行后建议第一次用右键菜单打开一次。后续正式消除这个提示，需要用 Apple Developer ID 签名并完成 notarization。

## 发布产物

Release 会同时包含 CLI 和桌面端：

- CLI：`m3u8dl-go_<version>_<os>_<arch>`，覆盖 darwin/linux/windows 的 amd64/arm64。
- 桌面端：`m3u8dl-go_<version>_desktop_macos_amd64.zip`、`m3u8dl-go_<version>_desktop_macos_arm64.zip`、`m3u8dl-go_<version>_desktop_linux_amd64.tar.gz`、`m3u8dl-go_<version>_desktop_windows_amd64.zip`。
- 构建产物不包含 `.sha256` 文件。

Release workflow 的桌面包 job 依赖两个门禁：

- 根目录 `go test -count=1 ./...`。
- `desktop` 模块 `go test -count=1 ./...` 和 `node --check desktop/frontend/dist/app.js`。

## 已验证样本

真实地址 `https://hd.ijycnd.com/play/dL9Zywje/index.m3u8` 已用于验证：

- CLI parse-only：识别为 HLS master playlist，1 条 1080p AES-128 视频流，38 个分片，约 2 分 17 秒。
- CLI 小范围下载：`--custom-range 0-2 -M format=mp4:muxer=ffmpeg` 成功输出 MP4。
- 桌面后端任务流：使用同一 URL、同一范围，通过 `StartDownload` 创建任务、监控进度、扫描完成文件，成功找到输出 MP4。
