package session

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// mintToken builds an unsigned JWT with the given expiry, which is all this
// package reads.
func mintToken(t *testing.T, email string, exp time.Time) string {
	t.Helper()
	payload := map[string]any{
		"id":           "test-id",
		"email":        email,
		"publicApiKey": "test-key",
		"env":          "production",
		"iat":          time.Now().Unix(),
		"exp":          exp.Unix(),
		"activeLicense": map[string]any{
			"icons": true, "vectors": true, "photos": true, "sounds": true,
			"expireAt": exp.Add(24 * time.Hour).UnixMilli(),
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	head := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	return head + "." + base64.RawURLEncoding.EncodeToString(body) + ".sig"
}

func writeDump(t *testing.T, dir, token string) string {
	t.Helper()
	path := filepath.Join(dir, "cookies.json")
	dump := []Cookie{
		{Domain: "icons8.com", Name: "i8token", Path: "/", Value: token, Expires: 1789114018},
		{Domain: "icons8.com", Name: "i8region", Path: "/", Value: "AU"},
	}
	raw, err := json.Marshal(dump)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBootstrapFromCookieDump(t *testing.T) {
	dir := t.TempDir()
	exp := time.Now().Add(10 * 24 * time.Hour)
	cookieFile := writeDump(t, dir, mintToken(t, "hi@example.test", exp))

	s, err := New(filepath.Join(dir, "session.json"), cookieFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Claims().Email; got != "hi@example.test" {
		t.Errorf("email = %q", got)
	}
	if s.PublicAPIKey() != "test-key" {
		t.Errorf("public api key = %q", s.PublicAPIKey())
	}
	if len(s.Fingerprint()) != 32 {
		t.Errorf("fingerprint should be 32 hex chars, got %q", s.Fingerprint())
	}
}

// A dump with no token has to load rather than fail. The server must start
// before it can be authorised, or the host reports a broken server and the user
// has no way in.
func TestMissingTokenLoadsUnauthorized(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cookies.json")
	if err := os.WriteFile(path, []byte(`[{"name":"i8region","value":"AU","domain":"icons8.com"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := New(filepath.Join(dir, "session.json"), path)
	if err != nil {
		t.Fatalf("a dump without i8token should still load: %v", err)
	}
	if s.Authorized() {
		t.Error("session should report itself unauthorized")
	}
	if len(s.Fingerprint()) != 32 {
		t.Errorf("fingerprint should still be generated, got %q", s.Fingerprint())
	}
}

func TestUnreadableDumpIsAnActionableError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cookies.json")
	if err := os.WriteFile(path, []byte(`not json at all`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := New(filepath.Join(dir, "session.json"), path)
	if err == nil {
		t.Fatal("expected an error for a dump that is not JSON")
	}
	if want := "cookie dump"; !strings.Contains(err.Error(), want) {
		t.Errorf("error should say what it failed to read, got: %v", err)
	}
}

func TestStatePersistsAndFingerprintIsStable(t *testing.T) {
	dir := t.TempDir()
	exp := time.Now().Add(10 * 24 * time.Hour)
	cookieFile := writeDump(t, dir, mintToken(t, "hi@example.test", exp))
	statePath := filepath.Join(dir, "session.json")

	first, err := New(statePath, cookieFile)
	if err != nil {
		t.Fatal(err)
	}
	second, err := New(statePath, cookieFile)
	if err != nil {
		t.Fatal(err)
	}
	if first.Fingerprint() != second.Fingerprint() {
		t.Error("fingerprint must survive a restart, or Icons8 sees a new browser each run")
	}
}

func TestSetTokenIgnoresAStaleToken(t *testing.T) {
	dir := t.TempDir()
	fresh := time.Now().Add(10 * 24 * time.Hour)
	cookieFile := writeDump(t, dir, mintToken(t, "hi@example.test", fresh))
	s, err := New(filepath.Join(dir, "session.json"), cookieFile)
	if err != nil {
		t.Fatal(err)
	}
	before := s.Token()

	stale := mintToken(t, "hi@example.test", time.Now().Add(time.Hour))
	if err := s.SetToken(stale); err != nil {
		t.Fatal(err)
	}
	if s.Token() != before {
		t.Error("a shorter-lived token must not replace a longer-lived one")
	}

	newer := mintToken(t, "hi@example.test", fresh.Add(24*time.Hour))
	if err := s.SetToken(newer); err != nil {
		t.Fatal(err)
	}
	if s.Token() != newer {
		t.Error("a longer-lived token should be adopted")
	}
}

func TestSetTokenRejectsGarbage(t *testing.T) {
	dir := t.TempDir()
	cookieFile := writeDump(t, dir, mintToken(t, "hi@example.test", time.Now().Add(24*time.Hour)))
	s, err := New(filepath.Join(dir, "session.json"), cookieFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetToken("not-a-jwt"); err == nil {
		t.Error("expected an error for a malformed token")
	}
}

func TestNeedsRefreshTracksExpiry(t *testing.T) {
	dir := t.TempDir()
	cookieFile := writeDump(t, dir, mintToken(t, "hi@example.test", time.Now().Add(2*time.Hour)))
	s, err := New(filepath.Join(dir, "session.json"), cookieFile)
	if err != nil {
		t.Fatal(err)
	}
	if !s.NeedsRefresh(6 * time.Hour) {
		t.Error("a token expiring in 2h should need refreshing within a 6h window")
	}
	if s.NeedsRefresh(time.Minute) {
		t.Error("a token expiring in 2h should not need refreshing within a 1m window")
	}
}

func TestApplySetsEverythingIcons8Checks(t *testing.T) {
	dir := t.TempDir()
	token := mintToken(t, "hi@example.test", time.Now().Add(24*time.Hour))
	cookieFile := writeDump(t, dir, token)
	s, err := New(filepath.Join(dir, "session.json"), cookieFile)
	if err != nil {
		t.Fatal(err)
	}

	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	s.Apply(req, "TestAgent/1.0", "en-US")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if want := "Bearer " + token; got.Get("Authorization") != want {
		t.Errorf("Authorization = %q", got.Get("Authorization"))
	}
	if got.Get("X-Icons8-Fingerprint") != s.Fingerprint() {
		t.Error("fingerprint header missing; Icons8 rejects some endpoints without it")
	}
	if got.Get("Origin") != "https://icons8.com" || got.Get("Referer") != "https://icons8.com/" {
		t.Error("Origin/Referer must look like the web app")
	}
	if got.Get("User-Agent") != "TestAgent/1.0" {
		t.Errorf("User-Agent = %q", got.Get("User-Agent"))
	}
}

func TestParseClaimsRejectsMalformedTokens(t *testing.T) {
	for _, tok := range []string{"", "a", "a.b", "a.b.c.d", "a.!!!.c"} {
		if _, err := ParseClaims(tok); err == nil {
			t.Errorf("ParseClaims(%q) should fail", tok)
		}
	}
}

func TestTokenFromCookiesPrefersLongestLived(t *testing.T) {
	short := "short"
	long := "long"
	got := tokenFromCookies([]Cookie{
		{Name: "i8token", Value: short, Expires: 100},
		{Name: "other", Value: "x", Expires: 999},
		{Name: "i8token", Value: long, Expires: 200},
	})
	if got != long {
		t.Errorf("got %q, want the later-expiring cookie %q", got, long)
	}
}
