<div align="center">

# better-icons8-mcp

**MCP server for the full Icons8 library: icons, illustrations, animated illustrations, 3D models and photos, in every format Icons8 serves, without an API key.**

[![CI](https://github.com/torkay/better-icons8-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/torkay/better-icons8-mcp/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/torkay/better-icons8-mcp.svg)](https://pkg.go.dev/github.com/torkay/better-icons8-mcp)
[![Go Report Card](https://goreportcard.com/badge/github.com/torkay/better-icons8-mcp)](https://goreportcard.com/report/github.com/torkay/better-icons8-mcp)
[![Release](https://img.shields.io/github/v/release/torkay/better-icons8-mcp?color=674EFF)](https://github.com/torkay/better-icons8-mcp/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

</div>

Works with any MCP host: Claude Code, Codex, Cursor, Windsurf, Copilot, Gemini CLI, Claude Desktop. It signs in to your existing Icons8 membership in a browser window and uses that session, so the free [GitHub Student Developer Pack](https://education.github.com/pack) plan is enough. Icons8's own MCP server serves icons only, and its SVGs need a separate $15/month subscription.

![Installing the plugin in Claude Code and signing in to Icons8](demo/quickstart.gif)

## Install

### Claude Code

```
/plugin marketplace add https://github.com/torkay/better-icons8-mcp.git
/plugin install icons8@better-icons8-mcp
```

Then prompt:

```
Connect to my icons8 account using the MCP.
```

That calls `icons8_authorize`, which opens a browser window on the Icons8 sign-in page and stores the session on your machine. It is the only setup step.

The plugin fetches the server binary for your platform on first run and checks it against the published sha256, so Go is not required. It also installs the `design-assets` skill.

> [!NOTE]
> Use the full git URL. The `owner/repo` shorthand clones over SSH first and prints a fallback notice when no key is configured.

### Any other host

Install the binary and sign in:

```sh
curl -fsSL https://raw.githubusercontent.com/torkay/better-icons8-mcp/main/scripts/install.sh | sh
icons8-mcp auth
```

It lands in `~/.local/bin`. Set `ICONS8_INSTALL_DIR` for somewhere else, or run `go install github.com/torkay/better-icons8-mcp/cmd/icons8-mcp@latest` to build it yourself.

Then register it.

Codex:

```sh
codex mcp add icons8 -- icons8-mcp
```

Gemini CLI:

```sh
gemini mcp add icons8 icons8-mcp
```

VS Code and Copilot:

```sh
code --add-mcp '{"name":"icons8","command":"icons8-mcp"}'
```

Claude Code without the plugin:

```sh
claude mcp add icons8 -s user -- icons8-mcp
```

Cursor writes `~/.cursor/mcp.json`, Windsurf `~/.codeium/windsurf/mcp_config.json`, Claude Desktop its own config file. All three take the same shape:

```json
{
  "mcpServers": {
    "icons8": {
      "command": "icons8-mcp"
    }
  }
}
```

`icons8-mcp` with no arguments is the MCP server speaking stdio, which is what a host runs. The subcommands are for the parts a person does by hand: `auth`, `status`, `tools`, and `import <file>` for machines with no display.

Downloads land in `~/.icons8-mcp/assets/{icons,illustrations,models3d,photos}/`. Download tools return the path they wrote, never the bytes. A 2 MB base64 PNG in a tool result is unusable context.

## Why not use the Official MCP by Icons8?

It serves icons only, and SVG needs a paid key.

`icons8/icons8-mcp` is a hosted HTTP endpoint with two tools: icon search and icon fetch. Illustrations, animated illustrations, 3D models and photos are not behind it at any price. Without a key it returns PNG previews that require attribution. Production SVG needs an API key from the [$15/month Icons plan](https://icons8.com/icons/pricing), which covers 100 downloads a month and then charges $0.20 an icon.

That subscription is the problem for anyone whose access came from the GitHub Student Developer Pack. The pack covers "downloads of all asset types with no limits on quantity, size, or format", and [Icons8 states](https://intercom.help/icons8-7fb7577e8170/en/articles/4729193-do-you-have-discounts-for-students) that it "does not include API access or the MCP server". The licence already permits every download; what is withheld is the machine-readable path to it.

|  | icons8-mcp@icons8 | better-icons8-mcp |
|---|---|---|
| Icons | ✅ | ✅ |
| Illustrations | ❌ | ✅ |
| Animated assets | ❌ | ✅ |
| 3D assets | ❌ | ✅ |
| Photos | ❌ | ✅ |
| Icon styles | 116 | 172 |
| Auth | Paid API key | Free student subscription |
| Runs | Hosted HTTP | Local binary over stdio |
| SVG | Paid tier | Whatever your plan covers |
| Tools | 2 | 19 |

> [!IMPORTANT]
> This is a client for an account you already have. It does not unlock anything a plan does not cover and does not bypass payment. Every request is authenticated as you and carries the same licence terms as clicking Download in the browser. See [Icons8's licensing](https://icons8.com/license).

## Tools

![Claude Code searching Icons8 and downloading matched assets](demo/usage.gif)

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
| `icons8_authorize` | Open the sign-in window and store the session. Run once |
| `icons8_account` | Identity, licence coverage, token expiry, asset directory |

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

## Style discipline

Connecting a server does not stop a model improvising artwork, and it does not stop it mixing styles. There are 172 icon styles and 345 illustration styles. Choosing one of each before searching, once, for the whole artefact, is the difference between an interface that looks designed and one that looks assembled.

Three things carry that rule, so it survives outside Claude Code:

- The server's MCP instructions, which any host reads on connect.
- `icons8_asset_plan`, a registered prompt that walks an agent through picking a style and sourcing what the artefact needs.
- [`design-assets`](plugins/icons8/skills/design-assets/SKILL.md), a skill the plugin installs. It adds the format table, a note that Icons8's "locked" badge is bookkeeping rather than a restriction, and a list of substitutions to avoid.

## How it works

`icons8.com` serves its HTML behind a Cloudflare managed challenge. Its API subdomains are not challenged. That is what makes a plain HTTP client viable.

- **Tool calls are plain HTTP.** Requests go to `search-app`, `api-icons`, `api-img`, `api-ouch` and `photos`, rate-limited, carrying the session JWT and a stable browser fingerprint. No browser process, no page loads, no captcha solver.
- **Sign-in is a real browser, once.** `icons8_authorize` opens a visible Chromium at the Icons8 login page and waits for the `i8token` cookie to appear with readable claims. That is the one signal common to email, Google, Apple and GitHub sign-in, which do not share a completion event.
- **The session renews itself.** `GET /user/v2` returns a freshly minted JWT on every call. A background loop keeps a 10-day token alive, so signing in is a one-off.
- **A headless browser handles recovery.** If a request is rejected as unauthorized and the cheap refresh does not fix it, [go-rod](https://github.com/go-rod/rod) with [`go-rod/stealth`](https://github.com/go-rod/stealth) drives a real Chromium through the Cloudflare challenge and harvests the resulting session. Measured at about 14 seconds by `go run ./cmd/reauthcheck`.

> [!NOTE]
> On a machine with no display, the sign-in window cannot open. Export cookies for `icons8.com` and run `icons8-mcp import <file>` instead. [`demo/cookies.example.json`](demo/cookies.example.json) shows the expected shape.

`docs/api.md` holds the endpoint map. Read it before changing a query. Several Icons8 parameters fail silently: a wrong parameter name returns HTTP 200 with unfiltered results instead of an error. Illustration filters are split across two mechanisms. `style_pretty_id` and `animated` are query parameters. `mood`, `technique` and `colors` belong inside a `meta` JSON blob. Sending one in the other's place is ignored.

## Configuration

Every setting is an environment variable with a working default.

| Variable | Default | Meaning |
|---|---|---|
| `ICONS8_MCP_HOME` | `~/.icons8-mcp` | State directory |
| `ICONS8_ASSET_DIR` | `$ICONS8_MCP_HOME/assets` | Where downloads land |
| `ICONS8_COOKIE_FILE` | `$ICONS8_MCP_HOME/cookies.json` | Session bootstrap for `import` |
| `ICONS8_RPS` | `6` | Requests per second ceiling |
| `ICONS8_CONCURRENCY` | `4` | Parallel requests |
| `ICONS8_REFRESH_INTERVAL` | `6h` | Rolling JWT refresh |
| `ICONS8_AUTH_TIMEOUT` | `5m` | How long the sign-in window waits |
| `ICONS8_BROWSER_FALLBACK` | `1` | `0` disables Chrome, including sign-in |
| `ICONS8_HEADFUL` | unset | `1` shows the recovery browser |
| `ICONS8_LOCALE` | `en-US` | Search language |
| `ICONS8_MCP_BIN` | unset | Path to a binary the plugin launcher should use instead of downloading one |

## Development

```sh
go test ./...                              # offline unit tests
go run ./cmd/smoke -bin ./dist/icons8-mcp  # 29 checks against the live API
go run ./cmd/reauthcheck                   # exercise headless-browser recovery
claude --plugin-dir ./plugins/icons8       # load the plugin without installing it
```

The smoke suite drives the built binary over stdio the way an MCP host does. Later checks reuse ids and styles taken from earlier results rather than hard-coded fixtures, so it fails when the API changes shape.

![Running the search half of the smoke suite against the live Icons8 API](demo/live.gif)

`cmd/recon` and `cmd/reconflow` are the discovery tools that produced `docs/api.md`. `reconflow` shims `fetch`, `XMLHttpRequest`, `URL.createObjectURL` and `HTMLAnchorElement.click` inside the page, because CDP's Network domain misses requests that finish as browser-level downloads. That is how the SVG download endpoint was found.

Demo GIFs are recorded with [VHS](https://github.com/charmbracelet/vhs): `make demo`.

## Acknowledgements

Built on the [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk), [go-rod](https://github.com/go-rod/rod) and [VHS](https://github.com/charmbracelet/vhs). Not affiliated with or endorsed by Icons8.
