#!/usr/bin/env bash
# Download Nexa into the current directory (macOS / Linux).
# 一键下载 Nexa 到当前目录（macOS / Linux）
#
# Usage / 用法:
#   curl -fsSL https://raw.githubusercontent.com/teexue/nexa/main/scripts/install.sh | bash
#   或：bash install.sh
set -euo pipefail

REPO="${NEXA_REPO:-teexue/nexa}"
OUT_NAME="nexa"

# UI language from LANG/LC_* (zh* → zh, else en).
resolve_lang() {
	local raw="${LC_ALL:-${LC_MESSAGES:-${LANG:-}}}"
	raw="$(printf '%s' "$raw" | tr '[:upper:]' '[:lower:]')"
	case "$raw" in
	zh | zh_* | zh-* | *zh_cn* | *zh-cn* | *zh_tw* | *zh-tw* | *zh_hk*) echo "zh" ;;
	*) echo "en" ;;
	esac
}

LANG_UI="$(resolve_lang)"

t() {
	local key="$1"
	shift
	local msg=""
	case "$LANG_UI:$key" in
	en:err_arch) msg='Unsupported CPU architecture: %s' ;;
	zh:err_arch) msg='不支持的 CPU 架构: %s' ;;
	en:err_arch_hint) msg='Download manually from https://github.com/%s/releases/latest' ;;
	zh:err_arch_hint) msg='请从 https://github.com/%s/releases/latest 手动下载。' ;;
	en:err_windows) msg='On Windows, use scripts/install.ps1.' ;;
	zh:err_windows) msg='请在 Windows 上使用 scripts/install.ps1。' ;;
	en:err_os) msg='Unsupported OS: %s' ;;
	zh:err_os) msg='不支持的系统: %s' ;;
	en:err_os_hint) msg='Use this script on macOS / Linux; on Windows use scripts/install.ps1.' ;;
	zh:err_os_hint) msg='macOS / Linux 用本脚本；Windows 用 scripts/install.ps1。' ;;
	en:err_downloader) msg='curl or wget is required.' ;;
	zh:err_downloader) msg='需要 curl 或 wget。' ;;
	en:downloading) msg='Downloading %s …' ;;
	zh:downloading) msg='正在下载 %s …' ;;
	en:saved) msg='Saved to:  %s' ;;
	zh:saved) msg='已下载到: %s' ;;
	en:run) msg='Run:       ./%s' ;;
	zh:run) msg='启动:      ./%s' ;;
	en:browser) msg='Browser:   http://localhost:8080' ;;
	zh:browser) msg='浏览器:    http://localhost:8080' ;;
	*) msg="$key" ;;
	esac
	# shellcheck disable=SC2059
	printf "$msg\n" "$@"
}

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$arch" in
x86_64 | amd64) arch="amd64" ;;
aarch64 | arm64) arch="arm64" ;;
*)
	t err_arch "$(uname -m)" >&2
	t err_arch_hint "$REPO" >&2
	exit 1
	;;
esac

case "$os" in
darwin | linux) ;;
msys* | mingw* | cygwin*)
	t err_windows >&2
	exit 1
	;;
*)
	t err_os "$(uname -s)" >&2
	t err_os_hint >&2
	exit 1
	;;
esac

asset="${OUT_NAME}-${os}-${arch}"
url="https://github.com/${REPO}/releases/latest/download/${asset}"
dest="$(pwd)/${OUT_NAME}"

t downloading "$asset"
echo "  ${url}"

tmp="$(mktemp "${TMPDIR:-/tmp}/nexa.XXXXXX")"
cleanup() { rm -f "$tmp"; }
trap cleanup EXIT

if command -v curl >/dev/null 2>&1; then
	curl -fL --progress-bar -o "$tmp" "$url"
elif command -v wget >/dev/null 2>&1; then
	wget -q --show-progress -O "$tmp" "$url"
else
	t err_downloader >&2
	exit 1
fi

mv -f "$tmp" "$dest"
trap - EXIT
chmod +x "$dest"

if [[ "$os" == "darwin" ]]; then
	xattr -d com.apple.quarantine "$dest" 2>/dev/null || true
fi

echo
t saved "$dest"
t run "$OUT_NAME"
t browser
