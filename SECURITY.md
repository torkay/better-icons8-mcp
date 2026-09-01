# Security

## What this program stores

An Icons8 session: the `i8token` JWT and the cookie dump it came from. Both live
under `$ICONS8_MCP_HOME`, which defaults to `~/.icons8-mcp`. Files are written
`0600` and the directory `0700`. Nothing is sent anywhere except `icons8.com`
and its API subdomains.

The token is a live credential for the account. Treat the state directory the
way you would treat `~/.ssh`. Never paste a cookie dump into an issue, a pull
request, or a screenshot.

The session is stored under `$ICONS8_MCP_HOME`, not in the plugin's data
directory, so uninstalling or updating the plugin does not scatter copies of it.

## Downloading the server binary

The plugin ships a launcher rather than five prebuilt binaries. On first run it
fetches the release archive for the platform over HTTPS and checks it against
`checksums.txt` from the same release before executing it. A mismatch is fatal.
Set `ICONS8_MCP_BIN` to a binary you built yourself to skip the download.

## Rotating a session

Signing out of Icons8 in the browser invalidates the token. To replace one:

```sh
rm -f ~/.icons8-mcp/session.json
icons8-mcp auth
```

## Reporting a vulnerability

Open a [private security advisory](https://github.com/torkay/better-icons8-mcp/security/advisories/new)
rather than a public issue. Expect confirmation of receipt within a few days.

Include what an attacker can do, what access they need first, and a minimal
reproduction. Do not include a real session token. Describe the shape of the
credential instead.
