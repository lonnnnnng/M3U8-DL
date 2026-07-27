# 真实样本验证记录

更新时间：2026-06-23 00:20:00（北京时间）

本文记录复刻版对真实 HLS 样本的验证结果。这里的验证只证明对应样本和命令覆盖的路径可用，不代表已经完成全量 HLS parity。

## 样本：jisuzyv AES-128 点播

- URL：`https://play.jisuzyv.com/play/bYE7AEMb/index.m3u8`
- 类型：HLS master playlist，单 1080p 子流
- 子 playlist：`https://play.jisuzyv.com/play/hls/bYE7AEMb/index.m3u8`
- 加密：`#EXT-X-KEY:METHOD=AES-128,URI="enc.key",IV=0x00000000000000000000000000000000`
- 分片：165 个 TS 分片
- 结束标记：包含 `#EXT-X-ENDLIST`
- key 验证：`enc.key` 可下载，长度 16 字节

### 覆盖能力

- master m3u8 解析和子 playlist 跳转。
- 相对 HLS key URL 合并。
- AES-128 key 加载与解密。
- `--custom-range 0-2` 分片裁剪。
- TS 分片真实下载。
- 二进制合并输出。
- ffprobe 媒体识别。

### 解析验证

命令：

```zsh
go run . 'https://play.jisuzyv.com/play/bYE7AEMb/index.m3u8' \
  --auto-select true \
  --skip-download true \
  --save-dir /tmp/m3u8dl-go-real-jisu/out \
  --tmp-dir /tmp/m3u8dl-go-real-jisu/tmp \
  --save-name jisuzyv-real \
  --disable-update-check true \
  --ui-language zh-CN
```

结果：

- 成功识别 `HTTP Live Streaming`。
- 成功解析 master 列表和子流。
- 解析出 1 条基本流，无可选音频流、无可选字幕流。
- 选中流信息：`Vid *AES-128 1920x1080 | 4096 Kbps | 165 Segments | ~07m48s`。
- 成功写出 `raw.m3u8`、`meta.json`、`meta_selected.json` 到显式 `--tmp-dir` 下。

### 小范围下载验证

命令：

```zsh
go run . 'https://play.jisuzyv.com/play/bYE7AEMb/index.m3u8' \
  --auto-select true \
  --custom-range 0-2 \
  --binary-merge true \
  --thread-count 3 \
  --save-dir /tmp/m3u8dl-go-real-jisu-download/out \
  --tmp-dir /tmp/m3u8dl-go-real-jisu-download/tmp \
  --save-name jisuzyv-real-range \
  --disable-update-check true \
  --ui-language zh-CN \
  --del-after-done false
```

结果：

- 成功按 `0-2` 裁剪为 3 个分片，日志显示约 `00m16s`。
- 成功下载 3/3 个分片。
- 成功输出：`/tmp/m3u8dl-go-real-jisu-download/out/jisuzyv-real-range.ts`。
- 输出文件大小：`1,652,708` 字节。
- 前 5 个 TS packet 同步字均为 `0x47`。
- `ffprobe` 可识别输出为 `mpegts`，包含：
  - 视频：H.264 High，1920x1080，24 fps
  - 音频：AAC LC，48000 Hz，stereo
  - 时长：约 16.6 秒

### 合并方式对比验证

同一 URL、同一范围 `--custom-range 0-2` 分别验证二进制直拼和默认 ffmpeg 合并。

二进制直拼命令额外添加：

```zsh
--binary-merge true
```

结果：

- 输出：`/tmp/m3u8dl-go-compare-binary/out/jisuzyv-binary.ts`
- 容器：`mpegts`
- 文件大小：`1,652,708` 字节
- 视频：H.264，1920x1080，24 fps，约 16.58 秒
- 音频：AAC，48000 Hz，stereo，约 16.51 秒
- TS packet 同步字正常，前 5 个 packet 都是 `0x47`
- `ffmpeg -v error -i ... -f null -` 无解码错误输出

默认 ffmpeg 合并不添加 `--binary-merge true`。

结果：

