// Command smoke drives the built MCP server over stdio exactly as a host would,
// calling every tool against the live Icons8 API and asserting the results.
//
//	go run ./cmd/smoke -bin ./dist/icons8-mcp
//
// It is an integration check, not a unit test, so it lives outside `go test`.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type check struct {
	name string
	run  func(ctx context.Context, cs *mcp.ClientSession) (string, error)
}

var (
	iconID         string
	animatedIconID string
	illustrationID string
	animatedIllID  string
	modelID        string
	photoID        string
	iconStyle      string
	illStyle       string
	packStyle      string
	packCategory   string
)

func main() {
	bin := flag.String("bin", "./dist/icons8-mcp", "path to the server binary")
	only := flag.String("only", "", "run only checks whose name contains this")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "icons8-smoke", Version: "1"}, nil)
	cmd := exec.Command(*bin)
	cmd.Stderr = os.Stderr
	cs, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		fatal("connect: %v", err)
	}
	defer cs.Close()

	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		fatal("list tools: %v", err)
	}
	fmt.Printf("server exposes %d tools\n\n", len(tools.Tools))

	pass, fail := 0, 0
	for _, c := range checks() {
		if *only != "" && !strings.Contains(c.name, *only) {
			continue
		}
		start := time.Now()
		note, err := c.run(ctx, cs)
		took := time.Since(start).Round(time.Millisecond)
		if err != nil {
			fail++
			fmt.Printf("FAIL  %-34s %8s  %v\n", c.name, took, err)
			continue
		}
		pass++
		fmt.Printf("ok    %-34s %8s  %s\n", c.name, took, note)
	}

	fmt.Printf("\n%d passed, %d failed\n", pass, fail)
	if fail > 0 {
		os.Exit(1)
	}
}

