package mcpserver

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/torkay/icons8-mcp-server/internal/assets"
	"github.com/torkay/icons8-mcp-server/internal/icons8"
)

// ---------- search ----------

type SearchIconsInput struct {
	Term     string `json:"term" jsonschema:"what to search for, e.g. 'shopping cart'"`
	Style    string `json:"style,omitempty" jsonschema:"restrict to one style (Icons8 calls it platform), e.g. fluency, ios7, ios_filled, color, material-outlined. Get valid values from icons8_icon_styles."`
	Category string `json:"category,omitempty" jsonschema:"restrict to a category api code, e.g. transport"`
	Author   string `json:"author,omitempty" jsonschema:"restrict to an author api code; 'icons8' for first-party icons only"`
	Animated string `json:"animated,omitempty" jsonschema:"'y' for animated icons only, 'n' to exclude them"`
	Amount   int    `json:"amount,omitempty" jsonschema:"how many results, default 30, max 100"`
	Offset   int    `json:"offset,omitempty" jsonschema:"pagination offset"`
}

type IconSummary struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Style      string `json:"style"`
	Category   string `json:"category,omitempty"`
	IsColor    bool   `json:"is_color"`
	IsAnimated bool   `json:"is_animated,omitempty"`
	Author     string `json:"author,omitempty"`
	PreviewURL string `json:"preview_url"`
}

type SearchIconsOutput struct {
	Total    int           `json:"total"`
	Offset   int           `json:"offset"`
	Returned int           `json:"returned"`
	Icons    []IconSummary `json:"icons"`
	Note     string        `json:"note,omitempty"`
}

func toIconSummary(i icons8.Icon) IconSummary {
	return IconSummary{
		ID: i.ID, Name: i.Name, Style: i.Platform, Category: i.Category,
		IsColor: i.IsColor, IsAnimated: i.IsAnimated, Author: i.AuthorAPICode,
		PreviewURL: icons8.IconPreviewURL(i.ID, 100),
	}
}

// ---------- styles ----------

type IconStylesInput struct {
	Term string `json:"term,omitempty" jsonschema:"optional search term; when given, only styles that actually have results for it are marked enabled"`
}

type StyleEntry struct {
	Name     string   `json:"name"`
	Value    string   `json:"value" jsonschema:"pass this as the style filter"`
	Enabled  bool     `json:"enabled"`
	Variants []string `json:"variants,omitempty" jsonschema:"sibling styles in the same family"`
}

type IconStylesOutput struct {
	Styles     []StyleEntry `json:"styles"`
	Categories []StyleEntry `json:"categories,omitempty"`
	Guidance   string       `json:"guidance"`
}

// ---------- download ----------

type DownloadIconInput struct {
	ID         string   `json:"id" jsonschema:"icon id from a search result"`
	Formats    []string `json:"formats,omitempty" jsonschema:"one or more of svg, png, pdf, eps, jpg, webp, and for animated icons gif, apng, json (Lottie). Defaults to svg."`
	Sizes      []int    `json:"sizes,omitempty" jsonschema:"pixel sizes to render; defaults to 256. SVG ignores this beyond the viewBox attribute."`
	Color      string   `json:"color,omitempty" jsonschema:"recolour the icon, hex without '#', e.g. FF6B00. Only meaningful for single-colour styles."`
	Simplified bool     `json:"simplified,omitempty" jsonschema:"request Icons8's reduced-node SVG, smaller but less precise"`
	Name       string   `json:"name,omitempty" jsonschema:"filename stem; defaults to the icon name"`
}

type DownloadIconOutput struct {
	ID    string         `json:"id"`
	Files []assets.Saved `json:"files"`
	Note  string         `json:"note,omitempty"`
}

// ---------- favicon / app icon set ----------

type FaviconInput struct {
	ID       string `json:"id" jsonschema:"icon id from a search result"`
	Platform string `json:"platform,omitempty" jsonschema:"favicon, web, ios, android, macos or windows. Defaults to favicon."`
	Name     string `json:"name,omitempty" jsonschema:"filename stem"`
	Color    string `json:"color,omitempty" jsonschema:"recolour the icon, hex without '#'"`
	ICO      bool   `json:"ico,omitempty" jsonschema:"also pack the sizes into a multi-resolution .ico file (Windows / classic favicon)"`
}

