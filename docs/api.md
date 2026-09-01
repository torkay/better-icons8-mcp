# The Icons8 private API

Mapped by recording the web app's own network traffic (`cmd/recon`,
`cmd/reconflow`) and reading its JS bundles. Everything here is verified against
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

## Illustrations and 3D models (Ouch)

Both live in one collection. `model=true` selects the 3D catalogue.

| Purpose | Endpoint |
|---|---|
| Search | `GET search-ouch-origin.icons8.com/api/illustrations/ouch/search?locale=&page=&per_page=&search=&meta=&style_pretty_id=&category_pretty_ids=&animated=true&model=true` |
| Browse (no term) | `GET api-ouch.icons8.com/api/frontend/v1/illustrations/watermarkless?locale=&page=&per_page=&meta=&style_pretty_id=&model=true` |
| Count | `GET search-ouch-origin.icons8.com/api/illustrations/count/v2?meta=&search=` |
| Facets | `GET search-ouch-origin.icons8.com/api/illustrations/ouch/filters?key_pretty_ids=mood,colors,technique&locale=&meta=&search=` |
| Styles | `GET api-ouch.icons8.com/api/frontend/v1/illustrations/styles?fields=title,pretty_id,icon,thumb1x,generator,free_distribution,backgroundColor` |
| Detail | `GET api-ouch.icons8.com/api/frontend/v1/illustrations/{id}?locale=` |
| Similar | `GET api-ouch.icons8.com/api/frontend/v1/illustrations/{id}/similars/watermarkless?page=&per_page=&style_pretty_id=&locale=` |
| Download | `GET api-ouch.icons8.com/api/frontend/v1/illustrations/{id}/download-url?media_format=` returns a signed R2 URL |
| Billing check | `GET api-icons.icons8.com/billing/v1/resources/illustration/download/info?id=&format=` |

Gotchas:

- The filters are split across two mechanisms. `style_pretty_id`,
  `category_pretty_ids` and `animated` are ordinary query parameters. `mood`,
  `technique` and `colors` go inside `meta` as a JSON object of arrays:
  `meta={"mood":["casual"],"technique":["3d"]}`. Sending one in the other's
  place is ignored rather than rejected. The request succeeds and returns
  unfiltered results.
- `animated=true` is a boolean. `animated=y` is ignored. There is no server-side
  static-only filter, so exclude those client-side.
- The download parameter is `media_format`, not `format`. A wrong name returns
  200 with the default (`png-low`) rather than an error.
- `media_format` values: `png-hd png png-low svg gif gif-low json webm mov-avc
  mov-hevc aep fbx-zip glb sources`. `json` is Lottie. `mov-hevc` is an `.mp4`.
- FBX is advertised as `fbx` in `downloadable_resources.available` but must be
  requested as `fbx-zip`. `icons8.MediaFormat` maps this.
- Search results carry the motion asset fields (`webm`, `json`, `gif_low`,
  `mov_hevc`) and omit `downloadable_resources`. The detail endpoint is the
  reverse. Detecting whether an item animates needs both.

## Photos (Moose)

| Purpose | Endpoint |
|---|---|
| Search | `GET photos.icons8.com/api/frontend/v1/images?query=&page=&per_page=&fields=&filter=all&sort_by=rising&locale=&category_id=&tag_id=&background=&type=` |
| Counts by kind | `GET photos.icons8.com/api/frontend/v1/images/count?query=` |
| Detail | `GET photos.icons8.com/api/frontend/v1/images/{id}?fields=` |
| Download | `GET photos.icons8.com/api/frontend/v1/images/{id}/download-url?width=&height=` returns a signed URL |
| Categories | `GET photos.icons8.com/api/frontend/v1/categories` |
| Autocomplete | `GET photos.icons8.com/api/frontend/v1/autocomplete?query=&limit=&locale=` |
| Account | `GET photos.icons8.com/api/frontend/v1/user/get_info` |

Gotchas:

- `fields` is a required-in-practice projection. Omitting it returns a thin
  object. See `photoListFields` in `internal/icons8/photos.go`.
- `download-url` requires both `width` and `height`. It 400s with
  `width is missing, height is missing` otherwise. The photo's native dimensions
  yield the original. Smaller values resize server-side.
- `filter=transparent` gives background-free cut-outs.

## Signed CDN URLs

`download-url` responses point at Cloudflare R2 or an imgproxy host with an
AWS4-HMAC signature. Fetch them without the session headers. The signature is
the authorisation, and some of these hosts reject a request that also carries an
`Authorization` header. `Client.FetchSigned` handles this.

## Re-running the discovery

```sh
go build ./cmd/recon ./cmd/reconflow

# Record every API call a page makes
./recon -cookies cookies.json -out recon.jsonl -pages icons,icon-detail,illustrations

# Record one interaction with full headers and response bodies
./reconflow -url https://icons8.com/icon/999/rocket \
  -click 'sel:button.btn-download,text:SVG,xy:292x649' \
  -js '() => window.__i8log'
```

`reconflow` shims `fetch`, `XMLHttpRequest`, `URL.createObjectURL` and
`HTMLAnchorElement.click` inside the page. CDP's Network domain misses fetches
that finish as browser-level downloads, which is how the SVG download request
was found.
