package icons8

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Icon is one result from icon search. `Platform` is Icons8's internal name for
// a style (e.g. "ios7", "fluency", "color"); the UI calls it Style.
type Icon struct {
	ID              string `json:"id"`
	CommonID        string `json:"commonId,omitempty"`
	Name            string `json:"name"`
	CommonName      string `json:"commonName"`
	Category        string `json:"category"`
	CategoryAPICode string `json:"categoryApiCode,omitempty"`
	Subcategory     string `json:"subcategory,omitempty"`
	Platform        string `json:"platform"`
	IsColor         bool   `json:"isColor"`
	IsAnimated      bool   `json:"isAnimated,omitempty"`
	IsExplicit      bool   `json:"isExplicit,omitempty"`
	NeedBackground  bool   `json:"needBackground,omitempty"`
	AuthorAPICode   string `json:"authorApiCode,omitempty"`
	IsExternal      bool   `json:"isExternal,omitempty"`
	SourceFormat    string `json:"sourceFormat,omitempty"`
	Free            bool   `json:"free,omitempty"`
}

type IconSearchResult struct {
	Success    bool `json:"success"`
	Parameters struct {
		Amount    int    `json:"amount"`
		CountAll  int    `json:"countAll"`
		Offset    int    `json:"offset"`
		MaxOffset int    `json:"maxOffset"`
		Term      string `json:"term"`
		Language  string `json:"language"`
	} `json:"parameters"`
	Icons []Icon `json:"icons"`
}

// IconSearchOptions mirrors the filters the web UI exposes on /icons.
type IconSearchOptions struct {
	Term     string
	Amount   int
	Offset   int
	Style    string // platform api code, e.g. "fluency", "ios7", "color"
	Category string // categoryApiCode, e.g. "transport"
	Author   string // authorApiCode, e.g. "icons8"
	Animated string // "y" to require animated, "n" to exclude
	Language string
	Exact    bool
}

