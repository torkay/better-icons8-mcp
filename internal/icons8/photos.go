package icons8

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// PhotoAsset is one rendition of a photo on the Moose CDN.
type PhotoAsset struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Type   string `json:"type,omitempty"`
	URL    string `json:"url"`
}

type PhotoTag struct {
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

type Photo struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Slug   string `json:"slug"`
	As     string `json:"as"` // "p" photo, "e" element, etc.
	Width  int    `json:"width"`
	Height int    `json:"height"`

	Thumb1x   *PhotoAsset `json:"thumb1x,omitempty"`
	Thumb2x   *PhotoAsset `json:"thumb2x,omitempty"`
	Preview1x *PhotoAsset `json:"preview1x,omitempty"`
	Preview2x *PhotoAsset `json:"preview2x,omitempty"`

	Tags          []PhotoTag `json:"tags,omitempty"`
	Photographers []struct {
		Title   string `json:"title"`
		Website string `json:"website"`
	} `json:"photographers,omitempty"`
	Purchased bool `json:"purchased,omitempty"`
}

type PhotoSearchResult struct {
	Total  int     `json:"total"`
	Images []Photo `json:"images"`
}

// photoListFields is the field projection the web app requests. Asking for
// everything is slower and returns editor-only blobs an agent cannot use.
const photoListFields = "id,title,slug,as,thumb1x,thumb2x,preview1x,preview2x,width,height,tags(title,slug),photographers(title,website),purchased"

const photoDetailFields = photoListFields + ",categories,models(title),associations(title,slug),createdAt"

// PhotoSearchOptions mirrors /photos filters.
type PhotoSearchOptions struct {
	Query   string
	Page    int
	PerPage int
	// Filter: "all", "transparent", "backgrounds", "elements".
	Filter string
	// SortBy: "rising", "new", "popular".
	SortBy     string
	CategoryID string
	TagID      string
	// Background: set to "transparent" to require a cut-out.
	Background string
	Type       string
	Locale     string
}

func (c *Client) SearchPhotos(ctx context.Context, o PhotoSearchOptions) (*PhotoSearchResult, error) {
	if o.Page <= 0 {
		o.Page = 1
	}
	if o.PerPage <= 0 {
		o.PerPage = 30
	}
	if o.PerPage > 100 {
		o.PerPage = 100
	}
	if o.Filter == "" {
		o.Filter = "all"
	}
	if o.SortBy == "" {
		o.SortBy = "rising"
	}
	if o.Locale == "" {
		o.Locale = c.cfg.Locale
	}
	q := map[string]string{
		"query": o.Query, "page": strconv.Itoa(o.Page), "per_page": strconv.Itoa(o.PerPage),
		"fields": photoListFields, "filter": o.Filter, "sort_by": o.SortBy,
		"locale": o.Locale, "category_id": o.CategoryID, "tag_id": o.TagID,
		"background": o.Background, "type": o.Type,
	}
	var out PhotoSearchResult
	if err := c.GetJSON(ctx, buildURL(HostPhotos, "/api/frontend/v1/images", q), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Photo(ctx context.Context, id string) (*Photo, error) {
	var out Photo
	u := buildURL(HostPhotos, "/api/frontend/v1/images/"+id, map[string]string{"fields": photoDetailFields})
	if err := c.GetJSON(ctx, u, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PhotoDownloadURL resolves a signed URL at the requested pixel size. Icons8
// requires both dimensions; passing the photo's native width/height yields the
// full-resolution original.
func (c *Client) PhotoDownloadURL(ctx context.Context, id string, width, height int) (*DownloadURL, error) {
	if width <= 0 || height <= 0 {
		p, err := c.Photo(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("resolve photo size for %s: %w", id, err)
		}
		width, height = p.Width, p.Height
		if width <= 0 || height <= 0 {
			return nil, fmt.Errorf("photo %s has no known dimensions; pass width and height", id)
		}
	}
	var out DownloadURL
	u := buildURL(HostPhotos, "/api/frontend/v1/images/"+id+"/download-url", map[string]string{
		"width": strconv.Itoa(width), "height": strconv.Itoa(height),
	})
	if err := c.GetJSON(ctx, u, &out); err != nil {
		return nil, err
	}
	if out.URL == "" {
		return nil, fmt.Errorf("icons8 returned no download url for photo %s", id)
	}
	if out.Width == 0 {
		out.Width, out.Height = width, height
	}
	return &out, nil
}

// PhotoCategories is the browse tree for photos.
func (c *Client) PhotoCategories(ctx context.Context) (any, error) {
	var out any
	if err := c.GetJSON(ctx, HostPhotos+"/api/frontend/v1/categories", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// PhotoAutocomplete suggests search terms that actually have results.
func (c *Client) PhotoAutocomplete(ctx context.Context, query string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	var out []struct {
		Name string `json:"name"`
	}
	u := buildURL(HostPhotos, "/api/frontend/v1/autocomplete", map[string]string{
		"query": query, "limit": strconv.Itoa(limit), "locale": c.cfg.Locale,
	})
	if err := c.GetJSON(ctx, u, &out); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(out))
	for _, o := range out {
		names = append(names, o.Name)
	}
	return names, nil
}

// Account is the signed-in identity plus what the licence covers.
type Account struct {
	ID           string   `json:"id"`
	Email        string   `json:"email"`
	PublicAPIKey string   `json:"publicApiKey"`
	Token        string   `json:"token"`
	History      []string `json:"searchHistory,omitempty"`
}

// Account fetches the current user. The response also carries a freshly minted
// JWT, which is how the session renews itself without touching a browser.
func (c *Client) Account(ctx context.Context) (*Account, error) {
	var out Account
	if err := c.GetJSON(ctx, HostIconAPI+"/user/v2", &out); err != nil {
		return nil, err
	}
	if out.Token != "" {
		if err := c.sess.SetToken(out.Token); err != nil {
			return nil, fmt.Errorf("adopt refreshed token: %w", err)
		}
	}
	return &out, nil
}

// RefreshToken renews the session JWT in place.
func (c *Client) RefreshToken(ctx context.Context) error {
	_, err := c.Account(ctx)
	return err
}

// FetchSigned downloads a signed CDN URL. These are pre-authorised, so they must
// not carry the session headers. Some CDNs reject a request that has both a
// signature and an Authorization header.
func (c *Client) FetchSigned(ctx context.Context, rawURL string) ([]byte, string, error) {
	if !strings.HasPrefix(rawURL, "http") {
		return nil, "", fmt.Errorf("not a URL: %q", rawURL)
	}
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, "", err
	}
	req, err := newPlainRequest(ctx, rawURL, c.cfg.UserAgent)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	body, err := readAllLimited(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", &APIError{Status: resp.StatusCode, URL: rawURL, Body: truncate(string(body), 300)}
	}
	return body, resp.Header.Get("Content-Type"), nil
}
