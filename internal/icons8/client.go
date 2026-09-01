// Package icons8 is a typed client for the private APIs behind the Icons8 web
// app: icons, illustrations (Ouch), 3D models, and photos (Moose).
//
// The web app talks to these hosts directly over CORS, so no Cloudflare
// challenge applies to them. Only icons8.com's own HTML is gated, so
// the fast path is plain HTTP with the session's bearer token, and the browser
// is only needed to bootstrap or repair the session.
package icons8

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/torkay/better-icons8-mcp/internal/config"
	"github.com/torkay/better-icons8-mcp/internal/session"
	"golang.org/x/time/rate"
)

// Hosts. Discovered by recording the web app's own traffic; see docs/api.md.
const (
	HostIconSearch = "https://search-app.icons8.com"
	HostIconAPI    = "https://api-icons.icons8.com"
	HostIconImg    = "https://api-img.icons8.com"
	HostIconCDN    = "https://img.icons8.com"
	HostOuch       = "https://api-ouch.icons8.com"
	HostOuchSearch = "https://search-ouch-origin.icons8.com"
	HostPhotos     = "https://photos.icons8.com"
	HostBilling    = "https://api-icons.icons8.com/billing/v1/resources"
)

// APIError carries a non-2xx response so tools can report it verbatim rather
// than collapsing every failure into "request failed".
type APIError struct {
	Status int
	URL    string
	Body   string
	Code   string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("icons8 %d %s: %s (%s)", e.Status, e.URL, e.Body, e.Code)
	}
	return fmt.Sprintf("icons8 %d %s: %s", e.Status, e.URL, e.Body)
}

// Unauthorized reports whether the failure looks like an expired session rather
// than a bad request, which is the signal to refresh and retry.
func (e *APIError) Unauthorized() bool { return e.Status == 401 || e.Status == 403 }

type Client struct {
	cfg  *config.Config
	sess *session.Session
	http *http.Client

	limiter *rate.Limiter
	sem     chan struct{}

	// reauth, when set, re-acquires the session through a real browser. It is
	// wired up by the server so this package does not depend on the browser.
	reauth func(ctx context.Context) error
}

// ErrNotAuthorized is returned before any request is attempted when no Icons8
// session is stored. It is a distinct error because the fix is a one-off user
// action rather than anything a retry can help with.
var ErrNotAuthorized = errors.New(
	"not connected to an Icons8 account: run the icons8_authorize tool, " +
		"or `icons8-mcp auth` in a terminal, to sign in once")

func New(cfg *config.Config, sess *session.Session) *Client {
	return &Client{
		cfg:     cfg,
		sess:    sess,
		http:    &http.Client{Timeout: cfg.HTTPTimeout},
		limiter: rate.NewLimiter(rate.Limit(cfg.RequestsPerSecond), int(math.Max(1, cfg.RequestsPerSecond))),
		sem:     make(chan struct{}, cfg.MaxConcurrent),
	}
}

func (c *Client) Session() *session.Session { return c.sess }
func (c *Client) Config() *config.Config    { return c.cfg }

// SetReauth installs the browser-backed recovery path.
func (c *Client) SetReauth(fn func(ctx context.Context) error) { c.reauth = fn }

// GetJSON fetches url and decodes it into out, refreshing the session once if
// the first attempt is rejected as unauthorized.
func (c *Client) GetJSON(ctx context.Context, rawURL string, out any) error {
	body, _, err := c.get(ctx, rawURL, true)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode %s: %w (body: %s)", rawURL, err, truncate(string(body), 300))
	}
	return nil
}

// GetBytes fetches url as raw bytes and returns the response content type.
func (c *Client) GetBytes(ctx context.Context, rawURL string) ([]byte, string, error) {
	return c.get(ctx, rawURL, true)
}

// PostJSON sends a JSON body and decodes a JSON response.
func (c *Client) PostJSON(ctx context.Context, rawURL string, in, out any) error {
	payload, err := json.Marshal(in)
	if err != nil {
		return err
	}
	body, _, err := c.do(ctx, http.MethodPost, rawURL, payload, true)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

func (c *Client) get(ctx context.Context, rawURL string, allowRetry bool) ([]byte, string, error) {
	return c.do(ctx, http.MethodGet, rawURL, nil, allowRetry)
}

func (c *Client) do(ctx context.Context, method, rawURL string, payload []byte, allowRetry bool) ([]byte, string, error) {
	if !c.sess.Authorized() {
		return nil, "", ErrNotAuthorized
	}
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, "", err
	}
	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return nil, "", ctx.Err()
	}

	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter; Icons8 rate-limits by IP and a
			// tight retry loop is what turns a hiccup into a block.
			delay := time.Duration(1<<attempt) * 400 * time.Millisecond
			delay += time.Duration(rand.Int63n(int64(delay / 2)))
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, "", ctx.Err()
			}
		}

		var reqBody io.Reader
		if payload != nil {
			reqBody = bytes.NewReader(payload)
		}
		req, err := http.NewRequestWithContext(ctx, method, rawURL, reqBody)
		if err != nil {
			return nil, "", err
		}
		c.sess.Apply(req, c.cfg.UserAgent, c.cfg.Locale)
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		ct := resp.Header.Get("Content-Type")
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			return body, ct, nil
		case resp.StatusCode == 429 || resp.StatusCode >= 500:
			lastErr = &APIError{Status: resp.StatusCode, URL: rawURL, Body: truncate(string(body), 400)}
			continue
		default:
			apiErr := &APIError{Status: resp.StatusCode, URL: rawURL, Body: truncate(string(body), 400), Code: errCode(body)}
			// A rejected token is worth exactly one repair attempt.
			if allowRetry && apiErr.Unauthorized() && c.reauth != nil {
				if rerr := c.reauth(ctx); rerr == nil {
					return c.do(ctx, method, rawURL, payload, false)
				}
			}
			return nil, "", apiErr
		}
	}
	return nil, "", fmt.Errorf("%s %s: %w", method, rawURL, lastErr)
}

func errCode(body []byte) string {
	var e struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(body, &e)
	return e.Code
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// buildURL assembles host+path with non-empty query values only, so callers can
// pass optional filters as zero values and have them dropped.
func buildURL(host, path string, q map[string]string) string {
	v := url.Values{}
	for k, val := range q {
		if val != "" {
			v.Set(k, val)
		}
	}
	if len(v) == 0 {
		return host + path
	}
	return host + path + "?" + v.Encode()
}
