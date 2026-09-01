# shellcheck shell=bash
# Linux/COSMIC wallpaper backend.
#
# Writes per-output entries into COSMIC's background config; cosmic-bg picks
# the change up live. Outputs are addressed by connector name (DP-1, eDP-1),
# which is what detect.sh reports on Linux.

COSMIC_CFG="${XDG_CONFIG_HOME:-$HOME/.config}/cosmic/com.system76.CosmicBackground/v1"
_cosmic_applied=()

apply_wallpaper() { # ID NAME WxH IMAGE_PATH
  local output=$1 image=$4
  mkdir -p "$COSMIC_CFG"
  cat > "$COSMIC_CFG/output.$output" <<EOF
(
    output: "$output",
    source: Path("$image"),
    filter_by_theme: false,
    rotation_frequency: 900,
    filter_method: Lanczos,
    scaling_mode: Zoom,
    sampling_method: Alphanumeric,
)
EOF
  _cosmic_applied+=("$output")
}

# cosmic-bg only reads the per-output entries once it is told which outputs
# exist and that they are configured individually.
commit_wallpapers() {
  [ "${#_cosmic_applied[@]}" -gt 0 ] || return 0
  printf '%s' 'false' > "$COSMIC_CFG/same-on-all"
  local list
  list=$(printf '"%s", ' "${_cosmic_applied[@]}")
  printf '[%s]' "${list%, }" > "$COSMIC_CFG/backgrounds"
}
