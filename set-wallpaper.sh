#!/usr/bin/env bash
# Pick a random dithered wallpaper for each connected monitor and apply it
# via COSMIC's background config (cosmic-bg live-reloads the change).
#
# Wallpapers come from screens/<WxH>/ (produced by generate-screens.sh).
# A monitor with no matching folder is skipped, leaving its wallpaper as-is.
set -euo pipefail

REPO="$(dirname "$(readlink -f "$0")")"
SCREENS="$REPO/screens"
CFG="${XDG_CONFIG_HOME:-$HOME/.config}/cosmic/com.system76.CosmicBackground/v1"
mkdir -p "$CFG"

# Serialize: if another instance (e.g. several terminals opened at once) holds
# the lock, bail out — it is already setting a fresh wallpaper. Prevents two
# writers from corrupting a config file mid-read by cosmic-bg.
exec 9>"${TMPDIR:-/tmp}/.cosmic-wallpaper.lock"
flock -n 9 || exit 0

# name<TAB>WxH for every connected output (xrandr gives post-rotation size).
mapfile -t MONITORS < <(xrandr --query | awk '/ connected/{
  for (i=1;i<=NF;i++) if ($i ~ /^[0-9]+x[0-9]+\+[0-9]+\+[0-9]+$/) { split($i,a,"+"); print $1"\t"a[1]; break }
}')

[ "${#MONITORS[@]}" -gt 0 ] || { echo "no connected monitors" >&2; exit 0; }

entry() { # $1=output name  $2=absolute image path
  cat <<EOF
(
    output: "$1",
    source: Path("$2"),
    filter_by_theme: false,
    rotation_frequency: 900,
    filter_method: Lanczos,
    scaling_mode: Zoom,
    sampling_method: Alphanumeric,
)
EOF
}

names=()
for line in "${MONITORS[@]}"; do
  name=${line%%$'\t'*}
  wh=${line##*$'\t'}
  dir="$SCREENS/$wh"

  # Random wallpaper for this resolution; skip monitor if none generated.
  pick=$(find "$dir" -maxdepth 1 -type f \( -iname '*.png' -o -iname '*.jpg' \) 2>/dev/null | shuf -n1 || true)
  if [ -z "$pick" ]; then
    echo "skip $name ($wh): no wallpapers in $dir" >&2
    continue
  fi

  entry "$name" "$pick" > "$CFG/output.$name"
  names+=("$name")
done

[ "${#names[@]}" -gt 0 ] || { echo "no wallpapers applied" >&2; exit 0; }

# Tell cosmic-bg to use per-output entries.
printf '%s' 'false' > "$CFG/same-on-all"
list=$(printf '"%s", ' "${names[@]}"); list=${list%, }
printf '[%s]' "$list" > "$CFG/backgrounds"
