# shellcheck shell=bash
# Screen discovery.
#
# detect_screens prints one tab-separated record per active screen:
#
#     ID <TAB> NAME <TAB> WIDTHxHEIGHT <TAB> ROTATION
#
# WIDTHxHEIGHT is always the *post-rotation* pixel geometry — the size the
# wallpaper actually has to be — so a monitor turned on its side reports a
# portrait geometry and is served portrait wallpapers. ID is whatever the
# platform's wallpaper setter needs in order to address that screen: the
# CoreGraphics display ID on macOS, the output name on Linux.

# _emit ID NAME W H DEGREES
# Normalises geometry against rotation. Sources disagree about whether they
# report the panel's native mode or the rotated framebuffer, so treat a
# rotated screen as portrait and swap only if the reported size still says
# landscape. This is idempotent, so it is safe for backends that have already
# applied the rotation themselves.
_emit() {
  local id=$1 name=$2 w=$3 h=$4 deg=$5
  deg=$(( (deg % 360 + 360) % 360 ))
  if [ "$deg" = 90 ] || [ "$deg" = 270 ]; then
    if [ "$w" -gt "$h" ]; then local t=$w; w=$h; h=$t; fi
  fi
  printf '%s\t%s\t%sx%s\t%s\n' "$id" "$name" "$w" "$h" "$deg"
}

# --- macOS: CoreGraphics display list via system_profiler -------------------
_detect_macos() {
  need jq "ships with macOS 15+; otherwise: brew install jq"
  local id name pixels rot fallback w h deg
  while IFS=$'\t' read -r id name pixels rot fallback; do
    # "3024 x 1964", or "spdisplays_3024x1964Retina" as a fallback.
    read -r w h <<<"$(printf '%s' "${pixels:-$fallback}" | awk -F'[^0-9]+' '{print $1, $2}')"
    [ -n "$w" ] && [ -n "$h" ] || { warn "skipping display '$name': no pixel size reported"; continue; }
    # system_profiler reports rotation as a capability string ("supported")
    # when unrotated and as an angle when rotated; digits or nothing.
    deg=$(printf '%s' "$rot" | tr -cd '0-9')
    _emit "$id" "$name" "$w" "$h" "${deg:-0}"
  done < <(
    system_profiler SPDisplaysDataType -json 2>/dev/null | jq -r '
      [.SPDisplaysDataType[]? | .spdisplays_ndrvs[]?] | .[]
      | select((.spdisplays_online // "spdisplays_yes") == "spdisplays_yes")
      # Mirrored screens share one framebuffer; the mirror source covers them.
      | select((.spdisplays_mirror // "spdisplays_off") == "spdisplays_off")
      | [ (._spdisplays_displayID // "0"),
          (._name // "Display"),
          (._spdisplays_pixels // ""),
          ((.spdisplays_rotation // "0") | tostring),
          (.spdisplays_pixelresolution // "") ]
      | @tsv'
  )
}

# --- Linux: X11 / XWayland --------------------------------------------------
# xrandr reports the post-rotation framebuffer geometry directly.
_detect_xrandr() {
  xrandr --query 2>/dev/null | awk '
    BEGIN { deg["normal"]=0; deg["left"]=90; deg["inverted"]=180; deg["right"]=270 }
    / connected/ {
      name = $1; geo = ""; rot = "normal"
      for (i = 2; i <= NF; i++) {
        if ($i ~ /^\(/) break                  # capability list — stop scanning
        if ($i ~ /^[0-9]+x[0-9]+\+-?[0-9]+\+-?[0-9]+$/) geo = $i
        else if ($i in deg) rot = $i
      }
      if (geo == "") next                      # connected but not active
      split(geo, a, "+"); split(a[1], s, "x")
      print name "\t" name "\t" s[1] "\t" s[2] "\t" deg[rot]
    }' | while IFS=$'\t' read -r id name w h deg; do _emit "$id" "$name" "$w" "$h" "$deg"; done
}

# --- Linux: bare Wayland (wlroots-based compositors) ------------------------
# Modes are the panel's native ones, so _emit applies the rotation.
_detect_wlr() {
  need jq
  wlr-randr --json 2>/dev/null | jq -r '
    .[] | select(.enabled)
    | . as $o | (.modes[] | select(.current)) as $m
    | [ $o.name, ($o.make + " " + $o.model | ltrimstr(" ")),
        $m.width, $m.height, ($o.transform // "normal") ] | @tsv' |
  while IFS=$'\t' read -r id name w h transform; do
    local deg
    case "$transform" in
      90|270|180) deg=$transform ;;
      *) deg=$(printf '%s' "$transform" | tr -cd '0-9'); deg=${deg:-0} ;;
    esac
    _emit "$id" "${name:-$id}" "$w" "$h" "$deg"
  done
}

_detect_linux() {
  if command -v xrandr >/dev/null 2>&1 && [ -n "${DISPLAY:-}" ]; then
    _detect_xrandr
  elif command -v wlr-randr >/dev/null 2>&1; then
    _detect_wlr
  else
    die "no way to enumerate screens: install xrandr (X11/XWayland) or wlr-randr (Wayland)"
  fi
}

detect_screens() {
  case "$(platform)" in
    macos) _detect_macos ;;
    linux) _detect_linux ;;
  esac
}
