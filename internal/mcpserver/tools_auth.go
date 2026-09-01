package mcpserver

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerAuthTools() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "icons8_authorize",
		Description: "Connect this server to an Icons8 account. Opens a browser window at the Icons8 sign-in page and " +
			"waits for the user to log in, then stores the session on this machine so it does not have to be done again. " +
			"Call this when any other tool reports that it is not connected. Tell the user a window is about to open and " +
			"that they need to sign in there.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
		if s.sess.Authorized() {
			claims := s.sess.Claims()
			out := map[string]any{
				"authorized": true,
				"email":      claims.Email,
				"note":       "already connected; nothing to do",
				"expires":    claims.Expiry().UTC().Format(time.RFC3339),
			}
			return textResult(out), out, nil
		}
		if s.browser == nil {
			return nil, nil, fmt.Errorf(
				"the browser is disabled (ICONS8_BROWSER_FALLBACK=0), so sign-in cannot open a window; " +
					"export cookies for icons8.com and run `icons8-mcp import <file>`")
		}

		ctx, cancel := context.WithTimeout(ctx, s.cfg.AuthTimeout+time.Minute)
		defer cancel()

		claims, err := s.browser.Authorize(ctx)
		if err != nil {
			return nil, nil, err
		}
		s.logf("authorized as %s", claims.Email)

		out := map[string]any{
			"authorized": true,
			"email":      claims.Email,
			"licence": map[string]any{
				"icons":   claims.License.Icons,
				"vectors": claims.License.Vectors,
				"photos":  claims.License.Photos,
				"sounds":  claims.License.Sounds,
			},
			"session_stored": s.cfg.StatePath(),
			"expires":        claims.Expiry().UTC().Format(time.RFC3339),
			"note": "the session refreshes itself from here on, so this is a one-off. " +
				"Every other tool is now available.",
		}
		return textResult(out), out, nil
	})
}
