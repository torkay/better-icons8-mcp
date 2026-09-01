package icons8

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Asset is one file variant attached to an illustration (a thumb, a preview, a
// downloadable source). Width/height are absent on some entries, so treat 0 as
// "unknown" rather than "empty".
type Asset struct {
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	URL      string `json:"url"`
	Filesize int    `json:"filesize,omitempty"`
}

type NamedRef struct {
	ID       string `json:"id"`
	PrettyID string `json:"pretty_id"`
	Title    string `json:"title"`
}

type IllustrationStyle struct {
	ID                string `json:"id"`
	PrettyID          string `json:"pretty_id"`
	Title             string `json:"title"`
	Animated          bool   `json:"animated"`
	FreeDistribution  bool   `json:"free_distribution"`
	IllustrationCount int    `json:"illustrations_count"`
	PrimaryColor      string `json:"primary_color,omitempty"`
	SecondaryColor    string `json:"secondary_color,omitempty"`
	BackgroundColor   string `json:"background_color,omitempty"`
	Thumb1x           *Asset `json:"thumb1x,omitempty"`
	Thumb2x           *Asset `json:"thumb2x,omitempty"`
}

// Illustration covers both flat illustrations and 3D models. Icons8 stores
// them in the same collection, distinguished by the `model` search flag and by
// which downloadable resources exist.
type Illustration struct {
	ID               string              `json:"id"`
	PrettyID         string              `json:"pretty_id"`
	Heading          string              `json:"heading"`
	Description      string              `json:"description,omitempty"`
	Type             string              `json:"type,omitempty"`
	FreeDistribution bool                `json:"free_distribution"`
	Styles           []IllustrationStyle `json:"styles"`
	Categories       []NamedRef          `json:"categories,omitempty"`
	Tags             []NamedRef          `json:"tags,omitempty"`
	CreatedAt        int64               `json:"created_at,omitempty"`

	ThumbXS1x  *Asset `json:"thumb_xs1x,omitempty"`
	Thumb1x    *Asset `json:"thumb1x,omitempty"`
	Preview1x  *Asset `json:"preview1x,omitempty"`
	Preview2x  *Asset `json:"preview2x,omitempty"`
	JPGPreview *Asset `json:"jpg_preview,omitempty"`
	PNG        *Asset `json:"png,omitempty"`
	WebM       *Asset `json:"webm,omitempty"`
	MovHEVC    *Asset `json:"mov_hevc,omitempty"`
	JSON       *Asset `json:"json,omitempty"`
	GifLow     *Asset `json:"gif_low,omitempty"`

	Author *struct {
		Name     string `json:"name"`
		Icons8ID string `json:"icons8_id"`
		External bool   `json:"external"`
	} `json:"author,omitempty"`

	DownloadableResources struct {
		Available []string        `json:"available"`
		Sources   json.RawMessage `json:"sources,omitempty"`
		FBXZip    json.RawMessage `json:"fbx-zip,omitempty"`
	} `json:"downloadable_resources"`
}

// HasMotion reports whether this illustration animates.
//
// It checks both sources because they do not overlap: search results carry the
// motion asset fields but omit downloadable_resources, while the detail endpoint
// populates downloadable_resources.
func (i Illustration) HasMotion() bool {
	if i.WebM != nil || i.JSON != nil || i.GifLow != nil || i.MovHEVC != nil {
		return true
	}
	for _, f := range i.DownloadableResources.Available {
		switch f {
		case "gif", "json", "webm", "mov-avc", "mov-hevc", "aep", "gif-low":
			return true
		}
	}
	return false
}

// Animated is HasMotion under the name the API uses for the same idea.
func (i Illustration) Animated() bool { return i.HasMotion() }

// Is3D reports whether 3D source files are attached.
func (i Illustration) Is3D() bool {
	for _, f := range i.DownloadableResources.Available {
		switch f {
		case "fbx", "fbx-zip", "glb", "gltf", "blend", "sources":
			return true
		}
	}
	return len(i.DownloadableResources.FBXZip) > 0 && string(i.DownloadableResources.FBXZip) != "null"
}

type IllustrationSearchResult struct {
	Total         int            `json:"total"`
	Illustrations []Illustration `json:"illustrations"`
}

// IllustrationSearchOptions mirrors the filters on /illustrations. Style,
// Category and Mood take Icons8 `pretty_id` slugs.
type IllustrationSearchOptions struct {
	Query    string
	Page     int
	PerPage  int
	Style    string
	Category string
	Mood     string
	Colors   string
	// Technique is a facet like "3d", "flat", "hand-drawn".
	Technique string
	// Animated: "y" restricts to animated, "n" to static, "" for both.
	Animated string
	// Models restricts to 3D models (the /threedio catalogue).
	Models bool
	Locale string
}

func (o *IllustrationSearchOptions) normalise(cfg func() string) {
	if o.Page <= 0 {
		o.Page = 1
	}
	if o.PerPage <= 0 {
		o.PerPage = 30
	}
	if o.PerPage > 100 {
		o.PerPage = 100
	}
	if o.Locale == "" {
		o.Locale = cfg()
	}
}

// meta carries the facet filters. Icons8 splits its illustration filters across
// two mechanisms: style/category/animated ride as ordinary query parameters,
// while mood, technique and colour go inside this JSON blob. Putting one in the
// other's place is silently ignored rather than rejected, so the split matters.
func (o IllustrationSearchOptions) meta() string {
	m := map[string][]string{}
	if o.Mood != "" {
		m["mood"] = splitCSV(o.Mood)
	}
	if o.Technique != "" {
		m["technique"] = splitCSV(o.Technique)
	}
	if o.Colors != "" {
		m["colors"] = splitCSV(o.Colors)
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// SearchIllustrations queries the Ouch catalogue. With Models set it returns 3D
// models instead; the endpoint and shape are otherwise identical.
func (c *Client) SearchIllustrations(ctx context.Context, o IllustrationSearchOptions) (*IllustrationSearchResult, error) {
	o.normalise(func() string { return c.cfg.Locale })

	q := map[string]string{
		"locale":              o.Locale,
		"page":                strconv.Itoa(o.Page),
		"per_page":            strconv.Itoa(o.PerPage),
		"meta":                o.meta(),
		"search":              o.Query,
		"style_pretty_id":     o.Style,
		"category_pretty_ids": o.Category,
	}
	if o.Models {
		q["model"] = "true"
	}
	// The API takes animated as a boolean flag that is only ever sent when
	// filtering to animated items; there is no "static only" value, so exclude
	// them client-side instead.
	if isAnimatedFilter(o.Animated) {
		q["animated"] = "true"
	}

	// With no free-text term the search host returns nothing useful; the app
	// switches to the browse endpoint, so mirror that.
	host, path := HostOuchSearch, "/api/illustrations/ouch/search"
	if strings.TrimSpace(o.Query) == "" {
		host, path = HostOuch, "/api/frontend/v1/illustrations/watermarkless"
		delete(q, "search")
	}

	var out IllustrationSearchResult
	if err := c.GetJSON(ctx, buildURL(host, path, q), &out); err != nil {
		return nil, err
	}
	// "static only" has no server-side representation, so apply it here. Search
	// results carry no downloadable_resources, so fall back to the presence of
	// motion assets on the item itself.
	if isStaticFilter(o.Animated) {
		kept := out.Illustrations[:0]
		for _, i := range out.Illustrations {
			if !i.HasMotion() {
				kept = append(kept, i)
			}
		}
		out.Illustrations = kept
	}
	return &out, nil
}

func isAnimatedFilter(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "y", "yes", "true", "animated":
		return true
	}
	return false
}

func isStaticFilter(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "n", "no", "false", "static":
		return true
	}
	return false
}

func (c *Client) Illustration(ctx context.Context, id string) (*Illustration, error) {
	var out Illustration
	u := buildURL(HostOuch, "/api/frontend/v1/illustrations/"+id, map[string]string{"locale": c.cfg.Locale})
	if err := c.GetJSON(ctx, u, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SimilarIllustrations returns items that read as the same visual family, which
// is what keeps a page's artwork coherent.
func (c *Client) SimilarIllustrations(ctx context.Context, id, style string, perPage int) (*IllustrationSearchResult, error) {
	if perPage <= 0 {
		perPage = 30
	}
	var out IllustrationSearchResult
	u := buildURL(HostOuch, "/api/frontend/v1/illustrations/"+id+"/similars/watermarkless", map[string]string{
		"page": "1", "per_page": strconv.Itoa(perPage),
		"style_pretty_id": style, "locale": c.cfg.Locale,
	})
	if err := c.GetJSON(ctx, u, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// IllustrationStyles lists every style in the catalogue, with counts. This is
// the menu an agent should pick from before searching, so a whole project can
// commit to one look.
func (c *Client) IllustrationStyles(ctx context.Context) ([]IllustrationStyle, error) {
	var out []IllustrationStyle
	u := buildURL(HostOuch, "/api/frontend/v1/illustrations/styles", map[string]string{
		"fields": "title,pretty_id,icon,thumb1x,generator,free_distribution,backgroundColor",
	})
	if err := c.GetJSON(ctx, u, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// IllustrationFilters returns the mood/colour/technique facets for a query.
func (c *Client) IllustrationFilters(ctx context.Context, query string, models bool) (map[string]any, error) {
	q := map[string]string{
		"key_pretty_ids": "mood,colors,technique",
		"locale":         c.cfg.Locale,
		"meta":           "{}",
		"search":         query,
	}
	if models {
		q["model"] = "true"
	}
	var out map[string]any
	if err := c.GetJSON(ctx, buildURL(HostOuchSearch, "/api/illustrations/ouch/filters", q), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// IllustrationFormats are the `media_format` values the download endpoint
// accepts. Which ones exist for a given item is in DownloadableResources.
var IllustrationFormats = []string{
	"png-hd", "png", "png-low", "svg",
	"gif", "gif-low", "json", "webm", "mov-avc", "mov-hevc", "aep",
	"fbx", "sources",
}

// FormatNotes explains what each format actually is, since the names are not
// self-describing (mov-hevc is an mp4 file, json is a Lottie animation).
var FormatNotes = map[string]string{
	"png-hd":   "PNG, full resolution (~2500px)",
	"png":      "PNG, medium resolution",
	"png-low":  "PNG, thumbnail resolution",
	"svg":      "vector SVG",
	"gif":      "animated GIF, full resolution",
	"gif-low":  "animated GIF, reduced resolution",
	"json":     "Lottie animation (JSON) for web/app playback",
	"webm":     "WebM video with alpha, best for web",
	"mov-avc":  "MOV (H.264) for video editors",
	"mov-hevc": "MP4/HEVC with alpha, best for Apple platforms",
	"aep":      "Adobe After Effects project",
	"fbx":      "3D model, FBX with textures, delivered as a zip",
	"sources":  "original source files",
}

type DownloadURL struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// IllustrationDownloadURL resolves a signed, time-limited CDN URL for one
// format. This call is the unlock: it goes through Icons8's billing check and
// records the download against the licence.
func (c *Client) IllustrationDownloadURL(ctx context.Context, id, mediaFormat string) (*DownloadURL, error) {
	if mediaFormat == "" {
		mediaFormat = "png-hd"
	}
	if !contains(IllustrationFormats, mediaFormat) {
		return nil, fmt.Errorf("unsupported illustration format %q (want one of %s)",
			mediaFormat, strings.Join(IllustrationFormats, ", "))
	}
	var out DownloadURL
	u := buildURL(HostOuch, "/api/frontend/v1/illustrations/"+id+"/download-url",
		map[string]string{"format": mediaFormat})
	if err := c.GetJSON(ctx, u, &out); err != nil {
		return nil, err
	}
	if out.URL == "" {
		return nil, fmt.Errorf("icons8 returned no download url for %s (%s)", id, mediaFormat)
	}
	return &out, nil
}

// Allowance is Icons8's answer to "may this account take this download".
type Allowance struct {
	Success   bool    `json:"success"`
	IsAllowed bool    `json:"isAllowed"`
	Price     float64 `json:"price"`
	Reason    struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"reason"`
}

// CheckAllowance asks the billing service whether a download is permitted.
// resource is "illustration" or "photo".
func (c *Client) CheckAllowance(ctx context.Context, resource, id, format string) (*Allowance, error) {
	var out Allowance
	u := buildURL(HostBilling, "/"+resource+"/download/info", map[string]string{
		"id": id, "format": format,
	})
	if err := c.GetJSON(ctx, u, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
