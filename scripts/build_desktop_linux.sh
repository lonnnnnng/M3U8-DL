#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
DESKTOP_DIR="${ROOT_DIR}/desktop"
WAILS_VERSION="${WAILS_VERSION:-v2.12.0}"
WAILS_BIN="${WAILS_BIN:-$(go env GOPATH)/bin/wails}"
ARCH_SUFFIX="$(go env GOARCH)"

resolve_release_version() {
  local raw_version

  if [[ -n "${RELEASE_VERSION:-}" ]]; then
    raw_version="${RELEASE_VERSION}"
  else
    # long: 桌面包名必须和 CLI 自报版本一致，避免用户下载时分不清核心和外壳是否同版。
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
TARGET_NAME="M3U8-DL_${RELEASE_VERSION}_desktop_linux_${ARCH_SUFFIX}"

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "Linux desktop package must be built on Linux." >&2
  exit 1
fi

if [[ ! -x "${WAILS_BIN}" ]]; then
  go install "github.com/wailsapp/wails/v2/cmd/wails@${WAILS_VERSION}"
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

CLI_HELPER="${TMP_DIR}/m3u8dl-go-cli"
go build -trimpath -ldflags="-s -w" -o "${CLI_HELPER}" "${ROOT_DIR}"

rm -rf "${DESKTOP_DIR}/build"
mkdir -p "${DESKTOP_DIR}/build"
# long: Linux 构建同样从受版本控制的图标源恢复资源，避免清理后回落到 Wails 默认图标。
cp "${DESKTOP_DIR}/assets/appicon.png" "${DESKTOP_DIR}/build/appicon.png"
(cd "${DESKTOP_DIR}" && "${WAILS_BIN}" build -clean)

APP_BIN="$(find "${DESKTOP_DIR}/build/bin" -maxdepth 1 -type f -perm -111 -name 'm3u8dl-go*' -print -quit)"
if [[ -z "${APP_BIN}" ]]; then
  echo "未找到 Wails 生成的 Linux 桌面程序" >&2
  exit 1
fi

PACKAGE_DIR="${TMP_DIR}/${TARGET_NAME}"
mkdir -p "${PACKAGE_DIR}" "${DIST_DIR}"
cp "${APP_BIN}" "${PACKAGE_DIR}/m3u8dl-go-desktop"
cp "${CLI_HELPER}" "${PACKAGE_DIR}/m3u8dl-go-cli"
cp "${ROOT_DIR}/README.md" "${PACKAGE_DIR}/"
chmod +x "${PACKAGE_DIR}/m3u8dl-go-desktop" "${PACKAGE_DIR}/m3u8dl-go-cli"

ARCHIVE="${DIST_DIR}/${TARGET_NAME}.tar.gz"
rm -f "${ARCHIVE}" "${ARCHIVE}.sha256"
LC_ALL=C tar -C "${TMP_DIR}" -czf "${ARCHIVE}" "${TARGET_NAME}"
"${ROOT_DIR}/scripts/verify_desktop_archive.sh" "${ARCHIVE}"

echo "${ARCHIVE}"
