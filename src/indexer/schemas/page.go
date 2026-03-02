package schemas

import "time"

type Page struct {
	NormalizedURL string
	HTML          string
	ContentType   string
	StatusCode    int
	LastCrawled   time.Time
}

func (p *Page) ToDocument() map[string]interface{} {
	return map[string]interface{}{
		"normalized_url": p.NormalizedURL,
		"html":           p.HTML,
		"content_type":   p.ContentType,
		"status_code":    p.StatusCode,
		"last_crawled":   p.LastCrawled,
	}
}
