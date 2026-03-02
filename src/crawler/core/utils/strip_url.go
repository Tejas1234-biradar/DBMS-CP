package utils

import (
	"fmt"
	"net/url"
	"strings"
)

func StripURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("Could not parse the URL [%w]", err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", fmt.Errorf("Invalid URL 'Scheme'")
	}
	if u.Host == "" {
		return "", fmt.Errorf("URL has no Field 'Host'")
	}
	strippedURL := u.Scheme + "://" + u.Host
	if u.Path != "" {
		trimmedPath := strings.TrimSuffix(u.Path, "/")
		strippedURL += trimmedPath
	}
	return strippedURL, nil

}