- 输出：`/tmp/m3u8dl-go-compare-ffmpeg/out/jisuzyv-ffmpeg.mp4`
- 容器：`mov,mp4,m4a,3gp,3g2,mj2`
- 文件大小：`1,570,491` 字节
- 视频：H.264，1920x1080，24 fps，约 16.58 秒
- 音频：AAC，48000 Hz，stereo，约 16.53 秒
- `ffmpeg -v error -i ... -f null -` 无解码错误输出

### 边界

- 本次没有完整下载 165 个分片，只验证了前 3 个分片的真实下载、解密和合并。
- 本样本是 AES-128 TS 点播，不覆盖 CENC、SAMPLE-AES、fMP4、直播刷新、多音轨、多字幕或最终 `-M` 混流。
- 本次没有在原版 .NET 项目本地运行对照；当前本机原版构建仍受 .NET SDK 版本限制。

## 真实样本：ijycnd AES-128 HLS

URL：

```text
https://hd.ijycnd.com/play/dL9Zywje/index.m3u8
```

### 解析验证

命令：

```zsh
go run . 'https://hd.ijycnd.com/play/dL9Zywje/index.m3u8' \
  --auto-select true \
  --skip-download true \
  --save-dir /tmp/m3u8dl-go-ijycnd-parse/out \
  --tmp-dir /tmp/m3u8dl-go-ijycnd-parse/tmp \
  --save-name ijycnd-parse \
  --disable-update-check true \
  --ui-language zh-CN
```

结果：

- 识别为 HLS master playlist。
- 共 1 条媒体流：`Vid *AES-128 1920x1080 | 4096 Kbps | 38 Segments | ~02m17s`。
- parse-only 成功写出 meta。

### 小范围下载与桌面后端验证

命令：

```zsh
go run . 'https://hd.ijycnd.com/play/dL9Zywje/index.m3u8' \
  --save-dir /tmp/m3u8dl-go-ijycnd-download/out \
  --tmp-dir /tmp/m3u8dl-go-ijycnd-download/tmp \
  --save-name ijycnd-range \
  --auto-select true \
  --ui-language zh-CN \
  --disable-update-check true \
  --thread-count 8 \
  --download-retry-count 3 \
  --use-system-proxy true \
  --custom-range 0-2 \
  -M format=mp4:muxer=ffmpeg
```

结果：

- 成功裁剪为 3 个分片，约 17 秒。
- 成功下载 3/3 个分片。
- 成功输出：`/tmp/m3u8dl-go-ijycnd-download/out/ijycnd-range.MUX.mp4`。
- `ffprobe` 可识别输出为 MP4，包含：
  - 视频：H.264 High，1920x1080，24 fps，约 17.9 秒。
  - 音频：AAC LC，44100 Hz，stereo，约 17.9 秒。

桌面后端任务流也用同一 URL 通过：

```zsh
M3U8DL_GO_REAL_SAMPLE=1 \
M3U8DL_GO_CLI=/tmp/m3u8dl-go-check \
go test -count=1 -run TestDesktopRealSampleDownload ./...
```

验证范围：

- `StartDownload` 创建任务并启动。
- 任务状态进入完成。
- 日志捕获下载进度。
- 完成后扫描到 MP4 输出文件。

### 桌面 App UI 验证

打包并打开 macOS 桌面版：

```zsh
./scripts/build_desktop_macos.sh
open desktop/build/bin/M3U8-DL.app
```

表单参数：

- m3u8 地址：`https://hd.ijycnd.com/play/dL9Zywje/index.m3u8`
- 输出目录：`/tmp/m3u8dl-go-ijycnd-ui/out`
- 保存名称：`ijycnd-ui-range`
- 下载范围：`0-2`
- 自动选轨：开启
- MP4 混流：开启

结果：

- 桌面 UI 成功创建并启动下载任务。
- 打包后的 `.app` 调用内置 `Contents/Resources/m3u8dl-go-cli`。
- 任务状态从下载中更新为完成。
- 完成文件列表显示 `ijycnd-ui-range.MUX.mp4`，大小约 5.3 MB。
- 任务日志能显示解析、下载、二进制合并和混流输出过程。
