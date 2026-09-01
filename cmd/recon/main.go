// Command recon drives a stealth headless Chrome through the Icons8 web app and
// records every XHR/fetch the app makes. It exists to discover the private API
// surface that the MCP server then calls directly over plain HTTP.
//
//	go run ./cmd/recon -cookies cookies.json -out recon.jsonl -pages icons,illustrations,photos,3d
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

type cookieDump struct {
	Domain   string  `json:"domain"`
	Name     string  `json:"name"`
	Path     string  `json:"path"`
	Value    string  `json:"value"`
	Secure   bool    `json:"secure"`
	HTTPOnly bool    `json:"httpOnly"`
	Expires  float64 `json:"expirationDate"`
}

type record struct {
	Page     string            `json:"page"`
	Method   string            `json:"method"`
	URL      string            `json:"url"`
	Status   int               `json:"status"`
	ReqBody  string            `json:"req_body,omitempty"`
	Headers  map[string]string `json:"req_headers,omitempty"`
	RespHead string            `json:"resp_head,omitempty"`
}

var pageURLs = map[string]string{
	"icons":         "https://icons8.com/icons/set/rocket",
	"icon-detail":   "https://icons8.com/icon/999/rocket",
	"illustrations": "https://icons8.com/illustrations/t/rocket",
	"animated":      "https://icons8.com/animated-illustrations",
	"photos":        "https://icons8.com/photos/search/office",
	"3d":            "https://icons8.com/3d-models/t/rocket",
	"styles":        "https://icons8.com/icons",
}

func main() {
	cookieFile := flag.String("cookies", "cookies.json", "browser cookie dump (JSON array)")
	out := flag.String("out", "recon.jsonl", "output JSONL file")
	pages := flag.String("pages", "icons,icon-detail,illustrations,photos,3d", "comma list of pages to visit")
	dwell := flag.Duration("dwell", 12*time.Second, "how long to sit on each page")
	headful := flag.Bool("headful", false, "show the browser")
	dumpHTML := flag.String("dump-html", "", "directory to write each page's rendered HTML")
	flag.Parse()

	cookies, err := loadCookies(*cookieFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cookies:", err)
		os.Exit(1)
	}

	l := launcher.New().
		Headless(!*headful).
		Set("disable-blink-features", "AutomationControlled").
		Set("no-sandbox").
		Set("disable-dev-shm-usage").
		Set("lang", "en-US,en").
		Set("window-size", "1440,900")
	u := l.MustLaunch()
	defer l.Cleanup()

	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	if err := browser.SetCookies(cookies); err != nil {
		fmt.Fprintln(os.Stderr, "set cookies:", err)
	}

	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "out:", err)
		os.Exit(1)
	}
	defer f.Close()
	enc := json.NewEncoder(f)

	var mu sync.Mutex
	seen := map[string]bool{}

	for _, name := range strings.Split(*pages, ",") {
		name = strings.TrimSpace(name)
		target, ok := pageURLs[name]
		if !ok {
			target = name // allow a raw URL
			name = "custom"
		}
		fmt.Fprintln(os.Stderr, "== visiting", name, target)

		page := stealth.MustPage(browser)
		page.MustSetUserAgent(&proto.NetworkSetUserAgentOverride{
			UserAgent:      "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36",
			AcceptLanguage: "en-US,en;q=0.9",
			Platform:       "Linux x86_64",
		})

		bodies := map[proto.NetworkRequestID]*record{}
		var bmu sync.Mutex

		go page.EachEvent(
			func(e *proto.NetworkRequestWillBeSent) {
				if !interesting(e.Request.URL) {
					return
				}
				h := map[string]string{}
				for k, v := range e.Request.Headers {
					lk := strings.ToLower(k)
					if lk == "authorization" || lk == "x-api-key" || strings.HasPrefix(lk, "x-") {
						h[lk] = redact(v.String())
					}
				}
				bmu.Lock()
				bodies[e.RequestID] = &record{
					Page: name, Method: e.Request.Method, URL: e.Request.URL,
					ReqBody: truncate(e.Request.PostData, 600), Headers: h,
				}
				bmu.Unlock()
			},
			func(e *proto.NetworkResponseReceived) {
				bmu.Lock()
				r := bodies[e.RequestID]
				bmu.Unlock()
				if r == nil {
					return
				}
				r.Status = e.Response.Status
				mu.Lock()
				key := r.Method + " " + stripVolatile(r.URL)
				if !seen[key] {
					seen[key] = true
					_ = enc.Encode(r)
					fmt.Fprintf(os.Stderr, "  %d %s %s\n", r.Status, r.Method, truncate(r.URL, 160))
				}
				mu.Unlock()
			},
		)()

		if err := page.Timeout(90 * time.Second).Navigate(target); err != nil {
			fmt.Fprintln(os.Stderr, "  nav err:", err)
		}
		_ = page.Timeout(90 * time.Second).WaitLoad()
		// Cloudflare interstitials resolve on their own; give them room, then scroll
		// to trigger lazy-loaded grid fetches.
		time.Sleep(*dwell)
		for i := 0; i < 4; i++ {
			_ = page.Mouse.Scroll(0, 1400, 6)
			time.Sleep(1500 * time.Millisecond)
		}

		if *dumpHTML != "" {
			_ = os.MkdirAll(*dumpHTML, 0o755)
			if html, err := page.HTML(); err == nil {
				_ = os.WriteFile(fmt.Sprintf("%s/%s.html", *dumpHTML, name), []byte(html), 0o644)
			}
		}
		title, _ := page.Eval("() => document.title")
		fmt.Fprintln(os.Stderr, "  title:", title.Value.Str())
		page.MustClose()
	}
	fmt.Fprintln(os.Stderr, "wrote", *out)
}

func interesting(u string) bool {
	if !strings.Contains(u, "icons8.com") && !strings.Contains(u, "ouch") {
		return false
	}
	for _, skip := range []string{".png", ".jpg", ".webp", ".woff", ".css", ".gif", "/_next/static/"} {
		if strings.Contains(u, skip) {
			return false
		}
	}
	return strings.Contains(u, "/api") || strings.Contains(u, "graphql") ||
		strings.Contains(u, "api-") || strings.Contains(u, "search.icons8")
}

func stripVolatile(u string) string {
	if i := strings.IndexByte(u, '?'); i >= 0 {
		return u[:i]
	}
	return u
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func redact(s string) string {
	if len(s) > 24 {
		return s[:12] + "...<redacted>"
	}
	return s
}

func loadCookies(path string) ([]*proto.NetworkCookieParam, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var dump []cookieDump
	if err := json.Unmarshal(b, &dump); err != nil {
		return nil, err
	}
	out := make([]*proto.NetworkCookieParam, 0, len(dump))
	for _, c := range dump {
		p := c.Path
		if p == "" {
			p = "/"
		}
		cp := &proto.NetworkCookieParam{
			Name: c.Name, Value: c.Value, Domain: c.Domain, Path: p,
			Secure: c.Secure, HTTPOnly: c.HTTPOnly,
		}
		if c.Expires > 0 {
			e := proto.TimeSinceEpoch(c.Expires)
			cp.Expires = e
		}
		out = append(out, cp)
	}
	return out, nil
}
