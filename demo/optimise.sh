#!/bin/bash
# Trim, shrink and speed up the recorded GIFs for a README.
#
# Three passes, in this order:
#
#   TRIM   Cut the tail once the screen stops changing. A tape has to allow for
#          the slowest a model turn might take, so a recording usually ends with
#          a still frame held for many seconds. A blinking cursor scores about
#          0.0004 on ffmpeg's scene metric and real output scores above 0.002,
#          so 0.001 separates them.
#   SPEED  Compress playback. A model turn takes as long as it takes, and a 90
#          second GIF is longer than anyone watches.
#   size   Halve the frame rate and generate a per-file palette. Takes the
#          Claude Code session from about 2.2 MB to 1.6 MB with no visible loss
#          at reading size.
#
#   SPEED=1.6 bash demo/optimise.sh usage.gif
#   TRIM=0 bash demo/optimise.sh live.gif
set -euo pipefail
cd "$(dirname "$0")"

speed="${SPEED:-1}"
trim="${TRIM:-1}"
dwell="${DWELL:-2}"   # seconds to hold the last frame that changed
fps="${FPS:-12}"
width="${WIDTH:-1200}"
colors="${COLORS:-128}"

for gif in "$@"; do
    [ -f "$gif" ] || { echo "no such file: $gif" >&2; exit 1; }
    before=$(stat -c %s "$gif")
    was=$(ffprobe -v error -show_entries format=duration -of csv=p=0 "$gif")

    cut=""
    if [ "$trim" = 1 ]; then
        last=$(ffmpeg -hide_banner -i "$gif" \
            -vf "select='gt(scene,0.001)',metadata=print:file=-" -f null - 2>/dev/null |
            awk -F'pts_time:' '/pts_time:/ {t=$2} END {printf "%.2f", t+0}')
        end=$(awk -v a="$last" -v b="$dwell" 'BEGIN {printf "%.2f", a+b}')
        # Only trim if it actually removes something.
        if awk -v e="$end" -v w="$was" 'BEGIN {exit !(e < w)}'; then
            cut="-t $end"
        fi
    fi

    # shellcheck disable=SC2086
    ffmpeg -v error $cut -i "$gif" \
        -vf "setpts=PTS/$speed,fps=$fps,scale=$width:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=$colors[p];[s1][p]paletteuse=dither=bayer:bayer_scale=3" \
        -y "$gif.opt.gif"
    mv "$gif.opt.gif" "$gif"

    after=$(stat -c %s "$gif")
    now=$(ffprobe -v error -show_entries format=duration -of csv=p=0 "$gif")
    printf '%s  %d KB -> %d KB  %.0fs -> %.0fs at %sx\n' \
        "$gif" $((before / 1024)) $((after / 1024)) "$was" "$now" "$speed"
done