func (c *Client) SearchIcons(ctx context.Context, o IconSearchOptions) (*IconSearchResult, error) {
	if o.Amount <= 0 {
		o.Amount = 30
	}
	if o.Language == "" {
		o.Language = c.cfg.Language()
	}
	q := map[string]string{
		"term":                    o.Term,
		"amount":                  strconv.Itoa(o.Amount),
		"offset":                  strconv.Itoa(o.Offset),
		"language":                o.Language,
		"isOuch":                  "true",
		"replaceNameWithSynonyms": "true",
		"platform":                o.Style,
		"category":                o.Category,
		"authorApiCode":           o.Author,
		"animated":                o.Animated,
	}
	if o.Exact {
		q["exact_amount"] = "1"
	}
	var out IconSearchResult
	if err := c.GetJSON(ctx, buildURL(HostIconSearch, "/api/iconsets/v7/search", q), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// IconVariant is the same glyph rendered in another style.
type IconVariant struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	URL         string `json:"url"`
	CommonID    string `json:"commonId"`
	CommonName  string `json:"commonName"`
	Category    string `json:"category"`
	Subcategory string `json:"subcategory,omitempty"`
	IsColor     bool   `json:"isColor"`
	IsAnimated  bool   `json:"isAnimated"`
	Free        bool   `json:"free"`
	IsExternal  bool   `json:"isExternal"`
}

func (c *Client) IconVariants(ctx context.Context, id string) ([]IconVariant, error) {
	var out struct {
		Success  bool          `json:"success"`
		Variants []IconVariant `json:"variants"`
	}
	u := buildURL(HostIconAPI, "/siteApi/icons/icon/"+id+"/variants", map[string]string{"language": c.cfg.Locale})
	if err := c.GetJSON(ctx, u, &out); err != nil {
		return nil, err
	}
	return out.Variants, nil
}

// vectorIcon is the shape the vector-similarity endpoint returns. It differs
// from the search endpoint: category, categoryApiCode and subcategory come back
// as arrays there and as plain strings here, so it needs its own type.
type vectorIcon struct {
	ID              string     `json:"id"`
	CommonID        string     `json:"commonId"`
	CommonName      string     `json:"commonName"`
	Name            string     `json:"name"`
	Category        StringList `json:"category"`
	CategoryAPICode StringList `json:"categoryApiCode"`
	Subcategory     StringList `json:"subcategory"`
	Platform        string     `json:"platform"`
	SourceFormat    string     `json:"sourceFormat"`
	IsColor         bool       `json:"isColor"`
	IsAnimated      bool       `json:"isAnimated"`
	IsExplicit      bool       `json:"isExplicit"`
	NeedBackground  bool       `json:"needBackground"`
	AuthorAPICode   string     `json:"authorApiCode"`
	IsExternal      bool       `json:"isExternal"`
}

func (v vectorIcon) toIcon() Icon {
	return Icon{
		ID: v.ID, CommonID: v.CommonID, Name: v.Name, CommonName: v.CommonName,
		Category: v.Category.First(), CategoryAPICode: v.CategoryAPICode.First(),
		Subcategory: v.Subcategory.First(), Platform: v.Platform,
		IsColor: v.IsColor, IsAnimated: v.IsAnimated, IsExplicit: v.IsExplicit,
		NeedBackground: v.NeedBackground, AuthorAPICode: v.AuthorAPICode,
		IsExternal: v.IsExternal, SourceFormat: v.SourceFormat,
	}
}

// SimilarIcons returns visually related icons across styles.
func (c *Client) SimilarIcons(ctx context.Context, id string, limit int) ([]Icon, error) {
	if limit <= 0 {
		limit = 30
	}
	var out struct {
		Icons []vectorIcon `json:"icons"`
	}
	u := buildURL(HostIconSearch, "/api/iconsets/vector/search/id", map[string]string{
		"id": id, "limit": strconv.Itoa(limit), "language": c.cfg.Language(),
	})
	if err := c.GetJSON(ctx, u, &out); err != nil {
		return nil, err
	}
	icons := make([]Icon, 0, len(out.Icons))
	for _, v := range out.Icons {
		icons = append(icons, v.toIcon())
	}
	return icons, nil
}

// FilterOption is one entry in the style/category filter tree the UI renders.
type FilterOption struct {
	Name      string         `json:"name"`
	Value     string         `json:"value"`
	URLValue  string         `json:"urlValue,omitempty"`
	IconURL   string         `json:"iconUrl,omitempty"`
	IsEnabled bool           `json:"isEnabled"`
	Options   []FilterOption `json:"options,omitempty"`
}

type FilterGroup struct {
	Name      string         `json:"name"`
	APICode   string         `json:"apiCode"`
	Type      string         `json:"type"`
	IsEnabled bool           `json:"isEnabled"`
	Options   []FilterOption `json:"options"`
}

// IconFilters returns the full style/category/technique filter tree. Passing a
// term scopes `isEnabled` to what that search actually has.
func (c *Client) IconFilters(ctx context.Context, term string) ([]FilterGroup, error) {
	var out struct {
		Success bool          `json:"success"`
		Docs    []FilterGroup `json:"docs"`
	}
	u := buildURL(HostIconAPI, "/siteApi/filters/v1/available", map[string]string{
		"term": term, "lang": c.cfg.Locale,
	})
	if err := c.GetJSON(ctx, u, &out); err != nil {
		return nil, err
	}
	return out.Docs, nil
}

// StyleGroup ties together styles that are variations of one family, which is
// what makes "keep the whole UI in one style" answerable.
type StyleGroup struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	APICode  string   `json:"apiCode"`
	Entities []string `json:"entities"`
}

func (c *Client) StyleGroups(ctx context.Context) ([]StyleGroup, error) {
	var out struct {
		Success bool         `json:"success"`
		Docs    []StyleGroup `json:"docs"`
	}
	if err := c.GetJSON(ctx, HostIconAPI+"/siteApi/groups/v1/platform/variations", &out); err != nil {
		return nil, err
	}
	return out.Docs, nil
}

// IconPack lists icons within one style + category, which is how you pull a
// coherent set rather than search results scattered across styles.
//
// Both style and category are required: the endpoint 400s without a category.
func (c *Client) IconPack(ctx context.Context, style, category, sortBy string, amount, offset int) (*IconSearchResult, error) {
	if style == "" {
		return nil, fmt.Errorf("style is required")
	}
	if amount <= 0 {
		amount = 50
	}
	if sortBy == "" {
		sortBy = "popular"
	}
	var raw struct {
		Success bool   `json:"success"`
		Icons   []Icon `json:"icons"`
	}
	u := buildURL(HostIconAPI, "/siteApi/icons/v1/packs/demarcation", map[string]string{
		"amount": strconv.Itoa(amount), "offset": strconv.Itoa(offset),
		"style": style, "category": category, "sortBy": sortBy, "language": c.cfg.Locale,
	})
	if err := c.GetJSON(ctx, u, &raw); err != nil {
		return nil, err
	}
	out := &IconSearchResult{Success: raw.Success, Icons: raw.Icons}
	out.Parameters.Amount = amount
	out.Parameters.Offset = offset
	out.Parameters.CountAll = len(raw.Icons)
	// Pack results omit `platform`, but every icon in a pack is in the style
	// that was asked for, so fill it in rather than returning blanks.
	for i := range out.Icons {
		if out.Icons[i].Platform == "" {
			out.Icons[i].Platform = style
		}
	}
	return out, nil
}

