package icons8

import (
	"context"
	"io"
	"net/http"
)

// maxAssetBytes caps a single asset download. Icons8's largest assets are AEP
// projects and 3D source zips; 256 MiB is far above those and still bounded.
const maxAssetBytes = 256 << 20

// newPlainRequest builds a browser-shaped request with no session credentials,
// for pre-signed CDN URLs.
func newPlainRequest(ctx context.Context, rawURL, userAgent string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", "https://icons8.com/")
	return req, nil
}

func readAllLimited(r io.Reader) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, maxAssetBytes))
}
