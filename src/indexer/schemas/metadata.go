package schemas

import "time"

type Metadata struct {
	ID          string
	Title       string
	Description string
	SummaryText string
	LastCrawled time.Time
	KeyWords    map[string]int
}

// metadata.to_dict equivalent
func (m *Metadata) ToDocument() map[string]interface{} {
	return map[string]interface{}{
		"_id":          m.ID,
		"title":        m.Title,
		"description":  m.Description,
		"summary_text": m.SummaryText,
		"last_crawled": m.LastCrawled,
		"keywords":     m.KeyWords,
	}
}
