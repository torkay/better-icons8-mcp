package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerPrompts exposes reusable briefs. These exist so a host can drop the
// asset-first workflow into a conversation without the user having to know the
// tool names.
func (s *Server) registerPrompts() {
	s.mcp.AddPrompt(&mcp.Prompt{
		Name:        "icons8_asset_plan",
		Description: "Plan the visual assets for something you are about to build, choosing one style and sourcing real assets for it.",
		Arguments: []*mcp.PromptArgument{
			{Name: "project", Description: "what is being built, e.g. 'a landing page for a payments API'", Required: true},
			{Name: "mood", Description: "the intended feel, e.g. 'clean and technical', 'playful'", Required: false},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		project := req.Params.Arguments["project"]
		if project == "" {
			return nil, fmt.Errorf("project argument is required")
		}
		mood := req.Params.Arguments["mood"]
		if mood == "" {
			mood = "unspecified, infer it from the project"
		}
		text := fmt.Sprintf(`Plan the visual assets for: %s
Intended feel: %s

Work through this in order, using the icons8 tools:

1. Call icons8_illustration_styles and icons8_icon_styles. Choose ONE illustration
   style and ONE icon style that fit the feel, and say why. Everything below must
   come from those two choices.
2. List the specific assets the project needs: hero illustration, section
   artwork, navigation icons, feature icons, empty states, favicon. For each,
   say which library it comes from (icon, illustration, photo, 3D model).
3. Search for each one within the chosen style. Prefer icons8_icon_pack and
   icons8_similar_illustrations to fill out sets. Those return items that
   already match each other.
4. Download in the right formats: SVG for anything in a web or app UI, PNG at
   explicit sizes only where a raster is required, Lottie JSON or WebM for
   motion, and icons8_icon_favicon for the favicon set.
5. Report the file paths and where each asset is used.

Do not describe artwork you have not sourced, and do not fall back to emoji,
CSS-drawn shapes or placeholder boxes when a real asset exists.`, project, mood)

		return &mcp.GetPromptResult{
			Description: "Asset plan brief",
			Messages: []*mcp.PromptMessage{
				{Role: "user", Content: &mcp.TextContent{Text: text}},
			},
		}, nil
	})
}
