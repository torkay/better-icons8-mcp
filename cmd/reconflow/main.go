// Command reconflow drives a specific interaction (open an asset page, click
// "Download", pick a format) and dumps every request the app makes *with full
// headers and response bodies*. This is how the paid-download / "unlock" flow
// gets reverse-engineered so the MCP server can perform it over plain HTTP.
//
//	go run ./cmd/reconflow -url https://icons8.com/icon/999/rocket -click 'Download'
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

type rec struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Status  int               `json:"status"`
	ReqHdr  map[string]string `json:"req_headers,omitempty"`
	ReqBody string            `json:"req_body,omitempty"`
	Body    string            `json:"resp_body,omitempty"`
	CT      string            `json:"content_type,omitempty"`
}

func main() {
	cookieFile := flag.String("cookies", "cookies.json", "cookie dump")
	target := flag.String("url", "https://icons8.com/icon/999/rocket", "page to open")
	clicks := flag.String("click", "", "comma-separated button texts to click in order")
	out := flag.String("out", "flow.jsonl", "output JSONL")
	filter := flag.String("filter", "", "only record URLs containing this substring")
	dwell := flag.Duration("dwell", 8*time.Second, "wait after each step")
	headful := flag.Bool("headful", false, "show browser")
	shot := flag.String("screenshot", "", "screenshot path")
	js := flag.String("js", "", "JS expression to evaluate after the clicks; result printed as JSON")
	sel := flag.String("click-sel", "", "comma-separated CSS selectors to click in order (runs after -click)")
	flag.Parse()

	cookies, err := loadCookies(*cookieFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cookies:", err)
		os.Exit(1)
	}

	l := launcher.New().Headless(!*headful).
		Set("disable-blink-features", "AutomationControlled").
		Set("no-sandbox").Set("disable-dev-shm-usage").
		Set("lang", "en-US,en").Set("window-size", "1600,1000")
	browser := rod.New().ControlURL(l.MustLaunch()).MustConnect()
	defer l.Cleanup()
	defer browser.MustClose()
	_ = browser.SetCookies(cookies)

	f, _ := os.Create(*out)
	defer f.Close()
	enc := json.NewEncoder(f)

	page := stealth.MustPage(browser)
	page.MustSetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent:      "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36",
		AcceptLanguage: "en-US,en;q=0.9", Platform: "Linux x86_64",
	})

	// CDP's Network domain misses fetches issued from workers and anything that
	// ends up as a browser-level download, so shim the page's own primitives too.
	_, _ = page.EvalOnNewDocument(`
		window.__i8log = [];
		const push = (o) => { try { window.__i8log.push(o) } catch(e){} };
		const of = window.fetch;
		window.fetch = function(input, init) {
			try {
				const url = typeof input === 'string' ? input : (input && input.url);
				const hdrs = {};
				const h = (init && init.headers) || (input && input.headers);
				if (h) { try { new Headers(h).forEach((v,k) => hdrs[k]=v) } catch(e){} }
				push({kind:'fetch', url, method:(init&&init.method)||'GET', headers:hdrs,
				      body: init && typeof init.body === 'string' ? init.body.slice(0,800) : undefined});
			} catch(e){}
			return of.apply(this, arguments);
		};
		const oo = XMLHttpRequest.prototype.open;
		XMLHttpRequest.prototype.open = function(m, u) { push({kind:'xhr', method:m, url:String(u)}); return oo.apply(this, arguments) };
		const oc = URL.createObjectURL;
		URL.createObjectURL = function(b) {
			const u = oc.apply(this, arguments);
			push({kind:'blob', url:u, type:b && b.type, size:b && b.size});
			try { if (b && b.text) b.text().then(t => push({kind:'blobtext', url:u, text:String(t).slice(0,1200)})) } catch(e){}
			return u;
		};
		const ac = HTMLAnchorElement.prototype.click;
		HTMLAnchorElement.prototype.click = function() {
			try { if (this.hasAttribute('download') || String(this.href).startsWith('blob:'))
			        push({kind:'anchor-download', href:this.href, name:this.getAttribute('download')}) } catch(e){}
			return ac.apply(this, arguments);
		};
	`)

	pending := map[proto.NetworkRequestID]*rec{}
	var mu sync.Mutex

	go page.EachEvent(
		func(e *proto.NetworkRequestWillBeSent) {
			if !keep(e.Request.URL, *filter) {
				return
			}
			h := map[string]string{}
			for k, v := range e.Request.Headers {
				h[strings.ToLower(k)] = v.String()
			}
			mu.Lock()
			pending[e.RequestID] = &rec{Method: e.Request.Method, URL: e.Request.URL, ReqHdr: h, ReqBody: e.Request.PostData}
			mu.Unlock()
		},
		func(e *proto.NetworkResponseReceived) {
			mu.Lock()
			r := pending[e.RequestID]
			mu.Unlock()
			if r == nil {
				return
			}
			r.Status = e.Response.Status
			r.CT = e.Response.MIMEType
		},
		func(e *proto.NetworkLoadingFinished) {
			mu.Lock()
			r := pending[e.RequestID]
			delete(pending, e.RequestID)
			mu.Unlock()
			if r == nil {
				return
			}
			if !strings.HasPrefix(r.CT, "image/") && !strings.Contains(r.CT, "octet-stream") {
				if b, err := (&proto.NetworkGetResponseBody{RequestID: e.RequestID}).Call(page); err == nil {
					r.Body = trunc(b.Body, 2500)
				}
			}
			mu.Lock()
			_ = enc.Encode(r)
			mu.Unlock()
			fmt.Fprintf(os.Stderr, "%d %-6s %s\n", r.Status, r.Method, trunc(r.URL, 150))
			if r.Method == "POST" && r.ReqBody != "" {
				fmt.Fprintf(os.Stderr, "      body> %s\n", trunc(r.ReqBody, 300))
			}
			if r.Body != "" && r.Status >= 400 {
				fmt.Fprintf(os.Stderr, "      err>  %s\n", trunc(r.Body, 300))
			}
		},
	)()

	fmt.Fprintln(os.Stderr, "== open", *target)
	navigate(page, *target, *dwell)

	// -click is an ordered step list. Each step is "sel:<css>" or "text:<label>"
	// (bare values are treated as text).
	for _, step := range strings.Split(*clicks, ",") {
		step = strings.TrimSpace(step)
		if step == "" {
			continue
		}
		fmt.Fprintf(os.Stderr, "== step %q\n", step)
		var err error
		switch {
		case strings.HasPrefix(step, "sel:"):
			err = clickBySelector(page, strings.TrimPrefix(step, "sel:"))
		case strings.HasPrefix(step, "xy:"):
			var x, y float64
			if _, e := fmt.Sscanf(strings.TrimPrefix(step, "xy:"), "%fx%f", &x, &y); e != nil {
				err = e
			} else if err = page.Mouse.MoveTo(proto.NewPoint(x, y)); err == nil {
				time.Sleep(150 * time.Millisecond)
				err = page.Mouse.Click(proto.InputMouseButtonLeft, 1)
			}
		case strings.HasPrefix(step, "wait:"):
			var d time.Duration
			d, err = time.ParseDuration(strings.TrimPrefix(step, "wait:"))
			if err == nil {
				time.Sleep(d)
			}
		default:
			err = clickByText(page, strings.TrimPrefix(step, "text:"))
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "   step failed:", err)
		}
		time.Sleep(*dwell)
	}

	for _, s := range strings.Split(*sel, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		fmt.Fprintf(os.Stderr, "== click-sel %q\n", s)
		if err := clickBySelector(page, s); err != nil {
			fmt.Fprintln(os.Stderr, "   click failed:", err)
		}
		time.Sleep(*dwell)
	}

	if *js != "" {
		res, err := page.Eval(*js)
		if err != nil {
			fmt.Fprintln(os.Stderr, "js err:", err)
		} else {
			fmt.Println(res.Value.JSON("", "  "))
		}
	}

	if *shot != "" {
		_ = os.WriteFile(*shot, page.MustScreenshot(), 0o644)
		fmt.Fprintln(os.Stderr, "screenshot ->", *shot)
	}
	fmt.Fprintln(os.Stderr, "wrote", *out)
}

