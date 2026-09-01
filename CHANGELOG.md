# Changelog

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-09-01

Renamed from `icons8-mcp-server` to `better-icons8-mcp`. The Go module path
moved with it, so `go install` needs the new path.

### Added

- Claude Code plugin. `/plugin marketplace add torkay/better-icons8-mcp` then
  `/plugin install icons8@better-icons8-mcp` installs the server and the
  `design-assets` skill together. The bundled launcher fetches the platform
  binary on first run and checks it against a published sha256, so Go is not
  required.
- `icons8_authorize` tool and `icons8-mcp auth`. Both open a browser window at
  the Icons8 sign-in page and store the session, replacing the manual cookie
  export.
- `checksums.txt` on every release.
- `ICONS8_AUTH_TIMEOUT` and `ICONS8_MCP_BIN`.

### Changed

- The server starts without a session instead of refusing to start. An MCP host
  reporting a broken server gives the user no way to fix it. Tools now return
  an actionable message naming `icons8_authorize`, and `icons8_account` reports
  `authorized: false`.
- The CLI takes subcommands (`auth`, `status`, `tools`, `import`, `version`)
  rather than flags. Running it with no arguments is still the MCP server.
- The skill moved to `plugins/icons8/skills/design-assets/`.

## [0.1.1] - 2026-09-01

### Changed

- README now shows recorded terminal demos of install, the tool listing, and a
  live run against the Icons8 API.
- CI and release workflows moved to current major versions of the GitHub
  Actions they use.

## [0.1.0] - 2026-09-01

First public release.

### Added

- MCP server over stdio with 18 tools covering icons, illustrations, animated
  illustrations, 3D models and photos, plus the `icons8_asset_plan` prompt.
- Session bootstrap from a browser cookie dump. A rolling `GET /user/v2`
  refresh keeps the 10-day JWT alive.
- Headless Chromium fallback (go-rod with stealth) for re-authentication when
  the refresh is rejected.
- Local ICO encoding, so `icons8_icon_favicon` returns a complete favicon or
  app-icon set. The Icons8 API rejects `format=ico`.
- `icons8_icon_embed` for CDN links, base64 data URIs and raw SVG markup.
- `-tools` flag listing the registered tools without an MCP client.
- `design-assets` skill, which requires one icon style and one illustration
  style to be chosen before sourcing anything.
- `docs/api.md`, the reverse-engineered endpoint map, including the parameters
  that fail silently.
- `cmd/smoke`, a 29-check live suite driving the built binary over stdio.

[Unreleased]: https://github.com/torkay/better-icons8-mcp/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/torkay/better-icons8-mcp/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/torkay/better-icons8-mcp/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/torkay/better-icons8-mcp/releases/tag/v0.1.0
