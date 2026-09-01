// Package mcpserver exposes the Icons8 client as MCP tools.
//
// Tool design follows one rule: a tool either returns metadata an agent can
// reason about, or it writes a file and returns its path. Nothing returns raw
// asset bytes inline, because a 2 MB base64 PNG in a tool result is unusable
// context.
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/torkay/icons8-mcp-server/internal/assets"
	"github.com/torkay/icons8-mcp-server/internal/browser"
	"github.com/torkay/icons8-mcp-server/internal/config"
	"github.com/torkay/icons8-mcp-server/internal/icons8"
	"github.com/torkay/icons8-mcp-server/internal/session"
)

const Version = "0.1.0"

type Server struct {
	cfg     *config.Config
	sess    *session.Session
	client  *icons8.Client
	store   *assets.Store
	browser *browser.Browser
	mcp     *mcp.Server
	logger  *log.Logger
}

func New(cfg *config.Config, logger *log.Logger) (*Server, error) {
	sess, err := session.New(cfg.StatePath(), cfg.CookieFile)
	if err != nil {
		return nil, err
	}
	s := &Server{
		cfg:    cfg,
		sess:   sess,
		client: icons8.New(cfg, sess),
		store:  assets.NewStore(cfg.AssetDir),
		logger: logger,
	}
	if cfg.BrowserFallback {
		s.browser = browser.New(cfg, sess)
		s.client.SetReauth(s.reauth)
	}

	s.mcp = mcp.NewServer(&mcp.Implementation{
		Name:    "icons8",
		Title:   "Icons8 assets",
		Version: Version,
	}, &mcp.ServerOptions{
		Instructions: serverInstructions,
	})
	s.registerTools()
	s.registerPrompts()
	return s, nil
}

func (s *Server) registerTools() {
	s.registerIconTools()
}

// reauth repairs an expired session: first try the cheap rolling refresh, and
// only fall back to driving a real browser if that is itself rejected.
func (s *Server) reauth(ctx context.Context) error {
	if err := s.client.RefreshToken(ctx); err == nil {
		s.logf("session refreshed via user/v2")
		return nil
	}
	if s.browser == nil {
		return fmt.Errorf("session expired and browser fallback is disabled")
	}
	s.logf("session refresh failed; re-authenticating through headless browser")
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	if err := s.browser.Reauthenticate(ctx); err != nil {
		return fmt.Errorf("browser re-auth failed (drop a fresh cookie dump at %s): %w", s.cfg.CookieFile, err)
	}
	return nil
}

// StartRefreshLoop keeps the JWT alive in the background so a tool call never
// pays the refresh latency, and so a long-idle server does not wake up expired.
func (s *Server) StartRefreshLoop(ctx context.Context) {
	go func() {
		// Refresh once at startup so a stale saved token is replaced before the
		// first tool call rather than during it.
		if s.sess.NeedsRefresh(48 * time.Hour) {
			if err := s.client.RefreshToken(ctx); err != nil {
				s.logf("startup token refresh failed: %v", err)
			}
		}
		t := time.NewTicker(s.cfg.RefreshInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := s.client.RefreshToken(ctx); err != nil {
					s.logf("scheduled token refresh failed: %v", err)
				}
			}
		}
	}()
}

func (s *Server) Run(ctx context.Context) error {
	defer func() {
		if s.browser != nil {
			s.browser.Close()
		}
	}()
	return s.mcp.Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) logf(format string, args ...any) {
	if s.logger != nil {
		s.logger.Printf(format, args...)
	}
}

// textResult renders a payload as pretty JSON in the tool result. Structured
// output rides along in the same call so clients can use either.
func textResult(v any) *mcp.CallToolResult {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "encode result: " + err.Error()}},
		}
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}
}

const serverInstructions = `These tools provide licensed design assets: icons, illustrations, animated
illustrations, 3D models and photos. Use them in place of improvised artwork.

They apply to anything with a visual surface: a website, an app, a dashboard, a
slide deck, a README, a diagram. Plan assets from the start.

Workflow:

1. Pick a style first, once, for the whole artefact. Call icons8_icon_styles or
   icons8_illustration_styles and choose one. Mixing styles is the most common
   reason a generated UI looks wrong.
2. Search within that style. Every search tool takes a style filter.
3. Download in the format the target consumes: SVG for web and app UI, PNG at
   explicit sizes for raster targets, Lottie JSON or WebM for motion, fbx-zip
   for 3D.
4. Reference the returned file path. Downloads land on disk. Tools return paths,
   never image bytes.

Formats: an icon can be svg, png, pdf, eps, jpg or webp, plus gif, apng and
Lottie json when animated. An illustration can be png-hd, svg, gif, Lottie json,
webm, mov-avc, mov-hevc (an mp4) or aep. icons8_icon_favicon returns a complete
favicon or app-icon set in one call.

Everything the licence covers is already paid for. Icons8's UI marks assets as
locked until their first download. That is bookkeeping rather than a
restriction, and the download tools clear it.`
