# The Icons8 private API

Mapped by recording the web app's own network traffic (`cmd/recon`)
and reading its JS bundles. Everything here is verified against
the live service.

## Auth

Three things travel with a request.

| What | Where | Notes |
|---|---|---|
| JWT | `Authorization: Bearer <i8token>` | From the `i8token` cookie. About 10 days of life. |
| Fingerprint | `X-Icons8-Fingerprint: <32 hex>` | Per-browser id. Any stable random value works. |
| Public API key | `token=` query param | Download hosts only. Read from the JWT's `publicApiKey`. |

Requests also need `Origin: https://icons8.com` and `Referer: https://icons8.com/`.

### Rolling refresh

```
GET https://api-icons.icons8.com/user/v2
```

Returns the account and a freshly minted JWT in `.token`. Refreshing before
expiry keeps the session alive. The browser is needed only to bootstrap, or to
recover after the JWT has fully lapsed.

### Cloudflare

`icons8.com` HTML sits behind a Cloudflare managed challenge (`cf-mitigated:
challenge`), so `curl` gets a 403. The API subdomains are not challenged, which
is why the fast path is plain HTTP. Only the browser fallback touches the gated
HTML. go-rod with stealth clears the challenge without a solver.