type FaviconOutput struct {
	ID       string         `json:"id"`
	Platform string         `json:"platform"`
	Sizes    []int          `json:"sizes"`
	Files    []assets.Saved `json:"files"`
	HTML     string         `json:"html_snippet,omitempty"`
}

// ---------- embed ----------

type EmbedIconInput struct {
	ID    string `json:"id" jsonschema:"icon id"`
	Size  int    `json:"size,omitempty" jsonschema:"pixel size for the raster forms, default 100"`
	Color string `json:"color,omitempty" jsonschema:"recolour, hex without '#'"`
}

type EmbedIconOutput struct {
	ID        string `json:"id"`
	CDNLink   string `json:"cdn_link" jsonschema:"hotlinkable PNG URL, no download credit consumed"`
	Base64PNG string `json:"base64_png" jsonschema:"data: URI for a PNG"`
	Base64SVG string `json:"base64_svg" jsonschema:"data: URI for the SVG"`
	SVGMarkup string `json:"svg_markup" jsonschema:"inline-able SVG source"`
	IMGTag    string `json:"img_tag"`
}

func (s *Server) registerIconTools() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "icons8_search_icons",
		Description: "Search the Icons8 icon library. Filter by style so an interface stays visually consistent. " +
			"Call icons8_icon_styles first and reuse one style value across a project. " +
			"Returns metadata and preview URLs. Use icons8_download_icon for files.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in SearchIconsInput) (*mcp.CallToolResult, SearchIconsOutput, error) {
		if strings.TrimSpace(in.Term) == "" && in.Style == "" && in.Category == "" {
			return nil, SearchIconsOutput{}, fmt.Errorf("give a term, or a style/category to browse")
		}
		if in.Amount > 100 {
			in.Amount = 100
		}
		res, err := s.client.SearchIcons(ctx, icons8.IconSearchOptions{
			Term: in.Term, Style: in.Style, Category: in.Category, Author: in.Author,
			Animated: in.Animated, Amount: in.Amount, Offset: in.Offset,
		})
		if err != nil {
			return nil, SearchIconsOutput{}, err
		}
		out := SearchIconsOutput{
			Total: res.Parameters.CountAll, Offset: res.Parameters.Offset,
			Returned: len(res.Icons),
			Icons:    make([]IconSummary, 0, len(res.Icons)),
		}
		for _, i := range res.Icons {
			out.Icons = append(out.Icons, toIconSummary(i))
		}
		if in.Style == "" && len(out.Icons) > 1 {
			out.Note = "Results span several styles. Pick one style value and re-search with it so the set looks like one family."
		}
		return textResult(out), out, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "icons8_icon_styles",
		Description: "List the available icon styles (Icons8 calls them platforms) and categories, optionally scoped to a search term. " +
			"Call this before searching so a project commits to one style. Variants show which styles belong to the same family.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in IconStylesInput) (*mcp.CallToolResult, IconStylesOutput, error) {
		groups, err := s.client.IconFilters(ctx, in.Term)
		if err != nil {
			return nil, IconStylesOutput{}, err
		}
		families, err := s.client.StyleGroups(ctx)
		if err != nil {
			// Family data is a nicety; a failure here should not sink the call.
			s.logf("style groups unavailable: %v", err)
		}
		siblings := map[string][]string{}
		for _, g := range families {
			for _, e := range g.Entities {
				siblings[e] = g.Entities
			}
		}

		out := IconStylesOutput{
			Guidance: "Pass `value` as the `style` argument to icons8_search_icons. Use one style for every icon in a single interface.",
		}
		for _, g := range groups {
			var dst *[]StyleEntry
			switch g.APICode {
			case "style", "platform":
				dst = &out.Styles
			case "category":
				dst = &out.Categories
			default:
				continue
			}
			var walk func(opts []FilterOptionLike, depth int)
			walk = func(opts []FilterOptionLike, depth int) {
				for _, o := range opts {
					// Family headers (value prefixed "family-") are not usable
					// filters themselves, only their children are.
					if !strings.HasPrefix(o.Value, "family-") {
						*dst = append(*dst, StyleEntry{
							Name: o.Name, Value: o.Value, Enabled: o.IsEnabled,
							Variants: siblings[o.Value],
						})
					}
					if depth < 3 && len(o.Options) > 0 {
						walk(toFilterOptionLike(o.Options), depth+1)
					}
				}
			}
			walk(toFilterOptionLike(g.Options), 0)
		}
		sort.SliceStable(out.Styles, func(i, j int) bool {
			if out.Styles[i].Enabled != out.Styles[j].Enabled {
				return out.Styles[i].Enabled
			}
			return out.Styles[i].Name < out.Styles[j].Name
		})
		return textResult(out), out, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "icons8_icon_variants",
		Description: "List the same glyph rendered in every other style, given one icon id. " +
			"Use it when an icon is right but its style does not match the interface.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in struct {
		ID string `json:"id" jsonschema:"icon id"`
	}) (*mcp.CallToolResult, map[string]any, error) {
		vs, err := s.client.IconVariants(ctx, in.ID)
		if err != nil {
			return nil, nil, err
		}
		list := make([]IconSummary, 0, len(vs))
		for _, v := range vs {
			list = append(list, IconSummary{
				ID: v.ID, Name: v.Name, Style: v.Platform, Category: v.Category,
				IsColor: v.IsColor, IsAnimated: v.IsAnimated,
				PreviewURL: icons8.IconPreviewURL(v.ID, 100),
			})
		}
		out := map[string]any{"variants": list, "count": len(list)}
		return textResult(out), out, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "icons8_similar_icons",
		Description: "Find icons visually similar to a given one. Fills out a set once the first icon fits.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in struct {
		ID    string `json:"id" jsonschema:"icon id"`
		Limit int    `json:"limit,omitempty" jsonschema:"how many, default 30"`
	}) (*mcp.CallToolResult, map[string]any, error) {
		list, err := s.client.SimilarIcons(ctx, in.ID, in.Limit)
		if err != nil {
			return nil, nil, err
		}
		sum := make([]IconSummary, 0, len(list))
		for _, i := range list {
			sum = append(sum, toIconSummary(i))
		}
		out := map[string]any{"icons": sum, "count": len(sum)}
		return textResult(out), out, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "icons8_icon_pack",
		Description: "Browse a whole style and category pack instead of searching. Every result is already in the same style, " +
			"which suits a navigation bar or a feature grid.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in struct {
		Style    string `json:"style" jsonschema:"style value from icons8_icon_styles"`
		Category string `json:"category,omitempty" jsonschema:"category api code"`
		SortBy   string `json:"sort_by,omitempty" jsonschema:"popular or new, default popular"`
		Amount   int    `json:"amount,omitempty" jsonschema:"default 50"`
		Offset   int    `json:"offset,omitempty"`
	}) (*mcp.CallToolResult, SearchIconsOutput, error) {
		res, err := s.client.IconPack(ctx, in.Style, in.Category, in.SortBy, in.Amount, in.Offset)
		if err != nil {
			return nil, SearchIconsOutput{}, err
		}
		out := SearchIconsOutput{Total: res.Parameters.CountAll, Offset: in.Offset, Returned: len(res.Icons)}
		for _, i := range res.Icons {
			out.Icons = append(out.Icons, toIconSummary(i))
		}
		return textResult(out), out, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "icons8_download_icon",
		Description: "Download an icon in one or more formats and sizes and return the file paths. " +
			"SVG for web and app UI. PNG when a raster is required. Lottie json, gif or apng for animated icons. " +
			"The call also clears Icons8's locked state for the icon.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in DownloadIconInput) (*mcp.CallToolResult, DownloadIconOutput, error) {
		if in.ID == "" {
			return nil, DownloadIconOutput{}, fmt.Errorf("id is required")
		}
		formats := in.Formats
		if len(formats) == 0 {
			formats = []string{"svg"}
		}
		sizes := in.Sizes
		if len(sizes) == 0 {
			sizes = []int{256}
		}
		stem := in.Name
		if stem == "" {
			stem = "icon-" + in.ID
		}

		out := DownloadIconOutput{ID: in.ID}
		for _, f := range formats {
			f = strings.ToLower(strings.TrimSpace(f))
			// Vector formats have no per-size rendering worth repeating.
			effective := sizes
			if f == "svg" || f == "pdf" || f == "eps" || f == "json" {
				effective = sizes[:1]
			}
			for _, size := range effective {
				data, ct, err := s.client.DownloadIcon(ctx, icons8.IconDownloadOptions{
					ID: in.ID, Format: f, Size: size, Color: in.Color,
					Simplified: in.Simplified, Name: stem + "." + f,
				})
				if err != nil {
					if isUnsupportedFormat(err) {
						out.Note = appendNote(out.Note, fmt.Sprintf("format %q is not available for this icon (animated-only formats need an animated icon)", f))
						break
					}
					return nil, DownloadIconOutput{}, err
				}
				name := stem
				if len(effective) > 1 {
					name = fmt.Sprintf("%s-%d", stem, size)
				}
				ext := assets.ExtFor(f)
				if ext == "bin" {
					ext = assets.ExtFromContentType(ct)
				}
				saved, err := s.store.Save("icons", name, ext, data, "")
				if err != nil {
					return nil, DownloadIconOutput{}, err
				}
				out.Files = append(out.Files, *saved)
			}
		}
		if len(out.Files) == 0 {
			return nil, out, fmt.Errorf("nothing downloaded for icon %s: %s", in.ID, out.Note)
		}
		return textResult(out), out, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "icons8_icon_favicon",
		Description: "Generate a favicon or app-icon set from one icon: every size the target platform expects, an optional " +
			"multi-resolution .ico, and an HTML snippet. Platforms: favicon, web, ios, android, macos, windows.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in FaviconInput) (*mcp.CallToolResult, FaviconOutput, error) {
		if in.ID == "" {
			return nil, FaviconOutput{}, fmt.Errorf("id is required")
		}
		platform := strings.ToLower(in.Platform)
		if platform == "" {
			platform = "favicon"
		}
		sizes, ok := assets.SizePresets[platform]
		if !ok {
			return nil, FaviconOutput{}, fmt.Errorf("unknown platform %q (want favicon, web, ios, android, macos or windows)", in.Platform)
		}
		stem := in.Name
		if stem == "" {
			stem = "favicon-" + in.ID
		}

		out := FaviconOutput{ID: in.ID, Platform: platform, Sizes: sizes}
		pngs := map[int][]byte{}
		for _, size := range sizes {
			data, _, err := s.client.DownloadIcon(ctx, icons8.IconDownloadOptions{
				ID: in.ID, Format: "png", Size: size, Color: in.Color,
				Name: fmt.Sprintf("%s-%d.png", stem, size),
			})
			if err != nil {
				return nil, FaviconOutput{}, err
			}
			pngs[size] = data
			saved, err := s.store.Save("icons/"+platform, fmt.Sprintf("%s-%dx%d", stem, size, size), "png", data, "")
			if err != nil {
				return nil, FaviconOutput{}, err
			}
			out.Files = append(out.Files, *saved)
		}

		// An SVG favicon is what modern browsers actually prefer.
		if svg, _, err := s.client.DownloadIcon(ctx, icons8.IconDownloadOptions{
			ID: in.ID, Format: "svg", Size: 512, Color: in.Color, Name: stem + ".svg",
		}); err == nil {
			if saved, err := s.store.Save("icons/"+platform, stem, "svg", svg, ""); err == nil {
				out.Files = append(out.Files, *saved)
			}
		}

		if in.ICO {
			icoPNGs := map[int][]byte{}
			for _, size := range assets.ICOSizes(sizes) {
				icoPNGs[size] = pngs[size]
			}
			ico, err := assets.EncodeICO(icoPNGs)
			if err != nil {
				return nil, FaviconOutput{}, fmt.Errorf("pack .ico: %w", err)
			}
			saved, err := s.store.Save("icons/"+platform, stem, "ico", ico, "")
			if err != nil {
				return nil, FaviconOutput{}, err
			}
			out.Files = append(out.Files, *saved)
		}

		if platform == "favicon" || platform == "web" {
			out.HTML = strings.Join([]string{
				`<link rel="icon" href="/favicon.ico" sizes="32x32">`,
				fmt.Sprintf(`<link rel="icon" href="/%s.svg" type="image/svg+xml">`, assets.Slug(stem)),
				fmt.Sprintf(`<link rel="apple-touch-icon" href="/%s-180x180.png">`, assets.Slug(stem)),
			}, "\n")
		}
		return textResult(out), out, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "icons8_icon_embed",
		Description: "Return every embeddable form of an icon without writing files: a hotlinkable CDN URL, base64 data URIs for " +
			"PNG and SVG, and raw SVG markup for inlining.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in EmbedIconInput) (*mcp.CallToolResult, EmbedIconOutput, error) {
		if in.ID == "" {
			return nil, EmbedIconOutput{}, fmt.Errorf("id is required")
		}
		size := in.Size
		if size <= 0 {
			size = 100
		}
		png, _, err := s.client.DownloadIcon(ctx, icons8.IconDownloadOptions{ID: in.ID, Format: "png", Size: size, Color: in.Color})
		if err != nil {
			return nil, EmbedIconOutput{}, err
		}
		svg, _, err := s.client.DownloadIcon(ctx, icons8.IconDownloadOptions{ID: in.ID, Format: "svg", Size: size, Color: in.Color})
		if err != nil {
			return nil, EmbedIconOutput{}, err
		}
		out := EmbedIconOutput{
			ID:        in.ID,
			CDNLink:   icons8.IconPreviewURL(in.ID, size),
			Base64PNG: "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
			Base64SVG: "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(svg),
			SVGMarkup: string(svg),
		}
		out.IMGTag = fmt.Sprintf(`<img width="%d" height="%d" src="%s" alt="icon"/>`, size, size, out.CDNLink)
		return textResult(out), out, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "icons8_check_unlock",
		Description: "Report which icon ids the account has already downloaded. Icons8 shows the rest as locked. " +
			"Downloading through icons8_download_icon unlocks an icon, so this is diagnostic rather than a gate.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in struct {
		IDs []string `json:"ids" jsonschema:"icon ids to check"`
	}) (*mcp.CallToolResult, map[string]any, error) {
		unlocked, err := s.client.UnlockedIconIDs(ctx, in.IDs)
		if err != nil {
			return nil, nil, err
		}
		set := map[string]bool{}
		for _, id := range unlocked {
			set[id] = true
		}
		var locked []string
		for _, id := range in.IDs {
			if !set[id] {
				locked = append(locked, id)
			}
		}
		out := map[string]any{
			"unlocked": unlocked,
			"locked":   locked,
			"note":     "Locked simply means not yet downloaded on this account. The licence still covers them; downloading clears it.",
		}
		return textResult(out), out, nil
	})
}

// FilterOptionLike lets the recursive walk work on both group and nested option
// levels, which the API models with the same shape at every depth.
type FilterOptionLike struct {
	Name      string
	Value     string
	IsEnabled bool
	Options   []icons8.FilterOption
}

func toFilterOptionLike(in []icons8.FilterOption) []FilterOptionLike {
	out := make([]FilterOptionLike, 0, len(in))
	for _, o := range in {
		out = append(out, FilterOptionLike{Name: o.Name, Value: o.Value, IsEnabled: o.IsEnabled, Options: o.Options})
	}
	return out
}

func isUnsupportedFormat(err error) bool {
	var apiErr *icons8.APIError
	if ok := asAPIError(err, &apiErr); !ok {
		return false
	}
	return apiErr.Code == "UNSUPPORTED_FORMAT" ||
		strings.Contains(apiErr.Body, "must be a valid enum value") ||
		strings.Contains(apiErr.Body, "does not exist in")
}

func appendNote(existing, add string) string {
	if existing == "" {
		return add
	}
	return existing + "; " + add
}
