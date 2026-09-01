// Package browser drives a stealth headless Chrome. It is the fallback path:
// everything the MCP server does day to day runs over plain HTTP, and the
// browser only comes out when the session needs repairing or a page is behind
// Cloudflare's managed challenge.
//
// go-rod plus go-rod/stealth clears icons8.com's Cloudflare challenge without a
// solver: the challenge is a browser-integrity check, and a real Chromium with
// the automation give-aways patched out passes it.
package browser

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/torkay/better-icons8-mcp/internal/config"
	"github.com/torkay/better-icons8-mcp/internal/session"
)

type Browser struct {
	cfg  *config.Config
	sess *session.Session

	mu       sync.Mutex
	launcher *launcher.Launcher
	browser  *rod.Browser
	lastUse  time.Time
}

func New(cfg *config.Config, sess *session.Session) *Browser {
	return &Browser{cfg: cfg, sess: sess}
}

// ensure lazily starts Chrome. The instance is kept warm between calls because
// the cold start (and the Cloudflare clearance that comes with it) is the
// expensive part.
func (b *Browser) ensure() (*rod.Browser, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.browser != nil {
		b.lastUse = time.Now()
		return b.browser, nil
	}

	l := launcher.New().
		Headless(!b.cfg.Headful).
		Set("disable-blink-features", "AutomationControlled").
		Set("no-sandbox").
		Set("disable-dev-shm-usage").
		Set("disable-gpu").
		Set("lang", b.cfg.Locale+",en").
		Set("window-size", "1440,900")

	ctrlURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch chrome: %w", err)
	}
	br := rod.New().ControlURL(ctrlURL)
	if err := br.Connect(); err != nil {
		l.Cleanup()
		return nil, fmt.Errorf("connect chrome: %w", err)
	}
	if err := b.applyCookies(br); err != nil {
		return nil, err
	}
	b.launcher, b.browser, b.lastUse = l, br, time.Now()
	return br, nil
}

func (b *Browser) applyCookies(br *rod.Browser) error {
	cookies := b.sess.Cookies()
	if len(cookies) == 0 {
		return nil
	}
	params := make([]*proto.NetworkCookieParam, 0, len(cookies))
	for _, c := range cookies {
		path := c.Path
		if path == "" {
			path = "/"
		}
		p := &proto.NetworkCookieParam{
			Name: c.Name, Value: c.Value, Domain: c.Domain, Path: path,
			Secure: c.Secure, HTTPOnly: c.HTTPOnly,
		}
		if c.Expires > 0 {
			p.Expires = proto.TimeSinceEpoch(c.Expires)
		}
		params = append(params, p)
	}
	// Keep the session's live token authoritative over whatever the dump held.
	if tok := b.sess.Token(); tok != "" {
		params = append(params, &proto.NetworkCookieParam{
			Name: "i8token", Value: tok, Domain: "icons8.com", Path: "/",
			Expires: proto.TimeSinceEpoch(float64(time.Now().Add(24 * time.Hour).Unix())),
		})
	}
	return br.SetCookies(params)
}

func (b *Browser) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.browser != nil {
		_ = b.browser.Close()
		b.browser = nil
	}
	if b.launcher != nil {
		b.launcher.Cleanup()
		b.launcher = nil
	}
}

// Page opens a stealth page with a browser-consistent UA override.
func (b *Browser) Page(ctx context.Context) (*rod.Page, error) {
	br, err := b.ensure()
	if err != nil {
		return nil, err
	}
	page, err := stealth.Page(br.Context(ctx))
	if err != nil {
		return nil, fmt.Errorf("open page: %w", err)
	}
	err = page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent:      b.cfg.UserAgent,
		AcceptLanguage: b.cfg.Locale + ",en;q=0.9",
		Platform:       "Linux x86_64",
	})
	if err != nil {
		_ = page.Close()
		return nil, err
	}
	return page, nil
}

// Open navigates to a URL and waits for the SPA to render, retrying transient
// Chrome network errors and Cloudflare interstitials.
func (b *Browser) Open(ctx context.Context, target string, settle time.Duration) (*rod.Page, error) {
	page, err := b.Page(ctx)
	if err != nil {
		return nil, err
	}
	if settle <= 0 {
		settle = 8 * time.Second
	}
	var lastState string
	for attempt := 1; attempt <= 3; attempt++ {
		if err := page.Timeout(90 * time.Second).Navigate(target); err != nil {
			lastState = err.Error()
		}
		_ = page.Timeout(90 * time.Second).WaitLoad()
		time.Sleep(settle)

		res, err := page.Eval(`() => ({ title: document.title, controls: document.querySelectorAll('button,a').length })`)
		if err != nil {
			lastState = err.Error()
			continue
		}
		title := res.Value.Get("title").Str()
		// "Just a moment..." is Cloudflare's interstitial; it clears itself.
		if strings.Contains(title, "Just a moment") {
			lastState = "cloudflare challenge"
			time.Sleep(6 * time.Second)
			continue
		}
		if res.Value.Get("controls").Int() > 20 {
			return page, nil
		}
		lastState = fmt.Sprintf("page rendered %d controls, title %q", res.Value.Get("controls").Int(), title)
		time.Sleep(time.Duration(attempt) * 2 * time.Second)
	}
	_ = page.Close()
	return nil, fmt.Errorf("could not load %s: %s", target, lastState)
}

