// Command reauthcheck exercises the headless-browser recovery path and reports
// what it actually recovers, so the fallback's real capability is measured
// rather than assumed.
//
//	go run ./cmd/reauthcheck
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/torkay/icons8-mcp-server/internal/browser"
	"github.com/torkay/icons8-mcp-server/internal/config"
	"github.com/torkay/icons8-mcp-server/internal/session"
)

func main() {
	flag.Parse()
	cfg := config.Load()

	sess, err := session.New(cfg.StatePath(), cfg.CookieFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "session:", err)
		os.Exit(1)
	}
	before := sess.Claims()
	fmt.Printf("before: token for %s expires %s\n", before.Email, before.Expiry().Format(time.RFC3339))

	b := browser.New(cfg, sess)
	defer b.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	start := time.Now()
	if err := b.Reauthenticate(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "reauthenticate:", err)
		os.Exit(1)
	}
	after := sess.Claims()
	fmt.Printf("after:  token for %s expires %s (%.1fs)\n",
		after.Email, after.Expiry().Format(time.RFC3339), time.Since(start).Seconds())

	switch {
	case after.ExpiresAt > before.ExpiresAt:
		fmt.Println("result: browser recovered a NEWER token")
	case after.Email != "":
		fmt.Println("result: browser cleared Cloudflare and kept the existing session (no newer token offered)")
	default:
		fmt.Println("result: no session recovered")
		os.Exit(1)
	}
}
