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

## Icons

| Purpose | Endpoint |
|---|---|
| Search | `GET search-app.icons8.com/api/iconsets/v7/search?term=&amount=&offset=&language=&platform=&category=&authorApiCode=&animated=&isOuch=true&replaceNameWithSynonyms=true` |
| Style and category filter tree | `GET api-icons.icons8.com/siteApi/filters/v1/available?term=&lang=` |
| Style families | `GET api-icons.icons8.com/siteApi/groups/v1/platform/variations` |
| Variants of one glyph | `GET api-icons.icons8.com/siteApi/icons/icon/{id}/variants?language=` |
| Style and category pack | `GET api-icons.icons8.com/siteApi/icons/v1/packs/demarcation?amount=&offset=&style=&category=&sortBy=&language=` |
| Visually similar | `GET search-app.icons8.com/api/iconsets/vector/search/id?id=&limit=&language=` |
| Popular terms | `GET search-app.icons8.com/api/iconsets/popularRequests?limit=&lang=` |
| Unlock state | `GET api-icons.icons8.com/user/v1/paidDownloadIds?type=iconDownload&objectIds=a,b,c` |
| Download | `GET api-img.icons8.com/?id=&format=&size=&name=&fromSite=true&token=<publicApiKey>[&color=&simplified=]` |
| Free preview PNG | `GET img.icons8.com/?id=&size=&format=png` (no auth, no download credit) |

Gotchas:

- `packs/demarcation` nests its results under `category.icons`, not `icons`, and
  400s when `category` is omitted. Its items also omit `platform`.
- `vector/search/id` returns `category`, `categoryApiCode` and `subcategory` as
  arrays. The search endpoint returns the same fields as strings.
  `icons8.StringList` accepts both.
- Download formats: `png svg pdf eps jpg webp`, plus `gif json apng` for icons
  with `isAnimated: true`. `json` is Lottie. Anything else 400s with
  `format must be a valid enum value`.
- `ico` is not supported. Favicons are assembled locally in
  `internal/assets/ico.go`.
- `format=svg` without the `token=` parameter returns `{"code":"PAID_FORMAT"}`.
  Without a valid session it returns `RESOURCE_CONSUMING_NOT_ALLOWED`. A
  successful call to this endpoint is the unlock. The icon then appears in
  `paidDownloadIds`.

