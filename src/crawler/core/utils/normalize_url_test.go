package utils

// Tests taken from www.boot.dev - Web Crawler

import (
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		inputURL string
		expected string
		wantErr  bool
	}{
		{
			name:     "remove https scheme",
			inputURL: "https://en.wikipedia.org/wiki/Mega_Man_X",
			expected: "en.wikipedia.org/wiki/Mega_Man_X",
			wantErr:  false,
		},
		{
			name:     "remove http scheme",
			inputURL: "http://en.wikipedia.org/wiki/Mega_Man_X",
			expected: "en.wikipedia.org/wiki/Mega_Man_X",
			wantErr:  false,
		},
		{
			name:     "remove trailing slash",
			inputURL: "http://en.wikipedia.org/wiki/Mega_Man_X/",
			expected: "en.wikipedia.org/wiki/Mega_Man_X",
			wantErr:  false,
		},
		{
			name:     "remove fragments",
			inputURL: "https://en.wikipedia.org/wiki/Mega_Man_X#Plot",
			expected: "en.wikipedia.org/wiki/Mega_Man_X",
			wantErr:  false,
		},
		{
			name:     "remove www.",
			inputURL: "https://www.mults.com/",
			expected: "mults.com",
			wantErr:  false,
		},
		{
			name:     "invalid scheme",
			inputURL: "htps://www.mults.com/",
			expected: "",
			wantErr:  true,
		},
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := NormalizeURL(tc.inputURL)
			if err != nil && !tc.wantErr {
				t.Errorf("Test %v - '%s' FAIL: unexpected error: %v", i, tc.name, err)
				return
			}

			if actual != tc.expected {
				t.Errorf("Test %v - '%s' FAIL: expected URL: %v, actual: %v", i, tc.name, tc.expected, actual)
			}
		})
	}
}