package utils

import (
	"github.com/PuerkitoBio/goquery"
	"regexp"
	"strings"
)

// Compiled regex patterns — equiv of module-level re.compile()
var (
	NameSplitPattern  = regexp.MustCompile(`[-_./\s]+`)
	URLSplitPattern   = regexp.MustCompile(`[.,\-_\/+\:\(\)]`)
	NewlinePattern    = regexp.MustCompile(`\n+`)
	WhitespacePattern = regexp.MustCompile(`\s+`)
	BracketsPattern   = regexp.MustCompile(`\[.*?\]`)
)

type HTMLData struct {
	Title       string
	Description string
	SummaryText string
	Text        []string
	Language    string
}

// SplitName equiv of split_name()
func SplitName(filename string) []string {
	parts := NameSplitPattern.Split(strings.ToLower(filename), -1)
	var result []string
	for _, part := range parts {
		if part == "" {
			continue
		}
		if _, isFileType := FileTypes[part]; isFileType {
			continue
		}
		if strings.Contains(part, "px") {
			continue
		}
		result = append(result, part)
	}
	return result
}

// SplitURL equiv of split_url()
func SplitURL(url string) []string {
	parts := URLSplitPattern.Split(strings.ToLower(url), -1)
	var result []string
	for _, part := range parts {
		if part == "" {
			continue
		}
		if _, isPopular := PopularDomains[part]; isPopular {
			continue
		}
		result = append(result, part)
	}
	return result
}

// GetMetaContent equiv of get_meta_content()
func GetMetaContent(doc *goquery.Document, propertyValue, nameValue string) string {
	if propertyValue != "" {
		if content, exists := doc.Find("meta[property='" + propertyValue + "']").Attr("content"); exists {
			return content
		}
	}
	if nameValue != "" {
		if content, exists := doc.Find("meta[name='" + nameValue + "']").Attr("content"); exists {
			return content
		}
	}
	return ""
}

// TokenizeLargeText equiv of tokenize_large_text()
func TokenizeLargeText(text string) []string {
	const chunkSize = 10000
	var tokens []string
	for i := 0; i < len(text); i += chunkSize {
		end := i + chunkSize
		if end > len(text) {
			end = len(text)
		}
		tokens = append(tokens, Tokenize(text[i:end])...)
	}
	return tokens
}

// DetectLanguage equiv of detect_language()
func DetectLanguage(text string) string {
	sample := text
	if len(sample) > 1000 {
		sample = sample[:1000]
	}
	latinCount, totalCount := 0, 0
	for _, r := range sample {
		if r > 32 && r < 127 {
			totalCount++
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				latinCount++
			}
		}
	}
	if totalCount == 0 {
		return "unknown"
	}
	if float64(latinCount)/float64(totalCount) > 0.7 {
		return "en"
	}
	return "unknown"
}

// GetHTMLData equiv of get_html_data()
func GetHTMLData(html string) (*HTMLData, error) {
	// equiv of BeautifulSoup(html, "lxml", parse_only=SoupStrainer(["meta", "p", "title"]))
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	// equiv of meta_tags dict comprehension
	metaTags := make(map[string]string)
	doc.Find("meta").Each(func(_ int, s *goquery.Selection) {
		content, exists := s.Attr("content")
		if !exists || content == "" {
			return
		}
		if property, ok := s.Attr("property"); ok && property != "" {
			metaTags[property] = content
		} else if name, ok := s.Attr("name"); ok && name != "" {
			metaTags[name] = content
		}
	})

	// title = meta_tags.get("og:title") or meta_tags.get("title")
	title := metaTags["og:title"]
	if title == "" {
		title = metaTags["title"]
	}
	if title == "" {
		title = strings.TrimSpace(doc.Find("title").Text())
	}

	// description = meta_tags.get("og:description") or meta_tags.get("description")
	description := metaTags["og:description"]
	if description == "" {
		description = metaTags["description"]
	}

	// equiv of soup.select("p") + BRACKETS_PATTERN.sub("", p.get_text()).strip()
	var paragraphs []string
	doc.Find("p").Each(func(_ int, s *goquery.Selection) {
		text := BracketsPattern.ReplaceAllString(s.Text(), "")
		text = strings.TrimSpace(text)
		if text != "" {
			paragraphs = append(paragraphs, text)
		}
	})
	pageText := strings.Join(paragraphs, " ")

	// summary_text: first 500 words
	words := strings.Fields(pageText)
	summaryText := pageText
	if len(words) >= 500 {
		summaryText = strings.Join(words[:500], " ")
	}

	// equiv of tokenize_large_text() + stop word filtering
	// note: Tokenize() already filters stop words and non-alphanumeric — equiv of:
	// [word.lower() for word in tokens if word.lower() not in stop_words_set and word.lower().isalnum()]
	filteredText := TokenizeLargeText(pageText)

	// equiv of detect_language(summary_text)
	language := DetectLanguage(summaryText)

	return &HTMLData{
		Title:       title,
		Description: description,
		SummaryText: summaryText,
		Text:        filteredText,
		Language:    language,
	}, nil
}

