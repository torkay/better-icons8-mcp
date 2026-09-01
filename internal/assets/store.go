// Package assets writes downloaded Icons8 files to disk under a predictable,
// collision-free layout so an agent can hand the path straight to a build.
package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Store struct{ root string }

func NewStore(root string) *Store { return &Store{root: root} }

func (s *Store) Root() string { return s.root }

// Saved describes a file on disk plus enough provenance to cite it.
type Saved struct {
	Path      string `json:"path"`
	Bytes     int    `json:"bytes"`
	Format    string `json:"format"`
	SourceURL string `json:"source_url,omitempty"`
	Reused    bool   `json:"reused"`
}

var unsafeChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// Slug turns an arbitrary title into a filename-safe stem.
func Slug(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = unsafeChars.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-._")
	if len(s) > 72 {
		s = s[:72]
	}
	if s == "" {
		s = "asset"
	}
	return s
}

// Save writes data under <root>/<kind>/<name>.<ext>. When a file of the same
// name already exists with identical content it is reused rather than rewritten,
// so repeated tool calls are idempotent; differing content gets a short hash
// suffix instead of silently clobbering the earlier file.
func (s *Store) Save(kind, name, ext string, data []byte, sourceURL string) (*Saved, error) {
	dir := filepath.Join(s.root, Slug(kind))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	ext = strings.TrimPrefix(strings.ToLower(ext), ".")
	if ext == "" {
		ext = "bin"
	}
	path := filepath.Join(dir, Slug(name)+"."+ext)

	if existing, err := os.ReadFile(path); err == nil {
		if sameBytes(existing, data) {
			return &Saved{Path: path, Bytes: len(existing), Format: ext, SourceURL: sourceURL, Reused: true}, nil
		}
		sum := sha256.Sum256(data)
		path = filepath.Join(dir, Slug(name)+"-"+hex.EncodeToString(sum[:4])+"."+ext)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, fmt.Errorf("write %s: %w", path, err)
	}
	return &Saved{Path: path, Bytes: len(data), Format: ext, SourceURL: sourceURL}, nil
}

func sameBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	sa, sb := sha256.Sum256(a), sha256.Sum256(b)
	return sa == sb
}

// ExtFor maps an Icons8 format token to the file extension it actually is.
// Several tokens lie: `json` is a Lottie file, `mov-hevc` ships as .mp4.
func ExtFor(format string) string {
	switch strings.ToLower(format) {
	case "png-hd", "png-low", "png":
		return "png"
	case "gif-low", "gif":
		return "gif"
	case "json":
		return "json"
	case "webm":
		return "webm"
	case "mov-avc":
		return "mov"
	case "mov-hevc":
		return "mp4"
	case "aep":
		return "aep"
	case "fbx-zip", "fbx", "sources":
		return "zip"
	case "glb":
		return "glb"
	case "apng":
		return "png"
	case "jpg", "jpeg":
		return "jpg"
	default:
		return strings.ToLower(format)
	}
}

// ExtFromContentType is the fallback when a signed URL does not reveal a format.
func ExtFromContentType(ct string) string {
	ct = strings.ToLower(strings.SplitN(ct, ";", 2)[0])
	switch ct {
	case "image/png", "image/apng":
		return "png"
	case "image/svg+xml":
		return "svg"
	case "image/jpeg":
		return "jpg"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	case "application/pdf":
		return "pdf"
	case "application/postscript":
		return "eps"
	case "application/json":
		return "json"
	case "video/webm":
		return "webm"
	case "video/mp4":
		return "mp4"
	case "video/quicktime":
		return "mov"
	case "application/zip":
		return "zip"
	default:
		return "bin"
	}
}
