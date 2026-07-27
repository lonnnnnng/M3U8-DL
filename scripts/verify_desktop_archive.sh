#!/usr/bin/env bash
set -euo pipefail

ARCHIVE="${1:-}"
if [[ -z "${ARCHIVE}" ]]; then
  echo "Usage: $0 <desktop-archive>" >&2
  exit 2
fi

if [[ ! -f "${ARCHIVE}" ]]; then
  echo "桌面包不存在: ${ARCHIVE}" >&2
  exit 1
fi

require_entry() {
  local entries="$1"
  local expected="$2"
  if ! grep -Fxq "${expected}" <<<"${entries}"; then
    echo "桌面包缺少必要文件: ${expected}" >&2
    exit 1
  fi
}

reject_sha256() {
  local entries="$1"
  if grep -Eq '(^|/)[^/]+\.sha256$' <<<"${entries}"; then
    echo "桌面包不应包含 .sha256 文件" >&2
    exit 1
  fi
}

case "${ARCHIVE}" in
  *_desktop_macos_*.zip)
    entries="$(zipinfo -1 "${ARCHIVE}")"
    require_entry "${entries}" "M3U8-DL.app/"
    require_entry "${entries}" "M3U8-DL.app/Contents/MacOS/m3u8dl-go-desktop"
    require_entry "${entries}" "M3U8-DL.app/Contents/Resources/m3u8dl-go-cli"
    reject_sha256 "${entries}"
    ;;
  *_desktop_linux_*.tar.gz)
    entries="$(LC_ALL=C tar -tzf "${ARCHIVE}")"
    package_name="$(basename "${ARCHIVE}" .tar.gz)"
    require_entry "${entries}" "${package_name}/"
    require_entry "${entries}" "${package_name}/m3u8dl-go-desktop"
    require_entry "${entries}" "${package_name}/m3u8dl-go-cli"
    reject_sha256 "${entries}"
    ;;
  *_desktop_windows_*.zip)
    entries="$(zipinfo -1 "${ARCHIVE}")"
    package_name="$(basename "${ARCHIVE}" .zip)"
    require_entry "${entries}" "${package_name}/"
    require_entry "${entries}" "${package_name}/m3u8dl-go-desktop.exe"
    require_entry "${entries}" "${package_name}/m3u8dl-go-cli.exe"
    reject_sha256 "${entries}"
    ;;
  *)
    echo "不支持的桌面包格式: ${ARCHIVE}" >&2
    exit 2
    ;;
esac

echo "桌面包结构检查通过: ${ARCHIVE}"