// navigate loads target, retrying transient Chrome network errors and waiting
// for the Nuxt app to actually paint rather than just for the load event.
func navigate(page *rod.Page, target string, dwell time.Duration) {
	for attempt := 1; attempt <= 4; attempt++ {
		if err := page.Timeout(90 * time.Second).Navigate(target); err != nil {
			fmt.Fprintln(os.Stderr, "   nav err:", err)
		}
		_ = page.Timeout(90 * time.Second).WaitLoad()
		time.Sleep(dwell)
		res, err := page.Eval(`() => ({ t: document.title, n: document.querySelectorAll('button,a').length })`)
		if err == nil && res.Value.Get("n").Int() > 20 {
			return
		}
		fmt.Fprintf(os.Stderr, "   page not ready (attempt %d), retrying\n", attempt)
		time.Sleep(time.Duration(attempt) * 2 * time.Second)
	}
}

func clickBySelector(page *rod.Page, css string) error {
	el, err := page.Timeout(20 * time.Second).Element(css)
	if err != nil {
		return err
	}
	_ = el.ScrollIntoView()
	time.Sleep(300 * time.Millisecond)
	return el.Click(proto.InputMouseButtonLeft, 1)
}

// clickByText finds the smallest visible clickable element whose trimmed text
// matches label (case-insensitive) and clicks it via a real mouse event.
func clickByText(page *rod.Page, label string) error {
	js := `(label) => {
		const want = label.toLowerCase();
		const hits = [...document.querySelectorAll('*')].filter(e => {
			const t = (e.innerText||e.textContent||'').trim().toLowerCase();
			if (!t || t.length > want.length + 24) return false;
			if (!t.includes(want)) return false;
			const r = e.getBoundingClientRect();
			return r.width > 4 && r.height > 4;
		});
		if (!hits.length) return null;
		// Smallest box first: that is the actual control, not an ancestor wrapper.
		hits.sort((a,b) => {
			const ra = a.getBoundingClientRect(), rb = b.getBoundingClientRect();
			return ra.width*ra.height - rb.width*rb.height;
		});
		const el = hits[0];
		el.scrollIntoView({block:'center'});
		const r = el.getBoundingClientRect();
		return {x: r.x + r.width/2, y: r.y + r.height/2};
	}`
	res, err := page.Eval(js, label)
	if err != nil {
		return err
	}
	if res.Value.Nil() {
		return fmt.Errorf("no element matching %q", label)
	}
	x := res.Value.Get("x").Num()
	y := res.Value.Get("y").Num()
	if err := page.Mouse.MoveTo(proto.NewPoint(x, y)); err != nil {
		return err
	}
	time.Sleep(180 * time.Millisecond)
	return page.Mouse.Click(proto.InputMouseButtonLeft, 1)
}

func keep(u, filter string) bool {
	if filter != "" {
		return strings.Contains(u, filter)
	}
	if !strings.Contains(u, "icons8.com") {
		return false
	}
	for _, s := range []string{"/_next/static/", ".css", ".woff", "sentry.", "growthbook", "analista"} {
		if strings.Contains(u, s) {
			return false
		}
	}
	return true
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
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
		cp := &proto.NetworkCookieParam{Name: c.Name, Value: c.Value, Domain: c.Domain, Path: p, Secure: c.Secure, HTTPOnly: c.HTTPOnly}
		if c.Expires > 0 {
			cp.Expires = proto.TimeSinceEpoch(c.Expires)
		}
		out = append(out, cp)
	}
	return out, nil
}
