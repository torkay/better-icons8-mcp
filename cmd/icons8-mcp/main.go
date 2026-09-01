// Command icons8-mcp is an MCP server for the Icons8 asset library.
//
// Run with no arguments it speaks MCP over stdio, which is how an MCP host
// starts it. The subcommands exist for the parts a person does by hand: signing
// in once, and checking that the result works.
//
//	icons8-mcp                 # serve MCP over stdio (what the host runs)
//	icons8-mcp auth            # sign in to Icons8 in a browser window
//	icons8-mcp status          # show the account and licence
//	icons8-mcp tools           # list the registered tools
//	icons8-mcp import a.json   # install a browser cookie dump instead of `auth`
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/torkay/better-icons8-mcp/internal/browser"
	"github.com/torkay/better-icons8-mcp/internal/config"
	"github.com/torkay/better-icons8-mcp/internal/icons8"
	"github.com/torkay/better-icons8-mcp/internal/mcpserver"
	"github.com/torkay/better-icons8-mcp/internal/session"
)

const usage = `icons8-mcp - MCP server for the Icons8 asset library

  icons8-mcp                serve MCP over stdio (what an MCP host runs)
  icons8-mcp auth           sign in to Icons8 in a browser window, once
  icons8-mcp status         show the connected account and licence
  icons8-mcp tools          list the registered tools
  icons8-mcp import <file>  install a browser cookie dump instead of auth
  icons8-mcp version        print the version
`

func main() {
	cfg := config.Load()
	args := os.Args[1:]

	command := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		command, args = args[0], args[1:]
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var err error
	switch command {
	case "":
		err = serve(ctx, cfg, hasFlag(args, "-v"))
	case "auth", "login":
		err = runAuth(ctx, cfg)
	case "status", "check":
		err = runStatus(ctx, cfg)
	case "tools":
		err = runTools(ctx, cfg)
	case "import":
		if len(args) == 0 {
			err = fmt.Errorf("usage: icons8-mcp import <cookies.json>")
			break
		}
		err = runImport(cfg, args[0])
	case "version":
		fmt.Println(mcpserver.Version)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "icons8-mcp: %v\n", err)
		os.Exit(1)
	}
}

func hasFlag(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

func serve(ctx context.Context, cfg *config.Config, verbose bool) error {
	// stdout is the MCP transport, so every log line must go to stderr.
	logger := log.New(io.Discard, "", 0)
	if verbose {
		logger = log.New(os.Stderr, "icons8-mcp ", log.LstdFlags|log.Lmsgprefix)
	}
	srv, err := mcpserver.New(cfg, logger)
	if err != nil {
		return err
	}
	srv.StartRefreshLoop(ctx)
	logger.Printf("serving on stdio; assets -> %s", cfg.AssetDir)
	if err := srv.Run(ctx); err != nil && ctx.Err() == nil {
		return err
	}
	return nil
}

// runAuth is the terminal twin of the icons8_authorize tool. Both open the same
// window and store the same session, so it does not matter which one a user
// reaches for.
func runAuth(ctx context.Context, cfg *config.Config) error {
	sess, err := session.New(cfg.StatePath(), cfg.CookieFile)
	if err != nil {
		return err
	}
	if sess.Authorized() {
		claims := sess.Claims()
		fmt.Printf("already signed in as %s (token valid until %s)\n",
			claims.Email, claims.Expiry().Format("2006-01-02 15:04 MST"))
		return nil
	}

	b := browser.New(cfg, sess)
	defer b.Close()

	fmt.Println("opening a browser window; sign in to Icons8 there.")
	claims, err := b.Authorize(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("signed in as %s\n", claims.Email)
	fmt.Printf("session stored at %s\n", cfg.StatePath())
	return nil
}

func runStatus(ctx context.Context, cfg *config.Config) error {
	sess, err := session.New(cfg.StatePath(), cfg.CookieFile)
	if err != nil {
		return err
	}
	if !sess.Authorized() {
		fmt.Println("not connected. Run `icons8-mcp auth` to sign in.")
		return nil
	}
	acct, err := icons8.New(cfg, sess).Account(ctx)
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

// runTools answers "is this thing actually wired up?" without needing an MCP
// client. The server talks to an in-process client over a pair of in-memory
// transports, so what gets printed is the real tool list, not a copy of it.
func runTools(ctx context.Context, cfg *config.Config) error {
	srv, err := mcpserver.New(cfg, log.New(io.Discard, "", 0))
	if err != nil {
		return err
	}
	serverT, clientT := mcp.NewInMemoryTransports()
	ss, err := srv.Connect(ctx, serverT)
	if err != nil {
		return err
	}
	defer ss.Close()

	cs, err := mcp.NewClient(&mcp.Implementation{Name: "icons8-mcp tools", Version: mcpserver.Version}, nil).
		Connect(ctx, clientT, nil)
	if err != nil {
		return err
	}
	defer cs.Close()

	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		return err
	}
	for _, t := range res.Tools {
		fmt.Printf("%-30s %s\n", t.Name, firstSentence(t.Description))
	}
	fmt.Printf("\n%d tools\n", len(res.Tools))
	return nil
}

// firstSentence keeps the listing one line per tool. Tool descriptions are
// written for a model to read, so they are several sentences long; the opening
// one is the part a human is scanning for.
func firstSentence(desc string) string {
	line, _, _ := strings.Cut(desc, "\n")
	if s, _, ok := strings.Cut(line, ". "); ok {
		line = s + "."
	}
	if len(line) > 96 {
		line = strings.TrimRight(line[:95], " ,") + "…"
	}
	return line
}

// runImport validates a dump before installing it, so a bad paste fails here
// rather than at the first tool call.
func runImport(cfg *config.Config, src string) error {
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
			fmt.Printf("found session for %s, token valid until %s\n",
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
	fmt.Printf("installed -> %s\n", cfg.CookieFile)
	return nil
}
