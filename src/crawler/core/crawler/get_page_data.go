package crawler

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Returns Html,status code,content-type and Error Code
func getPageData(rawURL string) (string, int, string, error) {
	res, err := http.Get(rawURL)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode > 399 {
		return "", res.StatusCode, "", fmt.Errorf("HTTP error: %d %s", res.StatusCode, http.StatusText(res.StatusCode))
	}
	contentType := res.Header.Get("content-type")
	if !strings.Contains(contentType, "text/html") {

		return "", res.StatusCode, contentType, fmt.Errorf("invalid content type:(only Supports html websites) %s", contentType)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", res.StatusCode, "text/html", fmt.Errorf("failed to read response body: %w", err)
	}
	return string(body), res.StatusCode, contentType, nil
}
