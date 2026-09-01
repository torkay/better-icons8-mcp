# Security

## What this program stores

An Icons8 session: the `i8token` JWT and the cookie dump it came from. Both live
under `$ICONS8_MCP_HOME`, which defaults to `~/.icons8-mcp`. Files are written
`0600` and the directory `0700`. Nothing is sent anywhere except `icons8.com`
and its API subdomains.

The token is a live credential for the account. Treat the state directory the
way you would treat `~/.ssh`. Never paste a cookie dump into an issue, a pull
request, or a screenshot.

## Rotating a session

Signing out of Icons8 in the browser invalidates the token. To replace one:

```sh
rm -f ~/.icons8-mcp/session.json
icons8-mcp -import /path/to/fresh-cookies.json
```

## Reporting a vulnerability

Open a [private security advisory](https://github.com/torkay/icons8-mcp-server/security/advisories/new)
rather than a public issue. Expect confirmation of receipt within a few days.

Include what an attacker can do, what access they need first, and a minimal
reproduction. Do not include a real session token. Describe the shape of the
credential instead.
