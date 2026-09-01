#!/bin/bash
# Shrink and speed up the recorded GIFs for a README.
#
# VHS records at the terminal's own size and frame rate, which is more of both
# than a README needs. Halving the frame rate and generating a per-file palette
# takes the Claude Code session from about 2.2 MB to 1.6 MB with no visible
# loss at reading size.
#
# SPEED compresses playback time. Model turns take as long as they take, and a
# 90 second GIF is longer than anyone watches, so the recordings are made at a
# readable typing speed and then played back faster.
#
#   SPEED=1.5 bash demo/optimise.sh usage.gif
set -euo pipefail
cd "$(dirname "$0")"

speed="${SPEED:-1}"

for gif in "$@"; do
    [ -f "$gif" ] || { echo "no such file: $gif" >&2; exit 1; }
    before=$(stat -c %s "$gif")
    ffmpeg -v error -i "$gif" \
        -vf "setpts=PTS/$speed,fps=12,scale=1200:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=128[p];[s1][p]paletteuse=dither=bayer:bayer_scale=3" \
        -y "$gif.opt.gif"
    mv "$gif.opt.gif" "$gif"
    after=$(stat -c %s "$gif")
    secs=$(ffprobe -v error -show_entries format=duration -of csv=p=0 "$gif")
    printf '%s  %d KB -> %d KB  %.0fs at %sx\n' "$gif" $((before / 1024)) $((after / 1024)) "$secs" "$speed"
done
