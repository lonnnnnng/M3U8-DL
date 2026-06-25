#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
DESKTOP_DIR="${ROOT_DIR}/desktop"
WAILS_VERSION="${WAILS_VERSION:-v2.12.0}"
WAILS_BIN="${WAILS_BIN:-$(go env GOPATH)/bin/wails}"
ARCH_NAME="$(uname -m)"

if [[ "${ARCH_NAME}" == "arm64" ]]; then
  TARGET_NAME="m3u8dl-go_desktop_macos_arm64"
else
  TARGET_NAME="m3u8dl-go_desktop_macos_amd64"
fi

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

mkdir -p "${DIST_DIR}"
ARCHIVE="${DIST_DIR}/${TARGET_NAME}.zip"
rm -f "${ARCHIVE}" "${ARCHIVE}.sha256"
ditto -c -k --sequesterRsrc --keepParent "${APP_PATH}" "${ARCHIVE}"

echo "${ARCHIVE}"
