package pages

import (
	"fmt"
	"strings"
)

// mainly used for pageRank strores outlinks and backlinks
type PageNode struct {
	NormalizedURL  string
	NormalizedURLs map[string]struct{}
}

func PageNodeConstructor(normalizedURL string) *PageNode {
	return &PageNode{
		NormalizedURL:  normalizedURL,
		NormalizedURLs: make(map[string]struct{}),
	}
}
func (b *PageNode) AddLink(newNormalizedLink string) {
	if b.NormalizedURLs == nil {
		b.NormalizedURLs = make(map[string]struct{})
	}
	b.NormalizedURLs[newNormalizedLink] = struct{}{}
}
func (b *PageNode) GetLinks() []string {
	var links []string
	for link := range b.NormalizedURLs {
		links = append(links, link)
	}
	return links
}

// toString from github
func (b *PageNode) ToString() string {
	var links []string

	for link := range b.NormalizedURLs {
		links = append(links, link)
	}

	return fmt.Sprintf(
		"\n-------------------------------------------------\n"+
			"%s has %d backlinks:\n"+
			"%v\n"+
			"-------------------------------------------------\n",
		b.NormalizedURL, len(b.NormalizedURLs), strings.Join(links, "\n"),
	)
}
