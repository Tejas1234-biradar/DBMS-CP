package utils

import (
	"fmt"
	"net/url"
	"strings"
)

func NormalizeURL(rawURL string) (string, error) {
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
	host := u.Host
	if strings.HasPrefix(host, "www.") {
		host = host[4:] //trim the www
	}
	normalizedURL := host
	if u.Path != "" {
		trimmedPath := strings.TrimSuffix(u.Path, "/")
		normalizedURL += trimmedPath
	}
	return normalizedURL, nil

}
