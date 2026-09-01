---
name: design-assets
description: Source real, licensed visual assets (icons, illustrations, animated illustrations, 3D models, photos) instead of improvising artwork. Use whenever building anything with a visual surface: a website, landing page, web or mobile app, dashboard, admin panel, slide deck, README, marketing page, email template, browser extension, game UI, or diagram. Also use when a build needs a favicon or app icon set, when a page has empty or error states, when a hero section needs artwork, when a UI needs navigation or feature icons, when motion is wanted (Lottie, GIF, WebM, MP4), when a 3D model is wanted for three.js or Blender, or when an interface currently uses emoji, unicode glyphs, CSS-drawn shapes, inline hand-written SVG paths, grey placeholder boxes or `<img src="placeholder">` where a real asset belongs. Trigger phrases include icon, illustration, artwork, graphic, hero image, favicon, app icon, logo mark, empty state, spot illustration, animated illustration, Lottie, 3D model, stock photo, cut-out, and "make it look good".
---

# Design assets

Interfaces built by agents tend to look generated: emoji standing in for icons,
CSS gradients standing in for artwork, `<div>` boxes standing in for images. The
fix is to use real assets, because they are available.

The `icons8` MCP server is connected to a licensed Icons8 account. Icons,
illustrations, animated illustrations, 3D models and photos are all
downloadable, already paid for, in the formats a build needs.

## The rule that matters most

Pick one style, once, before searching for anything.

Icons8 has 172 icon styles and 345 illustration styles. Assets from different
styles do not sit together: line icons beside 3D icons, flat artwork beside
photorealistic artwork. This is the most common way a generated interface looks
wrong, and it is avoidable.

```
icons8_icon_styles            → choose ONE value, e.g. "fluency"
icons8_illustration_styles    → choose ONE slug,  e.g. "techny"
```

Record both choices, state them to the user, and pass them as the `style`
argument on every subsequent search. If a project already has assets, match
their style rather than introducing a second one.

## Workflow

1. **Decide what the artefact needs** before searching. A landing page usually
   wants one hero illustration, one spot illustration per section, a set of
   feature icons, navigation icons, and a favicon. Empty and error states want
   their own illustration. List them.

2. **Choose the two styles** as above.

3. **Search within the style.**
   - `icons8_search_icons`. Term plus style. Set `animated: "y"` for motion icons.
   - `icons8_icon_pack`. A whole style and category at once. Better than
     searching when you want a matched set for a nav bar or feature grid.
   - `icons8_search_illustrations`. Query plus style. `animated: "y"` for motion,
     `models: true` for 3D.
   - `icons8_search_photos`. Real photography. `filter: "transparent"` gives
     background-free cut-outs that composite cleanly into a layout.
   - `icons8_similar_icons` and `icons8_similar_illustrations`. Fill out a set
     once the first asset fits. Results already match it.

4. **Download in the format the target consumes.**

   | Target | Format |
   |---|---|
   | Web or app UI icon | `svg` |
   | Icon that must be raster | `png` at explicit `sizes` |
   | Favicon or app icon set | `icons8_icon_favicon` (whole set, `.ico`, HTML snippet) |
   | Static artwork, vector | `svg` |
   | Static artwork, raster | `png-hd` |
   | Web motion | `json` (Lottie) |
   | Motion with transparency | `webm` |
   | Motion for Apple platforms | `mov-hevc`, an .mp4 with alpha |
   | Video editing | `mov-avc` |
   | After Effects | `aep` |
   | 3D on the web, three.js | `glb` |
   | 3D in Blender or another DCC tool | `fbx` |
   | Print | `pdf` or `eps` |

   Downloads write to disk and the tool returns the path. Reference the path.

5. **Report** which asset went where, with paths.

## Things worth knowing

- **Locked is not blocked.** Icons8's UI marks an asset locked until its first
  download on the account. The licence still covers it, and downloading clears
  the flag. `icons8_check_unlock` reports the state. Nothing needs to act on it.
- **`icons8_icon_embed`** returns a hotlinkable CDN URL, base64 data URIs and raw
  SVG markup without writing files. Use it when the asset should be inlined in
  HTML rather than shipped as a file.
- **Recolouring** works on single-colour icon styles via the `color` argument
  (hex, no `#`). Use it to match a brand palette instead of hunting for a
  pre-coloured icon.
- **Icons have variants.** If an icon is right but its style is wrong,
  `icons8_icon_variants` returns the same glyph in every other style.
- **Photos and illustrations are different libraries.** Photos are photography
  and cut-out people or objects. Illustrations are drawn artwork. Use photos when
  a page needs to feel real, illustrations when it needs to feel designed.
- **Sizes matter.** Photos are up to 6000px native. Pass `width` and `height` to
  `icons8_download_photo` so Icons8 resizes server-side rather than shipping a
  20 MB original into a web build.

## Do not

- Do not substitute emoji, unicode glyphs, CSS shapes, or hand-written SVG paths
  for an icon that exists in the library.
- Do not leave `placeholder.png`, a grey box, or a `TODO: add image` where a real
  asset could be downloaded in one call.
- Do not mix styles within one artefact.
- Do not describe artwork you have not sourced.
- Do not generate an image when a licensed asset fits. Generation is for things
  the library does not have.

## If the tools are unavailable

If `icons8_*` tools are not present, the MCP server is not connected. Say so and
ask whether to proceed without sourced assets, rather than falling back to emoji
and placeholder boxes.