func checks() []check {
	return []check{
		{"account", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_account", map[string]any{})
			if err != nil {
				return "", err
			}
			email, _ := m["email"].(string)
			if email == "" {
				return "", fmt.Errorf("no email in response")
			}
			return email, nil
		}},

		{"icon_styles", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_icon_styles", map[string]any{"term": "rocket"})
			if err != nil {
				return "", err
			}
			styles, _ := m["styles"].([]any)
			if len(styles) < 5 {
				return "", fmt.Errorf("expected many styles, got %d", len(styles))
			}
			for _, s := range styles {
				e := s.(map[string]any)
				if e["enabled"] == true {
					iconStyle, _ = e["value"].(string)
					break
				}
			}
			return fmt.Sprintf("%d styles, picked %q", len(styles), iconStyle), nil
		}},

		{"search_icons", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			args := map[string]any{"term": "rocket", "amount": 12}
			if iconStyle != "" {
				args["style"] = iconStyle
			}
			m, err := call(ctx, cs, "icons8_search_icons", args)
			if err != nil {
				return "", err
			}
			list, _ := m["icons"].([]any)
			if len(list) == 0 {
				return "", fmt.Errorf("no icons returned")
			}
			for _, it := range list {
				e := it.(map[string]any)
				if iconID == "" {
					iconID, _ = e["id"].(string)
					packStyle, _ = e["style"].(string)
					if cat, _ := e["category"].(string); cat != "" {
						packCategory = strings.ToLower(strings.ReplaceAll(cat, " ", "-"))
					}
				}
				if e["is_animated"] == true && animatedIconID == "" {
					animatedIconID, _ = e["id"].(string)
				}
			}
			return fmt.Sprintf("%v total, first id %s", m["total"], iconID), nil
		}},

		{"search_icons_animated", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_search_icons", map[string]any{"term": "rocket", "animated": "y", "amount": 5})
			if err != nil {
				return "", err
			}
			list, _ := m["icons"].([]any)
			if len(list) == 0 {
				return "", fmt.Errorf("no animated icons returned")
			}
			if animatedIconID == "" {
				animatedIconID, _ = list[0].(map[string]any)["id"].(string)
			}
			return fmt.Sprintf("%d animated, using %s", len(list), animatedIconID), nil
		}},

		{"icon_variants", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_icon_variants", map[string]any{"id": iconID})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%v variants", m["count"]), nil
		}},

		{"similar_icons", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_similar_icons", map[string]any{"id": iconID, "limit": 8})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%v similar", m["count"]), nil
		}},

		{"icon_pack", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			// Use the style and category that a real result actually has, so the
			// check tests the endpoint rather than a guessed combination.
			if packStyle == "" || packCategory == "" {
				return "", fmt.Errorf("no style/category captured from search")
			}
			m, err := call(ctx, cs, "icons8_icon_pack", map[string]any{
				"style": packStyle, "category": packCategory, "amount": 10,
			})
			if err != nil {
				return "", err
			}
			list, _ := m["icons"].([]any)
			if len(list) == 0 {
				return "", fmt.Errorf("pack %s/%s returned nothing", packStyle, packCategory)
			}
			return fmt.Sprintf("%d in %s/%s", len(list), packStyle, packCategory), nil
		}},

		{"download_icon_svg_png", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_download_icon", map[string]any{
				"id": iconID, "formats": []string{"svg", "png", "pdf"}, "sizes": []int{64, 256},
			})
			if err != nil {
				return "", err
			}
			files, _ := m["files"].([]any)
			if len(files) < 3 {
				return "", fmt.Errorf("expected >=3 files, got %d", len(files))
			}
			return describeFiles(files)
		}},

		{"download_icon_recolour", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_download_icon", map[string]any{
				"id": iconID, "formats": []string{"svg"}, "color": "FF6B00", "name": "rocket-orange",
			})
			if err != nil {
				return "", err
			}
			files, _ := m["files"].([]any)
			return describeFiles(files)
		}},

		{"download_icon_animated", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			if animatedIconID == "" {
				return "skipped (no animated icon found)", nil
			}
			m, err := call(ctx, cs, "icons8_download_icon", map[string]any{
				"id": animatedIconID, "formats": []string{"gif", "json", "apng"}, "sizes": []int{100},
			})
			if err != nil {
				return "", err
			}
			files, _ := m["files"].([]any)
			if len(files) < 3 {
				return "", fmt.Errorf("expected gif+json+apng, got %d files", len(files))
			}
			return describeFiles(files)
		}},

		{"icon_favicon_set", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_icon_favicon", map[string]any{
				"id": iconID, "platform": "favicon", "ico": true, "name": "smoke-favicon",
			})
			if err != nil {
				return "", err
			}
			files, _ := m["files"].([]any)
			hasICO := false
			for _, f := range files {
				if strings.HasSuffix(f.(map[string]any)["path"].(string), ".ico") {
					hasICO = true
				}
			}
			if !hasICO {
				return "", fmt.Errorf("no .ico produced")
			}
			return fmt.Sprintf("%d files incl. .ico", len(files)), nil
		}},

		{"icon_embed", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_icon_embed", map[string]any{"id": iconID, "size": 64})
			if err != nil {
				return "", err
			}
			svg, _ := m["svg_markup"].(string)
			b64, _ := m["base64_png"].(string)
			if !strings.Contains(svg, "<svg") || !strings.HasPrefix(b64, "data:image/png;base64,") {
				return "", fmt.Errorf("embed forms look wrong")
			}
			return fmt.Sprintf("svg %d bytes, data-uri %d bytes", len(svg), len(b64)), nil
		}},

		{"check_unlock", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_check_unlock", map[string]any{"ids": []string{iconID}})
			if err != nil {
				return "", err
			}
			unlocked, _ := m["unlocked"].([]any)
			if len(unlocked) == 0 {
				return "", fmt.Errorf("icon %s should be unlocked after download", iconID)
			}
			return fmt.Sprintf("%s unlocked", iconID), nil
		}},

		{"illustration_styles", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_illustration_styles", map[string]any{})
			if err != nil {
				return "", err
			}
			list, _ := m["styles"].([]any)
			if len(list) < 5 {
				return "", fmt.Errorf("expected many styles, got %d", len(list))
			}
			if illStyle == "" {
				illStyle, _ = list[0].(map[string]any)["slug"].(string)
			}
			return fmt.Sprintf("%d styles, picked %q", len(list), illStyle), nil
		}},

		{"search_illustrations", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_search_illustrations", map[string]any{"query": "rocket", "per_page": 10})
			if err != nil {
				return "", err
			}
			list, _ := m["illustrations"].([]any)
			if len(list) == 0 {
				return "", fmt.Errorf("no illustrations")
			}
			first := list[0].(map[string]any)
			illustrationID, _ = first["id"].(string)
			// Prefer a style that this query demonstrably has results in.
			if sl, _ := first["style_slug"].(string); sl != "" {
				illStyle = sl
			}
			return fmt.Sprintf("%v total, first %s", m["total"], illustrationID), nil
		}},

		{"search_illustrations_styled", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_search_illustrations", map[string]any{
				"query": "rocket", "style": illStyle, "per_page": 5,
			})
			if err != nil {
				return "", err
			}
			list, _ := m["illustrations"].([]any)
			for _, it := range list {
				if got, _ := it.(map[string]any)["style_slug"].(string); got != "" && got != illStyle {
					return "", fmt.Errorf("style filter leaked: asked %q, got %q", illStyle, got)
				}
			}
			return fmt.Sprintf("%d in style %q", len(list), illStyle), nil
		}},

		{"search_illustrations_animated", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_search_illustrations", map[string]any{
				"query": "rocket", "animated": "y", "per_page": 10,
			})
			if err != nil {
				return "", err
			}
			list, _ := m["illustrations"].([]any)
			if len(list) == 0 {
				return "", fmt.Errorf("no animated illustrations")
			}
			for _, it := range list {
				e := it.(map[string]any)
				if e["animated"] == true {
					animatedIllID, _ = e["id"].(string)
					break
				}
			}
			if animatedIllID == "" {
				return "", fmt.Errorf("animated filter returned %d items, none flagged animated", len(list))
			}
			return fmt.Sprintf("%d animated, using %s", len(list), animatedIllID), nil
		}},

		{"search_3d_models", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_search_illustrations", map[string]any{
				"query": "rocket", "models": true, "per_page": 5,
			})
			if err != nil {
				return "", err
			}
			list, _ := m["illustrations"].([]any)
			if len(list) == 0 {
				return "", fmt.Errorf("no 3D models")
			}
			modelID, _ = list[0].(map[string]any)["id"].(string)
			return fmt.Sprintf("%v models, first %s", m["total"], modelID), nil
		}},

		{"illustration_detail", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_illustration", map[string]any{"id": illustrationID})
			if err != nil {
				return "", err
			}
			formats, _ := m["formats"].([]any)
			if len(formats) == 0 {
				return "", fmt.Errorf("no formats listed")
			}
			names := make([]string, 0, len(formats))
			for _, f := range formats {
				names = append(names, f.(map[string]any)["format"].(string))
			}
			return strings.Join(names, ","), nil
		}},

		{"similar_illustrations", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_similar_illustrations", map[string]any{"id": illustrationID, "per_page": 6})
			if err != nil {
				return "", err
			}
			list, _ := m["illustrations"].([]any)
			return fmt.Sprintf("%d similar", len(list)), nil
		}},

		{"download_illustration", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_download_illustration", map[string]any{
				"id": illustrationID, "formats": []string{"png-hd", "svg"},
			})
			if err != nil {
				return "", err
			}
			files, _ := m["files"].([]any)
			if len(files) == 0 {
				return "", fmt.Errorf("no files")
			}
			return describeFiles(files)
		}},

		{"download_animated_illustration", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			if animatedIllID == "" {
				return "skipped (none found)", nil
			}
			m, err := call(ctx, cs, "icons8_download_illustration", map[string]any{
				"id": animatedIllID, "formats": []string{"gif", "json", "webm", "mov-avc", "mov-hevc"},
			})
			if err != nil {
				return "", err
			}
			files, _ := m["files"].([]any)
			if len(files) < 3 {
				return "", fmt.Errorf("expected several motion formats, got %d", len(files))
			}
			return describeFiles(files)
		}},

		{"download_3d_model", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			if modelID == "" {
				return "skipped (none found)", nil
			}
			m, err := call(ctx, cs, "icons8_download_illustration", map[string]any{
				"id": modelID, "formats": []string{"fbx", "glb", "png-hd"},
			})
			if err != nil {
				return "", err
			}
			files, _ := m["files"].([]any)
			if len(files) < 2 {
				return "", fmt.Errorf("expected 3D source files plus a preview, got %d (skipped: %v)", len(files), m["skipped"])
			}
			return describeFiles(files)
		}},

		{"search_photos", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_search_photos", map[string]any{"query": "office desk", "per_page": 6})
			if err != nil {
				return "", err
			}
			list, _ := m["photos"].([]any)
			if len(list) == 0 {
				return "", fmt.Errorf("no photos")
			}
			photoID, _ = list[0].(map[string]any)["id"].(string)
			return fmt.Sprintf("%v total, first %s", m["total"], photoID), nil
		}},

		{"search_photos_transparent", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_search_photos", map[string]any{
				"query": "person", "filter": "transparent", "per_page": 4,
			})
			if err != nil {
				return "", err
			}
			list, _ := m["photos"].([]any)
			return fmt.Sprintf("%d cut-outs", len(list)), nil
		}},

		{"photo_suggest", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_photo_suggest", map[string]any{"query": "offi"})
			if err != nil {
				return "", err
			}
			terms, _ := m["terms"].([]any)
			if len(terms) == 0 {
				return "", fmt.Errorf("no suggestions")
			}
			return fmt.Sprintf("%d terms", len(terms)), nil
		}},

		{"download_photo", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			m, err := call(ctx, cs, "icons8_download_photo", map[string]any{
				"id": photoID, "width": 1280, "height": 853,
			})
			if err != nil {
				return "", err
			}
			f, _ := m["file"].(map[string]any)
			return fmt.Sprintf("%v (%v bytes)", f["path"], f["bytes"]), nil
		}},

		{"prompt_asset_plan", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			res, err := cs.GetPrompt(ctx, &mcp.GetPromptParams{
				Name:      "icons8_asset_plan",
				Arguments: map[string]string{"project": "a landing page for a payments API"},
			})
			if err != nil {
				return "", err
			}
			if len(res.Messages) == 0 {
				return "", fmt.Errorf("prompt returned no messages")
			}
			return fmt.Sprintf("%d message(s)", len(res.Messages)), nil
		}},

		{"error_is_actionable", func(ctx context.Context, cs *mcp.ClientSession) (string, error) {
			// A deliberately bad format must surface a clear message, not a panic
			// or an opaque failure.
			res, err := cs.CallTool(ctx, &mcp.CallToolParams{
				Name:      "icons8_download_icon",
				Arguments: map[string]any{"id": iconID, "formats": []string{"tiff"}},
			})
			if err == nil && !res.IsError {
				return "", fmt.Errorf("expected an error for an unsupported format")
			}
			msg := errText(res, err)
			if !strings.Contains(strings.ToLower(msg), "format") {
				return "", fmt.Errorf("error does not mention the format: %s", msg)
			}
			return "clear error surfaced", nil
		}},
	}
}

