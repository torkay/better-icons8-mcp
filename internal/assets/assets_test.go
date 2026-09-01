package assets

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveIsIdempotentForIdenticalContent(t *testing.T) {
	s := NewStore(t.TempDir())
	data := []byte("<svg/>")

	first, err := s.Save("icons", "rocket", "svg", data, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Reused {
		t.Error("a first write should not report reuse")
	}

	second, err := s.Save("icons", "rocket", "svg", data, "")
	if err != nil {
		t.Fatal(err)
	}
	if !second.Reused || second.Path != first.Path {
		t.Errorf("identical content should reuse %s, got %+v", first.Path, second)
	}
}

func TestSaveDoesNotClobberDifferentContent(t *testing.T) {
	s := NewStore(t.TempDir())
	first, err := s.Save("icons", "rocket", "svg", []byte("<svg>a</svg>"), "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Save("icons", "rocket", "svg", []byte("<svg>b</svg>"), "")
	if err != nil {
		t.Fatal(err)
	}
	if second.Path == first.Path {
		t.Fatal("differing content must not overwrite the earlier file")
	}
	got, err := os.ReadFile(first.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "<svg>a</svg>" {
		t.Fatalf("original file was modified: %q", got)
	}
}

func TestSlugIsFilesystemSafe(t *testing.T) {
	cases := map[string]string{
		"Rocket in Flight":        "rocket-in-flight",
		"../../etc/passwd":        "etc-passwd",
		"weird/\\:*?\"<>|chars":   "weird-chars",
		"":                        "asset",
		"---":                     "asset",
		"Startup 3D — “launch”!!": "startup-3d-launch",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSlugStaysWithinTheStore(t *testing.T) {
	root := t.TempDir()
	s := NewStore(root)
	saved, err := s.Save("../../escape", "../../../etc/passwd", "svg", []byte("x"), "")
	if err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(saved.Path)
	if err != nil {
		t.Fatal(err)
	}
	rootAbs, _ := filepath.Abs(root)
	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil || len(rel) > 1 && rel[:2] == ".." {
		t.Fatalf("path escaped the store: %s", saved.Path)
	}
}

func TestExtForNamesTheRealFileType(t *testing.T) {
	// These tokens lie about the file they produce.
	cases := map[string]string{
		"mov-hevc": "mp4",
		"mov-avc":  "mov",
		"json":     "json",
		"png-hd":   "png",
		"gif-low":  "gif",
		"fbx":      "zip",
		"fbx-zip":  "zip",
		"glb":      "glb",
		"apng":     "png",
	}
	for in, want := range cases {
		if got := ExtFor(in); got != want {
			t.Errorf("ExtFor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExtFromContentType(t *testing.T) {
	cases := map[string]string{
		"image/svg+xml":              "svg",
		"image/png":                  "png",
		"application/pdf":            "pdf",
		"video/webm":                 "webm",
		"image/jpeg; charset=binary": "jpg",
		"application/octet-stream":   "bin",
		"application/postscript":     "eps",
	}
	for in, want := range cases {
		if got := ExtFromContentType(in); got != want {
			t.Errorf("ExtFromContentType(%q) = %q, want %q", in, got, want)
		}
	}
}

// fakePNG is enough bytes to stand in for a real image in header assertions.
func fakePNG(n byte) []byte {
	return append([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, bytes.Repeat([]byte{n}, 16)...)
}

func TestEncodeICOHeaderAndOffsets(t *testing.T) {
	pngs := map[int][]byte{16: fakePNG(1), 32: fakePNG(2), 256: fakePNG(3)}
	ico, err := EncodeICO(pngs)
	if err != nil {
		t.Fatal(err)
	}

	if got := binary.LittleEndian.Uint16(ico[0:2]); got != 0 {
		t.Errorf("reserved = %d, want 0", got)
	}
	if got := binary.LittleEndian.Uint16(ico[2:4]); got != 1 {
		t.Errorf("type = %d, want 1 (icon)", got)
	}
	count := binary.LittleEndian.Uint16(ico[4:6])
	if count != 3 {
		t.Fatalf("count = %d, want 3", count)
	}

	// Entries are sorted ascending, and 256 is stored as 0 in the byte field.
	wantDims := []byte{16, 32, 0}
	for i := 0; i < int(count); i++ {
		entry := ico[6+16*i : 6+16*(i+1)]
		if entry[0] != wantDims[i] || entry[1] != wantDims[i] {
			t.Errorf("entry %d dims = %d, want %d", i, entry[0], wantDims[i])
		}
		size := binary.LittleEndian.Uint32(entry[8:12])
		offset := binary.LittleEndian.Uint32(entry[12:16])
		if int(offset)+int(size) > len(ico) {
			t.Fatalf("entry %d points past the end of the file", i)
		}
		if got := ico[offset : offset+size]; !bytes.HasPrefix(got, []byte{0x89, 'P', 'N', 'G'}) {
			t.Errorf("entry %d does not point at PNG data", i)
		}
	}
}

func TestEncodeICORejectsBadInput(t *testing.T) {
	if _, err := EncodeICO(nil); err == nil {
		t.Error("expected an error for no images")
	}
	if _, err := EncodeICO(map[int][]byte{512: fakePNG(1)}); err == nil {
		t.Error("ICO cannot hold entries above 256px; expected an error")
	}
}

func TestICOSizesFiltersOversizedEntries(t *testing.T) {
	got := ICOSizes([]int{16, 32, 180, 256, 512, 1024})
	want := []int{16, 32, 180, 256}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestSizePresetsCoverTheAdvertisedPlatforms(t *testing.T) {
	for _, p := range []string{"favicon", "web", "ios", "android", "macos", "windows"} {
		if len(SizePresets[p]) == 0 {
			t.Errorf("preset %q is missing", p)
		}
	}
}
