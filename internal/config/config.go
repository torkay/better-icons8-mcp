// Package config resolves where the server keeps its state and assets, and how
// it is allowed to talk to Icons8.
package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	// StateDir holds the session state file (refreshed JWT, fingerprint).
	StateDir string
	// AssetDir is where downloaded assets land. Tools return paths under it.
	AssetDir string
	// CookieFile is the browser cookie dump used to bootstrap the session.
	CookieFile string

	UserAgent string
	Locale    string
	Region    string

	// RequestsPerSecond caps outbound calls so a burst of tool calls does not
	// look like scraping.
	RequestsPerSecond float64
	MaxConcurrent     int
	HTTPTimeout       time.Duration

	// RefreshInterval is how often the rolling JWT refresh runs. The token Icons8
	// mints lives ~10 days, so this is deliberately conservative.
	RefreshInterval time.Duration

	// BrowserFallback enables the headless-Chrome path for anything the plain
	// HTTP client cannot do (Cloudflare-gated pages, re-auth).
	BrowserFallback bool
	Headful         bool
}

const DefaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"

func Load() *Config {
	home, _ := os.UserHomeDir()
	base := envOr("ICONS8_MCP_HOME", filepath.Join(home, ".icons8-mcp"))

	c := &Config{
		StateDir:          base,
		AssetDir:          envOr("ICONS8_ASSET_DIR", filepath.Join(base, "assets")),
		CookieFile:        envOr("ICONS8_COOKIE_FILE", filepath.Join(base, "cookies.json")),
		UserAgent:         envOr("ICONS8_USER_AGENT", DefaultUserAgent),
		Locale:            envOr("ICONS8_LOCALE", "en-US"),
		Region:            envOr("ICONS8_REGION", "AU"),
		RequestsPerSecond: envFloat("ICONS8_RPS", 6),
		MaxConcurrent:     envInt("ICONS8_CONCURRENCY", 4),
		HTTPTimeout:       envDur("ICONS8_HTTP_TIMEOUT", 45*time.Second),
		RefreshInterval:   envDur("ICONS8_REFRESH_INTERVAL", 6*time.Hour),
		BrowserFallback:   envOr("ICONS8_BROWSER_FALLBACK", "1") != "0",
		Headful:           os.Getenv("ICONS8_HEADFUL") == "1",
	}
	_ = os.MkdirAll(c.StateDir, 0o700)
	_ = os.MkdirAll(c.AssetDir, 0o755)
	return c
}

func (c *Config) StatePath() string { return filepath.Join(c.StateDir, "session.json") }

// Language returns the two-letter code the icon APIs want, derived from Locale.
func (c *Config) Language() string {
	if len(c.Locale) >= 2 {
		return c.Locale[:2]
	}
	return "en"
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil && v > 0 {
		return v
	}
	return def
}

func envFloat(k string, def float64) float64 {
	if v, err := strconv.ParseFloat(os.Getenv(k), 64); err == nil && v > 0 {
		return v
	}
	return def
}

func envDur(k string, def time.Duration) time.Duration {
	if v, err := time.ParseDuration(os.Getenv(k)); err == nil && v > 0 {
		return v
	}
	return def
}
