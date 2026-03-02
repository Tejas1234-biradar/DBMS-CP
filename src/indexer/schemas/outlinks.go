package schemas

type Outlinks struct {
	ID    string
	Links map[string]struct{}
}

// o.To_dict equivalent
func (o *Outlinks) ToDocument() map[string]interface{} {
	links := make([]string, 0, len(o.Links))
	for link := range o.Links {
		links = append(links, link)
	}
	return map[string]interface{}{
		"_id":   o.ID,
		"links": links,
	}
}
