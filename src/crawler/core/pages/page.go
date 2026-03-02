package pages

import (
	"fmt"
	"time"
)

type Page struct {
	NormalizedURL string
	HTML          string
	ContentType   string
	LastCrawlTime time.Time
	StatusCode    int
}

func Constructor(normalizedUrl, html, contentype string, statuscode int) *Page {
	return &Page{
		NormalizedURL: normalizedUrl,
		HTML:          html,
		ContentType:   contentype,
		StatusCode:    statuscode,
		LastCrawlTime: time.Now(),
	}
}

func (p Page) ToString() string {
	htmlPreview := p.HTML

	// Truncate the HTML output
	if len(htmlPreview) > 15 {
		htmlPreview = htmlPreview[:15] + "..."
	}

	return fmt.Sprintf(
		"-------------------------------------------------\n"+
			"Normalized URL:    %-10s\n"+
			"HTML:              %-40s\n"+
			"Last Crawled:      %-30s\n"+
			"Status Code:       %-10d\n"+
			"Content Type:      %-20s\n"+
			"-------------------------------------------------\n",
		p.NormalizedURL, htmlPreview, p.LastCrawlTime.Format(time.RFC1123),
		p.StatusCode, p.ContentType,
	)
}