func call(ctx context.Context, cs *mcp.ClientSession, name string, args map[string]any) (map[string]any, error) {
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return nil, err
	}
	if res.IsError {
		return nil, fmt.Errorf("%s", errText(res, nil))
	}
	if len(res.Content) == 0 {
		return nil, fmt.Errorf("%s returned no content", name)
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		return nil, fmt.Errorf("%s returned %T, want text", name, res.Content[0])
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(text.Text), &m); err != nil {
		return nil, fmt.Errorf("%s returned non-JSON: %s", name, truncate(text.Text, 200))
	}
	return m, nil
}

func errText(res *mcp.CallToolResult, err error) string {
	if err != nil {
		return err.Error()
	}
	if res == nil {
		return "<nil result>"
	}
	var parts []string
	for _, c := range res.Content {
		if t, ok := c.(*mcp.TextContent); ok {
			parts = append(parts, t.Text)
		}
	}
	return strings.Join(parts, " ")
}

func describeFiles(files []any) (string, error) {
	var parts []string
	for _, f := range files {
		m := f.(map[string]any)
		path, _ := m["path"].(string)
		size, _ := m["bytes"].(float64)
		if size <= 0 {
			return "", fmt.Errorf("%s is empty", path)
		}
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("stat %s: %w", path, err)
		}
		parts = append(parts, fmt.Sprintf("%s(%.0fB)", trimPath(path), size))
	}
	return strings.Join(parts, " "), nil
}

func trimPath(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
