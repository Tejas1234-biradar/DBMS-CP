package utils

// Tests taken from www.boot.dev - Web Crawler

import (
	"testing"
)

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name     string
		inputURL string
		expected bool
	}{
		{
			name:     "valid url",
			inputURL: "https://en.wikipedia.org/wiki/Mega_Man_X",
			expected: true,
		},
		{
			name:     "valid normalized url",
			inputURL: "en.wikipedia.org/wiki/Mega_Man_X",
			expected: true,
		},
		{
			name:     "invalid url (japanese)",
			inputURL: "https://ja.wikipedia.org/wiki/仮面ライダーシリーズ",
			expected: false,
		},
		{
			name:     "invalid url (japanese 2)",
			inputURL: "wuu.wikipedia.org/wiki/假面骑士系列",
			expected: false,
		},
		{
			name:     "invalud url (cyrillic)",
			inputURL: "https://uk.wikipedia.org/wiki/Камен_Райдер_(франшиза)",
			expected: false,
		},
		{
			name:     "invalid url (weird)",
			inputURL: "https://zh-classical.wikipedia.org/wiki/%E7%B6%AD%E5%9F%BA%E5%A4%A7%E5%85%B8:%E5%B8%82%E9%9B%86",
			expected: false,
		},
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := IsValidURL(tc.inputURL)
			if actual != tc.expected {
				t.Errorf("Test %v - '%s' FAIL: expected URL: %v, actual: %v", i, tc.name, tc.expected, actual)
			}
		})
	}
}