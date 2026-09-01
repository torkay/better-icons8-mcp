<div align="center">

# icons8-mcp-server

**MCP server for the full Icons8 library: icons, illustrations, animated illustrations, 3D models and photos, in every format Icons8 serves.**

[![CI](https://github.com/torkay/icons8-mcp-server/actions/workflows/ci.yml/badge.svg)](https://github.com/torkay/icons8-mcp-server/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/torkay/icons8-mcp-server.svg)](https://pkg.go.dev/github.com/torkay/icons8-mcp-server)
[![Go Report Card](https://goreportcard.com/badge/github.com/torkay/icons8-mcp-server)](https://goreportcard.com/report/github.com/torkay/icons8-mcp-server)
[![Release](https://img.shields.io/github/v/release/torkay/icons8-mcp-server?color=674EFF)](https://github.com/torkay/icons8-mcp-server/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

</div>

It authenticates with a browser session from an Icons8 account you already have. Any MCP host (Claude Code, Cursor, Windsurf, VS Code agent mode) can then search 172 icon styles and 345 illustration styles, and write SVG, Lottie, WebM, MP4, FBX, GLB and photo files to disk.

The purpose is to give a coding agent licensed artwork to use, in place of emoji, CSS shapes and placeholder boxes.

![Installing icons8-mcp-server with go install, importing a browser cookie dump, and registering the server with Claude Code](demo/quickstart.gif)

## How this differs from the official Icons8 MCP server

Icons8 publishes [`icons8/icons8-mcp`](https://github.com/icons8/icons8-mcp). It is hosted, needs no setup, and serves icons: 368,865 of them across 116 design styles. It does not serve illustrations, animated illustrations, 3D models or photos. Its free tier returns 100x100 PNG previews that require attribution. Production SVG requires an API key from a paid Icons plan.

Icons8 access through the [GitHub Student Developer Pack](https://education.github.com/pack) covers "downloads of all asset types with no limits on quantity, size, or format". [Icons8 states](https://intercom.help/icons8-7fb7577e8170/en/articles/4729193-do-you-have-discounts-for-students) that it "does not include API access or the MCP server". Those require a separate Icons subscription. The licence permits downloading every asset in the catalogue by hand, and gives tooling no way to reach any of it.

This server uses the same web client the licence already covers. What an agent can reach is what you could have clicked.

|  | Official `icons8/icons8-mcp` | This server |
|---|---|---|
| Icons | Yes | Yes |
| Illustrations, animated, 3D, photos | No | Yes |
| Auth | API key from a paid Icons plan | Existing browser session |
| Runs | Hosted HTTP | Local binary over stdio |
| SVG | Paid tier | Whatever the plan covers |
| Tools | Icon search and fetch | 18, including favicon sets and embeds |

> [!IMPORTANT]
> This is a client for an account you already have. It does not unlock anything a plan does not cover and does not bypass payment. Every request is authenticated as you and carries the same licence terms as clicking Download in the browser. See [Icons8's licensing](https://icons8.com/license).

## Quick start

Requires Go 1.27 or later and an Icons8 account.

```sh
go install github.com/torkay/icons8-mcp-server/cmd/icons8-mcp@latest
```

Export cookies for `icons8.com` while signed in. Any "export cookies as JSON" browser extension produces the right format. The dump must contain the `i8token` cookie. [`demo/cookies.example.json`](demo/cookies.example.json) shows the expected shape.

```sh
icons8-mcp -import ~/Downloads/cookies.json
icons8-mcp -check
```

```
account:  you@example.com
licence:  icons=true vectors=true photos=true sounds=true
token:    valid until 2026-09-11 20:30 AEST
assets:   ~/.icons8-mcp/assets
```

Register it with an MCP host:

```sh
claude mcp add icons8 -s user -- icons8-mcp
```

<details>
<summary>Other MCP hosts (Cursor, Windsurf, VS Code, Claude Desktop)</summary>

```json
{
  "mcpServers": {
    "icons8": {
      "command": "icons8-mcp"
    }
  }
}
```

</details>

Downloads land in `~/.icons8-mcp/assets/{icons,illustrations,models3d,photos}/`. Download tools return the path they wrote, never the bytes. A 2 MB base64 PNG in a tool result is unusable context.

## Tools

`icons8-mcp -tools` prints the live list. It starts the server against an in-process client, so the output is the actual registration.

![Listing all 18 registered MCP tools](demo/tools.gif)

**Icons**

| Tool | Does |
|---|---|
| `icons8_search_icons` | Search by term, filtered by style, category, author or animation |
| `icons8_icon_styles` | 172 styles and their categories, with the family each belongs to |
| `icons8_icon_pack` | A whole style and category set, already visually matched |
| `icons8_icon_variants` | The same glyph in every other style |
| `icons8_similar_icons` | Visually similar icons, by vector search |
| `icons8_download_icon` | svg, png, pdf, eps, jpg, webp. Also gif, apng and Lottie json when animated. Multi-size, recolourable, simplified SVG |
| `icons8_icon_favicon` | A favicon or app-icon set for favicon/web/ios/android/macos/windows, a multi-resolution `.ico`, and the HTML snippet |
| `icons8_icon_embed` | CDN link, base64 PNG and SVG data URIs, raw markup. Writes no files |
| `icons8_check_unlock` | Which icons this account has already downloaded |

**Illustrations, animation and 3D**

| Tool | Does |
|---|---|
| `icons8_search_illustrations` | Illustrations, animated illustrations, or 3D models with `models`. Filters: style, category, mood, technique, colour |
| `icons8_illustration_styles` | 345 styles with item counts and which ones animate |
| `icons8_illustration` | Detail for one item, including which formats it has |
| `icons8_similar_illustrations` | Items in the same visual family |
| `icons8_download_illustration` | png-hd, png, png-low, svg, gif, gif-low, Lottie json, webm, mov-avc, mov-hevc (mp4), aep, fbx, glb |

**Photos**

| Tool | Does |
|---|---|
| `icons8_search_photos` | Photography and transparent cut-outs |
| `icons8_photo_suggest` | Search terms that return results |
| `icons8_download_photo` | Native resolution, or resized server-side |

**Session**

| Tool | Does |
|---|---|
| `icons8_account` | Identity, licence coverage, token expiry, asset directory |

One prompt is registered, `icons8_asset_plan`. It walks an agent through choosing a style and sourcing the assets an artefact needs.

### Which format for which target

| Target | Format |
|---|---|
| Web or app UI icon | `svg` |
| Icon that must be raster | `png` at explicit sizes |
| Favicon or app-icon set | `icons8_icon_favicon` |
| Static artwork | `svg`, or `png-hd` for raster |
| Web motion | `json` (Lottie) |
| Motion with transparency | `webm` |
| Motion on Apple platforms | `mov-hevc`, an mp4 with alpha |
| Video editing | `mov-avc` |
| After Effects | `aep` |
| 3D on the web, three.js | `glb` |
| 3D in Blender or another DCC tool | `fbx` |
| Print | `pdf` or `eps` |

## How it works

`icons8.com` serves its HTML behind a Cloudflare managed challenge. Its API subdomains are not challenged. That is what makes a plain HTTP client viable.

- **Tool calls are plain HTTP.** Requests go to `search-app`, `api-icons`, `api-img`, `api-ouch` and `photos`, rate-limited, carrying the session JWT and a stable browser fingerprint. No browser process, no page loads, no captcha solver.
- **The session renews itself.** `GET /user/v2` returns a freshly minted JWT on every call. A background loop keeps a 10-day token alive from one cookie dump. Cookies are imported once.
- **A headless browser is the fallback.** If a request is rejected as unauthorized and the cheap refresh does not fix it, [go-rod](https://github.com/go-rod/rod) with [`go-rod/stealth`](https://github.com/go-rod/stealth) drives a real Chromium through the Cloudflare challenge and harvests the resulting session. Measured at about 14 seconds by `go run ./cmd/reauthcheck`. The same tooling mapped the API in the first place.

> [!NOTE]
> Limit worth knowing: the only credential in a cookie dump is `i8token`. If it lapses completely the browser loads the site logged out and cannot recover. The server reports this and asks for a fresh dump. The rolling refresh is intended to prevent it.

`docs/api.md` holds the endpoint map. Read it before changing a query. Several Icons8 parameters fail silently: a wrong parameter name returns HTTP 200 with unfiltered results instead of an error. Illustration filters are split across two mechanisms. `style_pretty_id` and `animated` are query parameters. `mood`, `technique` and `colors` belong inside a `meta` JSON blob. Sending one in the other's place is ignored.

## The skill

[`skills/design-assets/SKILL.md`](skills/design-assets/SKILL.md) is the part that changes agent behaviour. Connecting a server does not stop a model improvising artwork. An instruction to treat assets as part of the plan does.

Its main rule is to pick one icon style and one illustration style before searching. There are 172 and 345 of them. Mixing styles is the most common reason a generated interface looks assembled rather than designed. The rest of the file is a format table, a note that the "locked" badge is bookkeeping rather than a restriction, and a list of substitutions to avoid.

```sh
mkdir -p ~/.claude/skills && cp -r skills/design-assets ~/.claude/skills/
```

## Configuration

Every setting is an environment variable with a working default.

| Variable | Default | Meaning |
|---|---|---|
| `ICONS8_MCP_HOME` | `~/.icons8-mcp` | State directory |
| `ICONS8_ASSET_DIR` | `$ICONS8_MCP_HOME/assets` | Where downloads land |
| `ICONS8_COOKIE_FILE` | `$ICONS8_MCP_HOME/cookies.json` | Session bootstrap |
| `ICONS8_RPS` | `6` | Requests per second ceiling |
| `ICONS8_CONCURRENCY` | `4` | Parallel requests |
| `ICONS8_REFRESH_INTERVAL` | `6h` | Rolling JWT refresh |
| `ICONS8_BROWSER_FALLBACK` | `1` | `0` disables headless Chrome |
| `ICONS8_HEADFUL` | unset | `1` shows the fallback browser |
| `ICONS8_LOCALE` | `en-US` | Search language |

## Development

```sh
go test ./...                              # offline unit tests
go run ./cmd/smoke -bin ./dist/icons8-mcp  # 29 checks against the live API
go run ./cmd/reauthcheck                   # exercise headless-browser recovery
```

The smoke suite drives the built binary over stdio the way an MCP host does. Later checks reuse ids and styles taken from earlier results rather than hard-coded fixtures, so it fails when the API changes shape.

![Running the search half of the smoke suite against the live Icons8 API](demo/live.gif)

`cmd/recon` and `cmd/reconflow` are the discovery tools that produced `docs/api.md`. `reconflow` shims `fetch`, `XMLHttpRequest`, `URL.createObjectURL` and `HTMLAnchorElement.click` inside the page, because CDP's Network domain misses requests that finish as browser-level downloads. That is how the SVG download endpoint was found.

Demo GIFs are recorded with [VHS](https://github.com/charmbracelet/vhs): `make demo`.

## Acknowledgements

Built on the [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk), [go-rod](https://github.com/go-rod/rod) and [VHS](https://github.com/charmbracelet/vhs). Not affiliated with or endorsed by Icons8.
