// Command icons8-mcp serves the Icons8 asset library as an MCP server over
// stdio.
//
//	icons8-mcp                 # run the server
//	icons8-mcp -import a.json  # install a browser cookie dump, then exit
//	icons8-mcp -check          # verify the session and print the account
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/torkay/icons8-mcp-server/internal/config"
	"github.com/torkay/icons8-mcp-server/internal/icons8"
	"github.com/torkay/icons8-mcp-server/internal/mcpserver"
	"github.com/torkay/icons8-mcp-server/internal/session"
)

func main() {
	importPath := flag.String("import", "", "install a browser cookie dump as the session bootstrap, then exit")
	check := flag.Bool("check", false, "verify the session against Icons8 and print the account, then exit")
	verbose := flag.Bool("v", false, "log to stderr")
	flag.Parse()

	cfg := config.Load()

	// stdout is the MCP transport, so every log line must go to stderr.
	logger := log.New(io.Discard, "", 0)
	if *verbose || *check || *importPath != "" {
		logger = log.New(os.Stderr, "icons8-mcp ", log.LstdFlags|log.Lmsgprefix)
	}

	if *importPath != "" {
		if err := importCookies(cfg, *importPath); err != nil {
			logger.Fatalf("import: %v", err)
		}
		return
	}

	srv, err := mcpserver.New(cfg, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "icons8-mcp: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *check {
		if err := runCheck(ctx, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "icons8-mcp: %v\n", err)
			os.Exit(1)
		}
		return
	}

	srv.StartRefreshLoop(ctx)
	logger.Printf("serving on stdio; assets -> %s", cfg.AssetDir)
	if err := srv.Run(ctx); err != nil && ctx.Err() == nil {
		fmt.Fprintf(os.Stderr, "icons8-mcp: %v\n", err)
		os.Exit(1)
	}
}

// importCookies validates a dump before installing it, so a bad paste fails here
// rather than at the first tool call.
func importCookies(cfg *config.Config, src string) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	var cookies []session.Cookie
	if err := json.Unmarshal(raw, &cookies); err != nil {
		return fmt.Errorf("not a JSON cookie dump: %w", err)
	}
	found := false
	for _, c := range cookies {
		if c.Name == "i8token" && c.Value != "" {
			claims, err := session.ParseClaims(c.Value)
			if err != nil {
				return fmt.Errorf("i8token is not a readable JWT: %w", err)
			}
			fmt.Fprintf(os.Stderr, "found session for %s, token valid until %s\n",
				claims.Email, claims.Expiry().Format("2006-01-02 15:04 MST"))
			found = true
		}
	}
	if !found {
		return fmt.Errorf("dump has no `i8token` cookie; export cookies for icons8.com while signed in")
	}
	if err := os.MkdirAll(filepath.Dir(cfg.CookieFile), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(cfg.CookieFile, raw, 0o600); err != nil {
		return err
	}
	// Drop any previously saved token so the new dump wins unambiguously.
	_ = os.Remove(cfg.StatePath())
	fmt.Fprintf(os.Stderr, "installed -> %s\n", cfg.CookieFile)
	return nil
}

func runCheck(ctx context.Context, cfg *config.Config) error {
	sess, err := session.New(cfg.StatePath(), cfg.CookieFile)
	if err != nil {
		return err
	}
	client := icons8.New(cfg, sess)
	acct, err := client.Account(ctx)
	if err != nil {
		return err
	}
	claims := sess.Claims()
	fmt.Printf("account:  %s\n", acct.Email)
	fmt.Printf("licence:  icons=%v vectors=%v photos=%v sounds=%v\n",
		claims.License.Icons, claims.License.Vectors, claims.License.Photos, claims.License.Sounds)
	fmt.Printf("token:    valid until %s\n", claims.Expiry().Format("2006-01-02 15:04 MST"))
	fmt.Printf("assets:   %s\n", cfg.AssetDir)
	return nil
}
