#!/bin/bash
# Build the GitHub social preview card (1280x640), matching the demo GIFs:
# VHS purple field, rounded dark terminal card, monospace type.
set -euo pipefail
OUT=demo/social-preview.png
MONO=/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf
BOLD=/usr/share/fonts/truetype/dejavu/DejaVuSansMono-Bold.ttf

convert -size 1280x640 xc:'#674EFF' \
  -fill '#1B1B29' -stroke none -draw 'roundrectangle 40,40 1239,599 18,18' \
  -fill '#3A3A55' -draw 'circle 84,84 84,90' \
  -fill '#3A3A55' -draw 'circle 112,84 112,90' \
  -fill '#3A3A55' -draw 'circle 140,84 140,90' \
  -font "$BOLD" -pointsize 58 -fill '#F5F5FA' -annotate +90+210 'better-icons8-mcp' \
  -font "$MONO" -pointsize 26 -fill '#A9A9C4' -annotate +90+266 'MCP server for the whole Icons8 library' \
  -font "$MONO" -pointsize 25 -fill '#9C8CFF' -annotate +90+350 'icons   illustrations   animated   3D models   photos' \
  -font "$MONO" -pointsize 22 -fill '#77778F' -annotate +90+396 'svg  png  pdf  eps  lottie  webm  mp4  gif  aep  fbx  glb' \
  -font "$MONO" -pointsize 22 -fill '#6E6E88' -annotate +90+500 '/plugin install icons8@better-icons8-mcp' \
  -font "$MONO" -pointsize 22 -fill '#5EC98A' -annotate +90+548 '19 tools    Go    stdio    MIT' \
  "$OUT"

identify "$OUT"
