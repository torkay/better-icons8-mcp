package mcpserver

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/torkay/better-icons8-mcp/internal/assets"
	"github.com/torkay/better-icons8-mcp/internal/icons8"
)

type PhotoSummary struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Width      int      `json:"width"`
	Height     int      `json:"height"`
	Tags       []string `json:"tags,omitempty"`
	PreviewURL string   `json:"preview_url,omitempty"`
}

func toPhotoSummary(p icons8.Photo) PhotoSummary {
	s := PhotoSummary{ID: p.ID, Title: p.Title, Width: p.Width, Height: p.Height}
	for i, t := range p.Tags {
		if i >= 12 {
			break
		}
		s.Tags = append(s.Tags, t.Title)
	}
	switch {
	case p.Preview1x != nil:
		s.PreviewURL = p.Preview1x.URL
	case p.Thumb2x != nil:
		s.PreviewURL = p.Thumb2x.URL
	case p.Thumb1x != nil:
		s.PreviewURL = p.Thumb1x.URL
	}
	return s
}

func (s *Server) registerPhotoTools() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "icons8_search_photos",
		Description: "Search the Icons8 photo library (Moose). Photos are a separate library from illustrations: real photography " +
			"and cut-out people or objects rather than drawn artwork. filter='transparent' returns background-free cut-outs.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in struct {
		Query   string `json:"query" jsonschema:"what to search for, e.g. 'person working at desk'"`
		Filter  string `json:"filter,omitempty" jsonschema:"all, transparent, backgrounds or elements. Default all."`
		SortBy  string `json:"sort_by,omitempty" jsonschema:"rising, new or popular. Default rising."`
		Page    int    `json:"page,omitempty" jsonschema:"1-based page"`
		PerPage int    `json:"per_page,omitempty" jsonschema:"default 30, max 100"`
	}) (*mcp.CallToolResult, map[string]any, error) {
		res, err := s.client.SearchPhotos(ctx, icons8.PhotoSearchOptions{
			Query: in.Query, Filter: in.Filter, SortBy: in.SortBy, Page: in.Page, PerPage: in.PerPage,
		})
		if err != nil {
			return nil, nil, err
		}
		list := make([]PhotoSummary, 0, len(res.Images))
		for _, p := range res.Images {
			list = append(list, toPhotoSummary(p))
		}
		out := map[string]any{"total": res.Total, "photos": list, "returned": len(list)}
		return textResult(out), out, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "icons8_photo_suggest",
		Description: "Suggest photo search terms that return results, for when a first query comes back thin.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in struct {
		Query string `json:"query" jsonschema:"partial term"`
		Limit int    `json:"limit,omitempty" jsonschema:"default 10"`
	}) (*mcp.CallToolResult, map[string]any, error) {
		terms, err := s.client.PhotoAutocomplete(ctx, in.Query, in.Limit)
		if err != nil {
			return nil, nil, err
		}
		out := map[string]any{"terms": terms}
		return textResult(out), out, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "icons8_download_photo",
		Description: "Download a photo at a chosen pixel size and return the path. Omit width and height for the native resolution. " +
			"Passing them makes Icons8 resize server-side, which avoids shipping a 6000px original into a web build.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in struct {
		ID     string `json:"id" jsonschema:"photo id from a search result"`
		Width  int    `json:"width,omitempty" jsonschema:"target width in pixels; omit for native size"`
		Height int    `json:"height,omitempty" jsonschema:"target height in pixels; omit for native size"`
		Name   string `json:"name,omitempty" jsonschema:"filename stem"`
	}) (*mcp.CallToolResult, map[string]any, error) {
		if in.ID == "" {
			return nil, nil, fmt.Errorf("id is required")
		}
		dl, err := s.client.PhotoDownloadURL(ctx, in.ID, in.Width, in.Height)
		if err != nil {
			return nil, nil, err
		}
		data, ct, err := s.client.FetchSigned(ctx, dl.URL)
		if err != nil {
			return nil, nil, err
		}
		stem := in.Name
		if stem == "" {
			stem = "photo-" + in.ID
		}
		saved, err := s.store.Save("photos", fmt.Sprintf("%s-%dx%d", stem, dl.Width, dl.Height),
			assets.ExtFromContentType(ct), data, dl.URL)
		if err != nil {
			return nil, nil, err
		}
		out := map[string]any{"id": in.ID, "file": saved, "width": dl.Width, "height": dl.Height}
		return textResult(out), out, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "icons8_account",
		Description: "Report the signed-in Icons8 account, what its licence covers, and when the session token expires. Use it to diagnose auth problems.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
		if !s.sess.Authorized() {
			out := map[string]any{
				"authorized": false,
				"next_step":  "call icons8_authorize to sign in once; nothing else works until then",
				"asset_dir":  s.store.Root(),
			}
			return textResult(out), out, nil
		}
		acct, err := s.client.Account(ctx)
		if err != nil {
			return nil, nil, err
		}
		claims := s.sess.Claims()
		out := map[string]any{
			"authorized": true,
			"email":      acct.Email,
			"licence": map[string]any{
				"icons":   claims.License.Icons,
				"vectors": claims.License.Vectors,
				"photos":  claims.License.Photos,
				"sounds":  claims.License.Sounds,
				"expires": time.UnixMilli(claims.License.ExpireAt).UTC().Format(time.RFC3339),
			},
			"session_token_expires": claims.Expiry().UTC().Format(time.RFC3339),
			"asset_dir":             s.store.Root(),
			"recent_searches":       firstN(acct.History, 10),
		}
		return textResult(out), out, nil
	})
}

func firstN(list []string, n int) []string {
	if len(list) <= n {
		return list
	}
	return list[:n]
}
