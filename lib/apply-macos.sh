# shellcheck shell=bash
# macOS wallpaper backend.
#
# System Events exposes one `desktop` per screen, keyed by the same
# CoreGraphics display ID that detect.sh reports, so wallpapers land on the
# right screen regardless of how the displays are arranged. Note that macOS
# applies the picture to the screen's *current* Space only; other Spaces keep
# whatever they had until they are set from within them.

apply_wallpaper() { # ID NAME WxH IMAGE_PATH
  local id=$1 name=$2 image=$4
  osascript - "$id" "$image" <<'APPLESCRIPT' >/dev/null
on run argv
  set targetID to (item 1 of argv) as integer
  set img to POSIX file (item 2 of argv)
  set applied to 0
  tell application "System Events"
    repeat with d in desktops
      if (id of d) is targetID then
        set picture of d to img
        set applied to applied + 1
      end if
    end repeat
  end tell
  if applied = 0 then error "no desktop with display id " & targetID
end run
APPLESCRIPT
}

# Nothing to commit — each apply_wallpaper call takes effect immediately.
commit_wallpapers() { :; }
