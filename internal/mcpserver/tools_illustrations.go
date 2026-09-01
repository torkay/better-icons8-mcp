package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/torkay/better-icons8-mcp/internal/assets"
	"github.com/torkay/better-icons8-mcp/internal/icons8"
)

type IllustrationSummary struct {
	ID         string   `json:"id"`
	Slug       string   `json:"slug"`
	Title      string   `json:"title"`
	Style      string   `json:"style,omitempty"`
	StyleSlug  string   `json:"style_slug,omitempty"`
	Animated   bool     `json:"animated"`
	Is3D       bool     `json:"is_3d,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Categories []string `json:"categories,omitempty"`
	PreviewURL string   `json:"preview_url,omitempty"`
	Formats    []string `json:"formats,omitempty"`
}

func toIllustrationSummary(i icons8.Illustration) IllustrationSummary {
	s := IllustrationSummary{
		ID: i.ID, Slug: i.PrettyID, Title: i.Heading,
		Animated: i.Animated(), Is3D: i.Is3D(),
		Formats: i.DownloadableResources.Available,
	}
	if len(i.Styles) > 0 {
		s.Style, s.StyleSlug = i.Styles[0].Title, i.Styles[0].PrettyID
	}
	for _, t := range i.Tags {
		s.Tags = append(s.Tags, t.Title)
	}
	for _, c := range i.Categories {
		s.Categories = append(s.Categories, c.Title)
	}
	switch {
	case i.Preview1x != nil:
		s.PreviewURL = i.Preview1x.URL
	case i.Thumb1x != nil:
		s.PreviewURL = i.Thumb1x.URL
	case i.ThumbXS1x != nil:
		s.PreviewURL = i.ThumbXS1x.URL
	}
	if len(s.Tags) > 12 {
		s.Tags = s.Tags[:12]
	}
	return s
}

type SearchIllustrationsInput struct {
	Query     string `json:"query,omitempty" jsonschema:"what to search for; leave empty to browse a style"`
	Style     string `json:"style,omitempty" jsonschema:"style slug from icons8_illustration_styles, e.g. techny, 3d-fluency, bright"`
	Category  string `json:"category,omitempty" jsonschema:"category slug, e.g. business, space"`
	Mood      string `json:"mood,omitempty" jsonschema:"mood facet, e.g. casual, funny, serious, retro"`
	Technique string `json:"technique,omitempty" jsonschema:"technique facet, e.g. 3d, flat, hand-drawn"`
	Colors    string `json:"colors,omitempty" jsonschema:"colour facet, comma-separated"`
	Animated  string `json:"animated,omitempty" jsonschema:"'y' for animated only, 'n' for static only"`
	Models    bool   `json:"models,omitempty" jsonschema:"search 3D models instead of flat illustrations"`
	Page      int    `json:"page,omitempty" jsonschema:"1-based page, default 1"`
	PerPage   int    `json:"per_page,omitempty" jsonschema:"default 30, max 100"`
}

type SearchIllustrationsOutput struct {
	Total         int                   `json:"total"`
	Page          int                   `json:"page"`
	Illustrations []IllustrationSummary `json:"illustrations"`
	Note          string                `json:"note,omitempty"`
}

type DownloadIllustrationInput struct {
	ID      string   `json:"id" jsonschema:"illustration id from a search result"`
	Formats []string `json:"formats,omitempty" jsonschema:"one or more of png-hd, png, png-low, svg, gif, gif-low, json (Lottie), webm, mov-avc, mov-hevc (an mp4), aep, and for 3D models fbx and glb. Defaults to svg, or png-hd when the item has no vector."`
	Name    string   `json:"name,omitempty" jsonschema:"filename stem; defaults to the illustration slug"`
}

type DownloadIllustrationOutput struct {
	ID        string         `json:"id"`
	Title     string         `json:"title"`
	Files     []assets.Saved `json:"files"`
	Skipped   []string       `json:"skipped,omitempty"`
	Available []string       `json:"available_formats,omitempty"`
	Note      string         `json:"note,omitempty"`
}

func (s *Server) registerIllustrationTools() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "icons8_search_illustrations",
		Description: "Search the Icons8 illustration library: flat illustrations, animated illustrations, and 3D models with models=true. " +
			"Filter by style so every illustration in a page or deck comes from the same visual family. Call icons8_illustration_styles first. " +
			"Set animated='y' for motion assets, which download as GIF, Lottie JSON, WebM or MP4.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in SearchIllustrationsInput) (*mcp.CallToolResult, SearchIllustrationsOutput, error) {
		res, err := s.client.SearchIllustrations(ctx, icons8.IllustrationSearchOptions{
			Query: in.Query, Style: in.Style, Category: in.Category, Mood: in.Mood,
			Technique: in.Technique, Colors: in.Colors,
			Animated: in.Animated, Models: in.Models, Page: in.Page, PerPage: in.PerPage,
		})
		if err != nil {
			return nil, SearchIllustrationsOutput{}, err
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		out := SearchIllustrationsOutput{Total: res.Total, Page: page}
		for _, i := range res.Illustrations {
			out.Illustrations = append(out.Illustrations, toIllustrationSummary(i))
		}
		if in.Style == "" && len(out.Illustrations) > 1 {
			out.Note = "Results span several styles. Pick one style slug and re-search with it so the artwork reads as one set."
		}
		return textResult(out), out, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "icons8_illustration_styles",
		Description: "List every illustration style with its item count and whether it has animated variants. " +
			"Choose one style before searching. Mixing styles is what makes a page's artwork look assembled from unrelated sources.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in struct {
		Animated bool `json:"animated_only,omitempty" jsonschema:"only styles that include animated illustrations"`
	}) (*mcp.CallToolResult, map[string]any, error) {
		styles, err := s.client.IllustrationStyles(ctx)
		if err != nil {
			return nil, nil, err
		}
		type entry struct {
			Slug     string `json:"slug" jsonschema:"pass as the style argument"`
			Title    string `json:"title"`
			Count    int    `json:"count"`
			Animated bool   `json:"animated"`
			Primary  string `json:"primary_color,omitempty"`
			Preview  string `json:"preview_url,omitempty"`
		}
		list := make([]entry, 0, len(styles))
		for _, st := range styles {
			if in.Animated && !st.Animated {
				continue
			}
			e := entry{Slug: st.PrettyID, Title: st.Title, Count: st.IllustrationCount, Animated: st.Animated, Primary: st.PrimaryColor}
			if st.Thumb1x != nil {
				e.Preview = st.Thumb1x.URL
			}
			list = append(list, e)
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Count > list[j].Count })
		out := map[string]any{
			"styles":   list,
			"count":    len(list),
			"guidance": "Pass `slug` as the `style` argument to icons8_search_illustrations. One style per artefact.",
		}
		return textResult(out), out, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "icons8_illustration",
		Description: "Full detail for one illustration or 3D model, including which download formats exist for it and what each format is.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in struct {
		ID string `json:"id" jsonschema:"illustration id"`
	}) (*mcp.CallToolResult, map[string]any, error) {
		ill, err := s.client.Illustration(ctx, in.ID)
		if err != nil {
			return nil, nil, err
		}
		formats := make([]map[string]string, 0)
		for _, f := range ill.DownloadableResources.Available {
			formats = append(formats, map[string]string{"format": f, "what_it_is": icons8.FormatNotes[f]})
		}
		out := map[string]any{
			"illustration": toIllustrationSummary(*ill),
			"formats":      formats,
		}
		return textResult(out), out, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "icons8_similar_illustrations",
		Description: "Find illustrations in the same visual family as a given one. Fills out a set once the first one fits.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in struct {
		ID      string `json:"id" jsonschema:"illustration id"`
		Style   string `json:"style,omitempty" jsonschema:"restrict to this style slug"`
		PerPage int    `json:"per_page,omitempty" jsonschema:"default 30"`
	}) (*mcp.CallToolResult, SearchIllustrationsOutput, error) {
		res, err := s.client.SimilarIllustrations(ctx, in.ID, in.Style, in.PerPage)
		if err != nil {
			return nil, SearchIllustrationsOutput{}, err
		}
		out := SearchIllustrationsOutput{Total: res.Total, Page: 1}
		for _, i := range res.Illustrations {
			out.Illustrations = append(out.Illustrations, toIllustrationSummary(i))
		}
		return textResult(out), out, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "icons8_download_illustration",
		Description: "Download an illustration, animated illustration or 3D model in one or more formats and return the paths. " +
			"Pick by target: svg or png-hd for static artwork, json (Lottie) or webm for web motion, mov-hevc for video editing, " +
			"fbx-zip for 3D. Requesting a format the item does not have is reported rather than failing the call.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in DownloadIllustrationInput) (*mcp.CallToolResult, DownloadIllustrationOutput, error) {
		if in.ID == "" {
			return nil, DownloadIllustrationOutput{}, fmt.Errorf("id is required")
		}
		ill, err := s.client.Illustration(ctx, in.ID)
		if err != nil {
			return nil, DownloadIllustrationOutput{}, err
		}
		// Index by the download-endpoint spelling so a caller asking for either
		// the advertised token or the request token is served.
		available := map[string]bool{}
		for _, f := range ill.DownloadableResources.Available {
			available[f] = true
			available[icons8.MediaFormat(f)] = true
		}

		formats := in.Formats
		if len(formats) == 0 {
			if available["svg"] {
				formats = []string{"svg"}
			} else {
				formats = []string{"png-hd"}
			}
		}
		stem := in.Name
		if stem == "" {
			stem = ill.PrettyID
		}
		if stem == "" {
			stem = "illustration-" + in.ID
		}

		out := DownloadIllustrationOutput{
			ID: in.ID, Title: ill.Heading,
			Available: ill.DownloadableResources.Available,
		}
		for _, f := range formats {
			f = strings.ToLower(strings.TrimSpace(f))
			if !available[f] {
				out.Skipped = append(out.Skipped, f)
				continue
			}
			f = icons8.MediaFormat(f)
			dl, err := s.client.IllustrationDownloadURL(ctx, in.ID, f)
			if err != nil {
				return nil, DownloadIllustrationOutput{}, err
			}
			data, ct, err := s.client.FetchSigned(ctx, dl.URL)
			if err != nil {
				return nil, DownloadIllustrationOutput{}, fmt.Errorf("fetch %s for %s: %w", f, in.ID, err)
			}
			ext := assets.ExtFor(f)
			if ext == "" || ext == "bin" {
				ext = assets.ExtFromContentType(ct)
			}
			name := stem
			if len(formats) > 1 {
				name = stem + "-" + assets.Slug(f)
			}
			kind := "illustrations"
			if ill.Is3D() {
				kind = "models3d"
			}
			saved, err := s.store.Save(kind, name, ext, data, dl.URL)
			if err != nil {
				return nil, DownloadIllustrationOutput{}, err
			}
			out.Files = append(out.Files, *saved)
		}
		if len(out.Skipped) > 0 {
			out.Note = fmt.Sprintf("not available for this item: %s", strings.Join(out.Skipped, ", "))
		}
		if len(out.Files) == 0 {
			return nil, out, fmt.Errorf("none of the requested formats exist for %s; available: %s",
				in.ID, strings.Join(ill.DownloadableResources.Available, ", "))
		}
		return textResult(out), out, nil
	})
}

// asAPIError is errors.As with a concrete target, kept in one place so the tool
// files do not each import errors for a single call.
func asAPIError(err error, target **icons8.APIError) bool {
	return errors.As(err, target)
}
