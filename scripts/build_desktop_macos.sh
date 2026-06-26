#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
DESKTOP_DIR="${ROOT_DIR}/desktop"
WAILS_VERSION="${WAILS_VERSION:-v2.12.0}"
WAILS_BIN="${WAILS_BIN:-$(go env GOPATH)/bin/wails}"
ARCH_NAME="$(uname -m)"

resolve_release_version() {
  local raw_version

  if [[ -n "${RELEASE_VERSION:-}" ]]; then
    raw_version="${RELEASE_VERSION}"
  else
    # 产物名要和程序自己报出的版本保持一致，所以默认直接从主程序里取版本号。
    raw_version="$(sed -nE 's/^const version = "m3u8dl-go ([^"]+)"$/\1/p' "${ROOT_DIR}/main.go" | head -n1)"
  fi

  if [[ -z "${raw_version}" ]]; then
    raw_version="dev"
  elif [[ "${raw_version}" != v* && "${raw_version}" != dev ]]; then
    raw_version="v${raw_version}"
  fi

  printf '%s' "${raw_version}"
}

RELEASE_VERSION="$(resolve_release_version)"

if [[ "${ARCH_NAME}" == "arm64" ]]; then
  ARCH_SUFFIX="arm64"
else
  ARCH_SUFFIX="amd64"
fi

TARGET_NAME="m3u8dl-go_${RELEASE_VERSION}_desktop_macos_${ARCH_SUFFIX}"

if [[ ! -x "${WAILS_BIN}" ]]; then
  go install "github.com/wailsapp/wails/v2/cmd/wails@${WAILS_VERSION}"
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

CLI_HELPER="${TMP_DIR}/m3u8dl-go-cli"
go build -trimpath -ldflags="-s -w" -o "${CLI_HELPER}" "${ROOT_DIR}"

rm -rf "${DESKTOP_DIR}/build"
(cd "${DESKTOP_DIR}" && "${WAILS_BIN}" build -clean)

APP_PATH="$(find "${DESKTOP_DIR}/build/bin" -maxdepth 1 -name '*.app' -print -quit)"
if [[ -z "${APP_PATH}" ]]; then
  echo "未找到 Wails 生成的 .app" >&2
  exit 1
fi

mkdir -p "${APP_PATH}/Contents/Resources"
cp "${CLI_HELPER}" "${APP_PATH}/Contents/Resources/m3u8dl-go-cli"
chmod +x "${APP_PATH}/Contents/Resources/m3u8dl-go-cli"

# long: 内置 CLI 是 Wails 签名后才注入的资源，必须重新签名，否则 macOS 会把 app 判定为密封资源被改动。
codesign --force --sign - "${APP_PATH}/Contents/Resources/m3u8dl-go-cli"
codesign --force --deep --sign - "${APP_PATH}"
codesign --verify --deep --strict --verbose=2 "${APP_PATH}"

mkdir -p "${DIST_DIR}"
ARCHIVE="${DIST_DIR}/${TARGET_NAME}.zip"
LEGACY_ARCHIVE="${DIST_DIR}/m3u8dl-go_desktop_macos_${ARCH_SUFFIX}.zip"
# 旧版无版本号 zip 容易被误认成当前构建结果，重新打包同一架构时一并清掉。
rm -f "${ARCHIVE}" "${ARCHIVE}.sha256" "${LEGACY_ARCHIVE}" "${LEGACY_ARCHIVE}.sha256"
ditto -c -k --norsrc --keepParent "${APP_PATH}" "${ARCHIVE}"
"${ROOT_DIR}/scripts/verify_desktop_archive.sh" "${ARCHIVE}"

echo "${ARCHIVE}"
