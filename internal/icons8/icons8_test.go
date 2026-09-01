package icons8

import (
	"encoding/json"
	"net/url"
	"testing"
)

func TestStringListAcceptsBothShapes(t *testing.T) {
	// The search endpoint returns strings here; the vector-similarity endpoint
	// returns arrays. Both must decode.
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"string", `{"category":"Transport"}`, "Transport"},
		{"array", `{"category":["Transport","Aircraft"]}`, "Transport"},
		{"empty array", `{"category":[]}`, ""},
		{"empty string", `{"category":""}`, ""},
		{"null", `{"category":null}`, ""},
		{"absent", `{}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var v struct {
				Category StringList `json:"category"`
			}
			if err := json.Unmarshal([]byte(tc.in), &v); err != nil {
				t.Fatalf("unmarshal %s: %v", tc.in, err)
			}
			if got := v.Category.First(); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStringListRejectsWrongType(t *testing.T) {
	var v struct {
		Category StringList `json:"category"`
	}
	if err := json.Unmarshal([]byte(`{"category":42}`), &v); err == nil {
		t.Fatal("expected an error for a numeric category")
	}
}

func TestMetaSplitsFacetsFromQueryParams(t *testing.T) {
	// Style and category are query parameters, not meta keys. Putting them in
	// meta is silently ignored by Icons8, so the split is load-bearing.
	o := IllustrationSearchOptions{
		Style: "techny", Category: "business",
		Mood: "casual", Technique: "3d", Colors: "blue, green",
	}
	var m map[string][]string
	if err := json.Unmarshal([]byte(o.meta()), &m); err != nil {
		t.Fatalf("meta is not JSON: %v", err)
	}
	if _, ok := m["style_pretty_ids"]; ok {
		t.Error("style must not be in meta")
	}
	if _, ok := m["category_pretty_ids"]; ok {
		t.Error("category must not be in meta")
	}
	if got := m["mood"]; len(got) != 1 || got[0] != "casual" {
		t.Errorf("mood = %v, want [casual]", got)
	}
	if got := m["technique"]; len(got) != 1 || got[0] != "3d" {
		t.Errorf("technique = %v, want [3d]", got)
	}
	if got := m["colors"]; len(got) != 2 || got[0] != "blue" || got[1] != "green" {
		t.Errorf("colors = %v, want [blue green] with whitespace trimmed", got)
	}
}

func TestMetaIsEmptyObjectWhenNoFacets(t *testing.T) {
	if got := (IllustrationSearchOptions{}).meta(); got != "{}" {
		t.Fatalf("meta = %q, want {}", got)
	}
}

func TestMediaFormatMapsFBX(t *testing.T) {
	// Items advertise "fbx" but the download endpoint only accepts "fbx-zip".
	if got := MediaFormat("fbx"); got != "fbx-zip" {
		t.Errorf("MediaFormat(fbx) = %q, want fbx-zip", got)
	}
	for _, f := range []string{"glb", "png-hd", "mov-hevc", "json"} {
		if got := MediaFormat(f); got != f {
			t.Errorf("MediaFormat(%q) = %q, want unchanged", f, got)
		}
	}
	if got := MediaFormat("  FBX "); got != "fbx-zip" {
		t.Errorf("MediaFormat is not normalising case/space: %q", got)
	}
}

func TestHasMotionUsesBothSignals(t *testing.T) {
	// Search results carry motion asset fields but no downloadable_resources.
	fromSearch := Illustration{WebM: &Asset{URL: "x"}}
	if !fromSearch.HasMotion() {
		t.Error("a search result with a webm asset should read as animated")
	}
	// Detail responses carry downloadable_resources but the asset fields may be
	// absent for some formats.
	var fromDetail Illustration
	fromDetail.DownloadableResources.Available = []string{"png-hd", "gif", "json"}
	if !fromDetail.HasMotion() {
		t.Error("a detail response listing gif/json should read as animated")
	}
	var static Illustration
	static.DownloadableResources.Available = []string{"png-hd", "png", "svg"}
	if static.HasMotion() {
		t.Error("a png/svg-only item should not read as animated")
	}
}

func TestIs3DRecognisesAdvertisedTokens(t *testing.T) {
	var model Illustration
	model.DownloadableResources.Available = []string{"png-low", "png", "png-hd", "fbx", "glb"}
	if !model.Is3D() {
		t.Error("fbx/glb should mark an item as 3D")
	}
	var flat Illustration
	flat.DownloadableResources.Available = []string{"png-hd", "svg"}
	if flat.Is3D() {
		t.Error("a flat illustration should not read as 3D")
	}
}

func TestBuildURLDropsEmptyValues(t *testing.T) {
	got := buildURL("https://example.test", "/search", map[string]string{
		"term": "rocket", "style": "", "amount": "10",
	})
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	q := u.Query()
	if q.Get("term") != "rocket" || q.Get("amount") != "10" {
		t.Fatalf("lost a value: %s", got)
	}
	if _, present := q["style"]; present {
		t.Fatalf("empty value should be dropped: %s", got)
	}
}

func TestBuildURLEscapesJSONMeta(t *testing.T) {
	got := buildURL("https://example.test", "/search", map[string]string{
		"meta": `{"mood":["casual"]}`,
	})
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if u.Query().Get("meta") != `{"mood":["casual"]}` {
		t.Fatalf("meta round-trip failed: %q", u.Query().Get("meta"))
	}
}

func TestAnimatedFilterParsing(t *testing.T) {
	for _, v := range []string{"y", "Y", "yes", "true", "animated"} {
		if !isAnimatedFilter(v) {
			t.Errorf("%q should mean animated-only", v)
		}
		if isStaticFilter(v) {
			t.Errorf("%q should not mean static-only", v)
		}
	}
	for _, v := range []string{"n", "no", "false", "static"} {
		if !isStaticFilter(v) {
			t.Errorf("%q should mean static-only", v)
		}
	}
	if isAnimatedFilter("") || isStaticFilter("") {
		t.Error("an empty filter should mean no restriction")
	}
}

func TestAPIErrorUnauthorized(t *testing.T) {
	if !(&APIError{Status: 401}).Unauthorized() {
		t.Error("401 should trigger a refresh")
	}
	if !(&APIError{Status: 403}).Unauthorized() {
		t.Error("403 should trigger a refresh")
	}
	if (&APIError{Status: 400}).Unauthorized() {
		t.Error("400 is a bad request, not an expired session")
	}
}