// PopularIconTerms is useful as a discovery aid when a caller has no term yet.
func (c *Client) PopularIconTerms(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	var out struct {
		Requests []string `json:"requests"`
		Docs     []string `json:"docs"`
	}
	u := buildURL(HostIconSearch, "/api/iconsets/popularRequests", map[string]string{
		"limit": strconv.Itoa(limit), "lang": c.cfg.Language(),
	})
	if err := c.GetJSON(ctx, u, &out); err != nil {
		return nil, err
	}
	if len(out.Requests) > 0 {
		return out.Requests, nil
	}
	return out.Docs, nil
}

// IconFormat values accepted by the download host. Icons8 rejects anything else
// with "format must be a valid enum value", so this list is the contract.
var IconFormats = []string{"png", "svg", "pdf", "eps", "jpg", "webp", "gif", "json", "apng"}

// AnimatedOnlyIconFormats are only served for icons with isAnimated=true.
var AnimatedOnlyIconFormats = map[string]bool{"gif": true, "json": true, "apng": true}

// IconDownloadOptions describes one concrete rendering of an icon.
type IconDownloadOptions struct {
	ID     string
	Format string
	Size   int
	// Color recolours the glyph, hex without '#', e.g. "FF0000".
	Color string
	// Simplified requests Icons8's reduced-node SVG.
	Simplified bool
	// Name becomes the suggested filename Icons8 stamps on the response.
	Name string
}

// IconDownloadURL builds the authenticated download URL. Fetching it is what
// registers the "unlock" against the account. There is no separate unlock call
// for icons, which is why an icon shows as locked until it has been downloaded
// once through this endpoint.
func (c *Client) IconDownloadURL(o IconDownloadOptions) (string, error) {
	if o.ID == "" {
		return "", fmt.Errorf("icon id required")
	}
	format := strings.ToLower(o.Format)
	if format == "" {
		format = "svg"
	}
	if !contains(IconFormats, format) {
		return "", fmt.Errorf("unsupported icon format %q (want one of %s)", format, strings.Join(IconFormats, ", "))
	}
	if o.Size <= 0 {
		o.Size = 256
	}
	q := map[string]string{
		"id":       o.ID,
		"format":   format,
		"size":     strconv.Itoa(o.Size),
		"fromSite": "true",
		"token":    c.sess.PublicAPIKey(),
		"name":     o.Name,
		"color":    strings.TrimPrefix(o.Color, "#"),
	}
	if o.Simplified && format == "svg" {
		q["simplified"] = "true"
	}
	return buildURL(HostIconImg, "/", q), nil
}

// IconPreviewURL is the unauthenticated CDN PNG. It costs no download credit and
// is the right thing to show in a picker before committing to a format.
func IconPreviewURL(id string, size int) string {
	if size <= 0 {
		size = 100
	}
	return buildURL(HostIconCDN, "/", map[string]string{
		"id": id, "size": strconv.Itoa(size), "format": "png",
	})
}

// DownloadIcon fetches the bytes for one icon rendering.
func (c *Client) DownloadIcon(ctx context.Context, o IconDownloadOptions) ([]byte, string, error) {
	u, err := c.IconDownloadURL(o)
	if err != nil {
		return nil, "", err
	}
	return c.GetBytes(ctx, u)
}

// UnlockedIconIDs reports which of the given icon ids the account has already
// paid a download for. Icons8 shows everything else as locked in the UI even
// when the licence covers it.
func (c *Client) UnlockedIconIDs(ctx context.Context, ids []string) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var out struct {
		Success bool     `json:"success"`
		IDs     []string `json:"ids"`
	}
	u := buildURL(HostIconAPI, "/user/v1/paidDownloadIds", map[string]string{
		"type": "iconDownload", "objectIds": strings.Join(ids, ","),
	})
	if err := c.GetJSON(ctx, u, &out); err != nil {
		return nil, err
	}
	return out.IDs, nil
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