// Authorize opens a visible browser at the Icons8 sign-in page, waits for the
// user to log in, then stores the session it produces. This is the whole setup
// step: no cookie export, no extension, no pasting a token.
//
// It polls rather than watching for a navigation because Icons8 signs in
// several ways (email, Google, Apple, GitHub) and they do not share a
// completion event. The i8token cookie appearing with readable claims is the
// one signal common to all of them.
func (b *Browser) Authorize(ctx context.Context) (session.Claims, error) {
	var zero session.Claims
	if err := displayAvailable(); err != nil {
		return zero, err
	}

	// A separate launcher from the warm fallback instance: this one is visible,
	// lives as long as the user needs, and is thrown away afterwards.
	l := launcher.New().
		Headless(false).
		Set("disable-blink-features", "AutomationControlled").
		Set("no-first-run").
		Set("no-default-browser-check").
		Set("window-size", "1180,900")
	ctrlURL, err := l.Launch()
	if err != nil {
		return zero, fmt.Errorf("open a browser to sign in: %w", err)
	}
	defer l.Cleanup()

	br := rod.New().ControlURL(ctrlURL)
	if err := br.Connect(); err != nil {
		return zero, fmt.Errorf("connect to the sign-in browser: %w", err)
	}
	defer br.Close()

	page, err := stealth.Page(br.Context(ctx))
	if err != nil {
		return zero, fmt.Errorf("open the sign-in page: %w", err)
	}
	if err := page.Navigate("https://icons8.com/login"); err != nil {
		return zero, fmt.Errorf("load the sign-in page: %w", err)
	}

	deadline := time.Now().Add(b.cfg.AuthTimeout)
	for {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}
		if time.Now().After(deadline) {
			return zero, fmt.Errorf("timed out after %s waiting for sign-in", b.cfg.AuthTimeout)
		}

		cookies, err := page.Cookies([]string{"https://icons8.com"})
		if err == nil {
			for _, c := range cookies {
				if c.Name != "i8token" || c.Value == "" {
					continue
				}
				claims, err := session.ParseClaims(c.Value)
				if err != nil || claims.Email == "" {
					continue
				}
				if err := b.sess.SetToken(c.Value); err != nil {
					return zero, err
				}
				out := make([]session.Cookie, 0, len(cookies))
				for _, k := range cookies {
					out = append(out, session.Cookie{
						Domain: k.Domain, Name: k.Name, Path: k.Path, Value: k.Value,
						Secure: k.Secure, HTTPOnly: k.HTTPOnly, Expires: float64(k.Expires),
					})
				}
				if err := b.sess.SetCookies(out); err != nil {
					return zero, err
				}
				return claims, nil
			}
		}
		time.Sleep(time.Second)
	}
}

// displayAvailable rejects the headful path early on a machine with no display,
// where Chrome would fail with something far less useful than this.
func displayAvailable() error {
	switch runtime.GOOS {
	case "darwin", "windows":
		return nil
	}
	if os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "" {
		return nil
	}
	return fmt.Errorf("no display available, so a sign-in window cannot be opened; " +
		"sign in on a desktop machine, or export cookies for icons8.com and run `icons8-mcp import <file>`")
}

// Reauthenticate loads icons8.com in a real browser and harvests the session
// cookies it ends up with. This recovers a session whose JWT has fully expired,
// provided the underlying login cookies are still valid.
func (b *Browser) Reauthenticate(ctx context.Context) error {
	page, err := b.Open(ctx, "https://icons8.com/icons", 10*time.Second)
	if err != nil {
		return err
	}
	defer page.Close()

	cookies, err := page.Cookies([]string{"https://icons8.com", "https://api-icons.icons8.com"})
	if err != nil {
		return fmt.Errorf("read cookies: %w", err)
	}
	out := make([]session.Cookie, 0, len(cookies))
	for _, c := range cookies {
		out = append(out, session.Cookie{
			Domain: c.Domain, Name: c.Name, Path: c.Path, Value: c.Value,
			Secure: c.Secure, HTTPOnly: c.HTTPOnly, Expires: float64(c.Expires),
		})
	}
	if err := b.sess.SetCookies(out); err != nil {
		return err
	}
	// The page also stashes the token where the SPA reads it; take that too if
	// the cookie jar did not carry a fresher one.
	if v, err := page.Eval(`() => localStorage.getItem('i8token') || localStorage.getItem('token') || ''`); err == nil {
		if tok := v.Value.Str(); strings.Count(tok, ".") == 2 {
			_ = b.sess.SetToken(tok)
		}
	}
	return nil
}
