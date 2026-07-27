# M3U8-DL 当前项目状态

更新时间：2026-07-27（北京时间）

## 当前目标

使用 Go 复刻 [`nilaoda/N_m3u8DL-RE`](https://github.com/nilaoda/N_m3u8DL-RE) 的 HLS 能力，尽量保持命令行语义和处理行为等价。DASH、MSS 不在本复刻范围，只保留输入类型识别和不支持提示。

## 仓库与发布

- 项目仓库：[`lonnnnnng/M3U8-DL`](https://github.com/lonnnnnng/M3U8-DL)
- 当前版本：[`v1.0.6`](https://github.com/lonnnnnng/M3U8-DL/releases/tag/v1.0.6)
- 原版源码目录：`../N_m3u8DL-RE`
- Go 复刻目录：`../m3u8dl-go`
- 用户可见应用、仓库和发布归档统一使用 `M3U8-DL`；CLI、内置核心和兼容日志名称继续使用 `m3u8dl-go`。
- 远端 `main` 已清理为单一根提交；最新提交以远端 `main` 为准。
- `v1.0.6` 为手工发布，包含 6 个 CLI 包、2 个 macOS 桌面包和 1 个 Windows amd64 桌面包，共 9 个资产；Linux 桌面包已配置构建流程，但尚未进入该 Release。

## 当前已完成重点

- HLS Master/Media 解析、HTTP 加载和重试、轨道选择、并发下载、限速、断点复用、Range 和 BYTERANGE。
- AES-128、AES-128-ECB、CHACHA20 内置解密，以及 CENC、SAMPLE-AES、SAMPLE-AES-CTR 外部工具入口。
- 二进制合并、FFmpeg 单轨合并、FFmpeg/mkvmerge 最终混流、字幕修复和抽取。
- 直播录制、实时合并和 PipeMux 基础路径。
- 三项直播正确性修复：
  - 每条直播轨道使用独立 parser 和刷新循环，慢轨不会阻塞其他轨道。
  - 使用毫秒级 `PROGRAM-DATE-TIME` 或分片序号去重，同一秒内多个分片不会相互覆盖。
  - 每轮直播刷新都会重新过滤广告分片，并同步排除匹配的广告 `EXT-X-MAP` init。
- Wails 桌面端任务队列、预检查、设置持久化、外部工具检测、敏感信息脱敏和完成文件管理。
- 桌面端浅色、深色、自动三态主题；自动模式跟随系统外观，选择会持久化。
- CLI 更新检查已迁移到 `lonnnnnng/M3U8-DL`；桌面任务禁用逐任务更新检查，避免重复联网。

## 当前能力边界

- 不实现 DASH/MSS 下载，也不解析 Bilibili 页面；需要用户提供真实 m3u8 URL 和必要的 headers、cookies、key。
- 复杂 DRM/CENC/PSSH/KID、SAMPLE-AES/SAMPLE-AES-CTR 仍缺系统性真实样本验证。
- MP4 实时解密尚未覆盖上游直播状态机的所有边缘分支。
- 直播完整多轨收尾仍弱于原版；Windows PipeMux 已实现和交叉编译，但未在真实 Windows 环境运行验证。
- 未完整复刻原版 Spectre Console 动态进度 UI。

完整差距和源码对比见 [`FEATURE_COMPARISON.md`](FEATURE_COMPARISON.md)，历史真实样本证据见 [`REAL_SAMPLE_VALIDATION.md`](REAL_SAMPLE_VALIDATION.md)。

## 继续开发前检查

```zsh
cd /Users/long/Documents/CodexProjects/m3u8/m3u8dl-go
git status --short --branch
git log --oneline -5
go test -count=1 ./...
(cd desktop && go test -count=1 ./...)
node --check desktop/frontend/dist/app.js
git diff --check
```

这些门禁只能证明当前自动化覆盖的 HLS 和桌面行为通过，不能证明已经与原版完整等价。
