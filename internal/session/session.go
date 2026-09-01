// Package session owns the authenticated Icons8 identity: the JWT that every
// API call carries, the browser fingerprint the API expects alongside it, and
// the cookies that let a real browser re-acquire both when the JWT lapses.
//
// The JWT is bootstrapped from a browser cookie dump (the `i8token` cookie) and
// then kept alive by a rolling refresh: GET /user/v2 returns a freshly minted
// token, so as long as the server refreshes before expiry the session never
// needs the browser again.
package session

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Cookie mirrors the shape of a browser-extension cookie dump.
type Cookie struct {
	Domain   string  `json:"domain"`
	Name     string  `json:"name"`
	Path     string  `json:"path"`
	Value    string  `json:"value"`
	Secure   bool    `json:"secure"`
	HTTPOnly bool    `json:"httpOnly"`
	HostOnly bool    `json:"hostOnly"`
	Expires  float64 `json:"expirationDate"`
}

// Claims is the subset of the i8token JWT payload the server acts on.
type Claims struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	PublicAPIKey string `json:"publicApiKey"`
	Env          string `json:"env"`
	IssuedAt     int64  `json:"iat"`
	ExpiresAt    int64  `json:"exp"`
	License      struct {
		Icons          bool  `json:"icons"`
		Vectors        bool  `json:"vectors"`
		Photos         bool  `json:"photos"`
		Sounds         bool  `json:"sounds"`
		MCPIconAPI     bool  `json:"mcpIconApi"`
		MCPIconMetaAPI bool  `json:"mcpIconMetaApi"`
		ExpireAt       int64 `json:"expireAt"`
	} `json:"activeLicense"`
}

func (c Claims) Expiry() time.Time { return time.Unix(c.ExpiresAt, 0) }

// state is what survives a restart.
type state struct {
	Token       string    `json:"token"`
	Fingerprint string    `json:"fingerprint"`
	Cookies     []Cookie  `json:"cookies,omitempty"`
	RefreshedAt time.Time `json:"refreshed_at"`
}

type Session struct {
	mu    sync.RWMutex
	st    state
	claim Claims
	path  string
}

// New loads persisted state, falling back to the cookie dump for bootstrap.
// Passing an explicit cookieFile that is newer than the saved state re-seeds the
// token, so dropping in a fresh dump is always the way to recover.
//
// A session with no token is a valid result, not an error. The server has to
// start before it can be authorised, otherwise the MCP host reports it as a
// broken server rather than an unauthorised one, and there is no way in to fix
// it. Callers gate on Authorized.
func New(statePath, cookieFile string) (*Session, error) {
	s := &Session{path: statePath}
	_ = s.loadState()

	if raw, err := os.ReadFile(cookieFile); err == nil {
		var cookies []Cookie
		if err := json.Unmarshal(raw, &cookies); err != nil {
			return nil, fmt.Errorf("parse cookie dump %s: %w", cookieFile, err)
		}
		tok := tokenFromCookies(cookies)
		if tok != "" {
			if c, err := ParseClaims(tok); err == nil {
				// Only adopt the dump's token if it beats what we already hold.
				if s.st.Token == "" || c.ExpiresAt > s.claim.ExpiresAt {
					s.st.Token, s.claim = tok, c
				}
			}
		}
		if len(cookies) > 0 {
			s.st.Cookies = cookies
		}
	}

	if s.st.Fingerprint == "" {
		s.st.Fingerprint = newFingerprint()
	}
	return s, s.save()
}

// tokenFromCookies pulls i8token out of a dump, preferring the longest-lived one
// when a dump contains duplicates across domains.
func tokenFromCookies(cookies []Cookie) string {
	var best Cookie
	for _, c := range cookies {
		if c.Name != "i8token" || c.Value == "" {
			continue
		}
		if best.Value == "" || c.Expires > best.Expires {
			best = c
		}
	}
	return best.Value
}

// ParseClaims decodes a JWT payload without verifying the signature. The server
// is a client of this token, not its issuer, so it only needs the metadata.
func ParseClaims(token string) (Claims, error) {
	var c Claims
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return c, fmt.Errorf("malformed JWT (%d segments)", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return c, fmt.Errorf("decode JWT payload: %w", err)
	}
	if err := json.Unmarshal(payload, &c); err != nil {
		return c, fmt.Errorf("parse JWT payload: %w", err)
	}
	return c, nil
}

func newFingerprint() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Session) Token() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.st.Token
}

// Authorized reports whether there is a token to act with. It says nothing
// about whether Icons8 still accepts that token; only a request can settle that.
func (s *Session) Authorized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.st.Token != ""
}

func (s *Session) Fingerprint() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.st.Fingerprint
}

func (s *Session) Claims() Claims {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.claim
}

// PublicAPIKey is the `token=` query parameter the download hosts want. It is
// not a secret in the JWT sense. It is the account's public download key.
func (s *Session) PublicAPIKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.claim.PublicAPIKey
}

func (s *Session) Cookies() []Cookie {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Cookie, len(s.st.Cookies))
	copy(out, s.st.Cookies)
	return out
}

// NeedsRefresh reports whether the token is close enough to expiry that the next
// call should renew it first.
func (s *Session) NeedsRefresh(window time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.claim.ExpiresAt == 0 {
		return true
	}
	return time.Until(time.Unix(s.claim.ExpiresAt, 0)) < window
}

// SetToken adopts a newly minted token, ignoring one that is not an improvement
// so a stale response cannot roll the session backwards.
func (s *Session) SetToken(token string) error {
	c, err := ParseClaims(token)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if c.ExpiresAt <= s.claim.ExpiresAt {
		s.mu.Unlock()
		return nil
	}
	s.st.Token, s.claim, s.st.RefreshedAt = token, c, time.Now()
	s.mu.Unlock()
	return s.save()
}

// SetCookies replaces the stored cookie jar, used after a browser re-auth.
func (s *Session) SetCookies(cookies []Cookie) error {
	if len(cookies) == 0 {
		return nil
	}
	s.mu.Lock()
	s.st.Cookies = cookies
	if tok := tokenFromCookies(cookies); tok != "" {
		if c, err := ParseClaims(tok); err == nil && c.ExpiresAt > s.claim.ExpiresAt {
			s.st.Token, s.claim, s.st.RefreshedAt = tok, c, time.Now()
		}
	}
	s.mu.Unlock()
	return s.save()
}

// Apply stamps a request with everything Icons8 checks: bearer token,
// fingerprint, and the browser-ish headers the API gateway expects.
func (s *Session) Apply(req *http.Request, userAgent, locale string) {
	s.mu.RLock()
	token, fp := s.st.Token, s.st.Fingerprint
	s.mu.RUnlock()

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Icons8-Fingerprint", fp)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", locale+",en;q=0.9")
	req.Header.Set("Origin", "https://icons8.com")
	req.Header.Set("Referer", "https://icons8.com/")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Sec-CH-UA", `"Chromium";v="141", "Not?A_Brand";v="24", "Google Chrome";v="141"`)
	req.Header.Set("Sec-CH-UA-Mobile", "?0")
	req.Header.Set("Sec-CH-UA-Platform", `"Linux"`)
}

func (s *Session) loadState() error {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	var st state
	if err := json.Unmarshal(raw, &st); err != nil {
		return err
	}
	c, err := ParseClaims(st.Token)
	if err != nil {
		return err
	}
	s.st, s.claim = st, c
	return nil
}

func (s *Session) save() error {
	s.mu.RLock()
	raw, err := json.MarshalIndent(s.st, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
