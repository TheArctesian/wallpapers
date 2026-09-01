# shellcheck shell=bash
# Shared helpers. Sourced by ./wallpapers, never executed directly.

REPO_ROOT=${REPO_ROOT:?common.sh expects REPO_ROOT}
RAW_DIR="$REPO_ROOT/raw"
SCREENS_DIR="$REPO_ROOT/screens"
IMAGE_TAG=${IMAGE_TAG:-wallpapers-dither}

die()  { printf 'wallpapers: %s\n' "$*" >&2; exit 1; }
warn() { printf 'wallpapers: %s\n' "$*" >&2; }
info() { printf '==> %s\n' "$*"; }

need() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1${2:+ ($2)}"
}

# list_wallpapers DIR — rendered wallpapers in DIR, one per line. Silent and
# empty for a screen geometry that has never been rendered, rather than a
# `find` failure that pipefail would turn into an abort.
list_wallpapers() {
  [ -d "$1" ] || return 0
  find "$1" -maxdepth 1 -type f \( -iname '*.png' -o -iname '*.jpg' \) 2>/dev/null
}

# platform: the name of the detect/apply backend pair to use.
platform() {
  case "$(uname -s)" in
    Darwin) echo macos ;;
    Linux)  echo linux ;;
    *)      die "unsupported platform: $(uname -s)" ;;
  esac
}
