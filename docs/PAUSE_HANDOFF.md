# 暂停交接记录

更新时间：2026-06-19 09:29:13（北京时间）

## 当前目标

使用 Go 复刻 `nilaoda/N_m3u8DL-RE` 的 HLS 能力，要求尽量与原版行为等价；DASH、MSS 不在本复刻范围。

## 当前仓库状态

- 原版源码目录：`../N_m3u8DL-RE`
- Go 复刻目录：`../m3u8dl-go`
- 父目录 `/Users/long/Documents/CodexProjects/m3u8` 保持非 git 仓库。
- Go 复刻仓库远端：`https://github.com/lonnnnnng/m3u8dl-go.git`
- 原版仓库远端：`https://github.com/nilaoda/N_m3u8DL-RE.git`

## 已完成的主要进度

- 已整理原版和复刻版为两个独立目录，并保持远端隔离。
- 已完成 HLS Master/Media 解析、HLS 内容预处理、HLS key 多来源加载、AES-128/AES-128-ECB/CHACHA20、CENC/SAMPLE-AES 外部工具入口、下载/合并/字幕/直播基础流程等大量主干能力。
- 已补齐多项上游边缘语义：子 playlist 预处理、Dolby Vision 禁用最终混流、直播录制上限按媒体时长累计、CENC/MuxAfterDone 自动二进制合并提示资源、HLS key 加载失败提示。
- 已建立本地门禁和 GitHub Actions：测试、多平台构建、tag release workflow。

## 本次暂停前的最后改动

本轮正在收口 HLS key HTTP 加载重试提示：

- 原版 `DefaultHLSKeyProcessor` 在每次 HTTP key 加载失败时输出错误和 `retryCount: N`。
- 原版会按 3、2、1、0 的顺序提示，最后输出 `cmd_loadKeyFailed` 并把加密方式降级为 `UNKNOWN`。
- Go 版已新增同等 retryCount 输出，并保留最终 `cmd_loadKeyFailed` 降级行为。
- 测试已扩展 `TestParseMediaKeyLoadRetriesBeforeDowngrade` 和 `TestParseMediaKeyLoadFailureUsesUpstreamRetryCount`，用零延迟避免单测等待 1 秒重试。

## 下次继续优先级

1. 优先从 `docs/FEATURE_COMPARISON.md` 的“当前最需要继续补齐的 HLS 差距”继续。
2. 下一块建议处理“真实证据不足”的能力：
   - 复杂 DRM/CENC/PSSH/KID 样本验证。
   - SAMPLE-AES/SAMPLE-AES-CTR 真实样本覆盖。
   - MP4 实时解密在直播状态机里的完整边缘分支。
3. 如果暂时没有真实媒体样本，则继续补可由源码证明的行为：
   - CLI 帮助的完整展示形态与原版仍有差距；`StaticText.cs` 资源 key 已全部登记到 Go 版资源表。
   - 直播 producer/consumer 多轨收尾细节。
   - PipeMux Windows 实机验证计划与可执行脚本。

## 下次恢复建议命令

```bash
cd /Users/long/Documents/CodexProjects/m3u8/m3u8dl-go
git status --short --branch
git log --oneline -5
go test -count=1 ./...
```

完成代码改动后继续使用当前门禁：

```bash
git diff --check
go test -count=1 ./...
go build -o /tmp/m3u8dl-go-check .
GOOS=windows GOARCH=amd64 go test -c -o /tmp/m3u8dl-go-windows.test.exe .
```
